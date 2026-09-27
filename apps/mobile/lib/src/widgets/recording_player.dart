import 'dart:async';

import 'package:flutter/material.dart';
import 'package:video_player/video_player.dart';

import '../models/recording.dart';

/// Owns one playback request. The future is created on selection, never in build.
class RecordingPlayer extends StatefulWidget {
  const RecordingPlayer({
    super.key,
    required this.recording,
    required this.loadUrl,
  });

  final Recording? recording;
  final Future<String> Function(Duration offset)? loadUrl;

  @override
  State<RecordingPlayer> createState() => _RecordingPlayerState();
}

class _RecordingPlayerState extends State<RecordingPlayer> {
  VideoPlayerController? _controller;
  Future<void>? _ready;
  int _selectionId = 0;
  Duration _sourceOffset = Duration.zero;

  Duration get _totalDuration {
    final seconds = widget.recording?.durationSeconds ?? 0;
    return Duration(
      microseconds: (seconds * Duration.microsecondsPerSecond).round(),
    );
  }

  @override
  void initState() {
    super.initState();
    _select();
  }

  @override
  void didUpdateWidget(RecordingPlayer oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.recording?.start != widget.recording?.start ||
        oldWidget.recording?.durationSeconds !=
            widget.recording?.durationSeconds) {
      _select();
    }
  }

  void _select({Duration offset = Duration.zero}) {
    final selectionId = ++_selectionId;
    final previous = _controller;
    _controller = null;
    _ready = null;
    _sourceOffset = _clampPosition(offset);
    if (previous != null) unawaited(previous.dispose());
    final loadUrl = widget.loadUrl;
    if (widget.recording == null || loadUrl == null) return;
    _ready = _loadAndInitialize(loadUrl, _sourceOffset, selectionId);
  }

  Future<void> _loadAndInitialize(
    Future<String> Function(Duration offset) loadUrl,
    Duration offset,
    int selectionId,
  ) async {
    final url = await loadUrl(offset);
    if (!mounted || selectionId != _selectionId) return;
    final controller = VideoPlayerController.networkUrl(Uri.parse(url));
    _controller = controller;
    await _initialize(controller);
  }

  Duration _clampPosition(Duration position) {
    final total = _totalDuration;
    if (position < Duration.zero) return Duration.zero;
    if (total <= const Duration(milliseconds: 100)) return Duration.zero;
    final latest = total - const Duration(milliseconds: 100);
    return position > latest ? latest : position;
  }

  Future<void> _initialize(VideoPlayerController controller) async {
    await controller.initialize().timeout(const Duration(seconds: 30));
    // A previous selection may finish after the user changes clips or leaves.
    if (!mounted || !identical(controller, _controller)) return;
    // Mobile browsers can reject autoplay after asynchronous initialization.
    // Initialization still succeeded, so leave the player ready for a direct
    // tap on the play button instead of reporting a loading failure.
    try {
      await controller.play();
    } catch (_) {
      // The visible play button provides the user gesture required by Safari.
    }
  }

  Future<void> _seek(Duration position) async {
    final target = _clampPosition(position);
    final controller = _controller;
    if (controller == null || !controller.value.isInitialized) return;
    final localTarget = target - _sourceOffset;
    final isBuffered =
        localTarget >= Duration.zero &&
        controller.value.buffered.any(
          (range) => localTarget >= range.start && localTarget <= range.end,
        );
    if (isBuffered) {
      await controller.seekTo(localTarget);
      if (!controller.value.isPlaying) await controller.play();
      return;
    }
    if (!mounted) return;
    setState(() => _select(offset: target));
  }

  @override
  void dispose() {
    _selectionId++;
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
            return const Center(
              child: Text('請選擇一段歷史錄影', style: TextStyle(color: Colors.white54)),
            );
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
                  const Text(
                    '影片載入失敗，請重試或重新整理錄影清單',
                    style: TextStyle(color: Colors.white),
                  ),
                  TextButton(
                    onPressed: () =>
                        setState(() => _select(offset: _sourceOffset)),
                    child: const Text('重試'),
                  ),
                ],
              ),
            );
          }
          final controller = _controller!;
          return ValueListenableBuilder<VideoPlayerValue>(
            valueListenable: controller,
            builder: (context, value, child) {
              final position = _clampTimelinePosition(
                _sourceOffset + value.position,
              );
              final buffered = value.buffered
                  .map(
                    (range) => (
                      start: _clampTimelinePosition(
                        _sourceOffset + range.start,
                      ),
                      end: _clampTimelinePosition(_sourceOffset + range.end),
                    ),
                  )
                  .toList();
              return Stack(
                fit: StackFit.expand,
                children: [
                  Center(
                    child: AspectRatio(
                      aspectRatio: value.aspectRatio,
                      child: VideoPlayer(controller),
                    ),
                  ),
                  Align(
                    alignment: Alignment.bottomCenter,
                    child: ColoredBox(
                      color: Colors.black54,
                      child: Padding(
                        padding: const EdgeInsets.fromLTRB(12, 6, 12, 8),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            _RecordingTimeline(
                              position: position,
                              duration: _totalDuration,
                              buffered: buffered,
                              onSeek: _seek,
                            ),
                            Row(
                              children: [
                                Text(
                                  _formatPosition(position),
                                  style: const TextStyle(
                                    color: Colors.white,
                                    fontSize: 12,
                                  ),
                                ),
                                const Spacer(),
                                const _ProgressLegend(
                                  color: Color(0xFF35E6A5),
                                  label: '已播放',
                                ),
                                const SizedBox(width: 12),
                                const _ProgressLegend(
                                  color: Colors.white54,
                                  label: '已緩衝',
                                ),
                                const SizedBox(width: 12),
                                Text(
                                  _formatPosition(_totalDuration),
                                  style: const TextStyle(
                                    color: Colors.white70,
                                    fontSize: 12,
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                  Center(
                    child: value.isBuffering
                        ? const CircularProgressIndicator()
                        : IconButton.filledTonal(
                            onPressed: () => value.isPlaying
                                ? controller.pause()
                                : controller.play(),
                            icon: Icon(
                              value.isPlaying ? Icons.pause : Icons.play_arrow,
                            ),
                          ),
                  ),
                ],
              );
            },
          );
        },
      ),
    );
  }

  Duration _clampTimelinePosition(Duration position) {
    if (position < Duration.zero) return Duration.zero;
    return position > _totalDuration ? _totalDuration : position;
  }

  String _formatPosition(Duration duration) {
    final hours = duration.inHours;
    final minutes = duration.inMinutes.remainder(60).toString().padLeft(2, '0');
    final seconds = duration.inSeconds.remainder(60).toString().padLeft(2, '0');
    return hours > 0 ? '$hours:$minutes:$seconds' : '$minutes:$seconds';
  }
}

class _RecordingTimeline extends StatefulWidget {
  const _RecordingTimeline({
    required this.position,
    required this.duration,
    required this.buffered,
    required this.onSeek,
  });

  final Duration position;
  final Duration duration;
  final List<({Duration start, Duration end})> buffered;
  final ValueChanged<Duration> onSeek;

  @override
  State<_RecordingTimeline> createState() => _RecordingTimelineState();
}

class _RecordingTimelineState extends State<_RecordingTimeline> {
  Duration? _preview;

  double _fraction(Duration value) {
    if (widget.duration <= Duration.zero) return 0;
    return (value.inMicroseconds / widget.duration.inMicroseconds).clamp(
      0.0,
      1.0,
    );
  }

  Duration _positionAt(double dx, double width) {
    final fraction = (dx / width).clamp(0.0, 1.0);
    return Duration(
      microseconds: (widget.duration.inMicroseconds * fraction).round(),
    );
  }

  void _previewAt(Offset localPosition, double width) {
    setState(() => _preview = _positionAt(localPosition.dx, width));
  }

  void _finishDrag() {
    final position = _preview;
    setState(() => _preview = null);
    if (position != null) widget.onSeek(position);
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.maxWidth;
        final shownPosition = _preview ?? widget.position;
        return GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTapUp: (details) {
            final position = _positionAt(details.localPosition.dx, width);
            widget.onSeek(position);
          },
          onHorizontalDragStart: (details) =>
              _previewAt(details.localPosition, width),
          onHorizontalDragUpdate: (details) =>
              _previewAt(details.localPosition, width),
          onHorizontalDragEnd: (_) => _finishDrag(),
          onHorizontalDragCancel: () => setState(() => _preview = null),
          child: SizedBox(
            height: 26,
            child: Stack(
              alignment: Alignment.centerLeft,
              children: [
                Positioned(
                  left: 0,
                  right: 0,
                  child: Container(height: 4, color: Colors.white24),
                ),
                for (final range in widget.buffered)
                  Positioned(
                    left: width * _fraction(range.start),
                    width:
                        width * (_fraction(range.end) - _fraction(range.start)),
                    child: Container(height: 4, color: Colors.white54),
                  ),
                Positioned(
                  left: 0,
                  width: width * _fraction(shownPosition),
                  child: Container(height: 4, color: const Color(0xFF35E6A5)),
                ),
                Positioned(
                  left: (width * _fraction(shownPosition) - 6).clamp(
                    0.0,
                    width - 12,
                  ),
                  child: Container(
                    width: 12,
                    height: 12,
                    decoration: const BoxDecoration(
                      color: Color(0xFF35E6A5),
                      shape: BoxShape.circle,
                    ),
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _ProgressLegend extends StatelessWidget {
  const _ProgressLegend({required this.color, required this.label});

  final Color color;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 10,
          height: 3,
          decoration: BoxDecoration(
            color: color,
            borderRadius: BorderRadius.circular(2),
          ),
        ),
        const SizedBox(width: 4),
        Text(
          label,
          style: const TextStyle(color: Colors.white70, fontSize: 11),
        ),
      ],
    );
  }
}
