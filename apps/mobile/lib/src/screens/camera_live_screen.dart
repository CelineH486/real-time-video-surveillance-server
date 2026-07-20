import 'package:flutter/material.dart';

import '../models/camera.dart';
import '../models/stream_session.dart';
import '../services/api_client.dart';
import '../widgets/whep_video_player.dart';

class CameraLiveScreen extends StatefulWidget {
  const CameraLiveScreen({
    super.key,
    required this.apiClient,
    required this.truckId,
    required this.cameraId,
  });

  final ApiClient apiClient;
  final String truckId;
  final String cameraId;

  @override
  State<CameraLiveScreen> createState() => _CameraLiveScreenState();
}

class _CameraLiveScreenState extends State<CameraLiveScreen> {
  late Future<_LiveViewData> _data;

  @override
  void initState() {
    super.initState();
    _data = _load();
  }

  Future<_LiveViewData> _load() async {
    final cameras = await widget.apiClient.getCameras(widget.truckId);
    final camera = cameras.firstWhere(
      (item) => item.cameraId == widget.cameraId,
      orElse: () => Camera(
        cameraId: widget.cameraId,
        truckId: widget.truckId,
        name: widget.cameraId.toUpperCase(),
        status: 'offline',
      ),
    );
    final session = await widget.apiClient.createPlaySession(
      truckId: widget.truckId,
      cameraId: widget.cameraId,
    );
    return _LiveViewData(camera: camera, session: session);
  }

  void _reload() {
    setState(() {
      _data = _load();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.cameraId.toUpperCase()),
        actions: [
          IconButton(
            onPressed: _reload,
            icon: const Icon(Icons.refresh),
            tooltip: '重新取得播放資訊',
          ),
        ],
      ),
      body: FutureBuilder<_LiveViewData>(
        future: _data,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(child: Text('取得播放資訊失敗：${snapshot.error}'));
          }

          final data = snapshot.requireData;
          final session = data.session;
          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Text(
                data.camera.name,
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 12),
              AspectRatio(
                aspectRatio: 16 / 9,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    color: Colors.black,
                    borderRadius: BorderRadius.circular(16),
                  ),
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(16),
                    child: WhepVideoPlayer(
                      url: session.url,
                      token: session.accessToken,
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 16),
              _InfoRow(label: 'Truck', value: session.truckId),
              _InfoRow(label: 'Camera', value: session.cameraId),
              _InfoRow(label: 'Quality', value: session.quality),
              _InfoRow(label: 'Protocol', value: session.protocol),
              _InfoRow(label: 'URL', value: session.url),
              _InfoRow(
                label: 'Expires',
                value: session.expiresAt.toLocal().toString(),
              ),
              const SizedBox(height: 16),
              FilledButton.icon(
                onPressed: _reload,
                icon: const Icon(Icons.play_arrow),
                label: const Text('重新取得播放 Token'),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _LiveViewData {
  const _LiveViewData({required this.camera, required this.session});

  final Camera camera;
  final StreamSession session;
}

class _InfoRow extends StatelessWidget {
  const _InfoRow({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: Theme.of(context).textTheme.labelLarge),
          const SizedBox(height: 4),
          SelectableText(value),
        ],
      ),
    );
  }
}
