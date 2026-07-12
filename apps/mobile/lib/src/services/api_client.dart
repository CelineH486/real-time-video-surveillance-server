import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config/api_config.dart';
import '../models/camera.dart';
import '../models/stream_session.dart';
import '../models/truck.dart';

class ApiClient {
  ApiClient({
    String baseUrl = ApiConfig.defaultBaseUrl,
    String apiToken = ApiConfig.defaultApiToken,
  })  : baseUri = Uri.parse(baseUrl),
        _apiToken = apiToken;

  final Uri baseUri;
  final String _apiToken;

  Uri _uri(String path, [Map<String, String>? queryParameters]) {
    return baseUri.replace(
      path: path,
      queryParameters: queryParameters,
    );
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

  void _ensureSuccess(http.Response response) {
    if (response.statusCode < 200 || response.statusCode >= 300) {
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
