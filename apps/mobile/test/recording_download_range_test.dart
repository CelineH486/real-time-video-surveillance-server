import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:surveillance_app/src/models/recording.dart';
import 'package:surveillance_app/src/utils/taiwan_time.dart';
import 'package:surveillance_app/src/widgets/recording_player.dart';

void main() {
  testWidgets('prevents selecting a recording still being written', (
    tester,
  ) async {
    final day = DateTime(2026, 11, 14);
    final start = taiwanDayStartUtc(day).add(const Duration(hours: 6));
    final row = Recording(start: start, durationSeconds: 60);
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: RecordingPlayer(
              date: day,
              recordings: [row],
              refreshRecordings: () async => [row],
              loadUrl: (_) async => 'http://localhost/test.mp4',
            ),
          ),
        ),
      ),
    );
    await tester.ensureVisible(find.byIcon(Icons.download));
    await tester.tap(find.byIcon(Icons.download));
    await tester.pumpAndSettle();
    expect(find.text('錄影尚未完成或檔案檢查中'), findsOneWidget);
    await tester.tap(find.textContaining('06:00:00'), warnIfMissed: false);
    await tester.pumpAndSettle();
    expect(find.text('選擇下載區段'), findsOneWidget);
  });
  testWidgets('downloads first recording without starting playback', (
    tester,
  ) async {
    final day = DateTime(2026, 11, 14);
    final start = taiwanDayStartUtc(day).add(const Duration(hours: 6));
    var loadCount = 0;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: RecordingPlayer(
              date: day,
              recordings: [
                Recording(
                  start: start.add(const Duration(hours: 1)),
                  durationSeconds: 60,
                  downloadReady: true,
                ),
                Recording(
                  start: start,
                  durationSeconds: 60,
                  downloadReady: true,
                ),
              ],
              loadUrl: (_) async {
                loadCount++;
                return 'http://localhost/test.mp4';
              },
              refreshRecordings: () async => [
                Recording(
                  start: start,
                  durationSeconds: 120,
                  downloadReady: true,
                ),
                Recording(
                  start: start.add(const Duration(hours: 1)),
                  durationSeconds: 60,
                  downloadReady: true,
                ),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.ensureVisible(find.byIcon(Icons.download));
    await tester.tap(find.byIcon(Icons.download));
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.textContaining('06:02:00'), findsOneWidget);
    await tester.tap(find.textContaining('07:00:00'));
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.textContaining('07:01:00'), findsOneWidget);
    expect(loadCount, 0);
    await tester.tap(find.text('取消'));
    await tester.pump(const Duration(milliseconds: 500));
  });
  testWidgets(
    'refreshes the selected recording before showing download range',
    (tester) async {
      final day = DateTime(2026, 11, 14);
      final start = taiwanDayStartUtc(day).add(const Duration(hours: 6));
      final original = Recording(
        start: start,
        durationSeconds: 60,
        downloadReady: true,
      );
      final pending = Completer<String>();
      var refreshCount = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: RecordingPlayer(
                date: day,
                recordings: [original],
                loadUrl: (_) => pending.future,
                refreshRecordings: () async {
                  refreshCount++;
                  return [
                    Recording(
                      start: start,
                      durationSeconds: 120,
                      downloadReady: true,
                    ),
                  ];
                },
              ),
            ),
          ),
        ),
      );
      final track = find.byKey(const ValueKey('recording-day-track'));
      await tester.ensureVisible(track);
      final rect = tester.getRect(track);
      await tester.tapAt(
        Offset(rect.left + rect.width * (6.0 / 24), rect.center.dy),
      );
      await tester.pump();
      await tester.ensureVisible(find.byIcon(Icons.download));
      await tester.tap(find.byIcon(Icons.download));
      await tester.pump(const Duration(milliseconds: 500));
      await tester.tap(find.textContaining('06:02:00'));
      await tester.pump(const Duration(milliseconds: 500));
      expect(refreshCount, 2);
      expect(find.textContaining('06:02:00'), findsOneWidget);
      expect(find.textContaining('此段錄影未滿 30 分鐘'), findsOneWidget);
      await tester.tap(find.text('取消'));
      await tester.pump(const Duration(milliseconds: 500));
      await tester.pumpWidget(const SizedBox());
      pending.complete('http://localhost/test.mp4');
      await tester.pump();
    },
  );
}
