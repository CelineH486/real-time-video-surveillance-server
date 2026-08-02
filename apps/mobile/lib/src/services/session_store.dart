import 'package:flutter_secure_storage/flutter_secure_storage.dart';

abstract interface class SessionStore {
  Future<String?> readToken();
  Future<void> writeToken(String token);
  Future<void> deleteToken();
  Future<String?> readEmail();
  Future<void> writeEmail(String email);
}

class SecureSessionStore implements SessionStore {
  const SecureSessionStore();

  static const _storage = FlutterSecureStorage();
  static const _tokenKey = 'surveillance_api_token';
  static const _emailKey = 'surveillance_login_email';

  @override
  Future<String?> readToken() => _storage.read(key: _tokenKey);

  @override
  Future<void> writeToken(String token) =>
      _storage.write(key: _tokenKey, value: token);

  @override
  Future<void> deleteToken() => _storage.delete(key: _tokenKey);

  @override
  Future<String?> readEmail() => _storage.read(key: _emailKey);

  @override
  Future<void> writeEmail(String email) =>
      _storage.write(key: _emailKey, value: email);
}
