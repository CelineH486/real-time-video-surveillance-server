import 'dart:async';

import 'package:flutter/material.dart';

import 'screens/camera_grid_screen.dart';
import 'screens/camera_live_screen.dart';
import 'screens/login_screen.dart';
import 'screens/truck_select_screen.dart';
import 'screens/truck_location_screen.dart';
import 'services/api_client.dart';
import 'services/session_store.dart';

class SurveillanceApp extends StatefulWidget {
  const SurveillanceApp({super.key, this.sessionStore, this.apiClient});

  final SessionStore? sessionStore;
  final ApiClient? apiClient;

  @override
  State<SurveillanceApp> createState() => _SurveillanceAppState();
}

class _SurveillanceAppState extends State<SurveillanceApp> {
  final _navigatorKey = GlobalKey<NavigatorState>();
  late final SessionStore _sessionStore;
  late final ApiClient _apiClient;
  bool _initializing = true;
  bool _authenticated = false;
  String _lastEmail = '';

  @override
  void initState() {
    super.initState();
    _sessionStore = widget.sessionStore ?? const SecureSessionStore();
    _apiClient = widget.apiClient ?? ApiClient(onUnauthorized: _expireSession);
    unawaited(_restoreSession());
  }

  Future<void> _restoreSession() async {
    final values = await Future.wait([
      _sessionStore.readToken(),
      _sessionStore.readEmail(),
    ]);
    final token = values[0];
    _lastEmail = values[1] ?? '';
    if (token != null && token.isNotEmpty) {
      _apiClient.apiToken = token;
      _authenticated = true;
    }
    if (mounted) setState(() => _initializing = false);
  }

  Future<void> _login(String email, String password) async {
    final token = await _apiClient.login(email: email, password: password);
    await Future.wait([
      _sessionStore.writeToken(token),
      _sessionStore.writeEmail(email),
    ]);
    if (mounted) {
      setState(() {
        _lastEmail = email;
        _authenticated = true;
      });
    }
  }

  Future<void> _logout() async {
    try {
      await _apiClient.logout();
    } finally {
      await _clearSession();
    }
  }

  void _expireSession() {
    unawaited(_clearSession());
  }

  Future<void> _clearSession() async {
    _apiClient.apiToken = '';
    await _sessionStore.deleteToken();
    if (!mounted) return;
    _navigatorKey.currentState?.popUntil((route) => route.isFirst);
    setState(() => _authenticated = false);
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      navigatorKey: _navigatorKey,
      title: '即時影像監控',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        brightness: Brightness.dark,
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF35E6A5),
          brightness: Brightness.dark,
          surface: const Color(0xFF111820),
        ),
        scaffoldBackgroundColor: const Color(0xFF090D12),
        cardTheme: const CardThemeData(
          color: Color(0xFF111820),
          margin: EdgeInsets.zero,
        ),
        appBarTheme: const AppBarTheme(
          backgroundColor: Color(0xFF090D12),
          foregroundColor: Colors.white,
          surfaceTintColor: Colors.transparent,
        ),
        inputDecorationTheme: const InputDecorationTheme(
          border: OutlineInputBorder(),
        ),
        useMaterial3: true,
      ),
      home: _initializing
          ? const Scaffold(body: Center(child: CircularProgressIndicator()))
          : _authenticated
          ? TruckSelectScreen(apiClient: _apiClient, onLogout: _logout)
          : LoginScreen(onLogin: _login, initialEmail: _lastEmail),
      onGenerateRoute: _authenticated ? _route : null,
    );
  }

  Route<dynamic> _route(RouteSettings settings) {
    final uri = Uri.parse(settings.name ?? '/');
    final segments = uri.pathSegments;
    Widget page = TruckSelectScreen(apiClient: _apiClient, onLogout: _logout);

    if (segments.length == 3 &&
        segments[0] == 'trucks' &&
        segments[2] == 'location') {
      page = TruckLocationScreen(
        apiClient: _apiClient,
        truckId: segments[1],
        onLogout: _logout,
      );
    } else if (segments.length == 3 &&
        segments[0] == 'trucks' &&
        segments[2] == 'cameras') {
      page = CameraGridScreen(
        apiClient: _apiClient,
        truckId: segments[1],
        onLogout: _logout,
      );
    } else if (segments.length == 4 &&
        segments[0] == 'trucks' &&
        segments[2] == 'cameras') {
      page = CameraLiveScreen(
        apiClient: _apiClient,
        truckId: segments[1],
        cameraId: segments[3],
        onLogout: _logout,
      );
    }
    return MaterialPageRoute(settings: settings, builder: (_) => page);
  }
}
