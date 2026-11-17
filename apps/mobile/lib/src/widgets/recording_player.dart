import 'dart:async';

import 'package:flutter/material.dart';
import 'package:video_player/video_player.dart';

import '../models/recording.dart';
import '../utils/taiwan_time.dart';
import 'recording_day_timeline.dart';

// Memory only: retained across routes, reset on a page reload.
double _sessionPlaybackSpeed = 1.0;

class RecordingPlayer extends StatefulWidget {
  const RecordingPlayer({
    super.key,
    required this.date,
    required this.recordings,
    required this.loadUrl,
  });

  final DateTime date;
  final List<Recording> recordings;
  final Future<String> Function(Recording) loadUrl;

  @override
  State<RecordingPlayer> createState() => _RecordingPlayerState();
}

class _RecordingPlayerState extends State<RecordingPlayer> {
  VideoPlayerController? _controller;
  Recording? _selected;
  DateTime? _position;
  bool _loading = false;
  String? _error;
  int _request = 0;
  bool _advanced = false;

  List<Recording> get _rows =>
      widget.recordings.where((r) => r.durationSeconds > 0).toList()
        ..sort((a, b) => a.start.compareTo(b.start));

  DateTime _end(Recording r) => r.start.add(
    Duration(microseconds: (r.durationSeconds * 1000000).round()),
  );

  Future<void> _selectTime(DateTime time) async {
    final recording = recordingAt(widget.recordings, time);
    if (recording == null) {
      ScaffoldMessenger.of(context)
        ..hideCurrentSnackBar()
        ..showSnackBar(const SnackBar(content: Text('此時段無錄影')));
      return;
    }
    await _play(recording, time.difference(recording.start));
  }

  Future<void> _play(Recording recording, Duration offset) async {
    final request = ++_request;
    var controller = _controller;
    final reuse =
        _selected == recording &&
        controller != null &&
        controller.value.isInitialized &&
        _error == null;
    setState(() {
      _selected = recording;
      _position = recording.start.add(offset);
      _loading = true;
      _error = null;
      _advanced = false;
    });
    try {
      if (!reuse) {
        final previous = controller;
        _controller = null;
        previous?.removeListener(_updated);
        if (previous != null) unawaited(previous.dispose());
        final url = await widget.loadUrl(recording);
        if (!mounted || request != _request) return;
        controller = VideoPlayerController.networkUrl(Uri.parse(url));
        _controller = controller;
        await controller.initialize().timeout(const Duration(seconds: 30));
        if (!mounted || request != _request) return;
        controller.addListener(_updated);
      }
      final active = controller;
      final last = active.value.duration - const Duration(milliseconds: 100);
      final target = offset < Duration.zero || last < Duration.zero
          ? Duration.zero
          : (offset > last ? last : offset);
      await active.seekTo(target);
      if (!mounted || request != _request) return;
      await active.setPlaybackSpeed(_sessionPlaybackSpeed);
      if (!mounted || request != _request) return;
      setState(() => _loading = false);
      // Browsers may require another direct tap after loading a new clip.
      try {
        await active.play();
      } catch (_) {}
    } catch (_) {
      if (!mounted || request != _request) return;
      setState(() {
        _loading = false;
        _error = '影片載入失敗，請重試';
      });
    }
  }

  Future<void> _changeSpeed(double speed) async {
    final controller = _controller;
    try {
      if (controller != null && controller.value.isInitialized) {
        await controller.setPlaybackSpeed(speed);
      }
      if (!mounted) return;
      setState(() => _sessionPlaybackSpeed = speed);
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('此裝置無法使用此播放速度')));
    }
  }

  void _updated() {
    final controller = _controller;
    final selected = _selected;
    if (!mounted || controller == null || selected == null || _loading) return;
    setState(() {
      _position = selected.start.add(controller.value.position);
      if (controller.value.hasError) _error = '影片播放失敗，請重試';
    });
    if (controller.value.isCompleted && !_advanced && _error == null) {
      _advanced = true;
      final end = _end(selected);
      final next = _rows.where(
        (r) => r.start.isAfter(selected.start) && _end(r).isAfter(end),
      );
      if (next.isNotEmpty) {
        final r = next.first;
        unawaited(
          _play(
            r,
            r.start.isBefore(end) ? end.difference(r.start) : Duration.zero,
          ),
        );
      }
    }
  }

  Future<void> _toggle() async {
    if (_loading) return;
    final controller = _controller;
    if (_selected == null) {
      if (_rows.isNotEmpty) {
        await _selectTime(
          _rows.first.start.isBefore(taiwanDayStartUtc(widget.date))
              ? taiwanDayStartUtc(widget.date)
              : _rows.first.start,
        );
      }
    } else if (_error != null) {
      await _play(
        _selected!,
        (_position ?? _selected!.start).difference(_selected!.start),
      );
    } else if (controller != null) {
      if (controller.value.isPlaying) {
        await controller.pause();
      } else {
        if (controller.value.isCompleted) {
          _advanced = false;
          await controller.seekTo(Duration.zero);
        }
        await controller.play();
      }
    }
  }

  @override
  void dispose() {
    _request++;
    _controller?.removeListener(_updated);
    if (_controller != null) unawaited(_controller!.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = _controller;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        AspectRatio(
          aspectRatio: 16 / 9,
          child: ClipRRect(
            borderRadius: BorderRadius.circular(16),
            child: ColoredBox(
              color: Colors.black,
              child: Stack(
                alignment: Alignment.center,
                children: [
                  if (controller != null &&
                      controller.value.isInitialized &&
                      _error == null)
                    Center(
                      child: AspectRatio(
                        aspectRatio: controller.value.aspectRatio,
                        child: VideoPlayer(controller),
                      ),
                    ),
                  if (_loading || (controller?.value.isBuffering ?? false))
                    const CircularProgressIndicator()
                  else if (_error != null)
                    Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text(_error!),
                        TextButton(onPressed: _toggle, child: const Text('重試')),
                      ],
                    )
                  else if (_selected == null)
                    Text(
                      _rows.isEmpty ? '這一天沒有錄影' : '點選下方綠色區段開始播放',
                      style: const TextStyle(color: Colors.white60),
                    ),
                ],
              ),
            ),
          ),
        ),
        const SizedBox(height: 12),
        RecordingDayTimeline(
          speed: _sessionPlaybackSpeed,
          onSpeedChanged: _changeSpeed,
          date: widget.date,
          recordings: widget.recordings,
          position: _position,
          playing: controller?.value.isPlaying ?? false,
          loading: _loading,
          onSeek: _selectTime,
          onToggle: _rows.isEmpty ? null : _toggle,
        ),
      ],
    );
  }
}
