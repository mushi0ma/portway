import '../core/api.dart';

// Defensive DTOs for the laptop bridge API (`/v1/codex/*`, contract v2).
// Older servers omit providers/usage/login fields: a missing or malformed
// field means "no data yet", never a crash and never an invented value.

String? _str(Object? v) => v is String && v.isNotEmpty ? v : null;
bool _bool(Object? v) => v == true;
double? _num(Object? v) => v is num ? v.toDouble() : null;
int? _int(Object? v) => v is int ? v : (v is num ? v.toInt() : null);
DateTime? _date(Object? v) => v is String ? DateTime.tryParse(v) : null;
List<Json> _items(Object? v) =>
    v is List ? v.whereType<Map<String, dynamic>>().toList() : const [];
List<String> _strings(Object? v) =>
    v is List ? v.whereType<String>().toList() : const [];

/// Display names for known providers; the server name wins when present.
String providerTitle(String id, [String? serverName]) =>
    _str(serverName) ??
    switch (id) {
      'codex' => 'Codex',
      'claude' => 'Claude Code',
      _ => id,
    };

/// Service whose account the provider uses (for UI copy only).
String providerService(String id) => switch (id) {
  'codex' => 'ChatGPT',
  'claude' => 'Claude',
  _ => id,
};

class UsageWindow {
  const UsageWindow({
    required this.id,
    required this.usedPercent,
    this.windowMinutes,
    this.resetsAt,
  });
  factory UsageWindow.fromJson(Json j) => UsageWindow(
    id: _str(j['id']) ?? 'window',
    usedPercent: (_num(j['used_percent']) ?? 0).clamp(0, 100).toDouble(),
    windowMinutes: _int(j['window_minutes']),
    resetsAt: _date(j['resets_at']),
  );
  final String id;
  final double usedPercent;
  final int? windowMinutes;
  final DateTime? resetsAt;
  bool get exhausted => usedPercent >= 100;
}

enum UsageStatus { ok, warning, exhausted, unknown }

class ProviderUsage {
  const ProviderUsage({
    required this.status,
    this.checkedAt,
    this.resetAt,
    this.windows = const [],
  });
  static ProviderUsage? fromJson(Object? raw) {
    if (raw is! Json) return null;
    final windows = _items(raw['windows']).map(UsageWindow.fromJson).toList();
    var status = switch (raw['status']) {
      'ok' => UsageStatus.ok,
      'warning' => UsageStatus.warning,
      'exhausted' => UsageStatus.exhausted,
      _ => UsageStatus.unknown,
    };
    // A window at 100 % is exhausted even if an older bridge forgot the flag.
    if (status != UsageStatus.exhausted && windows.any((w) => w.exhausted)) {
      status = UsageStatus.exhausted;
    }
    return ProviderUsage(
      status: status,
      checkedAt: _date(raw['checked_at']),
      resetAt: _date(raw['reset_at']),
      windows: windows,
    );
  }

  final UsageStatus status;
  final DateTime? checkedAt;
  final DateTime? resetAt;
  final List<UsageWindow> windows;
  bool get exhausted => status == UsageStatus.exhausted;

  /// When usage becomes possible again: explicit `reset_at`, otherwise the
  /// latest reset among exhausted windows.
  DateTime? get availableAt {
    if (resetAt != null) return resetAt;
    DateTime? latest;
    for (final w in windows.where((w) => w.exhausted)) {
      if (w.resetsAt != null &&
          (latest == null || w.resetsAt!.isAfter(latest))) {
        latest = w.resetsAt;
      }
    }
    return latest;
  }
}

class AiAccount {
  const AiAccount({
    required this.id,
    required this.label,
    this.email,
    this.plan,
    this.loggedIn = false,
    this.active = false,
    this.system = false,
  });
  factory AiAccount.fromJson(Json j) => AiAccount(
    id: _str(j['id']) ?? '',
    label: _str(j['label']) ?? _str(j['email']) ?? 'Аккаунт',
    email: _str(j['email']),
    plan: _str(j['plan']),
    loggedIn: _bool(j['logged_in']),
    active: _bool(j['active']),
    system: _bool(j['system']),
  );
  final String id, label;
  final String? email, plan;
  final bool loggedIn, active, system;
}

class AiModel {
  const AiModel({
    required this.id,
    required this.name,
    this.description,
    this.isDefault = false,
    this.efforts = const [],
    this.defaultEffort,
  });
  factory AiModel.fromJson(Json j) => AiModel(
    id: _str(j['id']) ?? '',
    name: _str(j['name']) ?? _str(j['id']) ?? '',
    description: _str(j['description']),
    isDefault: _bool(j['default']),
    efforts: _strings(j['efforts']),
    defaultEffort: _str(j['default_effort']),
  );
  final String id, name;
  final String? description, defaultEffort;
  final bool isDefault;
  final List<String> efforts;
}

class AiProvider {
  const AiProvider({
    required this.id,
    required this.name,
    this.installed = false,
    this.cliVersion,
    this.activeAccountId,
    this.accounts = const [],
    this.models = const [],
    this.usage,
  });
  factory AiProvider.fromJson(Json j) {
    final id = _str(j['id']) ?? '';
    return AiProvider(
      id: id,
      name: providerTitle(id, j['name'] as String?),
      installed: _bool(j['installed']),
      cliVersion: _str(j['cli_version']),
      activeAccountId: _str(j['active_account_id']),
      accounts: _items(j['accounts'])
          .map(AiAccount.fromJson)
          .where((a) => a.id.isNotEmpty)
          .toList(),
      models: _items(j['models'])
          .map(AiModel.fromJson)
          .where((m) => m.id.isNotEmpty)
          .toList(),
      usage: ProviderUsage.fromJson(j['usage']),
    );
  }
  final String id, name;
  final bool installed;
  final String? cliVersion, activeAccountId;
  final List<AiAccount> accounts;
  final List<AiModel> models;
  final ProviderUsage? usage;

  AiAccount? get activeAccount {
    for (final a in accounts) {
      if (a.id == activeAccountId || (activeAccountId == null && a.active)) {
        return a;
      }
    }
    return null;
  }

  AiModel? get defaultModel {
    for (final m in models) {
      if (m.isDefault) return m;
    }
    return models.isEmpty ? null : models.first;
  }

  bool get exhausted => usage?.exhausted == true;
  bool get hasLoggedInAccount => activeAccount?.loggedIn == true;
}

enum LoginState {
  starting,
  waitingUser,
  verifying,
  completed,
  failed,
  cancelled,
  expired,
}

class LoginFlow {
  const LoginFlow({
    required this.id,
    required this.provider,
    required this.method,
    required this.state,
    this.url,
    this.userCode,
    this.needsCode = false,
    this.errorCode,
    this.accountId,
    this.expiresAt,
    this.updatedAt,
  });
  static LoginFlow? fromJson(Object? raw) {
    if (raw is! Json || _str(raw['id']) == null) return null;
    final url = _str(raw['url']);
    return LoginFlow(
      id: raw['id'] as String,
      provider: _str(raw['provider']) ?? 'codex',
      method: _str(raw['method']) ?? 'phone',
      state: switch (raw['state']) {
        'waiting_user' => LoginState.waitingUser,
        'verifying' => LoginState.verifying,
        'completed' => LoginState.completed,
        'failed' => LoginState.failed,
        'cancelled' => LoginState.cancelled,
        'expired' => LoginState.expired,
        _ => LoginState.starting,
      },
      url: url != null && isAllowedLoginUrl(url) ? url : null,
      userCode: _str(raw['user_code']),
      needsCode: _bool(raw['needs_code']),
      errorCode: _str(raw['error_code']),
      accountId: _str(raw['account_id']),
      expiresAt: _date(raw['expires_at']),
      updatedAt: _date(raw['updated_at']),
    );
  }

  final String id, provider, method;
  final LoginState state;
  final String? url, userCode, errorCode, accountId;
  final bool needsCode;
  final DateTime? expiresAt, updatedAt;
  bool get terminal =>
      state == LoginState.completed ||
      state == LoginState.failed ||
      state == LoginState.cancelled ||
      state == LoginState.expired;
}

const _loginHosts = {
  'auth.openai.com',
  'chatgpt.com',
  'claude.ai',
  'claude.com',
  'platform.claude.com',
  'console.anthropic.com',
};

/// The phone opens only HTTPS login pages of the known services, even if a
/// compromised bridge reported something else.
bool isAllowedLoginUrl(String raw) {
  final u = Uri.tryParse(raw);
  return u != null &&
      u.scheme == 'https' &&
      u.userInfo.isEmpty &&
      _loginHosts.contains(u.host);
}

class BridgeCommand {
  const BridgeCommand({
    required this.id,
    required this.kind,
    required this.state,
    this.errorCode,
  });
  factory BridgeCommand.fromJson(Json j) => BridgeCommand(
    id: _str(j['id']) ?? '',
    kind: _str(j['kind']) ?? '',
    state: _str(j['state']) ?? 'queued',
    errorCode: _str(j['error_code']),
  );
  final String id, kind, state;
  final String? errorCode;
  bool get pending => state == 'queued' || state == 'delivered';
}

class BridgeStatus {
  const BridgeStatus({
    required this.connected,
    this.workspace = 'portway',
    this.sandbox = 'read-only',
    this.lastSeenAt,
    this.stateUpdatedAt,
    this.providers = const [],
    this.login,
    this.commands = const [],
  });
  factory BridgeStatus.fromJson(Json j) => BridgeStatus(
    connected: _bool(j['connected']),
    workspace: _str(j['workspace']) ?? 'portway',
    sandbox: _str(j['sandbox']) ?? 'read-only',
    lastSeenAt: _date(j['last_seen_at']),
    stateUpdatedAt: _date(j['state_updated_at']),
    providers: _items(j['providers'])
        .map(AiProvider.fromJson)
        .where((p) => p.id.isNotEmpty)
        .toList(),
    login: LoginFlow.fromJson(j['login']),
    commands: _items(j['commands'])
        .map(BridgeCommand.fromJson)
        .where((c) => c.id.isNotEmpty)
        .toList(),
  );

  final bool connected;
  final String workspace, sandbox;
  final DateTime? lastSeenAt, stateUpdatedAt;
  final List<AiProvider> providers;
  final LoginFlow? login;
  final List<BridgeCommand> commands;

  /// The bridge has reported providers at least once (new API + bridge).
  bool get hasProviders => providers.isNotEmpty;
  AiProvider? provider(String id) {
    for (final p in providers) {
      if (p.id == id) return p;
    }
    return null;
  }

  BridgeCommand? command(String id) {
    for (final c in commands) {
      if (c.id == id) return c;
    }
    return null;
  }
}

class RunTokens {
  const RunTokens(this.input, this.output);
  final int? input, output;
}

class CodexRun {
  CodexRun(this.data);
  final Json data;
  String get id => data['id'] as String;
  String get state => _str(data['state']) ?? 'unknown';
  String get prompt => _str(data['prompt']) ?? '';
  String get answer => _str(data['answer']) ?? '';

  /// Short preview for lists; the full prompt is only returned by GET one.
  String get preview => _str(data['prompt_preview']) ?? prompt;
  String? get provider => _str(data['provider']);
  String? get model => _str(data['resolved_model']) ?? _str(data['model']);
  String? get effort => _str(data['effort']);
  String? get accountLabel => _str(data['account_label']);
  String? get errorCode => _str(data['error_code']);
  RunTokens? get tokens {
    final t = data['tokens'];
    if (t is! Json) return null;
    final input = _int(t['input']), output = _int(t['output']);
    return input == null && output == null ? null : RunTokens(input, output);
  }

  DateTime? get createdAt => _date(data['created_at']);
  DateTime? get startedAt => _date(data['started_at']);
  DateTime? get finishedAt => _date(data['finished_at']);
  Duration? get duration {
    final start = startedAt ?? createdAt, end = finishedAt;
    if (start == null || end == null || end.isBefore(start)) return null;
    return end.difference(start);
  }

  bool get active => state == 'queued' || state == 'running';
}
