import 'dart:async';

import 'package:flutter/material.dart';
import 'package:video_player/video_player.dart';

/// Owns one playback request. The future is created on selection, never in build.
class RecordingPlayer extends StatefulWidget {
  const RecordingPlayer({super.key, required this.url});

  final String? url;

  @override
  State<RecordingPlayer> createState() => _RecordingPlayerState();
}

class _RecordingPlayerState extends State<RecordingPlayer> {
  VideoPlayerController? _controller;
  Future<void>? _ready;

  @override
  void initState() {
    super.initState();
    _select();
  }

  @override
  void didUpdateWidget(RecordingPlayer oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.url != widget.url) _select();
  }

  void _select() {
    final previous = _controller;
    _controller = null;
    _ready = null;
    if (previous != null) unawaited(previous.dispose());
    if (widget.url == null) return;
    final controller = VideoPlayerController.networkUrl(Uri.parse(widget.url!));
    _controller = controller;
    _ready = _initialize(controller);
  }

  Future<void> _initialize(VideoPlayerController controller) async {
    await controller.initialize().timeout(const Duration(seconds: 30));
    // A previous selection may finish after the user changes clips or leaves.
    if (!mounted || !identical(controller, _controller)) return;
    await controller.play();
  }

  @override
  void dispose() {
    final controller = _controller;
    _controller = null;
    if (controller != null) unawaited(controller.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: Colors.black,
      child: FutureBuilder<void>(
        future: _ready,
        builder: (context, snapshot) {
          if (_ready == null) {
            return const Center(child: Text('請選擇一段歷史錄影', style: TextStyle(color: Colors.white54)));
          }
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  CircularProgressIndicator(),
                  SizedBox(height: 16),
                  Text('影片載入中…', style: TextStyle(color: Colors.white)),
                ],
              ),
            );
          }
          if (snapshot.hasError) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Text('影片載入失敗，請重試或重新整理錄影清單', style: TextStyle(color: Colors.white)),
                  TextButton(onPressed: () => setState(_select), child: const Text('重試')),
                ],
              ),
            );
          }
          final controller = _controller!;
          return ValueListenableBuilder<VideoPlayerValue>(
            valueListenable: controller,
            builder: (context, value, child) => Stack(
              fit: StackFit.expand,
              children: [
                Center(child: AspectRatio(aspectRatio: value.aspectRatio, child: VideoPlayer(controller))),
                Align(
                  alignment: Alignment.bottomCenter,
                  child: VideoProgressIndicator(controller, allowScrubbing: true, padding: const EdgeInsets.all(12)),
                ),
                Center(
                  child: value.isBuffering
                      ? const CircularProgressIndicator()
                      : IconButton.filledTonal(
                          onPressed: () => value.isPlaying ? controller.pause() : controller.play(),
                          icon: Icon(value.isPlaying ? Icons.pause : Icons.play_arrow),
                        ),
                ),
              ],
            ),
          );
        },
      ),
    );
  }
}
