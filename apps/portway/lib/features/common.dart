import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../core/api.dart';
import '../core/format.dart';
import '../core/theme.dart';
import '../l10n/app_localizations.dart';
import 'bridge_models.dart';

extension Strings on BuildContext {
  AppLocalizations get l => AppLocalizations.of(this)!;
  PortwayColors get colors => PortwayColors.of(this);
  TextTheme get text => Theme.of(this).textTheme;
}

String errorLabel(BuildContext context, ApiFailure e) {
  final l = context.l;
  return switch (e.code) {
    'REQUEST_EXPIRED' => l.requestExpired,
    'OWNER_ALREADY_BOUND' => l.ownerAlreadyBound,
    'OWNER_REQUIRED' => l.ownerRequired,
    'REGISTRATION_UNCERTAIN' => l.registrationUncertain,
    'IDEMPOTENCY_CONFLICT' => l.registrationConflict,
    'RATE_LIMITED' => l.rateLimited,
    'INVALID_URL' => l.invalidUrl,
    'NETWORK' => l.networkError,
    'INVALID_RESPONSE' => l.responseError,
    'STORAGE' => l.storageError,
    'TOKEN_EXPIRED' || 'SESSION_EXPIRED' || 'INVALID_TOKEN' => l.sessionExpired,
    'DEVICE_REVOKED' => l.deviceRevoked,
    'SIGNED_OUT' => l.signedOut,
    'INVALID_CREDENTIALS' ||
    'AUTH_INVALID_CREDENTIALS' ||
    'AUTH_REQUIRED' => l.credentialsError,
    'EMAIL_NOT_CONFIRMED' => l.notConfirmed,
    'SETUP_PROOF_REQUIRED' ||
    'SETUP_PROOF_INVALID' ||
    'SETUP_DENIED' => l.setupError,
    'CODEX_OFFLINE' => l.errorOffline,
    'CODEX_DISABLED' => l.disabled,
    'CODEX_BUSY' => l.busyError,
    'AI_LIMIT_EXHAUSTED' => l.errorLimit,
    'AI_ACCOUNT_REQUIRED' => l.errorAccountRequired,
    'AI_PROVIDER_UNAVAILABLE' => l.errorProviderUnavailable,
    'AI_LOGIN_ACTIVE' => l.errorLoginActive,
    'AI_ACCOUNT_SYSTEM' => l.errorAccountSystem,
    'AI_LOGIN_NOT_ACTIVE' => l.errorLoginNotActive,
    'AI_COMMANDS_BUSY' => l.errorCommandsBusy,
    'INVALID_REQUEST' || 'VALIDATION_FAILED' => l.errorInvalidRequest,
    'INVALID_PROMPT' => l.promptRequired,
    'EVENT_GAP' => l.eventGap,
    'NOT_FOUND' => l.notFound,
    _ => l.serverError,
  };
}

/// Run failure codes (`error_code` / failed event `code`).
String runFailureLabel(BuildContext context, String? code) {
  final l = context.l;
  return switch (code) {
    'AI_LIMIT_EXHAUSTED' => l.runLimit,
    'AI_AUTH_REQUIRED' => l.runAuth,
    'CODEX_BRIDGE_LOST' => l.runBridgeLost,
    'DEVICE_REVOKED' => l.deviceRevoked,
    'AI_TIMEOUT' => l.runTimeout,
    'AI_RUN_FAILED' => l.runFailedGeneric,
    _ => l.runFailedGeneric,
  };
}

/// Outcome codes of account/usage commands reported by the bridge.
String commandErrorLabel(BuildContext context, String? code) {
  final l = context.l;
  return switch (code) {
    'COMMAND_EXPIRED' => l.commandExpired,
    'CODEX_BRIDGE_LOST' => l.runBridgeLost,
    'AI_ACCOUNT_NOT_FOUND' => l.accountNotFound,
    'AI_ACCOUNT_SYSTEM' => l.errorAccountSystem,
    'AI_ACCOUNT_BUSY' => l.accountBusy,
    'AI_LOGIN_NOT_ACTIVE' => l.errorLoginNotActive,
    'AI_COMMANDS_BUSY' => l.errorCommandsBusy,
    'CLI_MISSING' => l.cliMissingShort,
    'DEVICE_CODE_UNAVAILABLE' => l.deviceCodeUnavailable,
    _ => l.commandFailedGeneric,
  };
}

String stateLabel(BuildContext c, String state) => switch (state) {
  'queued' => c.l.queued,
  'running' => c.l.running,
  'completed' => c.l.completed,
  'failed' => c.l.failed,
  'stopped' => c.l.stopped,
  _ => c.l.unknownState,
};

enum Tone { neutral, accent, warning, danger }

Tone stateTone(String state) => switch (state) {
  'queued' || 'running' => Tone.accent,
  'completed' => Tone.neutral,
  'failed' => Tone.danger,
  'stopped' => Tone.warning,
  _ => Tone.neutral,
};

IconData stateIcon(String state) => switch (state) {
  'queued' => Icons.schedule_rounded,
  'running' => Icons.autorenew_rounded,
  'completed' => Icons.check_circle_outline_rounded,
  'failed' => Icons.error_outline_rounded,
  'stopped' => Icons.stop_circle_outlined,
  _ => Icons.help_outline_rounded,
};

({Color fg, Color bg}) toneColors(BuildContext context, Tone tone) {
  final c = context.colors;
  return switch (tone) {
    Tone.neutral => (fg: c.secondary, bg: c.raised),
    Tone.accent => (fg: c.accent, bg: c.accentContainer),
    Tone.warning => (fg: c.warning, bg: c.warningContainer),
    Tone.danger => (fg: c.danger, bg: c.dangerContainer),
  };
}

/// Small status pill: colour + icon + text, never colour alone.
class StatusBadge extends StatelessWidget {
  const StatusBadge(
    this.label, {
    super.key,
    this.tone = Tone.neutral,
    this.icon,
  });
  final String label;
  final Tone tone;
  final IconData? icon;
  @override
  Widget build(BuildContext context) {
    final t = toneColors(context, tone);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: t.bg,
        borderRadius: BorderRadius.circular(Radii.small),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 14, color: t.fg),
            const SizedBox(width: 4),
          ],
          Flexible(
            child: Text(
              label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: context.text.labelSmall?.copyWith(color: t.fg),
            ),
          ),
        ],
      ),
    );
  }
}

/// Live dot + text for connection state.
class ConnectionDot extends StatelessWidget {
  const ConnectionDot({super.key, required this.online, required this.label});
  final bool online;
  final String label;
  @override
  Widget build(BuildContext context) {
    final c = context.colors;
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 8,
          height: 8,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: online ? c.accent : c.disabled,
          ),
        ),
        const SizedBox(width: Space.sm),
        Flexible(
          child: Text(
            label,
            style: context.text.labelMedium?.copyWith(
              color: online ? c.accent : c.muted,
            ),
          ),
        ),
      ],
    );
  }
}

/// Flat card with a subtle outline; one level of surface only.
class AppCard extends StatelessWidget {
  const AppCard({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(Space.lg),
    this.onTap,
    this.color,
    this.borderColor,
  });
  final Widget child;
  final EdgeInsetsGeometry padding;
  final VoidCallback? onTap;
  final Color? color, borderColor;
  @override
  Widget build(BuildContext context) => Card(
    color: color,
    shape: borderColor == null
        ? null
        : RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(Radii.medium),
            side: BorderSide(color: borderColor!),
          ),
    child: onTap == null
        ? Padding(padding: padding, child: child)
        : InkWell(
            onTap: onTap,
            child: Padding(padding: padding, child: child),
          ),
  );
}

class SectionHeader extends StatelessWidget {
  const SectionHeader(this.title, {super.key, this.action, this.count});
  final String title;
  final Widget? action;
  final int? count;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(top: Space.lg, bottom: Space.xs),
    child: Row(
      children: [
        Expanded(
          child: Semantics(
            header: true,
            child: Text(
              count == null ? title : '$title · $count',
              style: context.text.titleSmall?.copyWith(
                color: context.colors.secondary,
              ),
            ),
          ),
        ),
        ?action,
      ],
    ),
  );
}

/// Banner for errors, warnings and explanations with optional actions.
class Notice extends StatelessWidget {
  const Notice({
    super.key,
    required this.title,
    this.message,
    this.tone = Tone.danger,
    this.icon,
    this.actions = const [],
    this.child,
  });
  final String title;
  final String? message;
  final Tone tone;
  final IconData? icon;
  final List<Widget> actions;
  final Widget? child;
  @override
  Widget build(BuildContext context) {
    final t = toneColors(context, tone), c = context.colors;
    return Semantics(
      liveRegion: true,
      container: true,
      child: Container(
        decoration: BoxDecoration(
          color: t.bg,
          borderRadius: BorderRadius.circular(Radii.medium),
        ),
        padding: const EdgeInsets.fromLTRB(
          Space.lg,
          Space.md + 2,
          Space.lg,
          Space.md + 2,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Padding(
                  padding: const EdgeInsets.only(top: 1),
                  child: Icon(
                    icon ??
                        switch (tone) {
                          Tone.danger => Icons.error_outline_rounded,
                          Tone.warning => Icons.warning_amber_rounded,
                          _ => Icons.info_outline_rounded,
                        },
                    size: 20,
                    color: t.fg,
                  ),
                ),
                const SizedBox(width: Space.md),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        title,
                        style: context.text.titleSmall?.copyWith(color: c.text),
                      ),
                      if (message != null) ...[
                        const SizedBox(height: Space.xs),
                        Text(
                          message!,
                          style: context.text.bodyMedium?.copyWith(
                            color: c.secondary,
                          ),
                        ),
                      ],
                      ?child,
                    ],
                  ),
                ),
              ],
            ),
            if (actions.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: Space.sm),
                child: TextButtonTheme(
                  // Actions read as tonal buttons inside the banner so a
                  // wrapped second row looks intentional, not like loose links.
                  data: TextButtonThemeData(
                    style: TextButton.styleFrom(
                      foregroundColor: tone == Tone.neutral ? c.accent : t.fg,
                      backgroundColor: (tone == Tone.neutral ? c.accent : t.fg)
                          .withValues(alpha: 0.12),
                      padding: const EdgeInsets.symmetric(horizontal: 14),
                      minimumSize: const Size(48, 40),
                      tapTargetSize: MaterialTapTargetSize.padded,
                      textStyle: context.text.labelLarge,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(Radii.small),
                      ),
                    ),
                  ),
                  child: Wrap(
                    spacing: Space.sm,
                    runSpacing: 0,
                    children: actions,
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class ErrorNotice extends StatelessWidget {
  const ErrorNotice(this.failure, {super.key, this.retry, this.title});
  final ApiFailure failure;
  final VoidCallback? retry;
  final String? title;
  @override
  Widget build(BuildContext context) => Notice(
    title: title ?? errorLabel(context, failure),
    message: title == null ? null : errorLabel(context, failure),
    tone: failure.code == 'CODEX_OFFLINE' ? Tone.warning : Tone.danger,
    actions: [
      if (retry != null)
        TextButton.icon(
          onPressed: retry,
          icon: const Icon(Icons.refresh_rounded, size: 18),
          label: Text(context.l.retry),
        ),
    ],
  );
}

class EmptyState extends StatelessWidget {
  const EmptyState({
    super.key,
    required this.icon,
    required this.title,
    required this.message,
    this.action,
  });
  final IconData icon;
  final String title, message;
  final Widget? action;
  @override
  Widget build(BuildContext context) {
    final c = context.colors;
    return Padding(
      padding: const EdgeInsets.symmetric(
        vertical: Space.xl,
        horizontal: Space.lg,
      ),
      child: Column(
        children: [
          Container(
            width: 56,
            height: 56,
            decoration: BoxDecoration(
              color: c.accentContainer,
              shape: BoxShape.circle,
            ),
            child: Icon(icon, size: 24, color: c.accent),
          ),
          const SizedBox(height: Space.lg),
          Text(
            title,
            textAlign: TextAlign.center,
            style: context.text.titleMedium,
          ),
          const SizedBox(height: Space.sm),
          Text(
            message,
            textAlign: TextAlign.center,
            style: context.text.bodyMedium?.copyWith(color: c.muted),
          ),
          if (action != null) ...[const SizedBox(height: Space.lg), action!],
        ],
      ),
    );
  }
}

/// Scrollable page content constrained to a readable width.
class PageBody extends StatelessWidget {
  const PageBody({
    super.key,
    required this.children,
    this.maxWidth = 640,
    this.padding = const EdgeInsets.fromLTRB(
      Space.lg,
      Space.sm,
      Space.lg,
      Space.xxl,
    ),
    this.onRefresh,
    this.controller,
  });
  final List<Widget> children;
  final double maxWidth;
  final EdgeInsets padding;
  final Future<void> Function()? onRefresh;
  final ScrollController? controller;
  @override
  Widget build(BuildContext context) {
    final list = LayoutBuilder(
      builder: (context, box) {
        final side = ((box.maxWidth - maxWidth) / 2).clamp(
          0.0,
          double.infinity,
        );
        return ListView(
          controller: controller,
          physics: const AlwaysScrollableScrollPhysics(),
          padding: padding.add(EdgeInsets.symmetric(horizontal: side)),
          children: children,
        );
      },
    );
    return onRefresh == null
        ? list
        : RefreshIndicator(onRefresh: onRefresh!, child: list);
  }
}

/// Label/value line used in meta blocks.
class MetaRow extends StatelessWidget {
  const MetaRow({
    super.key,
    required this.icon,
    required this.label,
    required this.value,
    this.mono = false,
    this.copyable = false,
  });
  final IconData icon;
  final String label, value;
  final bool mono, copyable;
  @override
  Widget build(BuildContext context) {
    final c = context.colors;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 18, color: c.muted),
          const SizedBox(width: Space.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, style: context.text.bodySmall),
                const SizedBox(height: 2),
                SelectableText(
                  value,
                  style: context.text.bodyMedium?.copyWith(
                    fontFamily: mono ? monoFamily : null,
                  ),
                ),
              ],
            ),
          ),
          if (copyable)
            IconButton(
              tooltip: context.l.copy,
              icon: const Icon(Icons.content_copy_rounded, size: 18),
              onPressed: () => copyText(context, value),
            ),
        ],
      ),
    );
  }
}

Future<void> copyText(BuildContext context, String value) async {
  await Clipboard.setData(ClipboardData(text: value));
  if (context.mounted) {
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(context.l.copied)));
  }
}

Future<bool> confirmAction(
  BuildContext context,
  String title,
  String message,
  String action, {
  bool destructive = false,
}) async =>
    await showDialog<bool>(
      context: context,
      builder: (c) => AlertDialog(
        title: Text(title),
        content: Text(message),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(c, false),
            child: Text(c.l.cancel),
          ),
          FilledButton(
            style: destructive
                ? FilledButton.styleFrom(
                    backgroundColor: c.colors.danger,
                    foregroundColor: c.colors.background,
                    minimumSize: const Size(64, 48),
                  )
                : FilledButton.styleFrom(minimumSize: const Size(64, 48)),
            onPressed: () => Navigator.pop(c, true),
            child: Text(action),
          ),
        ],
      ),
    ) ??
    false;

/// Label for a usage window id, falling back to its duration.
String windowLabel(BuildContext context, UsageWindow w) {
  final l = context.l;
  switch (w.id) {
    case 'five_hour':
      return l.windowFiveHour;
    case 'weekly':
      return l.windowWeekly;
    case 'weekly_opus':
      return l.windowWeeklyModel('Opus');
    case 'weekly_sonnet':
      return l.windowWeeklyModel('Sonnet');
  }
  final m = w.windowMinutes;
  if (m == null || m <= 0) return w.id;
  if (m < 60) return '$m мин';
  if (m < 1440) return '${(m / 60).round()} ч';
  return '${(m / 1440).round()} дн';
}

/// One usage window: label, percentage, bar and reset time.
class UsageMeter extends StatelessWidget {
  const UsageMeter({super.key, required this.window, required this.now});
  final UsageWindow window;
  final DateTime now;
  @override
  Widget build(BuildContext context) {
    final c = context.colors, l = context.l;
    final p = window.usedPercent;
    final color = p >= 100
        ? c.danger
        : p >= 80
        ? c.warning
        : c.accent;
    final reset = window.resetsAt;
    return Semantics(
      container: true,
      label: '${windowLabel(context, window)}: ${l.usedPercent(p.round())}',
      child: ExcludeSemantics(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    windowLabel(context, window),
                    style: context.text.labelMedium?.copyWith(
                      color: c.secondary,
                    ),
                  ),
                ),
                Text(
                  l.usedPercent(p.round()),
                  style: context.text.labelMedium?.copyWith(
                    color: p >= 80 ? color : c.text,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 6),
            ClipRRect(
              borderRadius: BorderRadius.circular(3),
              child: LinearProgressIndicator(
                value: (p / 100).clamp(0, 1),
                minHeight: 6,
                color: color,
                backgroundColor: c.raised,
              ),
            ),
            if (reset != null) ...[
              const SizedBox(height: 6),
              Text(
                l.resetsAt(formatMoment(reset, now), formatUntil(reset, now)),
                style: context.text.bodySmall,
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// Initial-letter avatar.
class InitialAvatar extends StatelessWidget {
  const InitialAvatar(this.name, {super.key, this.size = 40});
  final String name;
  final double size;
  @override
  Widget build(BuildContext context) {
    final c = context.colors;
    final letter = name.trim().isEmpty
        ? '?'
        : name.trim().characters.first.toUpperCase();
    return ExcludeSemantics(
      child: Container(
        width: size,
        height: size,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: c.accentContainer,
          shape: BoxShape.circle,
        ),
        child: Text(
          letter,
          style: TextStyle(
            color: c.accent,
            fontWeight: FontWeight.w700,
            fontSize: size * 0.42,
          ),
        ),
      ),
    );
  }
}
