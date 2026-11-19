import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:surveillance_app/src/models/recording.dart';
import 'package:surveillance_app/src/utils/taiwan_time.dart';
import 'package:surveillance_app/src/widgets/recording_player.dart';

void main() {
  testWidgets('gap clicks do not create playback requests; valid time does', (
    tester,
  ) async {
    final day = DateTime(2026, 11, 14);
    final start = taiwanDayStartUtc(day);
    final requests = <Recording>[];
    final pending = Completer<String>();
    final recording = Recording(
      start: start.add(const Duration(hours: 6)),
      durationSeconds: 3600,
    );
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: RecordingPlayer(
              date: day,
              recordings: [recording],
              loadUrl: (r) {
                requests.add(r);
                return pending.future;
              },
            ),
          ),
        ),
      ),
    );
    final track = find.byKey(const ValueKey('recording-day-track'));
    await tester.ensureVisible(track);
    final rect = tester.getRect(track);
    await tester.tapAt(Offset(rect.left + rect.width / 2, rect.center.dy));
    await tester.pump();
    expect(find.text('此時段無錄影'), findsOneWidget);
    expect(requests, isEmpty);
    await tester.tapAt(
      Offset(rect.left + rect.width * 6.5 / 24, rect.center.dy),
    );
    await tester.pump();
    expect(requests, [recording]);
    expect(find.text('2026-11-14 06:30:00'), findsOneWidget);
    // A request finishing after navigation must not initialize a video player.
    await tester.pumpWidget(const SizedBox());
    pending.complete('https://example.invalid/recording.mp4');
    await tester.pump();
    expect(tester.takeException(), isNull);
  });
}
