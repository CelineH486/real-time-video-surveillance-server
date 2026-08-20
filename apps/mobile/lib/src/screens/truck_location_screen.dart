import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../models/truck_location.dart';
import '../services/api_client.dart';

class TruckLocationScreen extends StatefulWidget {
  const TruckLocationScreen({
    super.key,
    required this.apiClient,
    required this.truckId,
    required this.onLogout,
  });

  final ApiClient apiClient;
  final String truckId;
  final Future<void> Function() onLogout;

  @override
  State<TruckLocationScreen> createState() => _TruckLocationScreenState();
}

class _TruckLocationScreenState extends State<TruckLocationScreen>
    with SingleTickerProviderStateMixin {
  static const _staleAfter = Duration(seconds: 30);
  static const _taipeiOffset = Duration(hours: 8);

  final _mapController = MapController();
  late final AnimationController _movementController;
  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _socketSubscription;
  Timer? _reconnectTimer;
  Timer? _freshnessTimer;
  TruckLocation? _location;
  LatLng? _markerPosition;
  LatLng? _movementStart;
  LatLng? _movementTarget;
  bool _mapReady = false;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _movementController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1100),
    )..addListener(_animateMarker);
    _freshnessTimer = Timer.periodic(
      const Duration(seconds: 1),
      (_) => mounted ? setState(() {}) : null,
    );
    unawaited(_loadInitialLocation());
  }

  @override
  void dispose() {
    _freshnessTimer?.cancel();
    _reconnectTimer?.cancel();
    unawaited(_socketSubscription?.cancel());
    unawaited(_channel?.sink.close());
    _movementController.dispose();
    _mapController.dispose();
    super.dispose();
  }

  Future<void> _loadInitialLocation() async {
    try {
      final location = await widget.apiClient.getTruckLocation(widget.truckId);
      if (!mounted) return;
      if (location != null) _applyLocation(location, animate: false);
      setState(() {
        _loading = false;
        _error = null;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = '$error';
      });
    } finally {
      if (mounted) _connectSocket();
    }
  }

  void _connectSocket() {
    _reconnectTimer?.cancel();
    unawaited(_socketSubscription?.cancel());
    unawaited(_channel?.sink.close());

    try {
      final channel = widget.apiClient.openTruckLocationSocket(widget.truckId);
      _channel = channel;
      _socketSubscription = channel.stream.listen(
        (message) {
          final json = jsonDecode(message as String) as Map<String, dynamic>;
          _applyLocation(TruckLocation.fromJson(json));
        },
        onError: (_) => _scheduleReconnect(),
        onDone: _scheduleReconnect,
        cancelOnError: true,
      );
    } catch (_) {
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    if (!mounted || _reconnectTimer?.isActive == true) return;
    _reconnectTimer = Timer(const Duration(seconds: 2), _connectSocket);
  }

  void _applyLocation(TruckLocation location, {bool animate = true}) {
    if (!mounted) return;
    final target = LatLng(location.latitude, location.longitude);
    _error = null;
    _location = location;

    if (_markerPosition == null || !animate) {
      _movementController.stop();
      _markerPosition = target;
      _movementTarget = target;
      setState(() {});
      _followMarker(target, initial: true);
      return;
    }

    if (_movementTarget == target) {
      setState(() {});
      return;
    }
    _movementStart = _markerPosition;
    _movementTarget = target;
    _movementController.forward(from: 0);
  }

  void _animateMarker() {
    final start = _movementStart;
    final target = _movementTarget;
    if (start == null || target == null || !mounted) return;
    final progress = _movementController.value;
    final current = LatLng(
      start.latitude + (target.latitude - start.latitude) * progress,
      start.longitude + (target.longitude - start.longitude) * progress,
    );
    setState(() => _markerPosition = current);
    _followMarker(current);
  }

  void _followMarker(LatLng position, {bool initial = false}) {
    if (!_mapReady) return;
    final zoom = initial ? 16.0 : _mapController.camera.zoom;
    _mapController.move(position, zoom);
  }

  bool get _stale {
    final location = _location;
    if (location == null) return true;
    return DateTime.now().toUtc().difference(location.receivedAt.toUtc()) >
        _staleAfter;
  }

  void _goBack() {
    final navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop();
      return;
    }
    navigator.pushReplacementNamed('/');
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        toolbarHeight: 82,
        leading: IconButton(
          onPressed: _goBack,
          icon: const Icon(Icons.arrow_back),
          tooltip: '返回車機列表',
        ),
        title: _PageTitle(truckId: widget.truckId),
        actions: [
          TextButton.icon(
            onPressed: _openCameras,
            icon: const Icon(Icons.videocam_outlined),
            label: const Text('即時監控'),
          ),
          const SizedBox(width: 8),
          IconButton(
            onPressed: () => unawaited(_loadInitialLocation()),
            icon: const Icon(Icons.refresh),
            tooltip: '更新位置',
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : LayoutBuilder(
              builder: (context, constraints) {
                final wide = constraints.maxWidth >= 900;
                final map = _MapPanel(
                  controller: _mapController,
                  location: _location,
                  markerPosition: _markerPosition,
                  stale: _stale,
                  truckId: widget.truckId,
                  onMapReady: () {
                    _mapReady = true;
                    final position = _markerPosition;
                    if (position != null) {
                      _followMarker(position, initial: true);
                    }
                  },
                  onMarkerTap: _showVehicleDetails,
                );
                final details = _LocationDetails(
                  location: _location,
                  stale: _stale,
                  error: _error,
                );

                return Padding(
                  padding: EdgeInsets.fromLTRB(
                    wide ? 36 : 16,
                    wide ? 22 : 16,
                    wide ? 36 : 16,
                    wide ? 32 : 16,
                  ),
                  child: wide
                      ? Row(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            Expanded(flex: 2, child: map),
                            const SizedBox(width: 20),
                            SizedBox(width: 360, child: details),
                          ],
                        )
                      : ListView(
                          children: [
                            SizedBox(height: 440, child: map),
                            const SizedBox(height: 16),
                            details,
                          ],
                        ),
                );
              },
            ),
    );
  }

  void _openCameras() {
    Navigator.of(context).pushNamed('/trucks/${widget.truckId}/cameras');
  }

  void _showVehicleDetails() {
    final location = _location;
    if (location == null) return;
    showDialog<void>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(widget.truckId.toUpperCase()),
        content: _LocationDetails(
          location: location,
          stale: _stale,
          compact: true,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            child: const Text('關閉'),
          ),
          FilledButton.icon(
            onPressed: () {
              Navigator.of(context).pop();
              _openCameras();
            },
            icon: const Icon(Icons.videocam_outlined),
            label: const Text('查看即時監控'),
          ),
        ],
      ),
    );
  }
}

class _MapPanel extends StatelessWidget {
  const _MapPanel({
    required this.controller,
    required this.location,
    required this.markerPosition,
    required this.stale,
    required this.truckId,
    required this.onMapReady,
    required this.onMarkerTap,
  });

  final MapController controller;
  final TruckLocation? location;
  final LatLng? markerPosition;
  final bool stale;
  final String truckId;
  final VoidCallback onMapReady;
  final VoidCallback onMarkerTap;

  @override
  Widget build(BuildContext context) {
    final position = markerPosition;
    return Card(
      clipBehavior: Clip.antiAlias,
      child: Stack(
        children: [
          FlutterMap(
            mapController: controller,
            options: MapOptions(
              initialCenter: position ?? const LatLng(23.7, 121),
              initialZoom: position == null ? 7 : 16,
              onMapReady: onMapReady,
            ),
            children: [
              TileLayer(
                urlTemplate: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                userAgentPackageName: 'com.example.surveillance_app',
              ),
              if (position != null)
                MarkerLayer(
                  markers: [
                    Marker(
                      point: position,
                      width: 116,
                      height: 84,
                      alignment: Alignment.bottomCenter,
                      child: GestureDetector(
                        onTap: onMarkerTap,
                        child: _VehicleMarker(truckId: truckId, stale: stale),
                      ),
                    ),
                  ],
                ),
              RichAttributionWidget(
                attributions: [
                  TextSourceAttribution(
                    'OpenStreetMap contributors',
                    onTap: () => launchUrl(
                      Uri.parse('https://www.openstreetmap.org/copyright'),
                    ),
                  ),
                ],
              ),
            ],
          ),
          if (location == null)
            const Center(
              child: Card(
                child: Padding(
                  padding: EdgeInsets.symmetric(horizontal: 20, vertical: 14),
                  child: Text('等待 GPS 定位資料'),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _VehicleMarker extends StatelessWidget {
  const _VehicleMarker({required this.truckId, required this.stale});

  final String truckId;
  final bool stale;

  @override
  Widget build(BuildContext context) {
    final color = stale ? Colors.redAccent : const Color(0xFF35E6A5);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
          decoration: BoxDecoration(
            color: const Color(0xEE111820),
            borderRadius: BorderRadius.circular(6),
            border: Border.all(color: color.withValues(alpha: .65)),
          ),
          child: Text(
            truckId.toUpperCase(),
            style: TextStyle(
              color: color,
              fontSize: 11,
              fontWeight: FontWeight.w800,
              letterSpacing: 1,
            ),
          ),
        ),
        Icon(Icons.location_on, size: 52, color: color),
      ],
    );
  }
}

class _LocationDetails extends StatelessWidget {
  const _LocationDetails({
    required this.location,
    required this.stale,
    this.error,
    this.compact = false,
  });

  final TruckLocation? location;
  final bool stale;
  final String? error;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final location = this.location;
    if (error != null && location == null) {
      return _MessageCard(icon: Icons.cloud_off, message: error!);
    }
    if (location == null) {
      return const _MessageCard(
        icon: Icons.location_searching,
        message: '尚未收到這台車的 GPS 定位資料',
      );
    }

    final color = stale
        ? Colors.redAccent
        : location.stoppedSince != null
        ? Colors.orangeAccent
        : const Color(0xFF35E6A5);
    final state = stale
        ? 'GPS 連線中斷'
        : location.stoppedSince != null
        ? '停留中'
        : '行駛中';
    final subtitle = stale
        ? '最後收到定位：${_relativeTime(location.receivedAt)}'
        : location.stoppedSince != null
        ? '已停留 ${_duration(DateTime.now().toUtc().difference(location.stoppedSince!.toUtc()))}'
        : '目前速度 ${location.speedKmh.toStringAsFixed(1)} km/h';

    final content = Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (!compact) ...[
          const Text(
            'CURRENT STATUS',
            style: TextStyle(
              color: Color(0xFF35E6A5),
              fontSize: 11,
              fontWeight: FontWeight.w800,
              letterSpacing: 1.6,
            ),
          ),
          const SizedBox(height: 18),
        ],
        Row(
          children: [
            Container(
              width: 16,
              height: 16,
              decoration: BoxDecoration(
                color: color,
                shape: BoxShape.circle,
                boxShadow: [BoxShadow(color: color, blurRadius: 12)],
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(state, style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 4),
                  Text(subtitle, style: const TextStyle(color: Colors.white60)),
                ],
              ),
            ),
          ],
        ),
        const SizedBox(height: 20),
        _MetricGrid(location: location),
      ],
    );

    if (compact) return content;
    return SingleChildScrollView(
      child: Card(
        child: Padding(padding: const EdgeInsets.all(20), child: content),
      ),
    );
  }
}

class _MetricGrid extends StatelessWidget {
  const _MetricGrid({required this.location});

  final TruckLocation location;

  @override
  Widget build(BuildContext context) {
    final items = <(String, String)>[
      ('緯度', location.latitude.toStringAsFixed(6)),
      ('經度', location.longitude.toStringAsFixed(6)),
      ('速度', '${location.speedKmh.toStringAsFixed(1)} km/h'),
      (
        '方向',
        location.headingDegrees == null
            ? '—'
            : '${location.headingDegrees!.toStringAsFixed(0)}°',
      ),
      (
        'GPS 精度',
        location.accuracyM == null
            ? '—'
            : '±${location.accuracyM!.toStringAsFixed(1)} m',
      ),
      ('衛星數', location.satellites?.toString() ?? '—'),
      ('GPS 定位時間（台灣時間）', _taipeiTime(location.recordedAt)),
      (
        '伺服器收到時間（台灣時間）',
        '${_taipeiTime(location.receivedAt)}（${_relativeTime(location.receivedAt)}）',
      ),
    ];
    return Wrap(
      spacing: 1,
      runSpacing: 1,
      children: items
          .map(
            (item) => Container(
              width: item.$1.contains('時間') ? double.infinity : 145,
              constraints: const BoxConstraints(minHeight: 82),
              padding: const EdgeInsets.all(14),
              color: const Color(0xFF0D141B),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    item.$1,
                    style: const TextStyle(color: Colors.white54, fontSize: 12),
                  ),
                  const SizedBox(height: 8),
                  Text(item.$2, style: const TextStyle(fontSize: 16)),
                ],
              ),
            ),
          )
          .toList(),
    );
  }
}

class _MessageCard extends StatelessWidget {
  const _MessageCard({required this.icon, required this.message});

  final IconData icon;
  final String message;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 42),
            const SizedBox(height: 12),
            Text(message, textAlign: TextAlign.center),
          ],
        ),
      ),
    );
  }
}

class _PageTitle extends StatelessWidget {
  const _PageTitle({required this.truckId});

  final String truckId;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          truckId.toUpperCase(),
          style: const TextStyle(
            color: Color(0xFF35E6A5),
            fontSize: 10,
            letterSpacing: 1.8,
            fontWeight: FontWeight.w800,
          ),
        ),
        const Text('車輛定位'),
      ],
    );
  }
}

String _taipeiTime(DateTime value) {
  final taipei = value.toUtc().add(_TruckLocationScreenState._taipeiOffset);
  String two(int number) => number.toString().padLeft(2, '0');
  return '${taipei.year}/${two(taipei.month)}/${two(taipei.day)} '
      '${two(taipei.hour)}:${two(taipei.minute)}:${two(taipei.second)}';
}

String _relativeTime(DateTime value) {
  final difference = DateTime.now().toUtc().difference(value.toUtc());
  final seconds = difference.isNegative ? 0 : difference.inSeconds;
  if (seconds < 60) return '$seconds 秒前';
  if (seconds < 3600) return '${seconds ~/ 60} 分鐘前';
  if (seconds < 86400) return '${seconds ~/ 3600} 小時前';
  return '${seconds ~/ 86400} 天前';
}

String _duration(Duration value) {
  final seconds = value.isNegative ? 0 : value.inSeconds;
  if (seconds < 60) return '$seconds 秒';
  if (seconds < 3600) return '${seconds ~/ 60} 分鐘';
  return '${seconds ~/ 3600} 小時 ${(seconds % 3600) ~/ 60} 分鐘';
}
