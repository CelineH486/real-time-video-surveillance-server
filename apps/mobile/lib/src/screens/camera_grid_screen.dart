import 'package:flutter/material.dart';

import '../models/camera.dart';
import '../services/api_client.dart';
import '../widgets/whep_video_player.dart';

class CameraGridScreen extends StatefulWidget {
  const CameraGridScreen({
    super.key,
    required this.apiClient,
    required this.truckId,
  });

  final ApiClient apiClient;
  final String truckId;

  @override
  State<CameraGridScreen> createState() => _CameraGridScreenState();
}

class _CameraGridScreenState extends State<CameraGridScreen> {
  late Future<List<Camera>> _cameras;

  @override
  void initState() {
    super.initState();
    _cameras = widget.apiClient.getCameras(widget.truckId);
  }

  void _reload() {
    setState(() {
      _cameras = widget.apiClient.getCameras(widget.truckId);
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.truckId.toUpperCase()),
        actions: [
          IconButton(
            onPressed: _reload,
            icon: const Icon(Icons.refresh),
            tooltip: '重新整理',
          ),
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
              final columns = constraints.maxWidth >= 900
                  ? 3
                  : constraints.maxWidth >= 560
                      ? 2
                      : 1;

              return GridView.builder(
                padding: const EdgeInsets.all(16),
                gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                  crossAxisCount: columns,
                  mainAxisSpacing: 12,
                  crossAxisSpacing: 12,
                  childAspectRatio: 16 / 11,
                ),
                itemCount: cameras.length,
                itemBuilder: (context, index) {
                  final camera = cameras[index];
                  return _CameraTile(
                    camera: camera,
                    onTap: () {
                      Navigator.of(context).pushNamed(
                        '/trucks/${widget.truckId}/cameras/${camera.cameraId}',
                      );
                    },
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
    final statusColor = camera.isOnline ? const Color(0xFF35E6A5) : Colors.redAccent;

    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Expanded(
              child: Container(
                color: Colors.black,
                child: Stack(
                  children: [
                    if (camera.isOnline &&
                        camera.subUrl != null &&
                        camera.subToken != null)
                      Positioned.fill(
                        child: WhepVideoPlayer(
                          url: camera.subUrl!,
                          token: camera.subToken!,
                        ),
                      )
                    else
                      const Center(
                        child: Icon(Icons.videocam, size: 48, color: Colors.white30),
                      ),
                    Positioned(
                      left: 10,
                      top: 10,
                      child: DecoratedBox(
                        decoration: BoxDecoration(
                          color: Colors.black54,
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: const Padding(
                          padding: EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                          child: Text('LIVE', style: TextStyle(fontSize: 12)),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.all(12),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(camera.cameraId.toUpperCase()),
                        const SizedBox(height: 4),
                        Text(camera.name, style: Theme.of(context).textTheme.titleMedium),
                      ],
                    ),
                  ),
                  Chip(
                    label: Text(camera.isOnline ? '在線' : '離線'),
                    side: BorderSide(color: statusColor),
                    labelStyle: TextStyle(color: statusColor),
                    backgroundColor: statusColor.withValues(alpha: .1),
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
