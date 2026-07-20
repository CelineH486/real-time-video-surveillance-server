import 'package:flutter_secure_storage/flutter_secure_storage.dart';

abstract interface class SessionStore {
  Future<String?> readToken();
  Future<void> writeToken(String token);
  Future<void> deleteToken();
}

class SecureSessionStore implements SessionStore {
  const SecureSessionStore();

  static const _storage = FlutterSecureStorage();
  static const _tokenKey = 'surveillance_api_token';

  @override
  Future<String?> readToken() => _storage.read(key: _tokenKey);

  @override
  Future<void> writeToken(String token) =>
      _storage.write(key: _tokenKey, value: token);

  @override
  Future<void> deleteToken() => _storage.delete(key: _tokenKey);
}
