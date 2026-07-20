import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config/api_config.dart';
import '../models/camera.dart';
import '../models/stream_session.dart';
import '../models/truck.dart';

class ApiClient {
  ApiClient({
    String baseUrl = ApiConfig.defaultBaseUrl,
    this._apiToken = '',
    this.onUnauthorized,
  }) : baseUri = Uri.parse(baseUrl);

  final Uri baseUri;
  String _apiToken;
  final void Function()? onUnauthorized;

  set apiToken(String value) => _apiToken = value;

  Future<String> login({
    required String email,
    required String password,
  }) async {
    final response = await http.post(
      _uri('/api/auth/login'),
      headers: const {'Content-Type': 'application/json'},
      body: jsonEncode({'email': email, 'password': password}),
    );
    _ensureSuccess(response, notifyUnauthorized: false);
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final token = body['token'] as String? ?? '';
    if (token.isEmpty) {
      throw const ApiException(statusCode: 500, message: '登入回應缺少 token');
    }
    _apiToken = token;
    return token;
  }

  Future<void> logout() async {
    if (_apiToken.isEmpty) return;
    final response = await http.post(
      _uri('/api/auth/logout'),
      headers: _headers,
    );
    _apiToken = '';
    if (response.statusCode != 204 && response.statusCode != 401) {
      _ensureSuccess(response);
    }
  }

  Uri _uri(String path, [Map<String, String>? queryParameters]) {
    return baseUri.replace(path: path, queryParameters: queryParameters);
  }

  Map<String, String> get _headers {
    if (_apiToken.isEmpty) {
      return const {};
    }
    return {'Authorization': 'Bearer $_apiToken'};
  }

  Future<List<Truck>> getTrucks() async {
    final response = await http.get(_uri('/api/trucks'), headers: _headers);
    _ensureSuccess(response);

    final rows = jsonDecode(response.body) as List<dynamic>;
    return rows
        .map((row) => Truck.fromJson(row as Map<String, dynamic>))
        .toList(growable: false);
  }

  Future<List<Camera>> getCameras(String truckId) async {
    final response = await http.get(
      _uri('/api/trucks/$truckId/cameras'),
      headers: _headers,
    );
    _ensureSuccess(response);

    final rows = jsonDecode(response.body) as List<dynamic>;
    return rows
        .map((row) => Camera.fromJson(row as Map<String, dynamic>))
        .toList(growable: false);
  }

  Future<StreamSession> createPlaySession({
    required String truckId,
    required String cameraId,
  }) async {
    final response = await http.post(
      _uri('/api/trucks/$truckId/cameras/$cameraId/play'),
      headers: _headers,
    );
    _ensureSuccess(response);

    return StreamSession.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  void _ensureSuccess(
    http.Response response, {
    bool notifyUnauthorized = true,
  }) {
    if (response.statusCode < 200 || response.statusCode >= 300) {
      if (response.statusCode == 401 && notifyUnauthorized) {
        onUnauthorized?.call();
      }
      throw ApiException(
        statusCode: response.statusCode,
        message: response.body.isEmpty ? response.reasonPhrase : response.body,
      );
    }
  }
}

class ApiException implements Exception {
  const ApiException({required this.statusCode, this.message});

  final int statusCode;
  final String? message;

  @override
  String toString() => 'API $statusCode ${message ?? ''}'.trim();
}
