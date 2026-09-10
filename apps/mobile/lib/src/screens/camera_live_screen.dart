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
  DateTime _historyDate = DateTime.now();

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
    final now = DateTime.now();
    final cutoff = now.subtract(const Duration(days: 7));
    if (_historyDate.isBefore(
          DateTime(cutoff.year, cutoff.month, cutoff.day),
        ) ||
        _historyDate.isAfter(now)) {
      _historyDate = now;
    }
    final start = DateTime(
      _historyDate.year,
      _historyDate.month,
      _historyDate.day,
    );
    final nextDay = DateTime(start.year, start.month, start.day + 1);
    final previous = _recordingController;
    _recordingController = null;
    if (previous != null) unawaited(previous.dispose());
    setState(() {
      _showHistory = true;
      _recordings = widget.apiClient.getRecordings(
        truckId: widget.truckId,
        cameraId: widget.cameraId,
        start: start.isBefore(cutoff) ? cutoff : start,
        end: nextDay.isAfter(now) ? now : nextDay,
      );
    });
  }

  void _goBack() {
    final navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop();
      return;
    }
    navigator.pushReplacementNamed('/trucks/${widget.truckId}/cameras');
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
        leading: IconButton(
          onPressed: _goBack,
          icon: const Icon(Icons.arrow_back),
          tooltip: '返回攝影機總覽',
        ),
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
                      if (_showHistory) ...[
                        Wrap(
                          spacing: 12,
                          runSpacing: 8,
                          crossAxisAlignment: WrapCrossAlignment.center,
                          children: [
                            Text('${_historyDate.year} 年 · 錄影保存最近 7 天'),
                            TextButton.icon(
                              onPressed: _loadHistory,
                              icon: const Icon(Icons.refresh),
                              label: const Text('重新整理'),
                            ),
                          ],
                        ),
                        _HistoryDates(
                          selected: _historyDate,
                          onSelect: (date) {
                            _historyDate = date;
                            _loadHistory();
                          },
                        ),
                        const SizedBox(height: 8),
                        const Text('最早一天僅提供最近 7 天內的時段。依整點／半點分組，實際錄影範圍列於下方。'),
                        const SizedBox(height: 12),
                      ],
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
                                  muted: true,
                                  onAuthenticationExpired: _reloadLive,
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
        final rows = (snapshot.data ?? const <Recording>[])
            .expand((recording) => recording.halfHourSegments())
            .toList();
        if (rows.isEmpty) return const Text('尚無歷史錄影');
        return Wrap(
          spacing: 10,
          runSpacing: 10,
          children: [
            for (final recording in rows)
              SizedBox(
                width: 220,
                height: 104,
                child: OutlinedButton(
                  style: OutlinedButton.styleFrom(
                    alignment: Alignment.centerLeft,
                    padding: const EdgeInsets.all(14),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(10),
                    ),
                  ),
                  onPressed: () => onPlay(recording),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Padding(
                        padding: EdgeInsets.only(top: 2),
                        child: Icon(Icons.play_arrow, size: 20),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            Text(
                              _formatSlot(recording),
                              style: const TextStyle(
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                            const SizedBox(height: 4),
                            Text(
                              _formatDuration(recording.durationSeconds),
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                            Text(
                              _formatRecordingStatus(recording),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ),
          ],
        );
      },
    );
  }

  String _formatRecordingStatus(Recording recording) {
    final local = recording.start.toLocal();
    final end = local.add(Duration(seconds: recording.durationSeconds.round()));
    final startsOnBoundary = local.minute % 30 == 0 && local.second == 0;
    final isComplete = startsOnBoundary && recording.durationSeconds >= 1799;
    if (isComplete) return '● 完整錄影';
    String clock(DateTime date) =>
        '${date.hour.toString().padLeft(2, '0')}:${date.minute.toString().padLeft(2, '0')}';
    if (!startsOnBoundary) return '${clock(local)} 開始 · 部分錄影';
    return '錄影至 ${clock(end)} · 部分錄影';
  }

  String _formatSlot(Recording recording) {
    final local = recording.start.toLocal();
    final slot = DateTime(
      local.year,
      local.month,
      local.day,
      local.hour,
      local.minute < 30 ? 0 : 30,
    );
    final end = slot.add(const Duration(minutes: 30));
    String clock(DateTime date) =>
        '${date.hour.toString().padLeft(2, '0')}:${date.minute.toString().padLeft(2, '0')}';
    return '${clock(slot)}–${clock(end)}';
  }

  String _formatDuration(double seconds) {
    final duration = Duration(seconds: seconds.round());
    final minutes = duration.inMinutes;
    final remainingSeconds = duration.inSeconds.remainder(60);
    return remainingSeconds == 0
        ? '$minutes 分鐘'
        : '$minutes 分 ${remainingSeconds.toString().padLeft(2, '0')} 秒';
  }
}

class _HistoryDates extends StatelessWidget {
  const _HistoryDates({required this.selected, required this.onSelect});

  final DateTime selected;
  final ValueChanged<DateTime> onSelect;

  @override
  Widget build(BuildContext context) {
    final now = DateTime.now();
    const weekdays = ['星期一', '星期二', '星期三', '星期四', '星期五', '星期六', '星期日'];
    // A rolling 168-hour retention window can overlap eight calendar dates.
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        for (var offset = 0; offset <= 7; offset++)
          Builder(
            builder: (context) {
              final date = DateTime(now.year, now.month, now.day - offset);
              final active = DateUtils.isSameDay(date, selected);
              return SizedBox(
                width: 104,
                child: Semantics(
                  selected: active,
                  child: OutlinedButton(
                    style: OutlinedButton.styleFrom(
                      backgroundColor: active
                          ? Theme.of(context).colorScheme.secondaryContainer
                          : null,
                      foregroundColor: active
                          ? Theme.of(context).colorScheme.onSecondaryContainer
                          : null,
                      padding: const EdgeInsets.symmetric(
                        vertical: 12,
                        horizontal: 4,
                      ),
                    ),
                    onPressed: () => onSelect(date),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text(weekdays[date.weekday - 1]),
                        Text('${date.month} 月 ${date.day} 日'),
                        Text(
                          offset == 0
                              ? '今天'
                              : offset == 7
                              ? '部分時段'
                              : '$offset 天前',
                          style: const TextStyle(fontSize: 11),
                        ),
                      ],
                    ),
                  ),
                ),
              );
            },
          ),
      ],
    );
  }
}

class _LiveViewData {
  const _LiveViewData({required this.camera, required this.session});

  final Camera camera;
  final StreamSession session;
}
