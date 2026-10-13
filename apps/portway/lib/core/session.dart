import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';

import 'api.dart';
import 'storage.dart';

enum SessionPhase { restoring, installation, login, ready }

typedef ApiFactory = ApiClient Function(Uri);

class SessionController extends ChangeNotifier {
  SessionController(this.vault, {ApiFactory? apiFactory})
    : factory = apiFactory ?? ((uri) => ApiClient(uri));
  final Vault vault;
  final ApiFactory factory;
  SessionPhase phase = SessionPhase.restoring;
  ApiClient? api;
  Json? installation;
  Json? credentials;
  Json? owner;
  ApiFailure? failure;
  bool busy = false;
  bool logoutPending = false;
  int epoch = 0;
  Future<void>? _refreshing;
  String get name => installation?['name'] as String? ?? '';
  String get baseUrl => api?.base.toString() ?? '';

  /// The client-config was fetched in this app run (not restored from disk),
  /// so `setup_required` can be trusted to open registration first.
  bool configFresh = false;

  bool confirmationReceived = false;
  void emailConfirmed() {
    // A public callback is a navigation hint, never authentication evidence.
    confirmationReceived = true;
    notifyListeners();
  }

  Future<void> restore() async {
    try {
      final saved = await vault.read('installation');
      if (saved == null) {
        phase = SessionPhase.installation;
        return;
      }
      final data = jsonDecode(saved) as Json;
      api = factory(installationUri(data['url'] as String));
      installation = data['config'] as Json;
      final auth = await vault.read('session');
      if (auth != null) {
        final stored = jsonDecode(auth) as Json;
        if (stored['url'] == baseUrl &&
            stored['installation_id'] == installation?['installation_id']) {
          credentials = stored['credentials'] as Json;
          owner = stored['owner'] as Json;
        }
      }
      phase = credentials == null ? SessionPhase.login : SessionPhase.ready;
      if (phase == SessionPhase.login) unawaited(_refreshConfig());
      await retryLogout();
    } catch (_) {
      credentials = null;
      phase = api == null ? SessionPhase.installation : SessionPhase.login;
      failure = const ApiFailure('STORAGE');
    } finally {
      notifyListeners();
    }
  }

  /// Updates the cached public config (name, setup_required) of the same
  /// installation; any failure keeps the cached copy.
  Future<void> _refreshConfig() async {
    final client = api, generation = epoch;
    if (client == null) return;
    try {
      final config = await client.request('GET', '/v1/client-config');
      if (generation != epoch ||
          config['installation_id'] != installation?['installation_id'] ||
          config['name'] is! String) {
        return;
      }
      installation = config;
      configFresh = true;
      await vault.write(
        'installation',
        jsonEncode({'url': baseUrl, 'config': config}),
      );
      notifyListeners();
    } catch (_) {}
  }

  Future<void> connect(String url) async {
    if (busy) return;
    busy = true;
    failure = null;
    notifyListeners();
    ApiClient? candidate;
    try {
      candidate = factory(installationUri(url));
      final config = await candidate.request('GET', '/v1/client-config');
      if (config['api_version'] != 'v1' ||
          config['installation_id'] is! String ||
          config['name'] is! String) {
        throw const ApiFailure('INVALID_RESPONSE');
      }
      await vault.write(
        'installation',
        jsonEncode({'url': candidate.base.toString(), 'config': config}),
      );
      api?.close();
      api = candidate;
      candidate = null;
      installation = config;
      configFresh = true;
      phase = SessionPhase.login;
      epoch++;
    } on ApiFailure catch (e) {
      failure = e;
    } catch (_) {
      failure = const ApiFailure('STORAGE');
    } finally {
      candidate?.close();
      busy = false;
      notifyListeners();
    }
  }

  Future<void> login(String email, String password) async {
    if (busy) return;
    busy = true;
    failure = null;
    notifyListeners();
    try {
      final data = await api!.request(
        'POST',
        '/v1/auth/login',
        body: {
          'email': email.trim(),
          'password': password,
          'device_label': 'Portway Android',
        },
      );
      final session = data['session'] as Json;
      if (session['access_token'] is! String ||
          session['refresh_token'] is! String ||
          data['device_proof'] is! String) {
        throw const ApiFailure('INVALID_RESPONSE');
      }
      credentials = {
        'session': session,
        'device_proof': data['device_proof'],
        'device_id': data['device_id'],
      };
      owner = data['owner'] as Json;
      await _save();
      phase = SessionPhase.ready;
      epoch++;
    } on ApiFailure catch (e) {
      failure = e;
      credentials = null;
    } catch (_) {
      failure = const ApiFailure('STORAGE');
      credentials = null;
    } finally {
      busy = false;
      notifyListeners();
    }
  }

  Future<void> _save() => vault.write(
    'session',
    jsonEncode({
      'url': baseUrl,
      'installation_id': installation!['installation_id'],
      'credentials': credentials,
      'owner': owner,
    }),
  );
  Map<String, String> get authHeaders => {
    'Authorization':
        'Bearer ${(credentials!['session'] as Json)['access_token']}',
    'X-Device-Session': credentials!['device_proof'] as String,
  };

  Future<Json> request(
    String method,
    String path, {
    Json? body,
    String? key,
  }) async {
    final generation = epoch;
    if (credentials == null) throw const ApiFailure('SESSION_EXPIRED');
    final token = (credentials!['session'] as Json)['access_token'];
    try {
      final result = await api!.request(
        method,
        path,
        body: body,
        headers: {...authHeaders, 'Idempotency-Key': ?key},
      );
      if (generation != epoch) throw const ApiFailure('SESSION_EXPIRED');
      return result;
    } on ApiFailure catch (e) {
      if (generation != epoch) throw const ApiFailure('SESSION_EXPIRED');
      if (e.code == 'DEVICE_REVOKED' ||
          (e.status == 401 && e.code != 'TOKEN_EXPIRED')) {
        await invalidate(e);
        rethrow;
      }
      if (e.code != 'TOKEN_EXPIRED') rethrow;
      // Concurrent 401s share one refresh, including responses arriving after it finishes.
      if ((credentials!['session'] as Json)['access_token'] == token) {
        await refresh();
      }
      if (generation != epoch || credentials == null) {
        throw const ApiFailure('SESSION_EXPIRED');
      }
      try {
        final result = await api!.request(
          method,
          path,
          body: body,
          headers: {...authHeaders, 'Idempotency-Key': ?key},
        );
        if (generation != epoch) throw const ApiFailure('SESSION_EXPIRED');
        return result;
      } on ApiFailure catch (e) {
        if (e.status == 401 && generation == epoch) await invalidate(e);
        rethrow;
      }
    }
  }

  Future<void> refresh() =>
      _refreshing ??= _doRefresh().whenComplete(() => _refreshing = null);
  Future<void> _doRefresh() async {
    final generation = epoch;
    final current = credentials;
    if (current == null) throw const ApiFailure('SESSION_EXPIRED');
    try {
      final session = await api!.request(
        'POST',
        '/v1/auth/refresh',
        body: {'refresh_token': (current['session'] as Json)['refresh_token']},
        headers: {'X-Device-Session': current['device_proof'] as String},
      );
      if (generation != epoch) throw const ApiFailure('SESSION_EXPIRED');
      credentials = {...current, 'session': session};
      await _save();
    } catch (e) {
      if (generation == epoch) {
        await invalidate(
          e is ApiFailure ? e : const ApiFailure('SESSION_EXPIRED'),
        );
      }
      rethrow;
    }
  }

  Future<void> invalidate(ApiFailure reason) async {
    epoch++;
    credentials = null;
    owner = null;
    phase = SessionPhase.login;
    failure = reason;
    try {
      await vault.write('session', null);
    } finally {
      notifyListeners();
    }
  }

  Future<void> logout() async {
    if (busy) return;
    busy = true;
    try {
      final proof = credentials?['device_proof'];
      final origin = baseUrl;
      await invalidate(const ApiFailure('SIGNED_OUT'));
      if (proof != null) {
        final raw = await vault.read('logout');
        final queue = raw == null ? <dynamic>[] : jsonDecode(raw) as List;
        queue.add({'url': origin, 'proof': proof});
        await vault.write('logout', jsonEncode(queue));
      }
      await vault.write('pendingCodex', null);
      await retryLogout();
    } catch (_) {
      failure = const ApiFailure('STORAGE');
    } finally {
      busy = false;
      notifyListeners();
    }
  }

  Future<void> retryLogout() async {
    final pending = await vault.read('logout');
    logoutPending = pending != null;
    if (pending == null) return;
    final queue = jsonDecode(pending) as List;
    final remaining = <dynamic>[];
    for (final raw in queue) {
      final data = raw as Json;
      final client = factory(installationUri(data['url'] as String));
      try {
        await client.request(
          'POST',
          '/v1/auth/logout',
          headers: {'X-Device-Session': data['proof'] as String},
        );
      } on ApiFailure catch (e) {
        if (e.code != 'DEVICE_REVOKED') remaining.add(data);
      } finally {
        client.close();
      }
    }
    await vault.write(
      'logout',
      remaining.isEmpty ? null : jsonEncode(remaining),
    );
    logoutPending = remaining.isNotEmpty;
  }

  Future<void> changeInstallation() async {
    if (credentials != null) await logout();
    epoch++;
    api?.close();
    api = null;
    installation = null;
    await vault.write('installation', null);
    await vault.write('session', null);
    credentials = null;
    owner = null;
    failure = null;
    phase = SessionPhase.installation;
    notifyListeners();
  }

  @override
  void dispose() {
    api?.close();
    super.dispose();
  }
}
