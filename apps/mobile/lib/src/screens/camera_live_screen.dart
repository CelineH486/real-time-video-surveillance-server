import 'dart:async';

import 'package:flutter/material.dart';
import 'package:video_player/video_player.dart';

import '../models/camera.dart';
import '../models/recording.dart';
import '../models/stream_session.dart';
import '../services/api_client.dart';
import '../widgets/whep_video_player.dart';

class CameraLiveScreen extends StatefulWidget {
  const CameraLiveScreen({
    super.key,
    required this.apiClient,
    required this.truckId,
    required this.cameraId,
    required this.onLogout,
  });

  final ApiClient apiClient;
  final String truckId;
  final String cameraId;
  final Future<void> Function() onLogout;

  @override
  State<CameraLiveScreen> createState() => _CameraLiveScreenState();
}

class _CameraLiveScreenState extends State<CameraLiveScreen> {
  late Future<_LiveViewData> _data;
  Future<List<Recording>>? _recordings;
  VideoPlayerController? _recordingController;
  bool _showHistory = false;

  @override
  void initState() {
    super.initState();
    _data = _load();
  }

  @override
  void dispose() {
    final controller = _recordingController;
    if (controller != null) unawaited(controller.dispose());
    super.dispose();
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

  void _reloadLive() {
    setState(() {
      _showHistory = false;
      _data = _load();
    });
  }

  void _loadHistory() {
    setState(() {
      _showHistory = true;
      _recordings = widget.apiClient.getRecordings(
        truckId: widget.truckId,
        cameraId: widget.cameraId,
      );
    });
  }

  Future<void> _playRecording(Recording recording) async {
    final previous = _recordingController;
    final controller = VideoPlayerController.networkUrl(
      Uri.parse(recording.url),
    );
    setState(() => _recordingController = controller);
    await previous?.dispose();
    try {
      await controller.initialize();
      await controller.play();
      if (mounted) setState(() {});
    } catch (_) {
      await controller.dispose();
      if (mounted && identical(_recordingController, controller)) {
        setState(() => _recordingController = null);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              widget.truckId.toUpperCase(),
              style: const TextStyle(
                color: Color(0xFF35E6A5),
                fontSize: 11,
                fontWeight: FontWeight.w800,
                letterSpacing: 1.5,
              ),
            ),
            Text(widget.cameraId.toUpperCase()),
          ],
        ),
        actions: [
          IconButton(
            onPressed: widget.onLogout,
            icon: const Icon(Icons.logout),
            tooltip: '登出',
          ),
          const SizedBox(width: 12),
        ],
      ),
      body: FutureBuilder<_LiveViewData>(
        future: _data,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(child: Text('載入攝影機失敗：${snapshot.error}'));
          }

          final data = snapshot.requireData;
          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 1100),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: Text(
                              data.camera.name,
                              style: Theme.of(context).textTheme.headlineSmall,
                            ),
                          ),
                          SegmentedButton<bool>(
                            segments: const [
                              ButtonSegment(
                                value: false,
                                icon: Icon(Icons.live_tv),
                                label: Text('即時影像'),
                              ),
                              ButtonSegment(
                                value: true,
                                icon: Icon(Icons.history),
                                label: Text('歷史錄影'),
                              ),
                            ],
                            selected: {_showHistory},
                            onSelectionChanged: (selection) {
                              if (selection.first) {
                                _loadHistory();
                              } else {
                                _reloadLive();
                              }
                            },
                          ),
                        ],
                      ),
                      const SizedBox(height: 16),
                      AspectRatio(
                        aspectRatio: 16 / 9,
                        child: ClipRRect(
                          borderRadius: BorderRadius.circular(16),
                          child: _showHistory
                              ? _RecordingStage(
                                  controller: _recordingController,
                                )
                              : WhepVideoPlayer(
                                  url: data.session.url,
                                  token: data.session.accessToken,
                                  muted: false,
                                ),
                        ),
                      ),
                      const SizedBox(height: 16),
                      if (_showHistory)
                        _RecordingList(
                          recordings: _recordings!,
                          onPlay: _playRecording,
                        )
                      else
                        Align(
                          alignment: Alignment.centerLeft,
                          child: FilledButton.icon(
                            onPressed: _reloadLive,
                            icon: const Icon(Icons.refresh),
                            label: const Text('重新取得串流'),
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _RecordingStage extends StatelessWidget {
  const _RecordingStage({required this.controller});

  final VideoPlayerController? controller;

  @override
  Widget build(BuildContext context) {
    if (controller == null || !controller!.value.isInitialized) {
      return const ColoredBox(
        color: Colors.black,
        child: Center(
          child: Text('請選擇一段歷史錄影', style: TextStyle(color: Colors.white54)),
        ),
      );
    }
    return Stack(
      fit: StackFit.expand,
      children: [
        ColoredBox(
          color: Colors.black,
          child: Center(
            child: AspectRatio(
              aspectRatio: controller!.value.aspectRatio,
              child: VideoPlayer(controller!),
            ),
          ),
        ),
        Align(
          alignment: Alignment.bottomCenter,
          child: VideoProgressIndicator(
            controller!,
            allowScrubbing: true,
            padding: const EdgeInsets.all(12),
          ),
        ),
        Center(
          child: IconButton.filledTonal(
            onPressed: () {
              controller!.value.isPlaying
                  ? controller!.pause()
                  : controller!.play();
            },
            icon: Icon(
              controller!.value.isPlaying ? Icons.pause : Icons.play_arrow,
            ),
          ),
        ),
      ],
    );
  }
}

class _RecordingList extends StatelessWidget {
  const _RecordingList({required this.recordings, required this.onPlay});

  final Future<List<Recording>> recordings;
  final Future<void> Function(Recording recording) onPlay;

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<Recording>>(
      future: recordings,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const Center(child: CircularProgressIndicator());
        }
        if (snapshot.hasError) {
          return Text('讀取歷史錄影失敗：${snapshot.error}');
        }
        final rows = snapshot.data ?? const [];
        if (rows.isEmpty) return const Text('尚無歷史錄影');
        return Wrap(
          spacing: 10,
          runSpacing: 10,
          children: [
            for (final recording in rows)
              OutlinedButton.icon(
                onPressed: () => onPlay(recording),
                icon: const Icon(Icons.play_arrow),
                label: Text(
                  '${recording.start.toLocal()} · '
                  '${recording.durationSeconds.round()} 秒',
                ),
              ),
          ],
        );
      },
    );
  }
}

class _LiveViewData {
  const _LiveViewData({required this.camera, required this.session});

  final Camera camera;
  final StreamSession session;
}
