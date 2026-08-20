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
  const SurveillanceApp({
    super.key,
    this.sessionStore,
    this.apiClient,
    this.initialRoute,
  });

  final SessionStore? sessionStore;
  final ApiClient? apiClient;
  final String? initialRoute;

  @override
  State<SurveillanceApp> createState() => _SurveillanceAppState();
}

class _SurveillanceAppState extends State<SurveillanceApp> {
  static const _sessionStoreTimeout = Duration(seconds: 2);

  GlobalKey<NavigatorState> _navigatorKey = GlobalKey<NavigatorState>();
  late final SessionStore _sessionStore;
  late final ApiClient _apiClient;
  late final String _initialRoute;
  bool _initializing = true;
  bool _authenticated = false;
  String _lastEmail = '';

  @override
  void initState() {
    super.initState();
    _sessionStore = widget.sessionStore ?? const SecureSessionStore();
    _apiClient = widget.apiClient ?? ApiClient(onUnauthorized: _expireSession);
    _initialRoute = _resolveInitialRoute();
    unawaited(_restoreSession());
  }

  String _resolveInitialRoute() {
    final configuredRoute = widget.initialRoute;
    if (configuredRoute != null) return configuredRoute;

    final platformRoute =
        WidgetsBinding.instance.platformDispatcher.defaultRouteName;
    if (platformRoute.isNotEmpty && platformRoute != '/') return platformRoute;

    // Flutter Web keeps named routes after the hash when using its default URL
    // strategy. Uri.base still contains that value after a full page refresh.
    final fragment = Uri.base.fragment;
    if (fragment.startsWith('/')) return Uri.parse(fragment).path;
    return '/';
  }

  Future<void> _restoreSession() async {
    List<String?> values;
    try {
      values = await Future.wait([
        _sessionStore.readToken(),
        _sessionStore.readEmail(),
      ]).timeout(_sessionStoreTimeout);
    } catch (error) {
      debugPrint('Restore session storage unavailable: $error');
      values = const [null, null];
    }
    final token = values[0];
    _lastEmail = values[1] ?? '';
    if (token != null && token.isNotEmpty) {
      _apiClient.apiToken = token;
      _authenticated = true;
    }
    if (mounted) {
      setState(() {
        _navigatorKey = GlobalKey<NavigatorState>();
        _initializing = false;
      });
    }
  }

  Future<void> _login(String email, String password) async {
    final token = await _apiClient.login(email: email, password: password);
    try {
      await Future.wait([
        _sessionStore.writeToken(token),
        _sessionStore.writeEmail(email),
      ]).timeout(_sessionStoreTimeout);
    } catch (error) {
      debugPrint('Persist session storage unavailable: $error');
    }
    if (mounted) {
      setState(() {
        _navigatorKey = GlobalKey<NavigatorState>();
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
    try {
      await _sessionStore.deleteToken().timeout(_sessionStoreTimeout);
    } catch (error) {
      debugPrint('Clear session storage unavailable: $error');
    }
    if (!mounted) return;
    setState(() {
      _navigatorKey = GlobalKey<NavigatorState>();
      _authenticated = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      key: ValueKey(
        _initializing
            ? 'initializing'
            : (_authenticated ? 'authenticated' : 'login'),
      ),
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
      home: _authenticated
          ? null
          : _initializing
          ? const Scaffold(body: Center(child: CircularProgressIndicator()))
          : LoginScreen(onLogin: _login, initialEmail: _lastEmail),
      initialRoute: !_initializing && _authenticated ? _initialRoute : '/',
      onGenerateInitialRoutes: !_initializing && _authenticated
          ? (routeName) => [_route(RouteSettings(name: routeName))]
          : null,
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
