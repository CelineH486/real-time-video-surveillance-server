import 'dart:ui' show SemanticsAction;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:surveillance_app/src/models/recording.dart';
import 'package:surveillance_app/src/utils/taiwan_time.dart';
import 'package:surveillance_app/src/widgets/recording_day_timeline.dart';

void main() {
  final day = DateTime(2026, 11, 14);
  final start = taiwanDayStartUtc(day);
  final first = Recording(
    start: start.add(const Duration(hours: 6)),
    durationSeconds: 3600,
  );
  final second = Recording(
    start: start.add(const Duration(hours: 8)),
    durationSeconds: 3600,
  );
  final rows = [first, second];

  test('maps Taiwan time to clips and preserves missing intervals', () {
    expect(recordingAt(rows, start.add(const Duration(hours: 6))), same(first));
    expect(recordingAt(rows, start.add(const Duration(hours: 7))), isNull);
    expect(
      recordingAt(rows, start.add(const Duration(hours: 7, minutes: 30))),
      isNull,
    );
    expect(
      recordingAt(rows, start.add(const Duration(hours: 8))),
      same(second),
    );
    expect(recordingAt(rows, start.add(const Duration(days: 1))), isNull);
  });

  testWidgets('hold opens the lens and drag advances at the magnified scale', (
    tester,
  ) async {
    DateTime? selected;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: RecordingDayTimeline(
            date: day,
            recordings: rows,
            position: start.add(const Duration(hours: 8)),
            playing: false,
            loading: false,
            onSeek: (time) => selected = time,
            onToggle: () {},
          ),
        ),
      ),
    );
    final rect = tester.getRect(
      find.byKey(const ValueKey('recording-day-track')),
    );
    final gesture = await tester.startGesture(
      Offset(rect.left + rect.width / 3, rect.center.dy),
    );
    await tester.pump(const Duration(milliseconds: 550));
    expect(
      find.byKey(const ValueKey('recording-timeline-lens')),
      findsOneWidget,
    );
    await gesture.moveBy(const Offset(26, 0));
    await tester.pump();
    expect(selected, isNull);
    await gesture.up();
    await tester.pump();
    expect(selected, start.add(const Duration(hours: 8, seconds: 30)));
    expect(find.byKey(const ValueKey('recording-timeline-lens')), findsNothing);
  });

  testWidgets('full day track maps taps and drag releases to real times', (
    tester,
  ) async {
    DateTime? selected;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: RecordingDayTimeline(
            date: day,
            recordings: rows,
            position: first.start,
            playing: false,
            loading: false,
            onSeek: (value) => selected = value,
            onToggle: () {},
          ),
        ),
      ),
    );
    expect(find.text('2026-11-14 06:00:00'), findsOneWidget);
    final track = find.byKey(const ValueKey('recording-day-track'));
    final rect = tester.getRect(track);
    await tester.tapAt(Offset(rect.left + rect.width / 3, rect.center.dy));
    expect(selected, start.add(const Duration(hours: 8)));
    selected = null;
    final gesture = await tester.startGesture(
      Offset(rect.left + rect.width / 4, rect.center.dy),
    );
    await gesture.moveTo(Offset(rect.left + rect.width / 2, rect.center.dy));
    await tester.pump();
    expect(selected, isNull);
    await gesture.up();
    expect(selected, start.add(const Duration(hours: 12)));
    expect(tester.takeException(), isNull);
  });

  testWidgets('empty narrow timeline remains usable without overflow', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(360, 640);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: RecordingDayTimeline(
            date: day,
            recordings: const [],
            position: null,
            playing: false,
            loading: false,
            onSeek: (_) {},
            onToggle: null,
          ),
        ),
      ),
    );
    expect(find.text('選擇錄影時間'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('timeline preserves pointer position and semantic adjustments', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    DateTime? selected;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: RecordingDayTimeline(
            date: day,
            recordings: rows,
            position: first.start,
            playing: false,
            loading: false,
            onSeek: (value) => selected = value,
            onToggle: () {},
          ),
        ),
      ),
    );
    final track = tester.getRect(
      find.byKey(const ValueKey('recording-day-track')),
    );
    await tester.tapAt(Offset(track.left + track.width / 3, track.center.dy));
    expect(selected, start.add(const Duration(hours: 8)));
    final node = tester.getSemantics(find.bySemanticsLabel('錄影時間軸'));
    node.owner!.performAction(
      node.id,
      SemanticsAction.increase,
    );
    expect(selected, start.add(const Duration(hours: 6, minutes: 30)));
    node.owner!.performAction(
      node.id,
      SemanticsAction.decrease,
    );
    expect(selected, start.add(const Duration(hours: 5, minutes: 30)));
    semantics.dispose();
  });
}
