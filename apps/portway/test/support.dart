import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:portway/core/api.dart';
import 'package:portway/core/session.dart';
import 'package:portway/core/storage.dart';

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

http.Response ok(Json data, {int status = 200}) => http.Response(
  jsonEncode({
    'data': data,
    'meta': {
      'server_time': '2026-10-13T06:00:00Z',
      'request_id': 'fixture-request',
    },
  }),
  status,
  headers: {'content-type': 'application/json'},
);
http.Response errorResponse(String code, int status) => http.Response(
  jsonEncode({
    'error': {'code': code},
    'meta': {},
  }),
  status,
);
SessionController authenticated(
  Future<http.Response> Function(http.Request) handler, {
  MemoryVault? vault,
}) {
  final s = SessionController(
    vault ?? MemoryVault(),
    apiFactory: (u) => ApiClient(u, client: MockClient(handler)),
  );
  s.api = s.factory(Uri.parse('https://fixture.example'));
  s.installation = {
    'installation_id': 'fixture-installation',
    'name': 'Рабочий ПК',
    'api_version': 'v1',
    'setup_required': false,
  };
  s.credentials = {
    'device_id': 'fixture-device',
    'device_proof': 'fixture-proof',
    'session': {
      'access_token': 'fixture-old',
      'refresh_token': 'fixture-refresh',
    },
  };
  s.owner = {'name': 'Разработчик', 'email': 'owner@example.test'};
  s.phase = SessionPhase.ready;
  return s;
}

/// Records requests and answers by "METHOD /path" (query ignored) or path.
class FakeServer {
  FakeServer(this.routes);
  final Map<String, Object Function(http.Request r)> routes;
  final requests = <http.Request>[];
  Future<http.Response> call(http.Request r) async {
    requests.add(r);
    final handler = routes['${r.method} ${r.url.path}'] ?? routes[r.url.path];
    if (handler == null) return errorResponse('NOT_FOUND', 404);
    final value = handler(r);
    if (value is http.Response) return value;
    return ok(value as Json);
  }

  List<http.Request> calls(String method, String path) =>
      requests.where((r) => r.method == method && r.url.path == path).toList();
}

http.Response problem(String code, int status, {Json? details}) =>
    http.Response(
      jsonEncode({
        'error': {'code': code, 'message': code, 'details': ?details},
        'meta': {'request_id': 'fixture-request'},
      }),
      status,
    );
