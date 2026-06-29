import 'package:flutter/material.dart';

import 'screens/camera_grid_screen.dart';
import 'screens/camera_live_screen.dart';
import 'screens/truck_select_screen.dart';
import 'services/api_client.dart';

class SurveillanceApp extends StatelessWidget {
  const SurveillanceApp({super.key});

  @override
  Widget build(BuildContext context) {
    final apiClient = ApiClient();

    return MaterialApp(
      title: '即時影像監控',
      debugShowCheckedModeBanner: false,
      initialRoute: '/trucks',
      onGenerateRoute: (settings) => _route(settings, apiClient),
      theme: ThemeData(
        brightness: Brightness.dark,
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF35E6A5),
          brightness: Brightness.dark,
        ),
        scaffoldBackgroundColor: const Color(0xFF090D12),
        cardTheme: const CardThemeData(
          color: Color(0xFF111820),
          margin: EdgeInsets.zero,
        ),
        appBarTheme: const AppBarTheme(
          backgroundColor: Color(0xFF090D12),
          foregroundColor: Colors.white,
          centerTitle: false,
        ),
        useMaterial3: true,
      ),
    );
  }

  Route<dynamic> _route(RouteSettings settings, ApiClient apiClient) {
    final uri = Uri.parse(settings.name ?? '/trucks');
    final segments = uri.pathSegments;

    Widget page = TruckSelectScreen(apiClient: apiClient);
    if (segments.length == 3 &&
        segments[0] == 'trucks' &&
        segments[2] == 'cameras') {
      page = CameraGridScreen(
        apiClient: apiClient,
        truckId: segments[1],
      );
    } else if (segments.length == 4 &&
        segments[0] == 'trucks' &&
        segments[2] == 'cameras') {
      page = CameraLiveScreen(
        apiClient: apiClient,
        truckId: segments[1],
        cameraId: segments[3],
      );
    }

    return MaterialPageRoute(
      settings: settings,
      builder: (_) => page,
    );
  }
}
