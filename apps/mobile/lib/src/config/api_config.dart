import 'package:flutter/foundation.dart';

class ApiConfig {
  static const _configuredBaseUrl = String.fromEnvironment('API_BASE_URL');

  static String get defaultBaseUrl {
    if (_configuredBaseUrl.isNotEmpty) return _configuredBaseUrl;
    if (kIsWeb) return Uri.base.origin;
    return 'http://localhost:8080';
  }
}
