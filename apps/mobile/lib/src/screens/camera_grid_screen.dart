import 'dart:async';

import 'package:flutter/material.dart';

import '../models/camera.dart';
import '../services/api_client.dart';
import '../widgets/whep_video_player.dart';

class CameraGridScreen extends StatefulWidget {
  const CameraGridScreen({
    super.key,
    required this.apiClient,
    required this.truckId,
    required this.onLogout,
  });

  final ApiClient apiClient;
  final String truckId;
  final Future<void> Function() onLogout;

  @override
  State<CameraGridScreen> createState() => _CameraGridScreenState();
}

class _CameraGridScreenState extends State<CameraGridScreen> {
  late Future<List<Camera>> _cameras;
  Timer? _refreshTimer;

  @override
  void initState() {
    super.initState();
    _reload();
    _refreshTimer = Timer.periodic(
      const Duration(seconds: 5),
      (_) => _reload(),
    );
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  void _reload() {
    if (!mounted) return;
    setState(() => _cameras = widget.apiClient.getCameras(widget.truckId));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        toolbarHeight: 76,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              widget.truckId.toUpperCase(),
              style: const TextStyle(
                color: Color(0xFF35E6A5),
                fontSize: 11,
                fontWeight: FontWeight.w800,
                letterSpacing: 1.6,
              ),
            ),
            const Text('車機攝影機總覽'),
          ],
        ),
        actions: [
          IconButton(
            onPressed: _reload,
            icon: const Icon(Icons.refresh),
            tooltip: '重新整理',
          ),
          IconButton(
            onPressed: widget.onLogout,
            icon: const Icon(Icons.logout),
            tooltip: '登出',
          ),
          const SizedBox(width: 12),
        ],
      ),
      body: FutureBuilder<List<Camera>>(
        future: _cameras,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(child: Text('讀取攝影機失敗：${snapshot.error}'));
          }

          final cameras = snapshot.data ?? const [];
          return LayoutBuilder(
            builder: (context, constraints) {
              final columns = constraints.maxWidth >= 1000
                  ? 3
                  : constraints.maxWidth >= 620
                  ? 2
                  : 1;
              return GridView.builder(
                padding: EdgeInsets.all(constraints.maxWidth >= 620 ? 20 : 12),
                gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                  crossAxisCount: columns,
                  mainAxisSpacing: 16,
                  crossAxisSpacing: 16,
                  childAspectRatio: 16 / 11.8,
                ),
                itemCount: cameras.length,
                itemBuilder: (context, index) {
                  final camera = cameras[index];
                  return _CameraTile(
                    camera: camera,
                    onTap: () => Navigator.of(context).pushNamed(
                      '/trucks/${widget.truckId}/cameras/${camera.cameraId}',
                    ),
                  );
                },
              );
            },
          );
        },
      ),
    );
  }
}

class _CameraTile extends StatelessWidget {
  const _CameraTile({required this.camera, required this.onTap});

  final Camera camera;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final statusColor = camera.isOnline
        ? const Color(0xFF35E6A5)
        : Colors.redAccent;

    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Expanded(
              child: Stack(
                fit: StackFit.expand,
                children: [
                  if (camera.isOnline &&
                      camera.subUrl != null &&
                      camera.subToken != null)
                    WhepVideoPlayer(
                      url: camera.subUrl!,
                      token: camera.subToken!,
                    )
                  else
                    const ColoredBox(
                      color: Colors.black,
                      child: Center(
                        child: Text(
                          '尚未取得畫面',
                          style: TextStyle(color: Colors.white38),
                        ),
                      ),
                    ),
                  const Positioned(
                    left: 10,
                    top: 10,
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        color: Color(0xCC030808),
                        borderRadius: BorderRadius.all(Radius.circular(6)),
                      ),
                      child: Padding(
                        padding: EdgeInsets.symmetric(
                          horizontal: 8,
                          vertical: 4,
                        ),
                        child: Text(
                          'LIVE',
                          style: TextStyle(
                            color: Color(0xFF35E6A5),
                            fontSize: 11,
                            fontWeight: FontWeight.w800,
                          ),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
            Padding(
              padding: const EdgeInsets.all(14),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          camera.cameraId.toUpperCase(),
                          style: const TextStyle(
                            color: Color(0xFF35E6A5),
                            fontSize: 12,
                            fontWeight: FontWeight.w800,
                            letterSpacing: 1.4,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          camera.name,
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                      ],
                    ),
                  ),
                  Chip(
                    label: Text(camera.isOnline ? '在線' : '離線'),
                    side: BorderSide.none,
                    labelStyle: TextStyle(
                      color: statusColor,
                      fontWeight: FontWeight.w700,
                    ),
                    backgroundColor: statusColor.withValues(alpha: .12),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
