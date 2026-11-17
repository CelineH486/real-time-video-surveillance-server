import 'package:flutter/material.dart';

import '../models/recording.dart';
import '../utils/taiwan_time.dart';

/// Half-open intervals ensure a clip boundary selects the following recording.
Recording? recordingAt(List<Recording> recordings, DateTime time) {
  for (final r in recordings) {
    final end = r.start.add(
      Duration(microseconds: (r.durationSeconds * 1000000).round()),
    );
    if (!time.isBefore(r.start) && time.isBefore(end)) return r;
  }
  return null;
}

class RecordingDayTimeline extends StatefulWidget {
  const RecordingDayTimeline({
    super.key,
    required this.date,
    required this.recordings,
    required this.position,
    required this.playing,
    required this.loading,
    required this.onSeek,
    required this.onToggle,
    this.speed = 1,
    this.onSpeedChanged,
  });

  final double speed;
  final ValueChanged<double>? onSpeedChanged;
  final DateTime date;
  final List<Recording> recordings;
  final DateTime? position;
  final bool playing;
  final bool loading;
  final ValueChanged<DateTime> onSeek;
  final VoidCallback? onToggle;

  @override
  State<RecordingDayTimeline> createState() => _RecordingDayTimelineState();
}

class _RecordingDayTimelineState extends State<RecordingDayTimeline> {
  DateTime? _preview;

  static const _lensSeconds = 300;
  static const _lensMaxWidth = 260.0;
  DateTime? _precisionAnchor;

  DateTime get _dayStart => taiwanDayStartUtc(widget.date);

  DateTime _at(double x, double width) =>
      _timeAt(x / width * Duration.millisecondsPerDay);

  DateTime _timeAt(double milliseconds) => _dayStart.add(
    Duration(
      milliseconds: milliseconds.round().clamp(
        0,
        Duration.millisecondsPerDay - 1,
      ),
    ),
  );

  void _finishDrag() {
    final target = _preview;
    _cancelDrag();
    if (target != null) widget.onSeek(target);
  }

  void _cancelDrag() => setState(() {
    _preview = null;
    _precisionAnchor = null;
  });

  String _label(DateTime time) {
    final t = time.toUtc().add(taiwanUtcOffset);
    String two(int n) => n.toString().padLeft(2, '0');
    return '${t.year}-${two(t.month)}-${two(t.day)} ${two(t.hour)}:${two(t.minute)}:${two(t.second)}';
  }

  @override
  Widget build(BuildContext context) {
    final shown = _preview ?? widget.position;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: const Color(0xFF0D141B),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: const Color(0xFF263540)),
      ),
      child: Column(
        children: [
          const Row(
            children: [
              Icon(Icons.history, color: Color(0xFF35E6A5), size: 22),
              SizedBox(width: 8),
              Text(
                '歷史回放',
                style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
              ),
              Spacer(),
              Text(
                '台灣時間',
                style: TextStyle(color: Colors.white54, fontSize: 12),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Text(
            shown == null ? '選擇錄影時間' : _label(shown),
            style: const TextStyle(
              fontSize: 18,
              fontFeatures: [FontFeature.tabularFigures()],
            ),
          ),
          const SizedBox(height: 12),
          SizedBox(
            height: 105,
            child: LayoutBuilder(
              builder: (context, constraints) {
                final preview = _preview;
                if (preview == null) {
                  return const Center(child: Text('拖曳定位 · 按住後拖曳可微調'));
                }
                final width = constraints.maxWidth.clamp(0.0, _lensMaxWidth);
                final elapsed = preview.difference(_dayStart).inMilliseconds;
                final lensStart = (elapsed - _lensSeconds * 500).clamp(
                  0,
                  Duration.millisecondsPerDay - _lensSeconds * 1000,
                );
                final x =
                    elapsed /
                    Duration.millisecondsPerDay *
                    constraints.maxWidth;
                return Stack(
                  children: [
                    Positioned(
                      left: (x - width / 2).clamp(
                        0.0,
                        constraints.maxWidth - width,
                      ),
                      width: width,
                      top: 0,
                      child: DecoratedBox(
                        key: const ValueKey('recording-timeline-lens'),
                        decoration: BoxDecoration(
                          color: const Color(0xFF0D141B),
                          border: Border.all(color: const Color(0xFF35E6A5)),
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 8),
                          child: CustomPaint(
                            size: const Size(double.infinity, 100),
                            painter: _DayPainter(
                              _dayStart.add(Duration(milliseconds: lensStart)),
                              _lensSeconds,
                              widget.recordings,
                              preview,
                            ),
                          ),
                        ),
                      ),
                    ),
                  ],
                );
              },
            ),
          ),
          LayoutBuilder(
            builder: (context, constraints) => MouseRegion(
              cursor: SystemMouseCursors.click,
              child: GestureDetector(
                behavior: HitTestBehavior.opaque,
                onTapUp: (d) => widget.onSeek(
                  _at(d.localPosition.dx, constraints.maxWidth),
                ),
                onHorizontalDragStart: (d) => setState(
                  () =>
                      _preview = _at(d.localPosition.dx, constraints.maxWidth),
                ),
                onHorizontalDragUpdate: (d) => setState(
                  () =>
                      _preview = _at(d.localPosition.dx, constraints.maxWidth),
                ),
                onHorizontalDragEnd: (_) {
                  _finishDrag();
                },
                onHorizontalDragCancel: _cancelDrag,
                onLongPressStart: (details) => setState(() {
                  _precisionAnchor = _at(
                    details.localPosition.dx,
                    constraints.maxWidth,
                  );
                  _preview = _precisionAnchor;
                }),
                onLongPressMoveUpdate: (details) {
                  final anchor = _precisionAnchor;
                  if (anchor == null) return;
                  final lensWidth = constraints.maxWidth.clamp(
                    1.0,
                    _lensMaxWidth,
                  );
                  setState(
                    () => _preview = _timeAt(
                      anchor.difference(_dayStart).inMilliseconds +
                          details.offsetFromOrigin.dx /
                              lensWidth *
                              _lensSeconds *
                              1000,
                    ),
                  );
                },
                onLongPressEnd: (_) => _finishDrag(),
                child: CustomPaint(
                  key: const ValueKey('recording-day-track'),
                  size: Size(constraints.maxWidth, 100),
                  painter: _DayPainter(
                    _dayStart,
                    86400,
                    widget.recordings,
                    shown,
                  ),
                ),
              ),
            ),
          ),
          Wrap(
            alignment: WrapAlignment.center,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              IconButton.filledTonal(
                tooltip: widget.playing ? '暫停' : '播放',
                onPressed: widget.loading ? null : widget.onToggle,
                icon: Icon(widget.playing ? Icons.pause : Icons.play_arrow),
              ),
              const SizedBox(width: 8),
              DropdownButton<double>(
                value: widget.speed,
                items: [0.25, 0.5, 1.0, 1.5, 2.0, 4.0]
                    .map(
                      (speed) => DropdownMenuItem(
                        value: speed,
                        child: Text(
                          '${speed == 1 ? '1.0' : speed.toString()}×',
                        ),
                      ),
                    )
                    .toList(),
                onChanged: widget.loading || widget.onSpeedChanged == null
                    ? null
                    : (value) {
                        if (value != null) widget.onSpeedChanged!(value);
                      },
              ),
              const SizedBox(width: 16),
              const Icon(Icons.square, color: Color(0xFF35E6A5), size: 12),
              const SizedBox(width: 5),
              const Text(
                '有錄影',
                style: TextStyle(fontSize: 12, color: Colors.white60),
              ),
              const SizedBox(width: 14),
              const Text(
                '空白：無錄影',
                style: TextStyle(fontSize: 12, color: Colors.white60),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _DayPainter extends CustomPainter {
  _DayPainter(this.start, this.span, this.recordings, this.position);
  final int span;
  final DateTime start;
  final List<Recording> recordings;
  final DateTime? position;

  double _x(DateTime time, double width) =>
      (time.difference(start).inMicroseconds / (span * 1000000)).clamp(
        0.0,
        1.0,
      ) *
      width;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint();
    canvas.drawRect(
      Rect.fromLTWH(0, 12, size.width, 48),
      paint..color = const Color(0xFF17232D),
    );
    for (final r in recordings) {
      final left = _x(r.start, size.width);
      final right = _x(
        r.start.add(
          Duration(microseconds: (r.durationSeconds * 1000000).round()),
        ),
        size.width,
      );
      if (right > left) {
        canvas.drawRect(
          Rect.fromLTRB(left, 12, right, 60),
          paint..color = const Color(0xFF35E6A5),
        );
      }
    }
    final divisions = size.width < 500 ? 4 : 6;
    for (var i = 0; i <= divisions; i++) {
      final x = size.width * i / divisions;
      final t = start
          .add(Duration(seconds: (span * i / divisions).round()))
          .toUtc()
          .add(taiwanUtcOffset);
      String two(int n) => n.toString().padLeft(2, '0');
      final text = i == divisions && t.hour == 0 && t.minute == 0
          ? '24:00'
          : '${two(t.hour)}:${two(t.minute)}${span <= 600 ? ':${two(t.second)}' : ''}';
      canvas.drawLine(
        Offset(x, 60),
        Offset(x, 69),
        paint
          ..color = Colors.white38
          ..strokeWidth = 1,
      );
      final label = TextPainter(
        text: TextSpan(
          text: text,
          style: const TextStyle(fontSize: 11, color: Colors.white60),
        ),
        textDirection: TextDirection.ltr,
      )..layout();
      label.paint(
        canvas,
        Offset((x - label.width / 2).clamp(0.0, size.width - label.width), 75),
      );
    }
    if (position != null &&
        !position!.isBefore(start) &&
        !position!.isAfter(start.add(Duration(seconds: span)))) {
      final x = _x(position!, size.width);
      canvas.drawLine(
        Offset(x, 4),
        Offset(x, 66),
        paint
          ..color = const Color(0xFFFF626C)
          ..strokeWidth = 2,
      );
      canvas.drawCircle(Offset(x, 5), 4, paint);
    }
  }

  @override
  bool shouldRepaint(_DayPainter old) => true;
}
