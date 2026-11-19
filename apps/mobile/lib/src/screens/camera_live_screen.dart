import 'package:flutter/material.dart';

import '../models/camera.dart';
import '../models/recording.dart';
import '../models/stream_session.dart';
import '../services/api_client.dart';
import '../utils/taiwan_time.dart';
import '../widgets/history_date_toolbar.dart';
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
  late Future<_LiveViewData> _data;
  Future<List<Recording>>? _recordings;
  bool _showHistory = false;
  DateTime _historyDate = taiwanCalendarDate(DateTime.now());

  List<DateTime> get _historyDates {
    final today = taiwanCalendarDate(DateTime.now());
    return List.generate(7, (index) => today.subtract(Duration(days: index)));
  }

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
    final firstDate = _historyDates.last;
    final lastDate = taiwanCalendarDate(nowUtc);
    if (_historyDate.isBefore(firstDate) || _historyDate.isAfter(lastDate)) {
      _historyDate = lastDate;
    }
    final dayStartUtc = taiwanDayStartUtc(_historyDate);
    final nextDayUtc = taiwanDayStartUtc(
      DateTime(_historyDate.year, _historyDate.month, _historyDate.day + 1),
    );
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

  void _goBack() {
    final navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop();
      return;
    }
    navigator.pushReplacementNamed('/trucks/${widget.truckId}/cameras');
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
                      LayoutBuilder(
                        builder: (context, constraints) {
                          final stacked =
                              constraints.maxWidth < 560 ||
                              MediaQuery.textScalerOf(context).scale(14) > 17;
                          final compact =
                              constraints.maxWidth < 380 ||
                              MediaQuery.textScalerOf(context).scale(14) > 20;
                          final title = Text(
                            data.camera.name,
                            style: Theme.of(context).textTheme.headlineSmall,
                          );
                          final switcher = SegmentedButton<bool>(
                            expandedInsets: stacked ? EdgeInsets.zero : null,
                            showSelectedIcon: false,
                            segments: [
                              ButtonSegment(
                                value: false,
                                icon: compact
                                    ? null
                                    : const Icon(Icons.live_tv),
                                label: const Text(
                                  '即時影像',
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                ),
                              ),
                              ButtonSegment(
                                value: true,
                                icon: compact
                                    ? null
                                    : const Icon(Icons.history),
                                label: const Text(
                                  '歷史錄影',
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                ),
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
                          );
                          if (stacked) {
                            return Column(
                              crossAxisAlignment: CrossAxisAlignment.stretch,
                              children: [
                                title,
                                const SizedBox(height: 8),
                                switcher,
                              ],
                            );
                          }
                          return Row(
                            children: [
                              Expanded(child: title),
                              const SizedBox(width: 12),
                              switcher,
                            ],
                          );
                        },
                      ),
                      const SizedBox(height: 16),
                      if (_showHistory) ...[
                        HistoryDateToolbar(
                          dates: _historyDates,
                          selectedDate: _historyDate,
                          summaryText: '選擇日期查看錄影',
                          onChanged: (date) {
                            _historyDate = date;
                            _loadHistory();
                          },
                          onRefresh: _loadHistory,
                        ),
                        const SizedBox(height: 12),
                      ],
                      if (_showHistory)
                        FutureBuilder<List<Recording>>(
                          future: _recordings,
                          builder: (context, history) {
                            if (history.connectionState !=
                                ConnectionState.done) {
                              return const Center(
                                child: CircularProgressIndicator(),
                              );
                            }
                            if (history.hasError) {
                              return const Text('讀取歷史錄影失敗，請重新整理');
                            }
                            return RecordingPlayer(
                              key: ObjectKey(_recordings),
                              date: _historyDate,
                              recordings: history.data ?? const [],
                              refreshRecordings: () =>
                                  widget.apiClient.getRecordings(
                                    truckId: widget.truckId,
                                    cameraId: widget.cameraId,
                                    start: taiwanDayStartUtc(
                                      _historyDate,
                                    ).subtract(const Duration(days: 1)),
                                    end: taiwanDayStartUtc(
                                      DateTime(
                                        _historyDate.year,
                                        _historyDate.month,
                                        _historyDate.day + 1,
                                      ),
                                    ),
                                  ),
                              loadUrl: (recording) =>
                                  widget.apiClient.createRecordingPlaySession(
                                    truckId: widget.truckId,
                                    cameraId: widget.cameraId,
                                    recording: recording,
                                  ),
                            );
                          },
                        )
                      else
                        AspectRatio(
                          aspectRatio: 16 / 9,
                          child: ClipRRect(
                            borderRadius: BorderRadius.circular(16),
                            child: WhepVideoPlayer(
                              url: data.session.url,
                              token: data.session.accessToken,
                              muted: true,
                              onAuthenticationExpired:
                                  _handleLiveAuthenticationExpired,
                            ),
                          ),
                        ),
                      const SizedBox(height: 16),
                      if (!_showHistory)
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

class _LiveViewData {
  const _LiveViewData({required this.camera, required this.session});

  final Camera camera;
  final StreamSession session;
}
