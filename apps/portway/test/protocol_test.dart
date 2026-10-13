import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'dart:io';

import 'package:intl/date_symbol_data_local.dart';
import 'package:portway/core/api.dart';
import 'package:portway/core/format.dart';
import 'package:portway/core/session.dart';
import 'package:portway/core/version.dart';
import 'package:portway/features/codex.dart';
import 'package:portway/features/tasks_screen.dart';

import 'fixtures.dart';
import 'support.dart';

/// Formatting uses non-breaking spaces between numbers and units.
String n(String s) => s.replaceAll('\u00a0', ' ');

void main() {
  setUpAll(() => initializeDateFormatting('ru'));
  test('installation accepts HTTPS origin only', () {
    for (final input in [
      'http://server.test',
      'https://localhost',
      'https://127.0.0.1',
      'https://user:pass@server.test',
      'https://server.test/v1',
      'https://server.test?a=b',
      'https://server.test#token',
    ]) {
      expect(() => installationUri(input), throwsA(isA<ApiFailure>()));
    }
    expect(
      installationUri(' https://server.test:8443/ ').toString(),
      'https://server.test:8443',
    );
  });
  test(
    'API does not follow credential redirects or expose raw error text',
    () async {
      final api = ApiClient(
        Uri.parse('https://fixture.example'),
        client: MockClient((r) async {
          expect(r.followRedirects, false);
          return http.Response('secret upstream body', 502);
        }),
      );
      await expectLater(
        api.request('POST', '/v1/auth/login', body: {'password': 'fixture'}),
        throwsA(
          isA<ApiFailure>().having((e) => e.code, 'code', 'INVALID_RESPONSE'),
        ),
      );
    },
  );
  test(
    'expired concurrent requests share one refresh and preserve mutation key',
    () async {
      int refreshes = 0;
      final s = authenticated((r) async {
        if (r.url.path.endsWith('refresh')) {
          refreshes++;
          await Future<void>.delayed(const Duration(milliseconds: 15));
          return ok({'access_token': 'new', 'refresh_token': 'new-refresh'});
        }
        expect(r.headers['X-Device-Session'], 'fixture-proof');
        expect(r.headers['Idempotency-Key'], 'same-key');
        return r.headers['Authorization'] == 'Bearer fixture-old'
            ? errorResponse('TOKEN_EXPIRED', 401)
            : ok({'accepted': true});
      });
      final result = await Future.wait([
        s.request('POST', '/resource', body: {}, key: 'same-key'),
        s.request('POST', '/resource', body: {}, key: 'same-key'),
      ]);
      expect(refreshes, 1);
      expect(result.every((e) => e['accepted'] == true), true);
    },
  );
  test('ambiguous refresh clears session and never retries token', () async {
    int refreshes = 0;
    final s = authenticated((r) async {
      if (r.url.path.endsWith('refresh')) {
        refreshes++;
        throw http.ClientException('fixture');
      }
      return errorResponse('TOKEN_EXPIRED', 401);
    });
    await expectLater(
      s.request('GET', '/resource'),
      throwsA(isA<ApiFailure>()),
    );
    expect(refreshes, 1);
    expect(s.credentials, null);
    expect(s.phase, SessionPhase.login);
  });
  test('late successful request cannot cross logout generation', () async {
    final response = Completer<http.Response>();
    final s = authenticated((_) => response.future);
    final request = s.request('GET', '/resource');
    await s.invalidate(const ApiFailure('SIGNED_OUT'));
    response.complete(ok({'old': 'data'}));
    await expectLater(
      request,
      throwsA(
        isA<ApiFailure>().having((e) => e.code, 'code', 'SESSION_EXPIRED'),
      ),
    );
  });
  test('revoked device closes session and removes persistent auth', () async {
    final v = MemoryVault()..values['session'] = 'fixture';
    final s = authenticated(
      (_) async => errorResponse('DEVICE_REVOKED', 401),
      vault: v,
    );
    await expectLater(
      s.request('GET', '/resource'),
      throwsA(isA<ApiFailure>()),
    );
    expect(s.phase, SessionPhase.login);
    expect(v.values.containsKey('session'), false);
  });
  test('lost create persists original body and key for explicit retry after restart', () async {
    final requests = <http.Request>[];
    final vault = MemoryVault();
    final s = authenticated((r) async {
      requests.add(r);
      if (requests.length == 1) throw http.ClientException('lost response');
      return ok({'id': 'accepted', 'state': 'queued'}, status: 202);
    }, vault: vault);
    final composer = TaskComposer(CodexRepository(s));
    expect(await composer.send('Исследовать проект'), null);
    expect(composer.pending, isNotNull);
    final restored = TaskComposer(CodexRepository(s));
    await restored.restore();
    expect(
      await restored.send('another body must not replace original'),
      'accepted',
    );
    expect(requests[0].body, requests[1].body);
    expect(
      requests[0].headers['Idempotency-Key'],
      requests[1].headers['Idempotency-Key'],
    );
    expect(vault.values.containsKey('pendingCodex'), false);
  });
  test('definitive busy response unlocks composer', () async {
    final c = TaskComposer(
      CodexRepository(
        authenticated((_) async => errorResponse('CODEX_BUSY', 409)),
      ),
    );
    await c.send('test');
    expect(c.pending, null);
    expect(c.failure?.code, 'CODEX_BUSY');
  });
  test('pending command from a different device is not restored', () async {
    final v = MemoryVault()
      ..values['pendingCodex'] = jsonEncode({
        'url': 'https://fixture.example',
        'device': 'other',
        'body': {'prompt': 'test'},
        'key': 'fixture',
      });
    final c = TaskComposer(
      CodexRepository(authenticated((_) async => ok({}), vault: v)),
    );
    await c.restore();
    expect(c.pending, null);
  });
  test(
    'offline logout removes auth but retains revoke-only tombstone',
    () async {
      final v = MemoryVault()..values['session'] = 'fixture';
      final s = authenticated(
        (_) async => throw http.ClientException('offline'),
        vault: v,
      );
      await s.logout();
      expect(s.credentials, null);
      expect(v.values['session'], null);
      expect(s.logoutPending, true);
      final tombstone = jsonDecode(v.values['logout']!) as List;
      expect((tombstone.single as Map).keys, unorderedEquals(['url', 'proof']));
    },
  );
  test('logout retry deletes tombstone after ACK', () async {
    final v = MemoryVault()
      ..values['logout'] = jsonEncode([
        {'url': 'https://fixture.example', 'proof': 'fixture'},
      ]);
    final s = authenticated((r) async {
      expect(r.headers.containsKey('Authorization'), false);
      return ok({'device_revoked': true});
    }, vault: v);
    await s.retryLogout();
    expect(v.values['logout'], null);
  });
  test(
    'event duplicate is ignored and foreground pause stops polling',
    () async {
      int events = 0;
      final s = authenticated((r) async {
        if (r.url.path.endsWith('/events')) {
          events++;
          return ok({
            'items': [
              {'sequence': 1, 'type': 'progress', 'text': 'reading'},
            ],
            'next_after': 1,
          });
        }
        return ok({'id': 'run', 'state': 'running', 'prompt': 'question'});
      });
      final c = RunController(CodexRepository(s), 'run');
      await c.refresh();
      await c.refresh();
      expect(c.events.length, 1);
      c.setForeground(false);
      await c.refresh();
      expect(events, 2);
      c.dispose();
    },
  );
  test('event gap is explicit and cursor is not advanced', () async {
    final s = authenticated(
      (r) async => r.url.path.endsWith('/events')
          ? ok({
              'items': [
                {'sequence': 2, 'type': 'progress'},
              ],
              'next_after': 2,
            })
          : ok({'id': 'run', 'state': 'running'}),
    );
    final c = RunController(CodexRepository(s), 'run');
    await c.refresh();
    expect(c.failure?.code, 'EVENT_GAP');
    expect(c.events, isEmpty);
    c.dispose();
  });

  group('bridge contract v2', () {
    test('full status parses providers, usage, login and commands', () {
      final s = BridgeStatus.fromJson(
        statusFixture(
          login: {
            'id': 'login-1',
            'provider': 'codex',
            'method': 'phone',
            'state': 'waiting_user',
            'url': 'https://auth.openai.com/codex/device',
            'user_code': 'ABCD-EFGH',
            'needs_code': false,
            'expires_at': iso(fixtureNow.add(const Duration(minutes: 15))),
          },
          commands: [
            {'id': 'cmd-1', 'kind': 'usage_refresh', 'state': 'delivered'},
          ],
        ),
      );
      expect(s.connected, true);
      expect(s.hasProviders, true);
      final codex = s.provider('codex')!;
      expect(codex.exhausted, true);
      expect(codex.activeAccount?.email, 'owner@example.test');
      expect(codex.defaultModel?.id, 'gpt-6.1-sol');
      expect(codex.defaultModel?.efforts, contains('xhigh'));
      expect(codex.usage!.windows.first.usedPercent, 100);
      expect(
        codex.usage!.availableAt!.toLocal(),
        fixtureNow.add(const Duration(hours: 3, minutes: 40)),
      );
      final claude = s.provider('claude')!;
      expect(claude.exhausted, false);
      expect(claude.usage!.windows[1].usedPercent, 18.5);
      expect(s.login!.state, LoginState.waitingUser);
      expect(s.login!.userCode, 'ABCD-EFGH');
      expect(s.login!.terminal, false);
      expect(s.command('cmd-1')!.pending, true);
    });

    test('old API status means "no data yet", not a failure', () {
      final s = BridgeStatus.fromJson(statusFixture(legacy: true));
      expect(s.connected, true);
      expect(s.hasProviders, false);
      expect(s.login, null);
      expect(s.commands, isEmpty);
    });

    test('malformed fields are ignored and percentages clamped', () {
      final s = BridgeStatus.fromJson({
        'connected': 'yes',
        'providers': [
          {
            'id': 'codex',
            'installed': true,
            'accounts': 'broken',
            'models': [
              {'name': 'no id'},
            ],
            'usage': {
              'status': 'ok',
              'windows': [
                {'id': 'mystery', 'used_percent': 140, 'window_minutes': 90},
                'garbage',
              ],
            },
          },
          {'name': 'missing id'},
        ],
        'login': {'state': 'waiting_user'},
        'commands': [42],
      });
      expect(s.connected, false);
      expect(s.providers.single.accounts, isEmpty);
      expect(s.providers.single.models, isEmpty);
      final usage = s.providers.single.usage!;
      expect(usage.windows.single.usedPercent, 100);
      // A full window is exhausted even if the status says ok.
      expect(usage.status, UsageStatus.exhausted);
      expect(s.login, null);
      expect(s.commands, isEmpty);
    });

    test('login URLs outside the allowlist are never opened', () {
      for (final url in [
        'http://auth.openai.com/x',
        'https://evil.example/codex',
        'https://user@claude.ai/x',
        'javascript:alert(1)',
      ]) {
        expect(isAllowedLoginUrl(url), false, reason: url);
        final flow = LoginFlow.fromJson({
          'id': 'l',
          'state': 'waiting_user',
          'url': url,
        });
        expect(flow!.url, null);
      }
      expect(isAllowedLoginUrl('https://claude.com/cai/oauth/authorize'), true);
    });

    test('run DTO exposes preview, meta, tokens and duration', () {
      final r = CodexRun(runFixture(full: true));
      expect(r.preview, startsWith('Кратко'));
      expect(r.prompt, endsWith('Ничего не изменяй.'));
      expect(r.provider, 'codex');
      expect(r.model, 'gpt-6.1-sol');
      expect(r.tokens!.input, 12840);
      expect(r.duration, const Duration(minutes: 1));
      final old = CodexRun({'id': 'x', 'state': 'completed'});
      expect(old.preview, '');
      expect(old.tokens, null);
      expect(old.provider, null);
    });

    test('problem details survive the envelope', () async {
      final api = ApiClient(
        Uri.parse('https://fixture.example'),
        client: MockClient(
          (_) async => problem(
            'AI_LIMIT_EXHAUSTED',
            429,
            details: {'provider': 'codex', 'reset_at': '2026-10-13T14:06:48Z'},
          ),
        ),
      );
      await expectLater(
        api.request('POST', '/v1/codex/runs', body: {}),
        throwsA(
          isA<ApiFailure>()
              .having((e) => e.code, 'code', 'AI_LIMIT_EXHAUSTED')
              .having((e) => e.uncertain, 'uncertain', false)
              .having((e) => e.details?['provider'], 'provider', 'codex'),
        ),
      );
    });

    test('composer sends provider, model and effort only when given', () async {
      final server = FakeServer({
        'POST /v1/codex/runs': (_) =>
            ok({'id': 'new', 'state': 'queued'}, status: 202),
      });
      final composer = TaskComposer(
        CodexRepository(authenticated(server.call)),
      );
      expect(
        await composer.send('Вопрос', provider: 'claude', model: 'sonnet'),
        'new',
      );
      expect(await composer.send('Старый сервер'), 'new');
      final bodies = server.requests
          .map((r) => jsonDecode(r.body) as Map)
          .toList();
      expect(bodies[0], {
        'prompt': 'Вопрос',
        'workspace': 'portway',
        'provider': 'claude',
        'model': 'sonnet',
      });
      expect(bodies[1].keys, unorderedEquals(['prompt', 'workspace']));
    });

    test(
      'limit exhausted from POST is definitive and unlocks composer',
      () async {
        final composer = TaskComposer(
          CodexRepository(
            authenticated(
              (_) async => problem(
                'AI_LIMIT_EXHAUSTED',
                429,
                details: {'provider': 'codex'},
              ),
            ),
          ),
        );
        expect(await composer.send('test', provider: 'codex'), null);
        expect(composer.failure?.code, 'AI_LIMIT_EXHAUSTED');
        expect(composer.pending, null);
      },
    );

    test('usage refresh is throttled and disabled on old servers', () async {
      final server = FakeServer({
        'GET /v1/codex/status': (_) => statusFixture(legacy: true),
        'GET /v1/codex/runs': (_) => {'items': []},
      });
      final b = BridgeController(
        CodexRepository(authenticated(server.call)),
        now: () => fixtureNow,
      );
      await b.requestUsageRefresh();
      expect(b.usageRefreshSupported, false);
      await b.requestUsageRefresh(force: true);
      expect(server.calls('POST', '/v1/codex/usage/refresh').length, 1);
      b.dispose();
    });

    test('account commands use the contract paths, bodies and keys', () async {
      final server = FakeServer({
        'GET /v1/codex/status': (_) => statusFixture(),
        'GET /v1/codex/runs': (_) => {'items': []},
        'POST /v1/codex/usage/refresh': (_) =>
            ok({'command_id': 'c0'}, status: 202),
        'POST /v1/codex/accounts/login': (_) =>
            ok({'command_id': 'c1', 'login_id': 'login-9'}, status: 202),
        'POST /v1/codex/accounts/login/login-9/code': (_) =>
            ok({'command_id': 'c2'}, status: 202),
        'POST /v1/codex/accounts/login/login-9/cancel': (_) =>
            ok({'command_id': 'c3'}, status: 202),
        'POST /v1/codex/accounts/codex-work/activate': (_) =>
            ok({'command_id': 'c4'}, status: 202),
        'DELETE /v1/codex/accounts/codex-work': (_) =>
            ok({'command_id': 'c5'}, status: 202),
      });
      final b = BridgeController(
        CodexRepository(authenticated(server.call)),
        now: () => fixtureNow,
      );
      expect(await b.startLogin('codex', 'phone'), null);
      expect(b.lastLoginId, 'login-9');
      expect(await b.submitLoginCode('login-9', 'secret-code'), null);
      expect(await b.cancelLogin('login-9'), null);
      expect(await b.activateAccount('codex', 'codex-work'), null);
      expect(await b.removeAccount('codex', 'codex-work'), null);
      expect(b.usageRefreshSupported, isNot(false));
      final login = server.calls('POST', '/v1/codex/accounts/login').single;
      expect(jsonDecode(login.body), {'provider': 'codex', 'method': 'phone'});
      expect(
        jsonDecode(
          server
              .calls('POST', '/v1/codex/accounts/login/login-9/code')
              .single
              .body,
        ),
        {'code': 'secret-code'},
      );
      expect(
        jsonDecode(
          server
              .calls('POST', '/v1/codex/accounts/codex-work/activate')
              .single
              .body,
        ),
        {'provider': 'codex'},
      );
      final remove = server
          .calls('DELETE', '/v1/codex/accounts/codex-work')
          .single;
      expect(remove.url.queryParameters['provider'], 'codex');
      final mutations = server.requests.where((r) => r.method != 'GET');
      expect(
        mutations.every(
          (r) => (r.headers['Idempotency-Key'] ?? '').length >= 32,
        ),
        true,
      );
      // Issued commands stay pending until status reports them done.
      expect(b.pendingWork, true);
      b.dispose();
    });

    test(
      'uncertain account command reuses its key; definitive one does not',
      () async {
        var calls = 0;
        final keys = <String?>[];
        final b = BridgeController(
          CodexRepository(
            authenticated((r) async {
              if (r.method == 'GET') {
                return ok(
                  r.url.path.endsWith('runs') ? {'items': []} : statusFixture(),
                );
              }
              keys.add(r.headers['Idempotency-Key']);
              calls++;
              if (calls == 1) throw http.ClientException('lost');
              if (calls == 2) return ok({'command_id': 'c1'}, status: 202);
              return problem('AI_ACCOUNT_SYSTEM', 409);
            }),
          ),
          now: () => fixtureNow,
        );
        expect(
          (await b.activateAccount('codex', 'codex-work'))?.code,
          'NETWORK',
        );
        expect(await b.activateAccount('codex', 'codex-work'), null);
        expect(keys[0], keys[1]);
        expect(
          (await b.removeAccount('codex', 'codex-system'))?.code,
          'AI_ACCOUNT_SYSTEM',
        );
        b.dispose();
      },
    );

    test('finished tracked command is reported once', () async {
      var done = false;
      final b = BridgeController(
        CodexRepository(
          authenticated((r) async {
            if (r.method == 'POST') {
              return ok({'command_id': 'c7'}, status: 202);
            }
            if (r.url.path.endsWith('runs')) return ok({'items': []});
            return ok(
              statusFixture(
                commands: [
                  {
                    'id': 'c7',
                    'kind': 'account_activate',
                    'state': done ? 'done' : 'delivered',
                  },
                ],
              ),
            );
          }),
        ),
        now: () => fixtureNow,
      );
      final results = <BridgeCommand>[];
      final sub = b.commandResults.listen(results.add);
      await b.activateAccount('codex', 'codex-work');
      await b.refresh();
      expect(b.accountBusy('codex-work'), true);
      done = true;
      await b.refresh();
      await b.refresh();
      await Future<void>.delayed(Duration.zero);
      expect(results.map((c) => c.state), ['done']);
      expect(b.accountBusy('codex-work'), false);
      await sub.cancel();
      b.dispose();
    });
  });

  group('session state machine', () {
    test(
      'login success stores host-bound session; wrong password does not',
      () async {
        final vault = MemoryVault();
        var attempt = 0;
        final s = authenticated((r) async {
          attempt++;
          if (attempt == 1) return problem('INVALID_CREDENTIALS', 401);
          return ok({
            'session': {'access_token': 'a', 'refresh_token': 'b'},
            'device_proof': 'proof',
            'device_id': 'dev-1',
            'owner': {
              'id': 'o',
              'name': 'Разработчик',
              'email': 'owner@example.test',
            },
          });
        }, vault: vault);
        s.credentials = null;
        s.phase = SessionPhase.login;
        await s.login('owner@example.test', 'wrong');
        expect(s.phase, SessionPhase.login);
        expect(s.failure?.code, 'INVALID_CREDENTIALS');
        expect(vault.values.containsKey('session'), false);
        final epoch = s.epoch;
        await s.login(' owner@example.test ', 'right');
        expect(s.phase, SessionPhase.ready);
        expect(s.epoch, epoch + 1);
        final saved = jsonDecode(vault.values['session']!) as Map;
        expect(saved['url'], 'https://fixture.example');
        expect((saved['credentials'] as Map)['device_id'], 'dev-1');
      },
    );

    test(
      'restore re-enters a saved session for the same installation',
      () async {
        final vault = MemoryVault()
          ..values['installation'] = jsonEncode({
            'url': 'https://fixture.example',
            'config': {
              'installation_id': 'i1',
              'name': 'ПК',
              'api_version': 'v1',
            },
          })
          ..values['session'] = jsonEncode({
            'url': 'https://fixture.example',
            'installation_id': 'i1',
            'credentials': {
              'device_id': 'd',
              'device_proof': 'p',
              'session': {'access_token': 'a', 'refresh_token': 'r'},
            },
            'owner': {'name': 'Разработчик'},
          });
        final s = SessionController(
          vault,
          apiFactory: (u) =>
              ApiClient(u, client: MockClient((_) async => ok({}))),
        );
        await s.restore();
        expect(s.phase, SessionPhase.ready);
        // A session saved for another installation is not reused.
        vault.values['installation'] = jsonEncode({
          'url': 'https://fixture.example',
          'config': {
            'installation_id': 'other',
            'name': 'ПК',
            'api_version': 'v1',
          },
        });
        final t = SessionController(
          vault,
          apiFactory: (u) =>
              ApiClient(u, client: MockClient((_) async => ok({}))),
        );
        await t.restore();
        expect(t.phase, SessionPhase.login);
      },
    );

    test('logout returns to login and protected requests fail', () async {
      final s = authenticated((_) async => ok({'device_revoked': true}));
      await s.logout();
      expect(s.phase, SessionPhase.login);
      expect(s.failure?.code, 'SIGNED_OUT');
      await expectLater(
        s.request('GET', '/v1/codex/status'),
        throwsA(
          isA<ApiFailure>().having((e) => e.code, 'code', 'SESSION_EXPIRED'),
        ),
      );
    });
  });

  group('formatting and history filter', () {
    test('Russian relative and absolute times', () {
      expect(
        n(formatSpan(const Duration(hours: 3, minutes: 40))),
        '3 ч 40 мин',
      );
      expect(n(formatSpan(const Duration(minutes: 12))), '12 мин');
      expect(n(formatSpan(const Duration(days: 2, hours: 4))), '2 дн 4 ч');
      expect(formatSpan(const Duration(seconds: 20)), 'меньше минуты');
      expect(
        n(
          formatUntil(
            fixtureNow.add(const Duration(hours: 3, minutes: 40)),
            fixtureNow,
          ),
        ),
        'через 3 ч 40 мин',
      );
      expect(formatMoment(DateTime(2026, 10, 13, 14, 6), fixtureNow), '14:06');
      expect(
        formatMoment(DateTime(2026, 10, 14, 9, 5), fixtureNow),
        'завтра, 09:05',
      );
      expect(
        formatAgo(fixtureNow.subtract(const Duration(minutes: 5)), fixtureNow),
        '5 мин назад',
      );
      expect(
        formatElapsed(const Duration(minutes: 3, seconds: 5)),
        '3 мин 05 с',
      );
    });

    test('history search and filters', () {
      final runs = historyFixture().map(CodexRun.new).toList();
      expect(filterRuns(runs, query: 'readme').single.id, 'run-2');
      expect(filterRuns(runs, query: 'SONNET').single.id, 'run-1');
      expect(filterRuns(runs, state: StateFilter.active).single.id, 'run-3');
      expect(filterRuns(runs, state: StateFilter.failed).single.id, 'run-2');
      expect(filterRuns(runs, provider: 'claude').single.id, 'run-1');
      expect(
        filterRuns(runs, provider: 'codex', state: StateFilter.completed),
        isEmpty,
      );
      // Old API rows without provider count as Codex.
      expect(
        filterRuns([
          CodexRun({'id': 'x', 'state': 'completed'}),
        ], provider: 'codex').length,
        1,
      );
    });

    test('displayed version matches pubspec', () {
      final line = File('pubspec.yaml')
          .readAsLinesSync()
          .firstWhere((l) => l.startsWith('version:'));
      final v = line.split(':').last.trim().split('+');
      expect(appVersion, '${v[0]} (${v[1]})');
    });
  });
}
