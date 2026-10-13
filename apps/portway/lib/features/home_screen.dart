import 'dart:async';

import 'package:flutter/material.dart';

import '../core/format.dart';
import '../core/session.dart';
import '../core/theme.dart';
import 'accounts_screen.dart';
import 'codex.dart';
import 'common.dart';
import 'compose_screen.dart';
import 'profile_screen.dart';
import 'run_screen.dart';
import 'tasks_screen.dart';

/// Navigation entry points shared by all tabs of the signed-in shell.
class ShellNav {
  const ShellNav({
    required this.selectTab,
    required this.compose,
    required this.openRun,
  });
  final void Function(int tab) selectTab;
  final Future<void> Function({String? provider}) compose;
  final Future<void> Function(String id) openRun;
  static const home = 0, tasks = 1, accounts = 2, profile = 3;
}

/// UI-05 shell: bottom navigation (rail on wide screens) with Главная ·
/// Задачи · Аккаунты ИИ · Профиль. Owns the bridge controller so all tabs
/// share one status poller.
class HomeScreen extends StatefulWidget {
  const HomeScreen({
    super.key,
    required this.session,
    required this.themeMode,
    required this.onTheme,
    this.now,
  });
  final SessionController session;
  final DateTime Function()? now;
  final ThemeMode themeMode;
  final ValueChanged<ThemeMode> onTheme;
  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> with WidgetsBindingObserver {
  late final CodexRepository repository = CodexRepository(widget.session);
  late final BridgeController bridge = BridgeController(
    repository,
    now: widget.now,
  );
  late final TaskComposer composer = TaskComposer(repository);
  int tab = 0;
  late final ShellNav nav = ShellNav(
    selectTab: (v) => setState(() => tab = v),
    compose: _compose,
    openRun: _openRun,
  );

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    composer.restore().catchError((Object _) {});
    bridge.start();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) =>
      bridge.setForeground(state == AppLifecycleState.resumed);

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    bridge.dispose();
    composer.dispose();
    super.dispose();
  }

  Future<void> _compose({String? provider}) async {
    if (provider != null) await bridge.setPreferredProvider(provider);
    if (!mounted) return;
    final id = await Navigator.of(context).push<String>(
      MaterialPageRoute(
        builder: (_) =>
            ComposeScreen(composer: composer, bridge: bridge, nav: nav),
      ),
    );
    if (!mounted) return;
    if (id != null) await _openRun(id);
    unawaited(bridge.refresh());
  }

  Future<void> _openRun(String id) async {
    await Navigator.of(context).push<void>(
      MaterialPageRoute(
        builder: (_) =>
            RunScreen(repository: repository, id: id, bridge: bridge, nav: nav),
      ),
    );
    if (mounted) unawaited(bridge.refresh());
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l;
    final items = [
      (Icons.home_outlined, Icons.home_rounded, l.home),
      (Icons.forum_outlined, Icons.forum_rounded, l.tasks),
      (
        Icons.manage_accounts_outlined,
        Icons.manage_accounts_rounded,
        l.aiAccounts,
      ),
      (Icons.person_outline_rounded, Icons.person_rounded, l.profile),
    ];
    final pages = [
      HomeTab(
        bridge: bridge,
        composer: composer,
        session: widget.session,
        nav: nav,
      ),
      TasksTab(bridge: bridge, nav: nav),
      AccountsTab(bridge: bridge),
      ProfileTab(
        session: widget.session,
        themeMode: widget.themeMode,
        onTheme: widget.onTheme,
        now: bridge.now,
      ),
    ];
    return PopScope(
      canPop: tab == 0,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) setState(() => tab = 0);
      },
      child: LayoutBuilder(
        builder: (context, box) {
          final wide = box.maxWidth >= 700;
          final body = IndexedStack(index: tab, children: pages);
          return Scaffold(
            body: wide
                ? Row(
                    children: [
                      SafeArea(
                        right: false,
                        child: NavigationRail(
                          selectedIndex: tab,
                          onDestinationSelected: (v) => setState(() => tab = v),
                          labelType: NavigationRailLabelType.all,
                          destinations: [
                            for (final i in items)
                              NavigationRailDestination(
                                icon: Icon(i.$1),
                                selectedIcon: Icon(i.$2),
                                label: Text(i.$3),
                              ),
                          ],
                        ),
                      ),
                      const VerticalDivider(width: 1),
                      Expanded(child: body),
                    ],
                  )
                : body,
            bottomNavigationBar: wide
                ? null
                : NavigationBar(
                    selectedIndex: tab,
                    onDestinationSelected: (v) => setState(() => tab = v),
                    destinations: [
                      for (final i in items)
                        NavigationDestination(
                          icon: Icon(i.$1),
                          selectedIcon: Icon(i.$2),
                          label: i.$3,
                          tooltip: i.$3,
                        ),
                    ],
                  ),
          );
        },
      ),
    );
  }
}

/// Main dashboard: laptop status, AI usage and limits, new task, recent tasks.
class HomeTab extends StatelessWidget {
  const HomeTab({
    super.key,
    required this.bridge,
    required this.composer,
    required this.session,
    required this.nav,
  });
  final BridgeController bridge;
  final TaskComposer composer;
  final SessionController session;
  final ShellNav nav;

  @override
  Widget build(BuildContext context) {
    final l = context.l;
    final ownerName = (session.owner?['name'] as String?)?.trim() ?? '';
    return Scaffold(
      appBar: AppBar(
        title: Text(l.home),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: Space.sm),
            child: IconButton(
              tooltip: l.profile,
              onPressed: () => nav.selectTab(ShellNav.profile),
              icon: InitialAvatar(
                ownerName.isEmpty ? session.name : ownerName,
                size: 32,
              ),
            ),
          ),
        ],
      ),
      body: ListenableBuilder(
        listenable: Listenable.merge([bridge, composer]),
        builder: (context, _) {
          final s = bridge.status, now = bridge.now();
          final preferred = s?.provider(bridge.preferredProvider);
          final recent = bridge.runs.take(3).toList();
          return PageBody(
            onRefresh: () => bridge.refresh(usage: true),
            children: [
              Text(
                l.greeting(
                  _dayPart(context, now),
                  ownerName.isEmpty ? l.ownerFallback : ownerName,
                ),
                style: context.text.titleMedium?.copyWith(
                  color: context.colors.secondary,
                ),
              ),
              const SizedBox(height: Space.lg),
              if (bridge.failure case final f?) ...[
                ErrorNotice(f, retry: () => bridge.refresh(usage: true)),
                const SizedBox(height: Space.md),
              ],
              if (preferred != null && preferred.exhausted) ...[
                LimitNotice(provider: preferred, bridge: bridge, nav: nav),
                const SizedBox(height: Space.md),
              ],
              ComputerCard(
                status: s,
                name: session.name,
                checkedAt: bridge.checkedAt,
                now: now,
                failed: bridge.failure != null,
              ),
              const SizedBox(height: Space.md),
              FilledButton.icon(
                onPressed: () => nav.compose(),
                icon: Icon(
                  composer.pending == null
                      ? Icons.add_rounded
                      : Icons.sync_problem_rounded,
                  size: 20,
                ),
                label: Text(
                  composer.pending == null ? l.newTask : l.pendingTitle,
                ),
              ),
              SectionHeader(
                l.aiModels,
                action: TextButton(
                  onPressed: () => nav.selectTab(ShellNav.accounts),
                  child: Text(l.manage),
                ),
              ),
              if (s == null && bridge.failure == null)
                AppCard(
                  child: Row(
                    children: [
                      SizedBox.square(
                        dimension: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          semanticsLabel: l.loading,
                        ),
                      ),
                      const SizedBox(width: Space.md),
                      Expanded(
                        child: Text(
                          l.usageLoading,
                          style: context.text.bodyMedium?.copyWith(
                            color: context.colors.secondary,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              if (s != null && !s.hasProviders)
                AppCard(
                  child: Row(
                    children: [
                      Icon(
                        Icons.hourglass_empty_rounded,
                        size: 20,
                        color: context.colors.muted,
                      ),
                      const SizedBox(width: Space.md),
                      Expanded(
                        child: Text(
                          s.connected ? l.providersPending : l.providersOffline,
                          style: context.text.bodyMedium?.copyWith(
                            color: context.colors.secondary,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              for (final p in s?.providers ?? const <AiProvider>[]) ...[
                ProviderCard(
                  provider: p,
                  preferred: p.id == bridge.preferredProvider,
                  stale: s?.connected != true,
                  now: now,
                  onTap: () => nav.selectTab(ShellNav.accounts),
                  onPrefer: p.id == bridge.preferredProvider
                      ? null
                      : () => bridge.setPreferredProvider(p.id),
                ),
                const SizedBox(height: Space.md),
              ],
              SectionHeader(
                l.recentTasks,
                action: bridge.runs.isEmpty
                    ? null
                    : TextButton(
                        onPressed: () => nav.selectTab(ShellNav.tasks),
                        child: Text(l.allTasks),
                      ),
              ),
              if (!bridge.loading &&
                  bridge.runs.isEmpty &&
                  bridge.failure == null)
                AppCard(
                  child: EmptyState(
                    icon: Icons.chat_bubble_outline_rounded,
                    title: l.noTasks,
                    message: l.noTasksHint,
                  ),
                ),
              for (final run in recent) ...[
                RunTile(run: run, now: now, onTap: () => nav.openRun(run.id)),
                const SizedBox(height: Space.sm),
              ],
            ],
          );
        },
      ),
    );
  }

  static String _dayPart(BuildContext context, DateTime now) {
    final l = context.l, h = now.hour;
    if (h >= 5 && h < 12) return l.morning;
    if (h >= 12 && h < 18) return l.day;
    if (h >= 18 && h < 23) return l.evening;
    return l.night;
  }
}

class ComputerCard extends StatelessWidget {
  const ComputerCard({
    super.key,
    required this.status,
    required this.name,
    required this.checkedAt,
    required this.now,
    required this.failed,
  });
  final BridgeStatus? status;
  final String name;
  final DateTime? checkedAt, now;
  final bool failed;
  @override
  Widget build(BuildContext context) {
    final l = context.l, c = context.colors, s = status;
    final online = s?.connected == true && !failed;
    final label = s == null
        ? l.statusUnknown
        : online
        ? l.online
        : l.offline;
    final lastSeen = s?.lastSeenAt;
    final detail = s == null
        ? l.statusChecking
        : online
        ? l.computerOnlineHint(s.workspace)
        : lastSeen == null
        ? l.computerNeverSeen
        : l.computerLastSeen(formatAgo(lastSeen, now!));
    return AppCard(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: online ? c.accentContainer : c.raised,
              borderRadius: BorderRadius.circular(Radii.small + 2),
            ),
            child: Icon(
              Icons.laptop_mac_rounded,
              size: 22,
              color: online ? c.accent : c.muted,
            ),
          ),
          const SizedBox(width: Space.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  name.isEmpty ? l.computer : name,
                  style: context.text.titleMedium,
                ),
                const SizedBox(height: Space.xs),
                Semantics(
                  liveRegion: true,
                  child: ConnectionDot(online: online, label: label),
                ),
                const SizedBox(height: Space.xs),
                Text(detail, style: context.text.bodySmall),
                if (!online && s != null) ...[
                  const SizedBox(height: Space.xs),
                  Text(l.offlineHint, style: context.text.bodySmall),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

String accountLine(AiProvider p, BuildContext context) {
  final a = p.activeAccount;
  if (a == null) return context.l.noActiveAccount(providerService(p.id));
  final parts = <String>[
    a.email ?? a.label,
    if (a.plan != null) planLabel(a.plan!),
  ];
  return parts.join(' · ');
}

String planLabel(String plan) => switch (plan.toLowerCase()) {
  'plus' => 'Plus',
  'pro' => 'Pro',
  'max' => 'Max',
  'team' => 'Team',
  'free' => 'Free',
  'claude_pro' => 'Pro',
  'claude_max' => 'Max',
  _ => plan,
};

/// Provider usage card: account, 5-hour/weekly meters, exhausted state.
class ProviderCard extends StatelessWidget {
  const ProviderCard({
    super.key,
    required this.provider,
    required this.preferred,
    required this.stale,
    required this.now,
    this.onTap,
    this.onPrefer,
  });
  final AiProvider provider;
  final bool preferred, stale;
  final DateTime now;
  final VoidCallback? onTap, onPrefer;
  @override
  Widget build(BuildContext context) {
    final l = context.l, c = context.colors, p = provider, u = p.usage;
    final (String badge, Tone tone, IconData icon) = !p.installed
        ? (l.notInstalled, Tone.neutral, Icons.block_rounded)
        : !p.hasLoggedInAccount
        ? (l.noLogin, Tone.warning, Icons.login_rounded)
        : p.exhausted
        ? (l.limitExhausted, Tone.danger, Icons.block_rounded)
        : u?.status == UsageStatus.warning
        ? (l.limitNear, Tone.warning, Icons.warning_amber_rounded)
        : u == null || u.status == UsageStatus.unknown
        ? (l.usageUnknown, Tone.neutral, Icons.help_outline_rounded)
        : (l.available, Tone.accent, Icons.check_rounded);
    return AppCard(
      onTap: onTap,
      borderColor: p.exhausted ? c.danger.withValues(alpha: 0.6) : null,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Icon(Icons.smart_toy_outlined, size: 20, color: c.accent),
              const SizedBox(width: Space.sm),
              Expanded(child: Text(p.name, style: context.text.titleMedium)),
              StatusBadge(badge, tone: tone, icon: icon),
            ],
          ),
          const SizedBox(height: Space.xs),
          Text(accountLine(p, context), style: context.text.bodySmall),
          if (u != null && u.windows.isNotEmpty) ...[
            for (final w in u.windows) ...[
              const SizedBox(height: Space.md),
              UsageMeter(window: w, now: now),
            ],
          ],
          if (p.exhausted &&
              u?.availableAt != null &&
              u!.windows.every((w) => w.resetsAt == null)) ...[
            const SizedBox(height: Space.md),
            Text(
              l.availableAgain(
                formatMoment(u.availableAt!, now),
                formatUntil(u.availableAt!, now),
              ),
              style: context.text.labelMedium?.copyWith(color: c.danger),
            ),
          ],
          const SizedBox(height: Space.md),
          Row(
            children: [
              Expanded(
                child: Text(
                  stale
                      ? l.usageStale
                      : u?.checkedAt == null
                      ? l.usageNotChecked
                      : l.usageChecked(formatAgo(u!.checkedAt!, now)),
                  style: context.text.bodySmall,
                ),
              ),
              if (preferred)
                StatusBadge(
                  l.forNewTasks,
                  tone: Tone.accent,
                  icon: Icons.bolt_rounded,
                )
              else if (onPrefer != null && p.installed)
                TextButton(onPressed: onPrefer, child: Text(l.useForTasks)),
            ],
          ),
        ],
      ),
    );
  }
}

/// Prominent "Лимит исчерпан" banner with reset time and next actions.
class LimitNotice extends StatelessWidget {
  const LimitNotice({
    super.key,
    required this.provider,
    required this.bridge,
    required this.nav,
    this.onSwitched,
  });
  final AiProvider provider;
  final BridgeController bridge;
  final ShellNav nav;
  final VoidCallback? onSwitched;
  @override
  Widget build(BuildContext context) {
    final l = context.l, now = bridge.now(), p = provider;
    final at = p.usage?.availableAt;
    AiProvider? other;
    for (final o in bridge.status?.providers ?? const <AiProvider>[]) {
      if (o.id != p.id && o.installed && o.hasLoggedInAccount && !o.exhausted) {
        other = o;
      }
    }
    final account = p.activeAccount;
    return Notice(
      tone: Tone.danger,
      icon: Icons.hourglass_bottom_rounded,
      title: l.limitTitle(p.name),
      message: [
        if (account != null)
          l.limitAccount(providerService(p.id), account.email ?? account.label),
        at == null
            ? l.limitNoReset
            : l.availableAgain(formatMoment(at, now), formatUntil(at, now)),
      ].join('\n'),
      actions: [
        TextButton(
          onPressed: () => nav.selectTab(ShellNav.accounts),
          child: Text(l.switchAccount),
        ),
        if (other != null)
          TextButton(
            onPressed: () {
              bridge.setPreferredProvider(other!.id);
              onSwitched?.call();
            },
            child: Text(l.switchTo(other.name)),
          ),
      ],
    );
  }
}

/// History row: prompt preview, state, provider/model and time.
class RunTile extends StatelessWidget {
  const RunTile({super.key, required this.run, required this.now, this.onTap});
  final CodexRun run;
  final DateTime now;
  final VoidCallback? onTap;
  @override
  Widget build(BuildContext context) {
    final l = context.l, c = context.colors;
    final preview = run.preview.trim();
    final meta = [
      if (run.provider != null) providerTitle(run.provider!),
      ?run.model,
    ].join(' · ');
    return AppCard(
      onTap: onTap,
      padding: const EdgeInsets.fromLTRB(
        Space.lg,
        Space.md + 2,
        Space.md,
        Space.md + 2,
      ),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  preview.isEmpty
                      ? (run.createdAt == null
                            ? l.taskTitle
                            : l.taskFrom(formatMoment(run.createdAt!, now)))
                      : preview,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: context.text.bodyLarge?.copyWith(
                    color: preview.isEmpty ? c.secondary : c.text,
                  ),
                ),
                const SizedBox(height: Space.sm),
                Wrap(
                  spacing: Space.sm,
                  runSpacing: Space.xs,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: [
                    StatusBadge(
                      stateLabel(context, run.state),
                      tone: stateTone(run.state),
                      icon: stateIcon(run.state),
                    ),
                    if (meta.isNotEmpty)
                      Text(meta, style: context.text.bodySmall),
                    if (run.createdAt != null)
                      Text(
                        formatMoment(run.createdAt!, now),
                        style: context.text.bodySmall,
                      ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(width: Space.sm),
          Icon(Icons.chevron_right_rounded, size: 20, color: c.muted),
        ],
      ),
    );
  }
}
