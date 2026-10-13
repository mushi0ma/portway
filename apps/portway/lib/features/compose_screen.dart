import 'dart:convert';

import 'package:flutter/material.dart';

import '../core/api.dart';
import '../core/format.dart';
import '../core/theme.dart';
import 'codex.dart';
import 'common.dart';
import 'home_screen.dart';

const effortOrder = ['low', 'medium', 'high', 'xhigh', 'max'];

String effortLabel(BuildContext context, String effort) {
  final l = context.l;
  return switch (effort) {
    'low' => l.effortLow,
    'medium' => l.effortMedium,
    'high' => l.effortHigh,
    'xhigh' => l.effortXhigh,
    'max' => l.effortMax,
    _ => effort,
  };
}

/// Why sending is blocked right now, with an optional next action.
class SendBlock {
  const SendBlock(
    this.title,
    this.message, {
    this.tone = Tone.warning,
    this.actions = const [],
  });
  final String title, message;
  final Tone tone;
  final List<Widget> actions;
}

/// UI-09 (Codex profile): chat-like composer with provider, model and effort.
class ComposeScreen extends StatefulWidget {
  const ComposeScreen({
    super.key,
    required this.composer,
    required this.bridge,
    required this.nav,
  });
  final TaskComposer composer;
  final BridgeController bridge;
  final ShellNav nav;
  @override
  State<ComposeScreen> createState() => _ComposeScreenState();
}

class _ComposeScreenState extends State<ComposeScreen> {
  final _text = TextEditingController();
  final _focus = FocusNode();
  String? _model, _effort, _modelProvider;

  @override
  void initState() {
    super.initState();
    _text.text = widget.composer.pendingBody?['prompt'] as String? ?? '';
    _text.addListener(() => setState(() {}));
    widget.composer.clearFailure();
    widget.bridge.refresh();
  }

  @override
  void dispose() {
    _text.dispose();
    _focus.dispose();
    super.dispose();
  }

  BridgeStatus? get _status => widget.bridge.status;
  String get _providerId => widget.bridge.preferredProvider;
  AiProvider? get _provider => _status?.provider(_providerId);

  /// Selected model (falls back to the provider default).
  AiModel? _selectedModel(AiProvider? p) {
    if (p == null) return null;
    if (_modelProvider != p.id) {
      _modelProvider = p.id;
      _model = null;
      _effort = null;
    }
    for (final m in p.models) {
      if (m.id == _model) return m;
    }
    return p.defaultModel;
  }

  String? _selectedEffort(AiModel? m) {
    if (m == null || m.efforts.isEmpty) return null;
    if (_effort != null && m.efforts.contains(_effort)) return _effort;
    if (m.defaultEffort != null && m.efforts.contains(m.defaultEffort)) {
      return m.defaultEffort;
    }
    return m.efforts.first;
  }

  SendBlock? _block() {
    final l = context.l, s = _status, p = _provider, now = widget.bridge.now();
    if (widget.composer.pending != null) return null;
    if (s == null) {
      return widget.bridge.failure == null
          ? null
          : SendBlock(
              l.blockNoServerTitle,
              errorLabel(context, widget.bridge.failure!),
              tone: Tone.danger,
            );
    }
    if (!s.connected) {
      return SendBlock(
        l.blockOfflineTitle,
        l.blockOffline,
        actions: [
          TextButton(
            onPressed: () => widget.bridge.refresh(usage: true),
            child: Text(l.refresh),
          ),
        ],
      );
    }
    if (!s.hasProviders || p == null) return null;
    if (!p.installed) {
      return SendBlock(
        l.blockNotInstalledTitle(p.name),
        l.blockNotInstalled,
        actions: _switchActions(p),
      );
    }
    if (!p.hasLoggedInAccount) {
      return SendBlock(
        l.blockNoAccountTitle(p.name),
        l.blockNoAccount(providerService(p.id)),
        actions: [
          TextButton(onPressed: _openAccounts, child: Text(l.connectAccount)),
          ..._switchActions(p),
        ],
      );
    }
    if (p.exhausted) {
      final at = p.usage?.availableAt;
      return SendBlock(
        l.limitTitle(p.name),
        at == null
            ? l.limitNoReset
            : l.availableAgain(formatMoment(at, now), formatUntil(at, now)),
        tone: Tone.danger,
        actions: [
          TextButton(onPressed: _openAccounts, child: Text(l.switchAccount)),
          ..._switchActions(p),
        ],
      );
    }
    return null;
  }

  List<Widget> _switchActions(AiProvider current) => [
    for (final o in _status?.providers ?? const <AiProvider>[])
      if (o.id != current.id &&
          o.installed &&
          o.hasLoggedInAccount &&
          !o.exhausted)
        TextButton(
          onPressed: () => widget.bridge.setPreferredProvider(o.id),
          child: Text(context.l.switchTo(o.name)),
        ),
  ];

  void _openAccounts() {
    Navigator.of(context).pop();
    widget.nav.selectTab(ShellNav.accounts);
  }

  Future<void> _send() async {
    FocusManager.instance.primaryFocus?.unfocus();
    final s = _status, p = _provider;
    String? provider, model, effort;
    // Older servers reject unknown fields: only send the choice when the
    // bridge has reported providers (contract v2).
    if (s != null && s.hasProviders && p != null) {
      final m = _selectedModel(p);
      provider = p.id;
      model = m == null || m.id == 'default' ? null : m.id;
      effort = _selectedEffort(m);
    }
    final id = await widget.composer.send(
      _text.text,
      provider: provider,
      model: model,
      effort: effort,
    );
    if (mounted && id != null) Navigator.pop(context, id);
  }

  Future<void> _pickModel(AiProvider p) async {
    final current = _selectedModel(p);
    final picked = await showModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      builder: (sheet) => SafeArea(
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.sizeOf(sheet).height * 0.75,
          ),
          child: ListView(
            shrinkWrap: true,
            padding: const EdgeInsets.only(bottom: Space.lg),
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(
                  Space.xl,
                  0,
                  Space.xl,
                  Space.sm,
                ),
                child: Text(
                  sheet.l.chooseModel(p.name),
                  style: sheet.text.titleMedium,
                ),
              ),
              RadioGroup<String>(
                groupValue: current?.id,
                onChanged: (v) => Navigator.pop(sheet, v),
                child: Column(
                  children: [
                    for (final m in p.models)
                      RadioListTile<String>(
                        value: m.id,
                        contentPadding: const EdgeInsets.symmetric(
                          horizontal: Space.lg,
                        ),
                        title: Row(
                          children: [
                            Flexible(child: Text(m.name)),
                            if (m.isDefault &&
                                m.name != sheet.l.defaultModel) ...[
                              const SizedBox(width: Space.sm),
                              StatusBadge(
                                sheet.l.defaultModel,
                                tone: Tone.accent,
                              ),
                            ],
                          ],
                        ),
                        subtitle: m.description == null
                            ? null
                            : Text(
                                m.description!,
                                maxLines: 2,
                                overflow: TextOverflow.ellipsis,
                              ),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
    if (picked != null && mounted) {
      setState(() {
        _model = picked;
        _effort = null;
      });
    }
  }

  @override
  Widget build(BuildContext context) => ListenableBuilder(
    listenable: Listenable.merge([widget.composer, widget.bridge]),
    builder: (context, _) {
      final l = context.l, c = context.colors, composer = widget.composer;
      final pending = composer.pending != null;
      final s = _status, p = _provider;
      final model = _selectedModel(p);
      final effort = _selectedEffort(model);
      final block = _block();
      final tooLong = utf8.encode(_text.text).length > 16384;
      final canSend =
          !composer.busy &&
          (pending ||
              (block == null && _text.text.trim().isNotEmpty && !tooLong));
      return Scaffold(
        appBar: AppBar(title: Text(l.newTask)),
        body: SafeArea(
          child: Column(
            children: [
              Expanded(
                child: PageBody(
                  children: [
                    if (s != null && s.hasProviders && !pending) ...[
                      Text(
                        l.executor,
                        style: context.text.titleSmall?.copyWith(
                          color: c.secondary,
                        ),
                      ),
                      const SizedBox(height: Space.sm),
                      SegmentedButton<String>(
                        showSelectedIcon: false,
                        segments: [
                          for (final x in s.providers)
                            ButtonSegment(
                              value: x.id,
                              label: Text(
                                x.name,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                              icon: Icon(
                                x.exhausted ||
                                        !x.installed ||
                                        !x.hasLoggedInAccount
                                    ? Icons.block_rounded
                                    : Icons.smart_toy_outlined,
                                size: 18,
                              ),
                            ),
                        ],
                        selected: {_providerId},
                        onSelectionChanged: (v) =>
                            widget.bridge.setPreferredProvider(v.first),
                      ),
                      if (p != null) ...[
                        const SizedBox(height: Space.sm),
                        Text(
                          accountLine(p, context),
                          style: context.text.bodySmall,
                        ),
                      ],
                      if (p != null && p.models.isNotEmpty) ...[
                        const SizedBox(height: Space.lg),
                        _OptionTile(
                          icon: Icons.tune_rounded,
                          label: l.model,
                          value: model?.name ?? l.defaultModel,
                          onTap: () => _pickModel(p),
                        ),
                      ],
                      if (model != null && model.efforts.isNotEmpty) ...[
                        const SizedBox(height: Space.lg),
                        Text(
                          l.effort,
                          style: context.text.titleSmall?.copyWith(
                            color: c.secondary,
                          ),
                        ),
                        const SizedBox(height: Space.sm),
                        Wrap(
                          spacing: Space.sm,
                          runSpacing: Space.sm,
                          children: [
                            for (final e in effortOrder.where(
                              model.efforts.contains,
                            ))
                              ChoiceChip(
                                label: Text(effortLabel(context, e)),
                                selected: e == effort,
                                onSelected: (_) => setState(() => _effort = e),
                              ),
                          ],
                        ),
                      ],
                    ] else if (!pending) ...[
                      AppCard(
                        child: Row(
                          children: [
                            Icon(
                              Icons.smart_toy_outlined,
                              size: 20,
                              color: c.accent,
                            ),
                            const SizedBox(width: Space.md),
                            Expanded(
                              child: Text(
                                l.codexDefault,
                                style: context.text.bodyMedium,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                    const SizedBox(height: Space.lg),
                    Row(
                      children: [
                        Icon(Icons.folder_outlined, size: 18, color: c.muted),
                        const SizedBox(width: Space.sm),
                        Expanded(
                          child: Text(
                            l.workspaceHint(s?.workspace ?? 'portway'),
                            style: context.text.bodySmall,
                          ),
                        ),
                      ],
                    ),
                    if (block != null) ...[
                      const SizedBox(height: Space.lg),
                      Notice(
                        tone: block.tone,
                        title: block.title,
                        message: block.message,
                        actions: block.actions,
                      ),
                    ],
                    if (composer.failure case final f?) ...[
                      const SizedBox(height: Space.lg),
                      _failureNotice(f),
                    ],
                    if (pending && !composer.busy) ...[
                      const SizedBox(height: Space.lg),
                      Notice(
                        tone: Tone.warning,
                        title: l.pendingTitle,
                        message: l.requestUnknown,
                        actions: [
                          TextButton(
                            onPressed: () async {
                              await composer.discardPending();
                              _text.clear();
                            },
                            child: Text(l.discardPending),
                          ),
                        ],
                      ),
                    ],
                    if (!pending && _text.text.isEmpty) ...[
                      const SizedBox(height: Space.xl),
                      Text(
                        l.suggestions,
                        style: context.text.titleSmall?.copyWith(
                          color: c.secondary,
                        ),
                      ),
                      const SizedBox(height: Space.sm),
                      for (final hint in [
                        l.suggestion1,
                        l.suggestion2,
                        l.suggestion3,
                      ])
                        Padding(
                          padding: const EdgeInsets.only(bottom: Space.sm),
                          child: AppCard(
                            padding: const EdgeInsets.symmetric(
                              horizontal: Space.lg,
                              vertical: Space.md,
                            ),
                            onTap: () {
                              _text.text = hint;
                              _focus.requestFocus();
                            },
                            child: Row(
                              children: [
                                Icon(
                                  Icons.lightbulb_outline_rounded,
                                  size: 18,
                                  color: c.accent,
                                ),
                                const SizedBox(width: Space.md),
                                Expanded(
                                  child: Text(
                                    hint,
                                    style: context.text.bodyMedium,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                    ],
                  ],
                ),
              ),
              _ComposerBar(
                controller: _text,
                focus: _focus,
                readOnly: pending || composer.busy,
                busy: composer.busy,
                canSend: canSend,
                retry: pending,
                tooLong: tooLong,
                onSend: _send,
              ),
            ],
          ),
        ),
      );
    },
  );

  Widget _failureNotice(ApiFailure f) {
    final l = context.l, now = widget.bridge.now();
    final p = _provider;
    switch (f.code) {
      case 'AI_LIMIT_EXHAUSTED':
        final resetRaw = f.details?['reset_at'];
        final at = resetRaw is String
            ? DateTime.tryParse(resetRaw)
            : p?.usage?.availableAt;
        final name = providerTitle(
          f.details?['provider'] as String? ?? _providerId,
        );
        return Notice(
          title: l.limitTitle(name),
          message: at == null
              ? l.limitNoReset
              : l.availableAgain(formatMoment(at, now), formatUntil(at, now)),
          actions: [
            TextButton(onPressed: _openAccounts, child: Text(l.switchAccount)),
            if (p != null) ..._switchActions(p),
          ],
        );
      case 'AI_ACCOUNT_REQUIRED':
        return Notice(
          title: errorLabel(context, f),
          actions: [
            TextButton(onPressed: _openAccounts, child: Text(l.connectAccount)),
          ],
        );
      case 'AI_PROVIDER_UNAVAILABLE':
        return Notice(
          title: errorLabel(context, f),
          actions: p == null ? const [] : _switchActions(p),
        );
      case 'CODEX_BUSY':
        return Notice(
          tone: Tone.warning,
          title: errorLabel(context, f),
          actions: [
            TextButton(
              onPressed: () {
                Navigator.of(context).pop();
                widget.nav.selectTab(ShellNav.tasks);
              },
              child: Text(l.openTasks),
            ),
          ],
        );
      default:
        return ErrorNotice(f);
    }
  }
}

class _OptionTile extends StatelessWidget {
  const _OptionTile({
    required this.icon,
    required this.label,
    required this.value,
    required this.onTap,
  });
  final IconData icon;
  final String label, value;
  final VoidCallback onTap;
  @override
  Widget build(BuildContext context) {
    final c = context.colors;
    return AppCard(
      onTap: onTap,
      padding: const EdgeInsets.symmetric(
        horizontal: Space.lg,
        vertical: Space.md,
      ),
      child: Row(
        children: [
          Icon(icon, size: 20, color: c.muted),
          const SizedBox(width: Space.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, style: context.text.bodySmall),
                Text(value, style: context.text.bodyLarge),
              ],
            ),
          ),
          Icon(Icons.unfold_more_rounded, size: 20, color: c.muted),
        ],
      ),
    );
  }
}

class _ComposerBar extends StatelessWidget {
  const _ComposerBar({
    required this.controller,
    required this.focus,
    required this.readOnly,
    required this.busy,
    required this.canSend,
    required this.retry,
    required this.tooLong,
    required this.onSend,
  });
  final TextEditingController controller;
  final FocusNode focus;
  final bool readOnly, busy, canSend, retry, tooLong;
  final VoidCallback onSend;
  @override
  Widget build(BuildContext context) {
    final l = context.l, c = context.colors;
    return Material(
      color: c.surface,
      child: Container(
        decoration: BoxDecoration(
          border: Border(top: BorderSide(color: c.divider)),
        ),
        padding: const EdgeInsets.fromLTRB(
          Space.lg,
          Space.md,
          Space.md,
          Space.md,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (tooLong)
              Padding(
                padding: const EdgeInsets.only(bottom: Space.sm),
                child: Text(
                  l.promptTooLong,
                  style: context.text.bodySmall?.copyWith(color: c.danger),
                ),
              ),
            Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Expanded(
                  child: TextField(
                    controller: controller,
                    focusNode: focus,
                    readOnly: readOnly,
                    minLines: 1,
                    maxLines: 6,
                    keyboardType: TextInputType.multiline,
                    textCapitalization: TextCapitalization.sentences,
                    decoration: InputDecoration(
                      hintText: l.promptHint,
                      labelText: null,
                      fillColor: c.background,
                      contentPadding: const EdgeInsets.symmetric(
                        horizontal: 16,
                        vertical: 14,
                      ),
                    ),
                  ),
                ),
                if (!retry) ...[
                  const SizedBox(width: Space.sm),
                  Padding(
                    padding: const EdgeInsets.only(bottom: 2),
                    child: busy
                        ? SizedBox.square(
                            dimension: 48,
                            child: Padding(
                              padding: const EdgeInsets.all(12),
                              child: CircularProgressIndicator(
                                strokeWidth: 2.5,
                                semanticsLabel: l.loading,
                              ),
                            ),
                          )
                        : IconButton.filled(
                            tooltip: l.send,
                            onPressed: canSend ? onSend : null,
                            style: IconButton.styleFrom(
                              minimumSize: const Size(52, 52),
                              disabledBackgroundColor: c.raised,
                              disabledForegroundColor: c.disabled,
                              shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(
                                  Radii.medium,
                                ),
                              ),
                            ),
                            icon: const Icon(
                              Icons.arrow_upward_rounded,
                              size: 24,
                            ),
                          ),
                  ),
                ],
              ],
            ),
            if (retry) ...[
              const SizedBox(height: Space.sm),
              FilledButton.icon(
                onPressed: canSend ? onSend : null,
                icon: busy
                    ? SizedBox.square(
                        dimension: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          semanticsLabel: l.loading,
                        ),
                      )
                    : const Icon(Icons.replay_rounded, size: 20),
                label: Text(l.retrySame),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
