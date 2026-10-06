import 'package:flutter/material.dart';

import '../models/camera.dart';
import '../models/recording.dart';
import '../models/stream_session.dart';
import '../services/api_client.dart';
import '../utils/taiwan_time.dart';
import '../widgets/recording_player.dart';
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
  final GlobalKey _historyPlayerKey = GlobalKey();
  late Future<_LiveViewData> _data;
  Future<List<Recording>>? _recordings;
  Recording? _selectedRecording;
  bool _showHistory = false;
  DateTime _historyDate = taiwanCalendarDate(DateTime.now());

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

  void _reloadLive() {
    setState(() {
      _showHistory = false;
      _data = _load();
    });
  }

  void _handleLiveAuthenticationExpired() {
    // A WHEP request can finish after the user has switched to history.
    // Ignore that stale live-player callback instead of changing their mode.
    if (!mounted || _showHistory) return;
    _reloadLive();
  }

  void _loadHistory() {
    final nowUtc = DateTime.now().toUtc();
    final cutoffUtc = nowUtc.subtract(const Duration(days: 7));
    final firstDate = taiwanCalendarDate(cutoffUtc);
    final lastDate = taiwanCalendarDate(nowUtc);
    if (_historyDate.isBefore(firstDate) || _historyDate.isAfter(lastDate)) {
      _historyDate = lastDate;
    }
    final dayStartUtc = taiwanDayStartUtc(_historyDate);
    final nextDayUtc = taiwanDayStartUtc(
      DateTime(_historyDate.year, _historyDate.month, _historyDate.day + 1),
    );
    _selectedRecording = null;
    setState(() {
      _showHistory = true;
      _recordings = widget.apiClient.getRecordings(
        truckId: widget.truckId,
        cameraId: widget.cameraId,
        start: dayStartUtc.isBefore(cutoffUtc) ? cutoffUtc : dayStartUtc,
        end: nextDayUtc.isAfter(nowUtc) ? nowUtc : nextDayUtc,
      );
    });
  }

  Future<void> _selectHistoryDate() async {
    final lastDate = taiwanCalendarDate(DateTime.now());
    final firstDate = lastDate.subtract(const Duration(days: 7));
    final initialDate = _historyDate.isBefore(firstDate)
        ? firstDate
        : _historyDate.isAfter(lastDate)
        ? lastDate
        : _historyDate;
    final selected = await showDatePicker(
      context: context,
      locale: const Locale('zh', 'TW'),
      initialDate: initialDate,
      firstDate: firstDate,
      lastDate: lastDate,
      initialEntryMode: DatePickerEntryMode.calendarOnly,
      helpText: '選擇錄影日期',
      cancelText: '取消',
      confirmText: '確定',
    );
    if (selected == null || !mounted) return;
    _historyDate = selected;
    _loadHistory();
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
    setState(() => _selectedRecording = recording);
    await WidgetsBinding.instance.endOfFrame;
    if (!mounted) return;
    final playerContext = _historyPlayerKey.currentContext;
    if (playerContext == null || !playerContext.mounted) return;
    await Scrollable.ensureVisible(
      playerContext,
      duration: const Duration(milliseconds: 450),
      curve: Curves.easeOutCubic,
      alignment: 0.08,
    );
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
                            OutlinedButton.icon(
                              onPressed: _selectHistoryDate,
                              icon: const Icon(Icons.calendar_month),
                              label: Text(
                                '${_historyDate.year}/${_historyDate.month.toString().padLeft(2, '0')}/${_historyDate.day.toString().padLeft(2, '0')}',
                              ),
                            ),
                            const Text('僅可選擇最近 7 天'),
                            TextButton.icon(
                              onPressed: _loadHistory,
                              icon: const Icon(Icons.refresh),
                              label: const Text('重新整理'),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                      ],
                      Column(
                        key: _historyPlayerKey,
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          if (_showHistory && _selectedRecording != null) ...[
                            _PlayingRecordingLabel(
                              recording: _selectedRecording!,
                            ),
                            const SizedBox(height: 10),
                          ],
                          AspectRatio(
                            aspectRatio: 16 / 9,
                            child: ClipRRect(
                              borderRadius: BorderRadius.circular(16),
                              child: _showHistory
                                  ? RecordingPlayer(
                                      recording: _selectedRecording,
                                      loadUrl: _selectedRecording == null
                                          ? null
                                          : (offset) => widget.apiClient
                                                .createRecordingPlaySession(
                                                  truckId: widget.truckId,
                                                  cameraId: widget.cameraId,
                                                  recording:
                                                      _selectedRecording!,
                                                  offset: offset,
                                                ),
                                    )
                                  : WhepVideoPlayer(
                                      url: data.session.url,
                                      token: data.session.accessToken,
                                      muted: true,
                                      onAuthenticationExpired:
                                          _handleLiveAuthenticationExpired,
                                    ),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 16),
                      if (_showHistory)
                        _RecordingList(
                          recordings: _recordings!,
                          onPlay: _playRecording,
                          selected: _selectedRecording,
                          cameraOnline: data.camera.status == 'online',
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

class _PlayingRecordingLabel extends StatelessWidget {
  const _PlayingRecordingLabel({required this.recording});

  final Recording recording;

  @override
  Widget build(BuildContext context) {
    final date = recording.taiwanStart;
    final dateLabel =
        '${date.year}/${date.month.toString().padLeft(2, '0')}/${date.day.toString().padLeft(2, '0')}';
    return Row(
      children: [
        const Icon(Icons.play_circle_outline, size: 19),
        const SizedBox(width: 8),
        Text(
          '目前播放：$dateLabel  ${recording.timeLabel}',
          style: Theme.of(
            context,
          ).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700),
        ),
      ],
    );
  }
}

class _RecordingList extends StatelessWidget {
  const _RecordingList({
    required this.recordings,
    required this.onPlay,
    required this.selected,
    required this.cameraOnline,
  });

  final Future<List<Recording>> recordings;
  final Future<void> Function(Recording recording) onPlay;
  final Recording? selected;
  final bool cameraOnline;

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<Recording>>(
      future: recordings,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const Center(child: CircularProgressIndicator());
        }
        if (snapshot.hasError) {
          return const Text('讀取歷史錄影失敗');
        }
        final rows = (snapshot.data ?? const <Recording>[]).toList();
        if (rows.isEmpty) return const Text('尚無歷史錄影');
        final latest = rows.reduce(
          (current, candidate) =>
              candidate.start.isAfter(current.start) ? candidate : current,
        );
        return Wrap(
          spacing: 10,
          runSpacing: 10,
          children: [
            for (final recording in rows)
              Builder(
                builder: (context) {
                  final isSelected = _isSameRecording(selected, recording);
                  final isRecording = _isRecording(recording, latest);
                  return SizedBox(
                    width: 270,
                    height: 58,
                    child: OutlinedButton(
                      style: OutlinedButton.styleFrom(
                        alignment: Alignment.centerLeft,
                        padding: const EdgeInsets.symmetric(horizontal: 12),
                        backgroundColor: isSelected
                            ? const Color(0xFF35E6A5).withValues(alpha: 0.12)
                            : null,
                        side: BorderSide(
                          color: isSelected
                              ? const Color(0xFF35E6A5)
                              : Theme.of(context).colorScheme.outline,
                          width: isSelected ? 1.5 : 1,
                        ),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(10),
                        ),
                      ),
                      onPressed: () => onPlay(recording),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.center,
                        children: [
                          Icon(
                            isSelected ? Icons.play_circle : Icons.play_arrow,
                            size: 20,
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Row(
                              children: [
                                Expanded(
                                  child: Text(
                                    _formatSlot(recording),
                                    style: const TextStyle(
                                      fontWeight: FontWeight.w700,
                                    ),
                                  ),
                                ),
                                Text(
                                  _formatDuration(recording.durationSeconds),
                                  style: Theme.of(context).textTheme.bodySmall,
                                ),
                                if (isRecording) ...[
                                  const SizedBox(width: 6),
                                  const _RecordingStatusBadge(),
                                ] else if (!_isComplete(recording)) ...[
                                  const SizedBox(width: 6),
                                  Text(
                                    '部分',
                                    style: Theme.of(
                                      context,
                                    ).textTheme.labelSmall,
                                  ),
                                ],
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                  );
                },
              ),
          ],
        );
      },
    );
  }

  bool _isComplete(Recording recording) {
    return recording.isComplete;
  }

  bool _isRecording(Recording recording, Recording latest) {
    if (!cameraOnline ||
        !_isSameRecording(recording, latest) ||
        recording.isComplete) {
      return false;
    }
    final nowInTaiwan = DateTime.now().toUtc().add(const Duration(hours: 8));
    final age = nowInTaiwan.difference(recording.taiwanEnd);
    return age >= const Duration(seconds: -10) &&
        age <= const Duration(minutes: 3);
  }

  bool _isSameRecording(Recording? first, Recording second) {
    return first?.start == second.start &&
        first?.durationSeconds == second.durationSeconds;
  }

  String _formatSlot(Recording recording) => recording.timeLabel;

  String _formatDuration(double seconds) {
    final duration = Duration(seconds: seconds.round());
    final minutes = duration.inMinutes;
    if (minutes > 0) return '$minutes 分';
    return '${duration.inSeconds} 秒';
  }
}

class _RecordingStatusBadge extends StatelessWidget {
  const _RecordingStatusBadge();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: const Color(0xFF35E6A5).withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(999),
      ),
      child: const Text(
        '錄製中',
        style: TextStyle(
          color: Color(0xFF35E6A5),
          fontSize: 10,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }
}

class _LiveViewData {
  const _LiveViewData({required this.camera, required this.session});

  final Camera camera;
  final StreamSession session;
}
