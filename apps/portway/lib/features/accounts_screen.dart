import 'dart:async';

import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../core/api.dart';
import '../core/format.dart';
import '../core/theme.dart';
import 'codex.dart';
import 'common.dart';
import 'home_screen.dart';

/// "Аккаунты ИИ": per provider accounts, usage, activate/remove and login.
class AccountsTab extends StatefulWidget {
  const AccountsTab({super.key, required this.bridge});
  final BridgeController bridge;
  @override
  State<AccountsTab> createState() => _AccountsTabState();
}

class _AccountsTabState extends State<AccountsTab> {
  StreamSubscription<BridgeCommand>? _results;

  @override
  void initState() {
    super.initState();
    _results = widget.bridge.commandResults.listen(_onResult);
  }

  @override
  void dispose() {
    _results?.cancel();
    super.dispose();
  }

  void _onResult(BridgeCommand c) {
    if (!mounted) return;
    final l = context.l;
    final text = switch ((c.kind, c.state)) {
      ('account_activate', 'done') => l.accountActivated,
      ('account_remove', 'done') => l.accountRemoved,
      ('account_activate' || 'account_remove', 'failed') => commandErrorLabel(
        context,
        c.errorCode,
      ),
      _ => null,
    };
    if (text == null) return;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(text)));
  }

  void _show(ApiFailure? f) {
    if (f == null || !mounted) return;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(errorLabel(context, f))));
  }

  Future<void> _connect(AiProvider p) async {
    final l = context.l;
    final method = await showModalBottomSheet<String>(
      context: context,
      builder: (sheet) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(Space.lg, 0, Space.lg, Space.lg),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                l.connectAccountTitle(providerService(p.id)),
                style: sheet.text.titleMedium,
              ),
              const SizedBox(height: Space.xs),
              Text(l.connectAccountHint, style: sheet.text.bodySmall),
              const SizedBox(height: Space.lg),
              _MethodTile(
                icon: Icons.smartphone_rounded,
                title: l.methodPhone,
                subtitle: p.id == 'codex'
                    ? l.methodPhoneCodex
                    : l.methodPhoneClaude,
                onTap: () => Navigator.pop(sheet, 'phone'),
              ),
              const SizedBox(height: Space.sm),
              _MethodTile(
                icon: Icons.laptop_mac_rounded,
                title: l.methodComputer,
                subtitle: l.methodComputerHint,
                onTap: () => Navigator.pop(sheet, 'computer'),
              ),
            ],
          ),
        ),
      ),
    );
    if (method == null || !mounted) return;
    await _start(p, method);
  }

  Future<void> _start(AiProvider p, String method) async {
    final b = widget.bridge, l = context.l;
    final f = await b.startLogin(p.id, method);
    if (!mounted) return;
    if (f?.code == 'AI_LOGIN_ACTIVE') {
      final previousId = f?.details?['login_id'] is String
          ? f!.details!['login_id'] as String
          : b.status?.login?.id;
      final cancel = await confirmAction(
        context,
        l.loginActiveTitle,
        l.loginActiveHint,
        l.cancelPrevious,
      );
      if (!cancel || !mounted || previousId == null) return;
      final c = await b.cancelLogin(previousId);
      if (c != null) return _show(c);
      await Future<void>.delayed(const Duration(seconds: 2));
      if (mounted) await _start(p, method);
      return;
    }
    if (f != null) return _show(f);
    final id = b.lastLoginId;
    if (id == null) return;
    await Navigator.of(context).push<void>(
      MaterialPageRoute(
        builder: (_) => AccountLoginScreen(
          bridge: b,
          loginId: id,
          provider: p.id,
          method: method,
        ),
      ),
    );
  }

  Future<void> _remove(AiProvider p, AiAccount a) async {
    final l = context.l;
    if (await confirmAction(
      context,
      l.removeAccountTitle,
      l.removeAccountHint(a.email ?? a.label, p.name),
      l.remove,
      destructive: true,
    )) {
      _show(await widget.bridge.removeAccount(p.id, a.id));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l;
    return Scaffold(
      appBar: AppBar(title: Text(l.aiAccounts)),
      body: ListenableBuilder(
        listenable: widget.bridge,
        builder: (context, _) {
          final b = widget.bridge, s = b.status, now = b.now();
          final online = s?.connected == true;
          final login = s?.login;
          return PageBody(
            onRefresh: () => b.refresh(usage: true),
            children: [
              Text(
                l.aiAccountsHint,
                style: context.text.bodyMedium?.copyWith(
                  color: context.colors.muted,
                ),
              ),
              const SizedBox(height: Space.lg),
              if (b.loading && s == null)
                LinearProgressIndicator(semanticsLabel: l.loading),
              if (b.failure case final f?) ...[
                ErrorNotice(f, retry: () => b.refresh(usage: true)),
                const SizedBox(height: Space.md),
              ],
              if (s != null && !online) ...[
                Notice(
                  tone: Tone.warning,
                  icon: Icons.cloud_off_rounded,
                  title: l.accountsOfflineTitle,
                  message: s.hasProviders
                      ? l.accountsOfflineStale
                      : l.offlineHint,
                ),
                const SizedBox(height: Space.md),
              ],
              if (login != null && !login.terminal) ...[
                Notice(
                  tone: Tone.accent,
                  icon: Icons.login_rounded,
                  title: l.loginInProgress(providerTitle(login.provider)),
                  message: l.loginInProgressHint,
                  actions: [
                    TextButton(
                      onPressed: () => Navigator.of(context).push<void>(
                        MaterialPageRoute(
                          builder: (_) => AccountLoginScreen(
                            bridge: b,
                            loginId: login.id,
                            provider: login.provider,
                            method: login.method,
                          ),
                        ),
                      ),
                      child: Text(l.open),
                    ),
                  ],
                ),
                const SizedBox(height: Space.md),
              ],
              if (s != null && !s.hasProviders)
                EmptyState(
                  icon: Icons.manage_accounts_outlined,
                  title: l.providersEmptyTitle,
                  message: online ? l.providersPending : l.providersOffline,
                ),
              for (final p in s?.providers ?? const <AiProvider>[]) ...[
                _ProviderSection(
                  provider: p,
                  now: now,
                  online: online,
                  bridge: b,
                  onConnect: () => _connect(p),
                  onActivate: (a) async =>
                      _show(await b.activateAccount(p.id, a.id)),
                  onRemove: (a) => _remove(p, a),
                ),
                const SizedBox(height: Space.lg),
              ],
            ],
          );
        },
      ),
    );
  }
}

class _ProviderSection extends StatelessWidget {
  const _ProviderSection({
    required this.provider,
    required this.now,
    required this.online,
    required this.bridge,
    required this.onConnect,
    required this.onActivate,
    required this.onRemove,
  });
  final AiProvider provider;
  final DateTime now;
  final bool online;
  final BridgeController bridge;
  final VoidCallback onConnect;
  final ValueChanged<AiAccount> onActivate, onRemove;
  @override
  Widget build(BuildContext context) {
    final l = context.l, c = context.colors, p = provider;
    final loginBusy =
        bridge.status?.login != null && !bridge.status!.login!.terminal;
    return AppCard(
      padding: EdgeInsets.zero,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(
              Space.lg,
              Space.lg,
              Space.lg,
              Space.md,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  children: [
                    Icon(Icons.smart_toy_outlined, size: 20, color: c.accent),
                    const SizedBox(width: Space.sm),
                    Expanded(
                      child: Text(p.name, style: context.text.titleMedium),
                    ),
                    StatusBadge(
                      p.installed
                          ? (p.cliVersion == null
                                ? l.installed
                                : l.installedVersion(p.cliVersion!))
                          : l.notInstalled,
                      tone: p.installed ? Tone.neutral : Tone.warning,
                      icon: p.installed
                          ? Icons.terminal_rounded
                          : Icons.block_rounded,
                    ),
                  ],
                ),
                const SizedBox(height: Space.xs),
                Text(
                  l.providerService(providerService(p.id)),
                  style: context.text.bodySmall,
                ),
                if (p.exhausted) ...[
                  const SizedBox(height: Space.md),
                  Notice(
                    icon: Icons.hourglass_bottom_rounded,
                    title: l.limitTitle(p.name),
                    message: p.usage?.availableAt == null
                        ? l.limitNoReset
                        : l.availableAgain(
                            formatMoment(p.usage!.availableAt!, now),
                            formatUntil(p.usage!.availableAt!, now),
                          ),
                  ),
                ],
                for (final w in p.usage?.windows ?? const <UsageWindow>[]) ...[
                  const SizedBox(height: Space.md),
                  UsageMeter(window: w, now: now),
                ],
              ],
            ),
          ),
          if (!p.installed)
            Padding(
              padding: const EdgeInsets.fromLTRB(
                Space.lg,
                0,
                Space.lg,
                Space.lg,
              ),
              child: Text(l.cliMissing(p.name), style: context.text.bodySmall),
            )
          else ...[
            const Divider(),
            if (p.accounts.isEmpty)
              Padding(
                padding: const EdgeInsets.all(Space.lg),
                child: Text(
                  l.noAccounts(providerService(p.id)),
                  style: context.text.bodyMedium?.copyWith(color: c.muted),
                ),
              ),
            for (final a in p.accounts) ...[
              _AccountRow(
                account: a,
                active:
                    a.id == p.activeAccountId ||
                    (p.activeAccountId == null && a.active),
                busy: bridge.accountBusy(a.id),
                enabled: online,
                onActivate: () => onActivate(a),
                onRemove: () => onRemove(a),
              ),
              const Divider(),
            ],
            Padding(
              padding: const EdgeInsets.all(Space.md),
              child: OutlinedButton.icon(
                onPressed: online && !loginBusy ? onConnect : null,
                icon: const Icon(Icons.add_rounded, size: 20),
                label: Text(l.connectAccount),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _AccountRow extends StatelessWidget {
  const _AccountRow({
    required this.account,
    required this.active,
    required this.busy,
    required this.enabled,
    required this.onActivate,
    required this.onRemove,
  });
  final AiAccount account;
  final bool active, busy, enabled;
  final VoidCallback onActivate, onRemove;
  @override
  Widget build(BuildContext context) {
    final l = context.l, a = account;
    final canActivate = enabled && !active && a.loggedIn && !busy;
    final canRemove = enabled && !a.system && !busy;
    return Padding(
      padding: const EdgeInsets.fromLTRB(
        Space.lg,
        Space.md,
        Space.xs,
        Space.md,
      ),
      child: Row(
        children: [
          InitialAvatar(a.email ?? a.label, size: 36),
          const SizedBox(width: Space.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  a.label,
                  style: context.text.bodyLarge,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                if (a.email != null && a.email != a.label)
                  Text(
                    a.email!,
                    style: context.text.bodySmall,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                const SizedBox(height: Space.xs),
                Wrap(
                  spacing: Space.xs,
                  runSpacing: Space.xs,
                  children: [
                    if (active)
                      StatusBadge(
                        l.activeAccount,
                        tone: Tone.accent,
                        icon: Icons.check_rounded,
                      ),
                    if (!a.loggedIn) StatusBadge(l.noLogin, tone: Tone.warning),
                    if (a.plan != null) StatusBadge(planLabel(a.plan!)),
                    if (a.system) StatusBadge(l.systemAccount),
                  ],
                ),
              ],
            ),
          ),
          if (busy)
            Padding(
              padding: const EdgeInsets.all(Space.md),
              child: SizedBox.square(
                dimension: 20,
                child: CircularProgressIndicator(
                  strokeWidth: 2,
                  semanticsLabel: l.loading,
                ),
              ),
            )
          else if (canActivate || canRemove)
            PopupMenuButton<String>(
              tooltip: l.accountActions,
              icon: const Icon(Icons.more_vert_rounded),
              onSelected: (v) => v == 'activate' ? onActivate() : onRemove(),
              itemBuilder: (_) => [
                if (canActivate)
                  PopupMenuItem(
                    value: 'activate',
                    child: ListTile(
                      leading: const Icon(Icons.check_circle_outline_rounded),
                      title: Text(l.makeActive),
                    ),
                  ),
                if (canRemove)
                  PopupMenuItem(
                    value: 'remove',
                    child: ListTile(
                      leading: Icon(
                        Icons.delete_outline_rounded,
                        color: context.colors.danger,
                      ),
                      title: Text(
                        l.remove,
                        style: TextStyle(color: context.colors.danger),
                      ),
                    ),
                  ),
              ],
            )
          else
            const SizedBox(width: 48),
        ],
      ),
    );
  }
}

class _MethodTile extends StatelessWidget {
  const _MethodTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });
  final IconData icon;
  final String title, subtitle;
  final VoidCallback onTap;
  @override
  Widget build(BuildContext context) {
    final c = context.colors;
    return AppCard(
      onTap: onTap,
      child: Row(
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: c.accentContainer,
              borderRadius: BorderRadius.circular(Radii.small + 2),
            ),
            child: Icon(icon, size: 22, color: c.accent),
          ),
          const SizedBox(width: Space.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: context.text.titleSmall),
                const SizedBox(height: 2),
                Text(subtitle, style: context.text.bodySmall),
              ],
            ),
          ),
          Icon(Icons.chevron_right_rounded, size: 20, color: c.muted),
        ],
      ),
    );
  }
}

/// Login flow driven by the laptop CLI: device code (Codex), code paste
/// (Claude) or a browser on the computer. Polls status every 2 s.
class AccountLoginScreen extends StatefulWidget {
  const AccountLoginScreen({
    super.key,
    required this.bridge,
    required this.loginId,
    required this.provider,
    required this.method,
  });
  final BridgeController bridge;
  final String loginId, provider, method;
  @override
  State<AccountLoginScreen> createState() => _AccountLoginScreenState();
}

class _AccountLoginScreenState extends State<AccountLoginScreen> {
  final _code = TextEditingController();
  bool _sending = false, _codeSent = false, _cancelling = false;
  ApiFailure? _failure;

  @override
  void initState() {
    super.initState();
    widget.bridge.refresh();
  }

  @override
  void dispose() {
    _code.clear();
    _code.dispose();
    super.dispose();
  }

  LoginFlow? get _flow {
    final f = widget.bridge.status?.login;
    return f?.id == widget.loginId ? f : null;
  }

  Future<void> _open(String url) async {
    final ok = await launchUrl(
      Uri.parse(url),
      mode: LaunchMode.externalApplication,
    );
    if (!ok && mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(context.l.openFailed)));
    }
  }

  Future<void> _sendCode() async {
    final code = _code.text.trim();
    if (code.isEmpty || code.length > 2048 || RegExp(r'\s').hasMatch(code)) {
      setState(() => _failure = const ApiFailure('INVALID_CODE'));
      return;
    }
    setState(() {
      _sending = true;
      _failure = null;
    });
    final f = await widget.bridge.submitLoginCode(widget.loginId, code);
    if (!mounted) return;
    setState(() {
      _sending = false;
      _failure = f;
      if (f == null) {
        _codeSent = true;
        _code.clear();
      }
    });
  }

  Future<void> _cancel() async {
    final flow = _flow;
    if (flow == null || flow.terminal) {
      Navigator.of(context).pop();
      return;
    }
    setState(() => _cancelling = true);
    final f = await widget.bridge.cancelLogin(widget.loginId);
    if (!mounted) return;
    setState(() {
      _cancelling = false;
      _failure = f;
    });
    if (f == null) Navigator.of(context).pop();
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l;
    return ListenableBuilder(
      listenable: widget.bridge,
      builder: (context, _) {
        final flow = _flow, now = widget.bridge.now();
        final name = providerTitle(widget.provider);
        return Scaffold(
          appBar: AppBar(
            title: Text(
              l.connectAccountTitle(providerService(widget.provider)),
            ),
          ),
          body: SafeArea(
            child: PageBody(
              maxWidth: 520,
              children: [
                Text(
                  widget.method == 'computer'
                      ? l.methodComputer
                      : l.methodPhone,
                  style: context.text.titleSmall?.copyWith(
                    color: context.colors.secondary,
                  ),
                ),
                const SizedBox(height: Space.xs),
                Text(name, style: context.text.headlineSmall),
                const SizedBox(height: Space.xl),
                ..._body(flow, now),
                if (_failure case final f?) ...[
                  const SizedBox(height: Space.lg),
                  Notice(
                    title: f.code == 'INVALID_CODE'
                        ? l.codeInvalid
                        : errorLabel(context, f),
                  ),
                ],
              ],
            ),
          ),
          bottomNavigationBar: SafeArea(
            child: Padding(
              padding: const EdgeInsets.fromLTRB(
                Space.lg,
                Space.sm,
                Space.lg,
                Space.md,
              ),
              child: flow != null && flow.terminal
                  ? FilledButton(
                      onPressed: () => Navigator.of(context).pop(),
                      child: Text(l.done),
                    )
                  : OutlinedButton(
                      onPressed: _cancelling ? null : _cancel,
                      child: Text(l.cancelLogin),
                    ),
            ),
          ),
        );
      },
    );
  }

  List<Widget> _body(LoginFlow? flow, DateTime now) {
    final l = context.l, c = context.colors;
    Widget waiting(String title, String message) => Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(top: 2),
          child: SizedBox.square(
            dimension: 20,
            child: CircularProgressIndicator(
              strokeWidth: 2.5,
              semanticsLabel: l.loading,
            ),
          ),
        ),
        const SizedBox(width: Space.md),
        Expanded(
          child: Semantics(
            liveRegion: true,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: context.text.titleMedium),
                const SizedBox(height: Space.xs),
                Text(
                  message,
                  style: context.text.bodyMedium?.copyWith(color: c.secondary),
                ),
              ],
            ),
          ),
        ),
      ],
    );
    if (flow == null || flow.state == LoginState.starting) {
      return [waiting(l.loginStarting, l.loginStartingHint)];
    }
    final expires = flow.expiresAt == null
        ? null
        : Text(
            l.loginExpires(formatMoment(flow.expiresAt!, now)),
            style: context.text.bodySmall,
          );
    switch (flow.state) {
      case LoginState.completed:
        return [
          Notice(
            tone: Tone.accent,
            icon: Icons.check_circle_rounded,
            title: l.loginCompleted,
            message: l.loginCompletedHint,
          ),
        ];
      case LoginState.failed || LoginState.expired || LoginState.cancelled:
        return [
          Notice(
            tone: flow.state == LoginState.cancelled
                ? Tone.neutral
                : Tone.danger,
            title: switch (flow.state) {
              LoginState.expired => l.loginExpired,
              LoginState.cancelled => l.loginCancelled,
              _ => l.loginFailed,
            },
            message: _errorHint(flow.errorCode),
          ),
        ];
      case LoginState.verifying:
        return [waiting(l.loginVerifying, l.loginVerifyingHint)];
      case LoginState.waitingUser || LoginState.starting:
        break;
    }
    if (flow.method == 'computer') {
      return [
        waiting(
          l.loginOnComputer,
          l.loginOnComputerHint(providerService(flow.provider)),
        ),
        if (expires != null) ...[const SizedBox(height: Space.lg), expires],
      ];
    }
    final steps = <Widget>[];
    if (flow.userCode != null) {
      steps.addAll([
        Text(
          l.deviceCodeTitle,
          style: context.text.titleSmall?.copyWith(color: c.secondary),
        ),
        const SizedBox(height: Space.sm),
        AppCard(
          color: c.accentContainer,
          borderColor: c.accentContainer,
          child: Row(
            children: [
              Expanded(
                child: SelectableText(
                  flow.userCode!,
                  style: TextStyle(
                    fontFamily: monoFamily,
                    fontSize: 30,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 3,
                    color: c.text,
                  ),
                ),
              ),
              IconButton.filled(
                style: IconButton.styleFrom(
                  backgroundColor: c.accent,
                  foregroundColor: c.onAccent,
                ),
                tooltip: l.copyCode,
                onPressed: () => copyText(context, flow.userCode!),
                icon: Icon(
                  Icons.content_copy_rounded,
                  size: 20,
                  color: c.onAccent,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: Space.md),
        Text(
          l.deviceCodeSteps,
          style: context.text.bodyMedium?.copyWith(color: c.secondary),
        ),
        const SizedBox(height: Space.lg),
      ]);
    } else {
      steps.addAll([
        Text(
          l.claudeSteps,
          style: context.text.bodyMedium?.copyWith(color: c.secondary),
        ),
        const SizedBox(height: Space.lg),
      ]);
    }
    if (flow.url != null) {
      // One primary action per state: with a code to paste, sending it is
      // the primary button and opening the page becomes secondary.
      final open = flow.needsCode ? OutlinedButton.icon : FilledButton.icon;
      steps.add(
        open(
          onPressed: () => _open(flow.url!),
          icon: const Icon(Icons.open_in_new_rounded, size: 20),
          label: Text(l.openLoginPage),
        ),
      );
      steps.add(const SizedBox(height: Space.xs));
      steps.add(
        Text(
          Uri.parse(flow.url!).host,
          style: context.text.bodySmall,
          textAlign: TextAlign.center,
        ),
      );
    } else {
      steps.add(waiting(l.loginStarting, l.loginUrlPending));
    }
    if (flow.needsCode) {
      steps.addAll([
        const SizedBox(height: Space.xl),
        if (_codeSent)
          waiting(l.loginVerifying, l.loginVerifyingHint)
        else ...[
          TextField(
            controller: _code,
            autocorrect: false,
            enableSuggestions: false,
            decoration: InputDecoration(
              labelText: l.pasteCode,
              prefixIcon: const Icon(Icons.key_outlined),
            ),
            onSubmitted: (_) => _sendCode(),
          ),
          const SizedBox(height: Space.md),
          FilledButton(
            onPressed: _sending ? null : _sendCode,
            child: Text(l.sendCode),
          ),
        ],
      ]);
    } else {
      steps.addAll([
        const SizedBox(height: Space.xl),
        waiting(l.loginWaiting, l.loginWaitingHint),
      ]);
    }
    if (expires != null) {
      steps.addAll([const SizedBox(height: Space.lg), expires]);
    }
    return steps;
  }

  String _errorHint(String? code) {
    final l = context.l;
    return switch (code) {
      'DEVICE_CODE_UNAVAILABLE' => l.deviceCodeUnavailable,
      'CODE_REJECTED' => l.codeRejected,
      'CLI_MISSING' => l.cliMissingShort,
      'LOGIN_TIMEOUT' => l.loginTimeout,
      'COMMAND_EXPIRED' => l.commandExpired,
      'CODEX_BRIDGE_LOST' => l.runBridgeLost,
      _ => l.loginFailedHint,
    };
  }
}
