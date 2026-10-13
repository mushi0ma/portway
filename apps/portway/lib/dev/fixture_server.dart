// In-process fixture server for the explicit demo build
// (`flutter build apk --dart-define=PORTWAY_FIXTURES=true`). It lets the
// owner and reviewers walk every screen of contract v2 before the new backend
// is deployed. The UI shows a permanent "ДЕМО" ribbon in this mode; nothing
// here is reachable from a normal build, and it never touches secure storage.

import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import '../core/api.dart';
import '../core/storage.dart';
import 'fixture_data.dart';

const fixtureMode = bool.fromEnvironment('PORTWAY_FIXTURES');

/// Volatile storage: the demo never writes the real secure storage.
class MemoryVault implements Vault {
  final values = <String, String>{};
  @override
  Future<String?> read(String key) async => values[key];
  @override
  Future<void> write(String key, String? value) async {
    if (value == null) {
      values.remove(key);
    } else {
      values[key] = value;
    }
  }
}

class FixtureServer {
  FixtureServer({DateTime Function()? clock}) : clock = clock ?? DateTime.now;
  final DateTime Function() clock;
  late final DateTime _start = clock();
  late final FixtureData _data = FixtureData(_start);

  final _runs = <Json>[];
  final _runStarted = <String, DateTime>{};
  final _commands = <Json>[];
  Json? _login;
  DateTime? _loginAt;
  bool _codeSent = false;
  late final List<Json> _providers = [
    _data.codexProvider(),
    _data.claudeProvider(),
  ];
  final _devices = <Json>[];
  int _seq = 0;

  http.Client client() => MockClient(_handle);

  String _id(String prefix) => '$prefix-${++_seq}';
  String _iso(DateTime t) => t.toUtc().toIso8601String();

  http.Response _ok(Object data, {int status = 200}) => http.Response.bytes(
    utf8.encode(
      jsonEncode({
        'data': data,
        'meta': {
          'request_id': '00000000-0000-4000-8000-000000000000',
          'server_time': _iso(clock()),
        },
      }),
    ),
    status,
    headers: {'content-type': 'application/json; charset=utf-8'},
  );

  http.Response _error(String code, int status, [Json? details]) =>
      http.Response.bytes(
        utf8.encode(
          jsonEncode({
            'error': {'code': code, 'message': code, 'details': ?details},
            'meta': {'request_id': '00000000-0000-4000-8000-000000000000'},
          }),
        ),
        status,
        headers: {'content-type': 'application/json; charset=utf-8'},
      );

  Future<http.Response> _handle(http.Request r) async {
    await Future<void>.delayed(const Duration(milliseconds: 250));
    if (_runs.isEmpty) {
      for (final run in _data.historyFixture().reversed) {
        _runs.insert(0, {...run, ..._fullFor(run)});
      }
      _devices.addAll([
        {
          'id': 'device-demo',
          'label': 'Portway Android',
          'last_seen_at': _iso(clock()),
          'revoked_at': null,
        },
        {
          'id': 'device-tablet',
          'label': 'Galaxy Tab',
          'last_seen_at': _iso(clock().subtract(const Duration(days: 1))),
          'revoked_at': null,
        },
      ]);
    }
    final path = r.url.path, m = r.method;
    final body = r.body.isEmpty
        ? <String, dynamic>{}
        : jsonDecode(r.body) as Json;
    switch ((m, path)) {
      case ('GET', '/v1/client-config'):
        return _ok({
          'api_version': 'v1',
          'installation_id': 'demo-installation',
          'name': 'Демо-компьютер',
          'setup_required': false,
        });
      case ('POST', '/v1/auth/login'):
        if (body['password'] != 'portway123') {
          return _error('INVALID_CREDENTIALS', 401);
        }
        return _ok({
          'session': {
            'access_token': 'demo-access',
            'refresh_token': 'demo-refresh',
          },
          'device_proof': 'demo-proof',
          'device_id': 'device-demo',
          'owner': {
            'id': 'owner-demo',
            'name': 'Алексей',
            'email': body['email'],
          },
        });
      case ('POST', '/v1/auth/register'):
        return _ok({
          'registration_id': '00000000-0000-4000-8000-000000000001',
          'state': 'pending_confirmation',
        }, status: 202);
      case ('POST', '/v1/auth/refresh'):
        return _ok({
          'access_token': 'demo-access',
          'refresh_token': 'demo-refresh',
        });
      case ('POST', '/v1/auth/logout'):
        return _ok({'device_revoked': true});
      case ('GET', '/v1/me'):
        return _ok({
          'id': 'owner-demo',
          'name': 'Алексей',
          'email': 'owner@example.test',
        });
      case ('GET', '/v1/devices'):
        return _ok({'items': _devices});
      case ('GET', '/v1/codex/status'):
        return _ok(_status());
      case ('GET', '/v1/codex/runs'):
        return _ok({
          'items': [
            for (final run in _runs)
              {..._advance(run)}
                ..remove('prompt')
                ..remove('answer'),
          ],
        });
      case ('POST', '/v1/codex/runs'):
        return _create(body);
      case ('POST', '/v1/codex/usage/refresh'):
        return _command('usage_refresh');
      case ('POST', '/v1/codex/accounts/login'):
        if (_login != null && !_terminal(_login!)) {
          return _error('AI_LOGIN_ACTIVE', 409);
        }
        final id = _id('login');
        _login = {
          'id': id,
          'provider': body['provider'],
          'method': body['method'],
          'state': 'starting',
          'url': null,
          'user_code': null,
          'needs_code': false,
          'error_code': null,
          'account_id': null,
          'expires_at': _iso(clock().add(const Duration(minutes: 15))),
          'updated_at': _iso(clock()),
        };
        _loginAt = clock();
        _codeSent = false;
        final c = _command('login_start');
        return _ok({
          'command_id': (jsonDecode(c.body) as Json)['data']['command_id'],
          'login_id': id,
        }, status: 202);
    }
    final segments = r.url.pathSegments;
    if (m == 'DELETE' && path.startsWith('/v1/devices/')) {
      for (final d in _devices) {
        if (d['id'] == segments.last) d['revoked_at'] = _iso(clock());
      }
      return _ok({'id': segments.last, 'revoked': true});
    }
    if (path.startsWith('/v1/codex/runs/')) {
      final run = _runs.where((x) => x['id'] == segments[3]).firstOrNull;
      if (run == null) return _error('NOT_FOUND', 404);
      if (segments.length == 4 && m == 'GET') return _ok(_advance(run));
      if (segments.length == 5 && segments[4] == 'events') {
        return _ok({'items': _events(_advance(run)), 'next_after': 0});
      }
      if (segments.length == 5 && segments[4] == 'stop') {
        run['state'] = 'stopped';
        run['finished_at'] = _iso(clock());
        return _ok(run, status: 202);
      }
    }
    if (path.startsWith('/v1/codex/accounts/')) {
      if (segments.length == 6 && segments[3] == 'login') {
        if (segments[5] == 'code') {
          _codeSent = true;
          _login?['state'] = 'verifying';
          _loginAt = clock();
          return _command('login_code');
        }
        if (segments[5] == 'cancel') {
          _login?['state'] = 'cancelled';
          return _command('login_cancel');
        }
      }
      final accountId = segments[3];
      final provider = m == 'DELETE'
          ? r.url.queryParameters['provider']
          : body['provider'];
      final p = _providers.where((x) => x['id'] == provider).firstOrNull;
      if (p == null) return _error('INVALID_REQUEST', 400);
      final accounts = (p['accounts'] as List).cast<Json>();
      if (m == 'DELETE') {
        final a = accounts.where((x) => x['id'] == accountId).firstOrNull;
        if (a?['system'] == true) return _error('AI_ACCOUNT_SYSTEM', 409);
        accounts.removeWhere((x) => x['id'] == accountId);
        return _command('account_remove');
      }
      if (segments.length == 5 && segments[4] == 'activate') {
        for (final a in accounts) {
          a['active'] = a['id'] == accountId;
        }
        p['active_account_id'] = accountId;
        if (provider == 'codex') {
          _setCodexExhausted(accountId == 'codex-system');
        }
        return _command('account_activate');
      }
    }
    return _error('NOT_FOUND', 404);
  }

  void _setCodexExhausted(bool exhausted) {
    final i = _providers.indexWhere((p) => p['id'] == 'codex');
    final fresh = FixtureData(clock()).codexProvider(exhausted: exhausted);
    _providers[i] = {..._providers[i], 'usage': fresh['usage']};
  }

  http.Response _command(String kind) {
    final id = _id('cmd');
    _commands.insert(0, {
      'id': id,
      'kind': kind,
      'state': 'done',
      'error_code': null,
      'updated_at': _iso(clock()),
    });
    if (_commands.length > 10) _commands.removeLast();
    return _ok({'command_id': id}, status: 202);
  }

  bool _terminal(Json login) => const {
    'completed',
    'failed',
    'cancelled',
    'expired',
  }.contains(login['state']);

  Json _status() {
    final login = _login;
    if (login != null && !_terminal(login)) {
      final age = clock().difference(_loginAt!);
      if (login['state'] == 'starting' && age > const Duration(seconds: 2)) {
        login['state'] = 'waiting_user';
        if (login['method'] == 'phone') {
          if (login['provider'] == 'codex') {
            login['url'] = 'https://auth.openai.com/codex/device';
            login['user_code'] = 'K7QM-4XPD';
          } else {
            login['url'] = 'https://claude.com/cai/oauth/authorize?code=true';
            login['needs_code'] = true;
          }
        }
      } else if (login['state'] == 'verifying' &&
          _codeSent &&
          age > const Duration(seconds: 3)) {
        login['state'] = 'completed';
        login['account_id'] = 'claude-demo';
      }
    }
    return {
      ..._data.statusFixture(),
      'last_seen_at': _iso(clock()),
      'providers': _providers,
      'login': login,
      'commands': _commands,
    };
  }

  http.Response _create(Json body) {
    final provider = body['provider'] as String? ?? 'codex';
    final p = _providers.where((x) => x['id'] == provider).firstOrNull;
    final usage = p?['usage'] as Json?;
    if (usage?['status'] == 'exhausted') {
      return _error('AI_LIMIT_EXHAUSTED', 429, {
        'provider': provider,
        'reset_at': usage?['reset_at'],
      });
    }
    final id = _id('run');
    final prompt = body['prompt'] as String;
    final run = {
      'id': id,
      'state': 'queued',
      'workspace': 'portway',
      'sandbox': 'read-only',
      'provider': provider,
      'model':
          body['model'] ?? (provider == 'codex' ? 'gpt-6.1-sol' : 'sonnet'),
      'effort': body['effort'],
      'resolved_model': null,
      'account_label': provider == 'codex' ? 'ChatGPT' : 'Claude',
      'tokens': null,
      'prompt': prompt,
      'prompt_preview': prompt.length > 120 ? prompt.substring(0, 120) : prompt,
      'answer': null,
      'error_code': null,
      'created_at': _iso(clock()),
      'started_at': null,
      'finished_at': null,
    };
    _runs.insert(0, run);
    _runStarted[id] = clock();
    return _ok(run, status: 202);
  }

  Json _fullFor(Json run) => {
    'prompt': '${run['prompt_preview']}. Ничего не изменяй.',
    if (run['state'] == 'completed') 'answer': _answer,
  };

  /// Moves a fresh demo run through queued → running → completed.
  Json _advance(Json run) {
    final started = _runStarted[run['id']];
    if (started == null || run['state'] == 'stopped') return run;
    final age = clock().difference(started);
    if (age > const Duration(seconds: 1) && run['state'] == 'queued') {
      run['state'] = 'running';
      run['started_at'] = _iso(started.add(const Duration(seconds: 1)));
    }
    if (age > const Duration(seconds: 7) && run['state'] == 'running') {
      run['state'] = 'completed';
      run['finished_at'] = _iso(clock());
      run['answer'] = _answer;
      run['resolved_model'] = run['model'];
      run['tokens'] = {'input': 18342, 'output': 1206};
    }
    return run;
  }

  List<Json> _events(Json run) {
    final started = _runStarted[run['id']];
    final age = started == null
        ? const Duration(minutes: 5)
        : clock().difference(started);
    final all = <Json>[
      {'type': 'started'},
      {'type': 'progress', 'text': 'Читаю README.md и docs/README.md'},
      {
        'type': 'progress',
        'text': 'Просматриваю apps/portway/lib и cmd/portway',
      },
      {'type': 'progress', 'text': 'Составляю краткий ответ'},
      if (run['state'] == 'completed') {'type': 'completed'},
      if (run['state'] == 'failed')
        {'type': 'failed', 'code': run['error_code']},
      if (run['state'] == 'stopped') {'type': 'stopped'},
    ];
    final visible = started == null
        ? all.length
        : (age.inMilliseconds ~/ 1500 + 1).clamp(0, all.length);
    return [
      for (var i = 0; i < visible; i++) {'sequence': i + 1, ...all[i]},
    ];
  }

  static const _answer = '''## Назначение

Portway позволяет управлять **ИИ-агентами на компьютере** с телефона: отправлять задачи Codex или Claude Code, следить за ходом и читать ответы.

### Основные части

1. `apps/portway` — Android-клиент на Flutter.
2. `cmd/portway` — Go API и мост на ноутбуке.
3. `docs/` — спецификации и протоколы проверок.

```bash
cd apps/portway
flutter test
```

> Проект открыт только на чтение: файлы не изменялись.''';
}
