import 'package:flutter/material.dart';

import '../core/api.dart';
import '../core/format.dart';
import '../core/session.dart';
import '../core/theme.dart';
import '../core/version.dart';
import 'common.dart';

/// UI-17: owner, installation, device sessions, theme, version, logout.
class ProfileTab extends StatefulWidget {
  const ProfileTab({
    super.key,
    required this.session,
    required this.themeMode,
    required this.onTheme,
    required this.now,
  });
  final SessionController session;
  final ThemeMode themeMode;
  final ValueChanged<ThemeMode> onTheme;
  final DateTime Function() now;
  @override
  State<ProfileTab> createState() => _ProfileTabState();
}

class _ProfileTabState extends State<ProfileTab> {
  Json? _me;
  List<Json>? _devices;
  ApiFailure? _failure;
  final Set<String> _revoking = {};
  final Map<String, String> _revokeKeys = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final s = widget.session;
    try {
      final me = await s.request('GET', '/v1/me');
      final devices = await s.request('GET', '/v1/devices');
      if (!mounted) return;
      setState(() {
        _me = me;
        _devices = (devices['items'] as List? ?? const [])
            .whereType<Json>()
            .toList();
        _failure = null;
      });
    } on ApiFailure catch (e) {
      if (mounted) setState(() => _failure = e);
    }
  }

  Future<void> _revoke(Json device) async {
    final l = context.l, id = device['id'] as String;
    if (!await confirmAction(
      context,
      l.revokeTitle,
      l.revokeHint(device['label'] as String? ?? l.unknownDevice),
      l.revoke,
      destructive: true,
    )) {
      return;
    }
    setState(() => _revoking.add(id));
    try {
      await widget.session.request(
        'DELETE',
        '/v1/devices/${Uri.encodeComponent(id)}',
        key: _revokeKeys[id] ??= operationKey(),
      );
      _revokeKeys.remove(id);
      await _load();
    } on ApiFailure catch (e) {
      if (!e.uncertain) _revokeKeys.remove(id);
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(errorLabel(context, e))));
      }
    } finally {
      if (mounted) setState(() => _revoking.remove(id));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l,
        s = widget.session,
        c = context.colors,
        now = widget.now();
    final name = (_me?['name'] ?? s.owner?['name']) as String? ?? '';
    final email = (_me?['email'] ?? s.owner?['email']) as String? ?? '';
    final currentDevice = s.credentials?['device_id'];
    return Scaffold(
      appBar: AppBar(title: Text(l.profile)),
      body: PageBody(
        onRefresh: _load,
        children: [
          AppCard(
            child: Row(
              children: [
                InitialAvatar(name.isEmpty ? email : name, size: 56),
                const SizedBox(width: Space.lg),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        name.isEmpty ? l.ownerFallback : name,
                        style: context.text.titleLarge,
                      ),
                      const SizedBox(height: 2),
                      SelectableText(
                        email,
                        style: context.text.bodyMedium?.copyWith(
                          color: c.muted,
                        ),
                      ),
                      const SizedBox(height: Space.sm),
                      StatusBadge(
                        l.ownerRole,
                        tone: Tone.accent,
                        icon: Icons.verified_user_outlined,
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          if (_failure case final f?) ...[
            const SizedBox(height: Space.md),
            ErrorNotice(f, retry: _load),
          ],
          SectionHeader(l.connection),
          AppCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                MetaRow(
                  icon: Icons.dns_outlined,
                  label: l.installation,
                  value: s.name,
                ),
                MetaRow(
                  icon: Icons.link_rounded,
                  label: l.apiUrl,
                  value: s.baseUrl,
                  copyable: true,
                ),
                const SizedBox(height: Space.sm),
                Align(
                  alignment: Alignment.centerLeft,
                  child: TextButton.icon(
                    onPressed: () async {
                      if (await confirmAction(
                        context,
                        l.changeInstallationTitle,
                        l.changeConfirm,
                        l.continueAction,
                      )) {
                        await s.changeInstallation();
                      }
                    },
                    icon: const Icon(Icons.swap_horiz_rounded, size: 20),
                    label: Text(l.changeInstallation),
                  ),
                ),
              ],
            ),
          ),
          SectionHeader(l.deviceSessions, count: _devices?.length),
          AppCard(
            padding: EdgeInsets.zero,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                if (_devices == null && _failure == null)
                  Padding(
                    padding: const EdgeInsets.all(Space.lg),
                    child: LinearProgressIndicator(semanticsLabel: l.loading),
                  ),
                if (_devices?.isEmpty == true)
                  Padding(
                    padding: const EdgeInsets.all(Space.lg),
                    child: Text(l.noDevices),
                  ),
                for (final (i, d) in (_devices ?? const <Json>[]).indexed) ...[
                  if (i > 0) const Divider(),
                  _DeviceRow(
                    device: d,
                    current: d['id'] == currentDevice,
                    now: now,
                    busy: _revoking.contains(d['id']),
                    onRevoke: () => _revoke(d),
                  ),
                ],
              ],
            ),
          ),
          SectionHeader(l.theme),
          SegmentedButton<ThemeMode>(
            showSelectedIcon: false,
            segments: [
              ButtonSegment(
                value: ThemeMode.system,
                label: Text(l.systemTheme),
              ),
              ButtonSegment(value: ThemeMode.light, label: Text(l.lightTheme)),
              ButtonSegment(value: ThemeMode.dark, label: Text(l.darkTheme)),
            ],
            selected: {widget.themeMode},
            onSelectionChanged: (v) => widget.onTheme(v.first),
          ),
          SectionHeader(l.about),
          AppCard(
            child: Column(
              children: [
                MetaRow(
                  icon: Icons.info_outline_rounded,
                  label: l.appVersion,
                  value: appVersion,
                ),
                MetaRow(
                  icon: Icons.shield_outlined,
                  label: l.security,
                  value: l.securityValue,
                ),
              ],
            ),
          ),
          const SizedBox(height: Space.xl),
          OutlinedButton.icon(
            style: OutlinedButton.styleFrom(
              foregroundColor: c.danger,
              side: BorderSide(color: c.danger),
            ),
            onPressed: s.busy
                ? null
                : () async {
                    if (await confirmAction(
                      context,
                      l.logoutTitle,
                      l.logoutHint,
                      l.logout,
                      destructive: true,
                    )) {
                      await s.logout();
                    }
                  },
            icon: const Icon(Icons.logout_rounded, size: 20),
            label: Text(l.logout),
          ),
          const SizedBox(height: Space.sm),
          Text(
            l.logoutFootnote,
            textAlign: TextAlign.center,
            style: context.text.bodySmall,
          ),
        ],
      ),
    );
  }
}

class _DeviceRow extends StatelessWidget {
  const _DeviceRow({
    required this.device,
    required this.current,
    required this.now,
    required this.busy,
    required this.onRevoke,
  });
  final Json device;
  final bool current, busy;
  final DateTime now;
  final VoidCallback onRevoke;
  @override
  Widget build(BuildContext context) {
    final l = context.l, c = context.colors;
    final revoked = device['revoked_at'] != null;
    final seen = DateTime.tryParse(device['last_seen_at'] as String? ?? '');
    return Padding(
      padding: const EdgeInsets.fromLTRB(
        Space.lg,
        Space.md,
        Space.xs,
        Space.md,
      ),
      child: Row(
        children: [
          Icon(
            Icons.smartphone_rounded,
            size: 22,
            color: revoked ? c.disabled : (current ? c.accent : c.secondary),
          ),
          const SizedBox(width: Space.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  device['label'] as String? ?? l.unknownDevice,
                  style: context.text.bodyLarge?.copyWith(
                    color: revoked ? c.muted : c.text,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  revoked
                      ? l.deviceRevokedLabel
                      : seen == null
                      ? l.deviceNeverSeen
                      : l.deviceSeen(formatAgo(seen, now)),
                  style: context.text.bodySmall,
                ),
                if (current) ...[
                  const SizedBox(height: Space.xs),
                  StatusBadge(l.thisDevice, tone: Tone.accent),
                ],
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
          else if (!current && !revoked)
            IconButton(
              tooltip: l.revoke,
              onPressed: onRevoke,
              icon: Icon(Icons.phonelink_erase_rounded, color: c.danger),
            )
          else
            const SizedBox(width: 48),
        ],
      ),
    );
  }
}
