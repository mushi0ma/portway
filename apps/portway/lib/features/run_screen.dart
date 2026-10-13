import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:markdown/markdown.dart' as md;
import 'package:url_launcher/url_launcher.dart';

import '../core/api.dart';
import '../core/format.dart';
import '../core/theme.dart';
import 'codex.dart';
import 'common.dart';
import 'compose_screen.dart' show effortLabel;
import 'home_screen.dart';

/// UI-10/13 (Codex profile): prompt, live progress, Markdown answer, meta.
class RunScreen extends StatefulWidget {
  const RunScreen({
    super.key,
    required this.repository,
    required this.id,
    this.bridge,
    this.nav,
    this.now,
  });
  final CodexRepository repository;
  final String id;
  final BridgeController? bridge;
  final ShellNav? nav;

  /// Clock for tests when no bridge controller is passed.
  final DateTime Function()? now;
  @override
  State<RunScreen> createState() => _RunScreenState();
}

class _RunScreenState extends State<RunScreen> with WidgetsBindingObserver {
  late final RunController controller;
  bool? _timelineOpen;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    controller = RunController(widget.repository, widget.id)..refresh();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) =>
      controller.setForeground(state == AppLifecycleState.resumed);
  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    controller.dispose();
    super.dispose();
  }

  DateTime get _now =>
      widget.bridge?.now() ?? widget.now?.call() ?? DateTime.now();

  @override
  Widget build(BuildContext context) => ListenableBuilder(
    listenable: controller,
    builder: (context, _) {
      final l = context.l, r = controller.run, c = context.colors;
      final open = _timelineOpen ?? (r == null || r.active || r.answer.isEmpty);
      return Scaffold(
        appBar: AppBar(
          title: Text(l.taskTitle),
          actions: [
            if (r != null && r.answer.isNotEmpty)
              IconButton(
                tooltip: l.copyAnswer,
                icon: const Icon(Icons.content_copy_rounded),
                onPressed: () => copyText(context, r.answer),
              ),
          ],
        ),
        body: SafeArea(
          child: PageBody(
            onRefresh: controller.refresh,
            children: [
              if (r == null && controller.busy)
                LinearProgressIndicator(semanticsLabel: l.loading),
              if (controller.failure case final f?) ...[
                ErrorNotice(f, retry: controller.refresh),
                const SizedBox(height: Space.md),
              ],
              if (r != null) ...[
                if (r.prompt.isNotEmpty || r.preview.isNotEmpty)
                  _PromptBubble(
                    text: r.prompt.isNotEmpty ? r.prompt : r.preview,
                    time: r.createdAt,
                    now: _now,
                  ),
                const SizedBox(height: Space.lg),
                Wrap(
                  spacing: Space.sm,
                  runSpacing: Space.sm,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: [
                    Semantics(
                      liveRegion: true,
                      child: StatusBadge(
                        stateLabel(context, r.state),
                        tone: stateTone(r.state),
                        icon: stateIcon(r.state),
                      ),
                    ),
                    Text(
                      [
                        providerTitle(r.provider ?? 'codex'),
                        ?r.model,
                        if (r.effort != null) effortLabel(context, r.effort!),
                      ].join(' · '),
                      style: context.text.bodySmall,
                    ),
                  ],
                ),
                if (r.state == 'failed') ...[
                  const SizedBox(height: Space.md),
                  _failure(r),
                ],
                if (r.state == 'stopped') ...[
                  const SizedBox(height: Space.md),
                  Notice(
                    tone: Tone.warning,
                    title: l.stoppedTitle,
                    message: l.stoppedHint,
                  ),
                ],
                const SizedBox(height: Space.lg),
                _timeline(r, open),
                if (r.answer.isNotEmpty) ...[
                  const SizedBox(height: Space.lg),
                  _answer(r),
                ] else if (r.active) ...[
                  const SizedBox(height: Space.lg),
                  Text(l.waitingResult, style: context.text.bodySmall),
                ],
                const SizedBox(height: Space.lg),
                _details(r),
              ],
            ],
          ),
        ),
        bottomNavigationBar: r?.active == true
            ? SafeArea(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(
                    Space.lg,
                    Space.sm,
                    Space.lg,
                    Space.md,
                  ),
                  child: OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(
                      foregroundColor: c.danger,
                      side: BorderSide(color: c.danger),
                    ),
                    onPressed: controller.stopping
                        ? null
                        : () async {
                            if (await confirmAction(
                              context,
                              l.stopTitle,
                              l.stopHint,
                              l.stop,
                              destructive: true,
                            )) {
                              await controller.stop();
                            }
                          },
                    icon: controller.stopping
                        ? SizedBox.square(
                            dimension: 18,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              semanticsLabel: l.loading,
                            ),
                          )
                        : const Icon(Icons.stop_circle_outlined, size: 20),
                    label: Text(l.stop),
                  ),
                ),
              )
            : null,
      );
    },
  );

  Widget _failure(CodexRun r) {
    final l = context.l;
    String? code = r.errorCode;
    for (final e in controller.events.reversed) {
      if (e['type'] == 'failed' && e['code'] is String) {
        code ??= e['code'] as String;
        break;
      }
    }
    if (code == 'AI_LIMIT_EXHAUSTED') {
      final p = widget.bridge?.status?.provider(r.provider ?? 'codex');
      final at = p?.usage?.availableAt;
      return Notice(
        icon: Icons.hourglass_bottom_rounded,
        title: l.runLimit,
        message: at == null
            ? l.limitNoReset
            : l.availableAgain(formatMoment(at, _now), formatUntil(at, _now)),
        actions: [
          if (widget.nav != null)
            TextButton(
              onPressed: () {
                Navigator.of(context).pop();
                widget.nav!.selectTab(ShellNav.accounts);
              },
              child: Text(l.switchAccount),
            ),
        ],
      );
    }
    return Notice(title: runFailureLabel(context, code), message: l.failedHint);
  }

  Widget _timeline(CodexRun r, bool open) {
    final l = context.l, c = context.colors, events = controller.events;
    return AppCard(
      padding: EdgeInsets.zero,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          InkWell(
            onTap: () => setState(() => _timelineOpen = !open),
            child: ConstrainedBox(
              constraints: const BoxConstraints(minHeight: 52),
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: Space.lg,
                  vertical: Space.md,
                ),
                child: Row(
                  children: [
                    Icon(Icons.timeline_rounded, size: 20, color: c.muted),
                    const SizedBox(width: Space.md),
                    Expanded(
                      child: Text(
                        l.progressCount(events.length),
                        style: context.text.titleSmall,
                      ),
                    ),
                    Icon(
                      open
                          ? Icons.expand_less_rounded
                          : Icons.expand_more_rounded,
                      size: 22,
                      color: c.muted,
                    ),
                  ],
                ),
              ),
            ),
          ),
          if (r.active)
            LinearProgressIndicator(minHeight: 2, semanticsLabel: l.running)
          else if (open)
            const Divider(),
          if (open) ...[
            Padding(
              padding: const EdgeInsets.fromLTRB(
                Space.lg,
                Space.md,
                Space.lg,
                Space.md,
              ),
              child: events.isEmpty
                  ? Text(
                      r.active ? l.emptyEventsActive : l.emptyEvents,
                      style: context.text.bodySmall,
                    )
                  : Column(
                      children: [
                        for (var i = 0; i < events.length; i++)
                          _EventRow(
                            event: events[i],
                            last: i == events.length - 1,
                          ),
                      ],
                    ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _answer(CodexRun r) {
    final l = context.l, c = context.colors, theme = Theme.of(context);
    final base = theme.textTheme.bodyLarge!;
    final sheet = MarkdownStyleSheet.fromTheme(theme).copyWith(
      p: base,
      h1: theme.textTheme.titleLarge,
      h2: theme.textTheme.titleMedium?.copyWith(fontSize: 18),
      h3: theme.textTheme.titleMedium,
      listBullet: base,
      code: base.copyWith(
        fontFamily: monoFamily,
        fontSize: 14,
        backgroundColor: c.raised,
        color: c.text,
      ),
      a: base.copyWith(color: c.accent, decoration: TextDecoration.underline),
      blockquoteDecoration: BoxDecoration(
        color: c.raised,
        borderRadius: BorderRadius.circular(Radii.small),
        border: Border(left: BorderSide(color: c.accent, width: 3)),
      ),
      tableBorder: TableBorder.all(color: c.divider),
      tableHead: base.copyWith(fontWeight: FontWeight.w600),
      tableBody: theme.textTheme.bodyMedium,
      horizontalRuleDecoration: BoxDecoration(
        border: Border(top: BorderSide(color: c.divider)),
      ),
      blockSpacing: 12,
    );
    return AppCard(
      padding: EdgeInsets.zero,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(
              Space.lg,
              Space.sm,
              Space.xs,
              Space.xs,
            ),
            child: Row(
              children: [
                Icon(Icons.smart_toy_outlined, size: 20, color: c.accent),
                const SizedBox(width: Space.md),
                Expanded(
                  child: Text(
                    l.answerFrom(providerTitle(r.provider ?? 'codex')),
                    style: context.text.titleSmall,
                  ),
                ),
                TextButton.icon(
                  onPressed: () => copyText(context, r.answer),
                  icon: const Icon(Icons.content_copy_rounded, size: 18),
                  label: Text(l.copy),
                ),
              ],
            ),
          ),
          const Divider(),
          Padding(
            padding: const EdgeInsets.all(Space.lg),
            child: MarkdownBody(
              data: r.answer,
              selectable: true,
              styleSheet: sheet,
              extensionSet: md.ExtensionSet.gitHubFlavored,
              builders: {'pre': _CodeBlockBuilder()},
              onTapLink: (text, href, title) {
                final uri = href == null ? null : Uri.tryParse(href);
                if (uri != null && uri.scheme == 'https') {
                  launchUrl(uri, mode: LaunchMode.externalApplication);
                }
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _details(CodexRun r) {
    final l = context.l, now = _now;
    final t = r.tokens;
    return AppCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(l.details, style: context.text.titleSmall),
          const SizedBox(height: Space.sm),
          MetaRow(
            icon: Icons.smart_toy_outlined,
            label: l.executor,
            value: [
              providerTitle(r.provider ?? 'codex'),
              r.model ?? l.defaultModel,
            ].join(' · '),
          ),
          if (r.accountLabel != null)
            MetaRow(
              icon: Icons.account_circle_outlined,
              label: l.account,
              value: r.accountLabel!,
            ),
          if (t != null)
            MetaRow(
              icon: Icons.data_usage_rounded,
              label: l.tokens,
              value: l.tokensValue(
                formatCount(t.input ?? 0),
                formatCount(t.output ?? 0),
              ),
            ),
          if (r.duration != null)
            MetaRow(
              icon: Icons.timer_outlined,
              label: l.duration,
              value: formatElapsed(r.duration!),
            ),
          if (r.createdAt != null)
            MetaRow(
              icon: Icons.schedule_rounded,
              label: l.created,
              value: formatMoment(r.createdAt!, now),
            ),
          MetaRow(
            icon: Icons.folder_outlined,
            label: l.workspace,
            value: l.workspaceValue(
              r.data['workspace'] as String? ?? 'portway',
            ),
          ),
          MetaRow(
            icon: Icons.tag_rounded,
            label: l.runId,
            value: r.id,
            mono: true,
            copyable: true,
          ),
        ],
      ),
    );
  }
}

class _PromptBubble extends StatelessWidget {
  const _PromptBubble({
    required this.text,
    required this.time,
    required this.now,
  });
  final String text;
  final DateTime? time;
  final DateTime now;
  @override
  Widget build(BuildContext context) {
    final c = context.colors;
    return Align(
      alignment: Alignment.centerRight,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 520),
        child: FractionallySizedBox(
          widthFactor: 0.92,
          alignment: Alignment.centerRight,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: Space.lg,
                  vertical: Space.md,
                ),
                decoration: BoxDecoration(
                  color: c.accentContainer,
                  borderRadius: const BorderRadius.only(
                    topLeft: Radius.circular(16),
                    topRight: Radius.circular(16),
                    bottomLeft: Radius.circular(16),
                    bottomRight: Radius.circular(4),
                  ),
                ),
                child: SelectableText(text, style: context.text.bodyLarge),
              ),
              if (time != null) ...[
                const SizedBox(height: Space.xs),
                Text(formatMoment(time!, now), style: context.text.bodySmall),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _EventRow extends StatelessWidget {
  const _EventRow({required this.event, required this.last});
  final Json event;
  final bool last;
  @override
  Widget build(BuildContext context) {
    final c = context.colors, l = context.l;
    final type = event['type'] as String? ?? '';
    final (IconData icon, Color color, String title) = switch (type) {
      'started' => (Icons.play_arrow_rounded, c.accent, l.eventStarted),
      'completed' => (Icons.check_rounded, c.accent, l.eventCompleted),
      'failed' => (Icons.close_rounded, c.danger, l.eventFailed),
      'stopped' => (Icons.stop_rounded, c.warning, l.eventStopped),
      _ => (Icons.circle, c.muted, l.eventProgress),
    };
    final code = event['code'] as String?;
    final text =
        event['text'] as String? ??
        (code == null ? null : runFailureLabel(context, code));
    return IntrinsicHeight(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 20,
            child: Column(
              children: [
                const SizedBox(height: 2),
                Icon(icon, size: type == 'progress' ? 8 : 16, color: color),
                if (!last)
                  Expanded(
                    child: Container(
                      width: 1,
                      margin: const EdgeInsets.symmetric(vertical: 4),
                      color: c.divider,
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(width: Space.md),
          Expanded(
            child: Padding(
              padding: EdgeInsets.only(bottom: last ? 0 : Space.md),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    style: context.text.labelMedium?.copyWith(
                      color: color == c.muted ? c.secondary : color,
                    ),
                  ),
                  if (text != null && text.isNotEmpty) ...[
                    const SizedBox(height: 2),
                    SelectableText(
                      text,
                      style: context.text.bodyMedium?.copyWith(
                        color: c.secondary,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// Code blocks: mono, horizontally scrollable, with their own copy button.
class _CodeBlockBuilder extends MarkdownElementBuilder {
  @override
  bool isBlockElement() => true;

  @override
  Widget? visitElementAfterWithContext(
    BuildContext context,
    md.Element element,
    TextStyle? preferredStyle,
    TextStyle? parentStyle,
  ) {
    final code = element.textContent.replaceAll(RegExp(r'\n$'), '');
    String? language;
    final child = element.children?.firstOrNull;
    if (child is md.Element) {
      final cls = child.attributes['class'];
      if (cls != null && cls.startsWith('language-')) {
        language = cls.substring(9);
      }
    }
    return _CodeBlock(code: code, language: language);
  }
}

class _CodeBlock extends StatelessWidget {
  const _CodeBlock({required this.code, this.language});
  final String code;
  final String? language;
  @override
  Widget build(BuildContext context) {
    final c = context.colors;
    return Container(
      margin: const EdgeInsets.symmetric(vertical: Space.xs),
      decoration: BoxDecoration(
        color: c.raised,
        borderRadius: BorderRadius.circular(Radii.small),
        border: Border.all(color: c.divider),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              const SizedBox(width: Space.md),
              Expanded(
                child: Text(
                  language ?? context.l.code,
                  style: context.text.labelSmall?.copyWith(color: c.muted),
                ),
              ),
              IconButton(
                tooltip: context.l.copyCode,
                icon: const Icon(Icons.content_copy_rounded, size: 16),
                onPressed: () => copyText(context, code),
              ),
            ],
          ),
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.fromLTRB(Space.md, 0, Space.md, Space.md),
            child: SelectableText(
              code,
              style: TextStyle(
                fontFamily: monoFamily,
                fontSize: 13,
                height: 1.45,
                color: c.text,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
