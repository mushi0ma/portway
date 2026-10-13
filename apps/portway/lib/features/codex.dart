import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';

import '../core/api.dart';
import '../core/session.dart';
import 'bridge_models.dart';

export 'bridge_models.dart';

/// HTTP boundary of the laptop bridge API. Screens and controllers depend on
/// this class only, so a fake can replace it in tests or before a backend
/// rollout; it never fabricates server results itself.
class CodexRepository {
  CodexRepository(this.session);
  final SessionController session;

  Future<BridgeStatus> status() async =>
      BridgeStatus.fromJson(await session.request('GET', '/v1/codex/status'));
  Future<List<CodexRun>> list() async {
    final data = await session.request('GET', '/v1/codex/runs');
    final items = data['items'];
    return items is List
        ? items
              .whereType<Json>()
              .where((e) => e['id'] is String)
              .map(CodexRun.new)
              .toList()
        : <CodexRun>[];
  }

  Future<CodexRun> get(String id) async => CodexRun(
    await session.request('GET', '/v1/codex/runs/${Uri.encodeComponent(id)}'),
  );
  Future<Json> events(String id, int after) => session.request(
    'GET',
    '/v1/codex/runs/${Uri.encodeComponent(id)}/events?after=$after',
  );
  Future<Json> create(Json body, String key) =>
      session.request('POST', '/v1/codex/runs', body: body, key: key);
  Future<Json> stop(String id, String key) => session.request(
    'POST',
    '/v1/codex/runs/${Uri.encodeComponent(id)}/stop',
    body: {},
    key: key,
  );

  Future<Json> refreshUsage(String key) =>
      session.request('POST', '/v1/codex/usage/refresh', body: {}, key: key);
  Future<Json> startLogin(String provider, String method, String key) =>
      session.request(
        'POST',
        '/v1/codex/accounts/login',
        body: {'provider': provider, 'method': method},
        key: key,
      );
  Future<Json> submitLoginCode(String loginId, String code, String key) =>
      session.request(
        'POST',
        '/v1/codex/accounts/login/${Uri.encodeComponent(loginId)}/code',
        body: {'code': code},
        key: key,
      );
  Future<Json> cancelLogin(String loginId, String key) => session.request(
    'POST',
    '/v1/codex/accounts/login/${Uri.encodeComponent(loginId)}/cancel',
    body: {},
    key: key,
  );
  Future<Json> activateAccount(String accountId, String provider, String key) =>
      session.request(
        'POST',
        '/v1/codex/accounts/${Uri.encodeComponent(accountId)}/activate',
        body: {'provider': provider},
        key: key,
      );
  Future<Json> removeAccount(String accountId, String provider, String key) =>
      session.request(
        'DELETE',
        '/v1/codex/accounts/${Uri.encodeComponent(accountId)}'
            '?provider=${Uri.encodeQueryComponent(provider)}',
        key: key,
      );
}

/// Shared bridge state for the signed-in shell: status, recent runs, usage
/// refresh and account commands. Polls every 10 s, every 2 s while a command
/// or login is pending; pauses in background.
class BridgeController extends ChangeNotifier {
  BridgeController(this.repository, {DateTime Function()? now})
    : _now = now ?? DateTime.now;
  final CodexRepository repository;
  final DateTime Function() _now;

  BridgeStatus? status;
  List<CodexRun> runs = const [];
  ApiFailure? failure;
  bool loading = true;
  DateTime? checkedAt;
  String preferredProvider = 'codex';

  /// null = unknown, false = server predates usage refresh (old API).
  bool? usageRefreshSupported;
  // Command id -> when this phone issued it. Forgotten after 2 minutes so a
  // command missing from the status window cannot force fast polling forever.
  final Map<String, DateTime> _tracked = {};
  final Map<String, String> _keys = {};
  DateTime? _lastUsageRefresh;
  bool _foreground = true, _disposed = false, _refreshing = false;
  Timer? _timer;

  DateTime now() => _now();

  Future<void> start() async {
    try {
      final saved = await repository.session.vault.read('aiProvider');
      if (saved == 'codex' || saved == 'claude') preferredProvider = saved!;
    } catch (_) {}
    await refresh(usage: true);
  }

  Future<void> setPreferredProvider(String id) async {
    if (preferredProvider == id) return;
    preferredProvider = id;
    _notify();
    try {
      await repository.session.vault.write('aiProvider', id);
    } catch (_) {}
  }

  bool get pendingWork {
    final s = status;
    if (s == null) return false;
    if (s.login != null && !s.login!.terminal) return true;
    final now = _now();
    for (final e in _tracked.entries) {
      if (now.difference(e.value) > const Duration(minutes: 2)) continue;
      final c = s.command(e.key);
      if (c == null || c.pending) return true;
    }
    return false;
  }

  Future<void> refresh({bool usage = false}) async {
    if (_disposed) return;
    if (usage) unawaited(requestUsageRefresh());
    if (_refreshing) return;
    _refreshing = true;
    _timer?.cancel();
    try {
      final next = await repository.status();
      final nextRuns = await repository.list();
      if (_disposed) return;
      status = next;
      runs = nextRuns;
      failure = null;
      checkedAt = _now();
      final now = _now();
      _tracked.removeWhere((id, at) {
        final c = next.command(id);
        if (now.difference(at) > const Duration(minutes: 2)) {
          _accountCommands.remove(id);
          return true;
        }
        if (c == null || c.pending) return false;
        _accountCommands.remove(id);
        if (!_results.isClosed) _results.add(c);
        return true;
      });
    } on ApiFailure catch (e) {
      if (!_disposed) failure = e;
    } finally {
      _refreshing = false;
      loading = false;
      _notify();
      _schedule();
    }
  }

  void _schedule() {
    _timer?.cancel();
    if (_disposed || !_foreground) return;
    _timer = Timer(
      pendingWork ? const Duration(seconds: 2) : const Duration(seconds: 10),
      refresh,
    );
  }

  /// Asks the bridge to re-read usage (no inference cost). Throttled; silently
  /// disabled on servers that predate the endpoint.
  Future<void> requestUsageRefresh({bool force = false}) async {
    if (usageRefreshSupported == false || _disposed) return;
    final last = _lastUsageRefresh;
    if (!force &&
        last != null &&
        _now().difference(last) < const Duration(seconds: 30)) {
      return;
    }
    _lastUsageRefresh = _now();
    try {
      final data = await repository.refreshUsage(operationKey());
      usageRefreshSupported = true;
      final id = data['command_id'];
      if (id is String) _tracked[id] = _now();
      _schedule();
    } on ApiFailure catch (e) {
      if (e.unsupported) usageRefreshSupported = false;
    }
  }

  /// Runs one account mutation. The same logical operation reuses its
  /// idempotency key after an uncertain transport failure.
  Future<ApiFailure?> _command(
    String signature,
    Future<Json> Function(String key) call, {
    void Function(Json data)? onAccepted,
  }) async {
    final key = _keys[signature] ??= operationKey();
    try {
      final data = await call(key);
      _keys.remove(signature);
      final id = data['command_id'];
      if (id is String) _tracked[id] = _now();
      onAccepted?.call(data);
      unawaited(refresh());
      return null;
    } on ApiFailure catch (e) {
      if (!e.uncertain) _keys.remove(signature);
      return e;
    }
  }

  String? lastLoginId;
  Future<ApiFailure?> startLogin(String provider, String method) => _command(
    'login:$provider:$method',
    (k) => repository.startLogin(provider, method, k),
    onAccepted: (d) => lastLoginId = d['login_id'] as String?,
  );
  Future<ApiFailure?> submitLoginCode(String loginId, String code) => _command(
    'code:$loginId:${code.hashCode}',
    (k) => repository.submitLoginCode(loginId, code, k),
  );
  Future<ApiFailure?> cancelLogin(String loginId) =>
      _command('cancel:$loginId', (k) => repository.cancelLogin(loginId, k));

  /// True while a command this phone issued for [accountId] is unfinished.
  bool accountBusy(String accountId) =>
      _inflight.contains(accountId) ||
      _accountCommands.entries.any(
        (e) => e.value == accountId && _tracked.containsKey(e.key),
      );
  final Set<String> _inflight = {};
  final Map<String, String> _accountCommands = {};

  Future<ApiFailure?> activateAccount(String provider, String accountId) =>
      _forAccount(
        accountId,
        'activate:$provider:$accountId',
        (k) => repository.activateAccount(accountId, provider, k),
      );
  Future<ApiFailure?> removeAccount(String provider, String accountId) =>
      _forAccount(
        accountId,
        'remove:$provider:$accountId',
        (k) => repository.removeAccount(accountId, provider, k),
      );
  Future<ApiFailure?> _forAccount(
    String accountId,
    String signature,
    Future<Json> Function(String key) call,
  ) async {
    _inflight.add(accountId);
    _notify();
    final result = await _command(
      signature,
      call,
      onAccepted: (d) {
        final id = d['command_id'];
        if (id is String) _accountCommands[id] = accountId;
      },
    );
    _inflight.remove(accountId);
    _notify();
    return result;
  }

  /// Finished commands issued from this phone (for one-shot feedback).
  final _results = StreamController<BridgeCommand>.broadcast();
  Stream<BridgeCommand> get commandResults => _results.stream;

  void setForeground(bool value) {
    if (_foreground == value) return;
    _foreground = value;
    _timer?.cancel();
    if (value) refresh(usage: true);
  }

  void _notify() {
    if (!_disposed) notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    _timer?.cancel();
    _results.close();
    super.dispose();
  }
}

/// One explicit pending operation. No automatic replay of a task on reconnect.
class TaskComposer extends ChangeNotifier {
  TaskComposer(this.repository);
  final CodexRepository repository;
  bool _disposed = false;
  @override
  void notifyListeners() {
    if (!_disposed) super.notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }

  Json? pending;
  bool busy = false;
  ApiFailure? failure;
  Json? get pendingBody => pending?['body'] as Json?;

  Future<void> restore() async {
    final saved = await repository.session.vault.read('pendingCodex');
    if (saved != null) {
      final data = jsonDecode(saved) as Json;
      if (data['url'] == repository.session.baseUrl &&
          data['device'] == repository.session.credentials?['device_id']) {
        pending = data;
      }
    }
    notifyListeners();
  }

  void clearFailure() {
    if (failure == null) return;
    failure = null;
    notifyListeners();
  }

  /// Sends a new task, or explicitly retries the stored pending one with the
  /// original body and key. `provider/model/effort` are sent only when given
  /// (older servers reject unknown fields).
  Future<String?> send(
    String prompt, {
    String? provider,
    String? model,
    String? effort,
  }) async {
    if (busy) return null;
    if (pending == null &&
        (prompt.trim().isEmpty || utf8.encode(prompt).length > 16384)) {
      failure = const ApiFailure('INVALID_PROMPT');
      notifyListeners();
      return null;
    }
    if (pending != null) {
      final created = DateTime.tryParse(
        pending!['created_at'] as String? ?? '',
      );
      if (created == null ||
          DateTime.now().toUtc().difference(created) >=
              const Duration(days: 6)) {
        failure = const ApiFailure('REQUEST_EXPIRED');
        notifyListeners();
        return null;
      }
    }
    busy = true;
    failure = null;
    notifyListeners();
    try {
      pending ??= {
        'created_at': DateTime.now().toUtc().toIso8601String(),
        'url': repository.session.baseUrl,
        'device': repository.session.credentials?['device_id'],
        'key': operationKey(),
        'body': {
          'prompt': prompt.trim(),
          'workspace': 'portway',
          'provider': ?provider,
          'model': ?model,
          'effort': ?effort,
        },
      };
      await repository.session.vault.write('pendingCodex', jsonEncode(pending));
      final data = await repository.create(
        pending!['body'] as Json,
        pending!['key'] as String,
      );
      pending = null;
      await repository.session.vault.write('pendingCodex', null);
      return data['id'] as String;
    } on ApiFailure catch (e) {
      failure = e;
      if (!e.uncertain ||
          e.code == 'CODEX_OFFLINE' ||
          e.code == 'CODEX_DISABLED') {
        pending = null;
        await repository.session.vault.write('pendingCodex', null);
      }
      return null;
    } catch (_) {
      failure = const ApiFailure('STORAGE');
      return null;
    } finally {
      busy = false;
      notifyListeners();
    }
  }

  /// Drops a stored uncertain request after the owner checked the history.
  Future<void> discardPending() async {
    pending = null;
    failure = null;
    await repository.session.vault.write('pendingCodex', null);
    notifyListeners();
  }
}

class RunController extends ChangeNotifier {
  RunController(this.repository, this.id);
  final CodexRepository repository;
  final String id;
  CodexRun? run;
  final List<Json> events = [];
  ApiFailure? failure;
  bool busy = false, stopping = false, foreground = true, _disposed = false;
  int _after = 0;
  String? _stopKey;
  Timer? _timer;
  Future<void> refresh() async {
    if (busy || _disposed || !foreground) return;
    busy = true;
    _timer?.cancel();
    try {
      final next = await repository.get(id);
      run = next;
      for (var page = 0; page < 3; page++) {
        final batch = await repository.events(id, _after);
        if (_disposed) return;
        for (final raw in batch['items'] as List) {
          final event = raw as Json;
          final seq = event['sequence'] as int;
          if (seq <= _after) continue;
          if (seq != _after + 1) throw const ApiFailure('EVENT_GAP');
          events.add(event);
          _after = seq;
        }
        if ((batch['items'] as List).length < 100 || _after >= 256) break;
      }
      // Server protocol is capped at 256 events; retain no unbounded feed.
      if (events.length > 256) events.removeRange(0, events.length - 256);
      failure = null;
    } on ApiFailure catch (e) {
      if (!_disposed) failure = e;
    } finally {
      busy = false;
      if (!_disposed) {
        notifyListeners();
        if (foreground && (run == null || run!.active || failure != null)) {
          _timer = Timer(Duration(seconds: failure == null ? 2 : 5), refresh);
        }
      }
    }
  }

  Future<void> stop() async {
    if (stopping || _disposed) return;
    stopping = true;
    notifyListeners();
    try {
      await repository.stop(id, _stopKey ??= operationKey());
      _stopKey = null;
      await refresh();
    } on ApiFailure catch (e) {
      failure = e;
    } finally {
      stopping = false;
      if (!_disposed) notifyListeners();
    }
  }

  void setForeground(bool value) {
    foreground = value;
    _timer?.cancel();
    if (value) refresh();
  }

  @override
  void dispose() {
    _disposed = true;
    _timer?.cancel();
    super.dispose();
  }
}
