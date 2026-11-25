import 'dart:async';

import 'package:flutter/material.dart';
import 'package:video_player/video_player.dart';
import 'package:url_launcher/url_launcher.dart';

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
    this.refreshRecordings,
  });

  final DateTime date;
  final List<Recording> recordings;
  final Future<String> Function(Recording) loadUrl;
  final Future<List<Recording>> Function()? refreshRecordings;

  @override
  State<RecordingPlayer> createState() => _RecordingPlayerState();
}

class _RecordingPlayerState extends State<RecordingPlayer> {
  VideoPlayerController? _controller;
  Recording? _selected;
  DateTime? _position;
  bool _loading = false;
  bool _downloading = false;
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
    var recording = recordingAt(widget.recordings, time);
    if (recording != null &&
        !recording.playReady &&
        widget.refreshRecordings != null) {
      try {
        recording = recordingAt(await widget.refreshRecordings!(), time);
        if (!mounted) return;
      } catch (_) {
        if (!mounted) return;
        ScaffoldMessenger.of(context)
          ..hideCurrentSnackBar()
          ..showSnackBar(const SnackBar(content: Text('無法更新錄影狀態，請重試')));
        return;
      }
    }
    if (recording == null) {
      ScaffoldMessenger.of(context)
        ..hideCurrentSnackBar()
        ..showSnackBar(const SnackBar(content: Text('此時段無錄影')));
      return;
    }
    await _play(recording, time.difference(recording.start));
  }

  Future<void> _play(Recording recording, Duration offset) async {
    if (!recording.playReady) {
      ScaffoldMessenger.of(context)
        ..hideCurrentSnackBar()
        ..showSnackBar(const SnackBar(content: Text('此段錄影尚未上傳完成')));
      return;
    }
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

  Future<void> _download() async {
    if (_rows.isEmpty || _downloading) return;
    setState(() => _downloading = true);
    try {
      final refresh = widget.refreshRecordings;
      if (refresh == null) throw StateError('recording refresh unavailable');
      final dayStart = taiwanDayStartUtc(widget.date);
      final dayEnd = taiwanDayStartUtc(
        DateTime(widget.date.year, widget.date.month, widget.date.day + 1),
      );
      final available =
          (await refresh())
              .where(
                (r) =>
                    r.durationSeconds > 0 &&
                    r.start.isBefore(dayEnd) &&
                    _end(r).isAfter(dayStart),
              )
              .toList()
            ..sort((a, b) => a.start.compareTo(b.start));
      if (!mounted) return;
      if (available.isEmpty) throw StateError('no recordings');
      String format(DateTime value) {
        final t = value.toUtc().add(const Duration(hours: 8));
        String two(int n) => n.toString().padLeft(2, '0');
        return '${t.year}/${two(t.month)}/${two(t.day)} ${two(t.hour)}:${two(t.minute)}:${two(t.second)}';
      }

      final recording = await showDialog<Recording>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('選擇下載區段'),
          content: SizedBox(
            width: 420,
            child: ListView.builder(
              shrinkWrap: true,
              itemCount: available.length,
              itemBuilder: (context, index) {
                final row = available[index];
                return ListTile(
                  title: Text('${format(row.start)} ～ ${format(_end(row))}'),
                  subtitle: row.downloadReady
                      ? (row.durationSeconds < 1800
                            ? const Text('此段錄影未滿 30 分鐘')
                            : null)
                      : Text(
                          row.uploadStatus == 'failed'
                              ? '上傳失敗，系統將重試'
                              : row.uploadStatus == 'uploading'
                              ? '上傳中'
                              : row.uploadStatus == null
                              ? '錄影尚未完成或檔案檢查中'
                              : '錄影尚未上傳完成',
                        ),
                  enabled: row.downloadReady,
                  onTap: row.downloadReady
                      ? () => Navigator.pop(context, row)
                      : null,
                );
              },
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('取消'),
            ),
          ],
        ),
      );
      if (!mounted || recording == null) return;
      final rows = await refresh();
      final matches = rows
          .where(
            (r) =>
                r.start.isAtSameMomentAs(recording.start) &&
                r.durationSeconds > 0,
          )
          .toList();
      if (!mounted) return;
      if (matches.length != 1) {
        throw StateError('recording no longer available');
      }
      final latest = matches.single;
      if (!latest.downloadReady) throw StateError('recording not ready');
      final end = latest.start.add(
        Duration(
          microseconds:
              (latest.durationSeconds * Duration.microsecondsPerSecond).floor(),
        ),
      );
      final approved = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('下載目前這段錄影'),
          content: Text(
            '可下載範圍（台灣時間）\n${format(latest.start)} ～ ${format(end)}\n\n${latest.durationSeconds < 1800 ? '此段錄影未滿 30 分鐘。\n' : ''}下載範圍以此畫面顯示的時間為準',
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('下載 MP4'),
            ),
          ],
        ),
      );
      if (!mounted || approved != true) return;
      final url = Uri.parse(await widget.loadUrl(latest));
      final download = url.replace(
        queryParameters: {...url.queryParameters, 'download': '1'},
      );
      if (!await launchUrl(download, webOnlyWindowName: '_self')) {
        throw Exception('download failed');
      }
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('下載失敗，請重試')));
      }
    } finally {
      if (mounted) setState(() => _downloading = false);
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
        Align(
          alignment: Alignment.centerRight,
          child: IconButton(
            tooltip: '下載 MP4',
            onPressed: _rows.isEmpty || _downloading ? null : _download,
            icon: const Icon(Icons.download),
          ),
        ),
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
