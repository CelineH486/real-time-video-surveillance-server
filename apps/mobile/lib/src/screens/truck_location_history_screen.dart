import 'dart:async';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';
import 'package:url_launcher/url_launcher.dart';

import '../models/truck_location.dart';
import '../services/api_client.dart';
import '../utils/route_direction.dart';
import '../utils/taiwan_time.dart';
import '../widgets/history_date_toolbar.dart';

class TruckLocationHistoryScreen extends StatefulWidget {
  const TruckLocationHistoryScreen({
    super.key,
    required this.apiClient,
    required this.truckId,
  });

  final ApiClient apiClient;
  final String truckId;

  @override
  State<TruckLocationHistoryScreen> createState() =>
      _TruckLocationHistoryScreenState();
}

class _TruckLocationHistoryScreenState
    extends State<TruckLocationHistoryScreen> {
  final _mapController = MapController();
  late DateTime _selectedDate;
  List<TruckLocation> _locations = const [];
  bool _loading = true;
  bool _mapReady = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _selectedDate = taiwanCalendarDate(DateTime.now());
    unawaited(_load());
  }

  @override
  void dispose() {
    _mapController.dispose();
    super.dispose();
  }

  List<DateTime> get _dates {
    final today = taiwanCalendarDate(DateTime.now());
    return List.generate(7, (index) => today.subtract(Duration(days: index)));
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final locations = await widget.apiClient.getTruckLocationHistory(
        truckId: widget.truckId,
        date: _selectedDate,
      );
      if (!mounted) return;
      setState(() => _locations = locations);
      _fitRoute();
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _locations = const [];
        _error = '$error';
      });
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _fitRoute() {
    if (!_mapReady || _locations.isEmpty) return;
    final points = _locations
        .map((item) => LatLng(item.latitude, item.longitude))
        .toList(growable: false);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (points.length == 1) {
        _mapController.move(points.first, 16);
      } else {
        _mapController.fitCamera(
          CameraFit.bounds(
            bounds: LatLngBounds.fromPoints(points),
            padding: const EdgeInsets.all(48),
          ),
        );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final points = _locations
        .map((item) => LatLng(item.latitude, item.longitude))
        .toList(growable: false);
    return Scaffold(
      appBar: AppBar(
        title: Text('${widget.truckId.toUpperCase()}・移動軌跡歷史'),
        actions: [
          IconButton(
            onPressed: _loading ? null : () => unawaited(_load()),
            icon: const Icon(Icons.refresh),
            tooltip: '重新載入',
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: Column(
        children: [
          HistoryDateToolbar(
            dates: _dates,
            selectedDate: _selectedDate,
            loading: _loading,
            summaryText: '${_locations.length} 個定位點',
            onChanged: (date) {
              setState(() => _selectedDate = date);
              unawaited(_load());
            },
          ),
          Expanded(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Card(
                clipBehavior: Clip.antiAlias,
                child: Stack(
                  children: [
                    FlutterMap(
                      mapController: _mapController,
                      options: MapOptions(
                        initialCenter: const LatLng(23.7, 121),
                        initialZoom: 7,
                        onMapReady: () {
                          _mapReady = true;
                          _fitRoute();
                        },
                      ),
                      children: [
                        TileLayer(
                          urlTemplate:
                              'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                          userAgentPackageName: 'com.example.surveillance_app',
                        ),
                        if (points.length >= 2)
                          PolylineLayer(
                            polylines: [
                              Polyline(
                                points: points,
                                strokeWidth: 5,
                                color: const Color(0xFF35E6A5),
                              ),
                            ],
                          ),
                        if (points.isNotEmpty)
                          MarkerLayer(
                            markers: [
                              Marker(
                                point: points.first,
                                width: 46,
                                height: 46,
                                child: _EndpointMarker(
                                  tooltip: '起點',
                                  color: Color(0xFF35E6A5),
                                  rotation: routeStartAngle(points),
                                  marker: const CustomPaint(
                                    size: Size(27, 17),
                                    painter: _DirectionTrianglePainter(),
                                  ),
                                ),
                              ),
                              if (points.length > 1)
                                Marker(
                                  point: points.last,
                                  width: 46,
                                  height: 46,
                                  child: const _EndpointMarker(
                                    tooltip: '終點',
                                    color: Color(0xFFFFBD66),
                                    marker: Icon(Icons.stop, size: 23),
                                  ),
                                ),
                            ],
                          ),
                        RichAttributionWidget(
                          attributions: [
                            TextSourceAttribution(
                              'OpenStreetMap contributors',
                              onTap: () => launchUrl(
                                Uri.parse(
                                  'https://www.openstreetmap.org/copyright',
                                ),
                              ),
                            ),
                          ],
                        ),
                      ],
                    ),
                    if (_loading)
                      const Center(child: CircularProgressIndicator()),
                    if (!_loading && _error != null)
                      _MapMessage(icon: Icons.cloud_off, message: _error!),
                    if (!_loading && _error == null && points.isEmpty)
                      const _MapMessage(
                        icon: Icons.route,
                        message: '這一天沒有移動軌跡',
                      ),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _EndpointMarker extends StatelessWidget {
  const _EndpointMarker({
    required this.tooltip,
    required this.color,
    required this.marker,
    this.rotation = 0,
  });

  final String tooltip;
  final Color color;
  final Widget marker;
  final double rotation;

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      child: CircleAvatar(
        backgroundColor: color,
        foregroundColor: const Color(0xFF071017),
        child: Transform.rotate(angle: rotation, child: marker),
      ),
    );
  }
}

class _DirectionTrianglePainter extends CustomPainter {
  const _DirectionTrianglePainter();

  @override
  void paint(Canvas canvas, Size size) {
    final path = ui.Path()
      ..moveTo(size.width, size.height / 2)
      ..lineTo(0, 0)
      ..lineTo(0, size.height)
      ..close();

    canvas.drawPath(
      path,
      Paint()
        ..color = const Color(0xFF071017)
        ..style = PaintingStyle.fill,
    );
  }

  @override
  bool shouldRepaint(_DirectionTrianglePainter oldDelegate) => false;
}

class _MapMessage extends StatelessWidget {
  const _MapMessage({required this.icon, required this.message});

  final IconData icon;
  final String message;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Card(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 18),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [Icon(icon), const SizedBox(width: 12), Text(message)],
          ),
        ),
      ),
    );
  }
}
