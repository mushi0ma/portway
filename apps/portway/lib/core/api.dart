import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:http/http.dart' as http;

typedef Json = Map<String, dynamic>;

class ApiFailure implements Exception {
  const ApiFailure(this.code, {this.status = 0, this.requestId, this.details});
  final String code;
  final int status;
  final String? requestId;

  /// Optional typed details from the problem envelope, e.g. the provider and
  /// `reset_at` of `AI_LIMIT_EXHAUSTED`. Never contains secrets.
  final Json? details;
  bool get uncertain => status == 0 || status >= 500;

  /// The route or operation does not exist on this server version.
  bool get unsupported => status == 404 || status == 405 || code == 'NOT_FOUND';
  @override
  String toString() => 'ApiFailure($code, $status)';
}

String operationKey() {
  final random = Random.secure();
  final bytes = List.generate(16, (_) => random.nextInt(256));
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  final hex = bytes.map((b) => b.toRadixString(16).padLeft(2, '0')).join();
  return '${hex.substring(0, 8)}-${hex.substring(8, 12)}-${hex.substring(12, 16)}-${hex.substring(16, 20)}-${hex.substring(20)}';
}

Uri installationUri(String input) {
  final u = Uri.tryParse(input.trim());
  if (u == null ||
      u.scheme != 'https' ||
      u.host.isEmpty ||
      u.userInfo.isNotEmpty ||
      u.hasQuery ||
      u.hasFragment ||
      (u.path.isNotEmpty && u.path != '/') ||
      u.host == 'localhost' ||
      u.host.endsWith('.localhost') ||
      u.host == '::1' ||
      u.host == '0.0.0.0' ||
      u.host.startsWith('127.')) {
    throw const ApiFailure('INVALID_URL');
  }
  return u.replace(path: '');
}

class ApiClient {
  ApiClient(this.base, {http.Client? client})
    : client = client ?? http.Client();
  final Uri base;
  final http.Client client;
  DateTime? serverTime;
  Future<Json> request(
    String method,
    String path, {
    Json? body,
    Map<String, String> headers = const {},
  }) async {
    try {
      final request = http.Request(method, base.resolve(path))
        ..followRedirects = false
        ..headers.addAll({'Accept': 'application/json', ...headers});
      if (body != null) {
        request.headers['Content-Type'] = 'application/json';
        request.body = jsonEncode(body);
      }
      final response = await client
          .send(request)
          .timeout(const Duration(seconds: 20));
      final bytes = <int>[];
      await for (final chunk in response.stream.timeout(
        const Duration(seconds: 20),
      )) {
        bytes.addAll(chunk);
        if (bytes.length > 2 * 1024 * 1024) {
          throw const ApiFailure('INVALID_RESPONSE');
        }
      }
      final value = jsonDecode(utf8.decode(bytes)) as Json;
      final meta = value['meta'] as Json?;
      serverTime = DateTime.tryParse(meta?['server_time'] as String? ?? '');
      if (response.statusCode < 200 || response.statusCode >= 300) {
        final error = value['error'] is Json ? value['error'] as Json : null;
        throw ApiFailure(
          error?['code'] is String ? error!['code'] as String : 'SERVER_ERROR',
          status: response.statusCode,
          requestId: meta?['request_id'] as String?,
          details: error?['details'] is Json ? error!['details'] as Json : null,
        );
      }
      return value['data'] as Json;
    } on ApiFailure {
      rethrow;
    } on TimeoutException {
      throw const ApiFailure('NETWORK');
    } on http.ClientException {
      throw const ApiFailure('NETWORK');
    } on FormatException {
      throw const ApiFailure('INVALID_RESPONSE');
    } on TypeError {
      throw const ApiFailure('INVALID_RESPONSE');
    }
  }

  void close() => client.close();
}
