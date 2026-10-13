import 'package:flutter_secure_storage/flutter_secure_storage.dart';

abstract interface class Vault {
  Future<String?> read(String key);
  Future<void> write(String key, String? value);
}

final class SecureVault implements Vault {
  const SecureVault();
  static const _storage = FlutterSecureStorage();
  @override
  Future<String?> read(String key) => _storage.read(key: key);
  @override
  Future<void> write(String key, String? value) => value == null
      ? _storage.delete(key: key)
      : _storage.write(key: key, value: value);
}
