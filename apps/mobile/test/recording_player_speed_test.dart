import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:surveillance_app/src/models/recording.dart';
import 'package:surveillance_app/src/widgets/recording_player.dart';
import 'package:video_player_platform_interface/video_player_platform_interface.dart';

class FakeVideoPlatform extends VideoPlayerPlatform {
  int nextId = 0;
  final events = <int, StreamController<VideoEvent>>{};
  final rates = <int, double>{};
  final seeks = <Duration>[];
  @override
  Future<void> init() async {}
  @override
  Future<int?> createWithOptions(VideoCreationOptions options) async {
    final id = ++nextId;
    events[id] = StreamController<VideoEvent>.broadcast();
    return id;
  }

  @override
  Stream<VideoEvent> videoEventsFor(int id) => events[id]!.stream;
  @override
  Future<void> dispose(int id) async {
    await events[id]?.close();
  }

  @override
  Future<void> play(int id) async {}
  @override
  Future<void> pause(int id) async {}
  @override
  Future<void> setLooping(int id, bool looping) async {}
  @override
  Future<void> setVolume(int id, double volume) async {}
  @override
  Future<void> setPlaybackSpeed(int id, double speed) async {
    rates[id] = speed;
  }

  @override
  Future<void> seekTo(int id, Duration position) async {
    seeks.add(position);
  }

  @override
  Future<Duration> getPosition(int id) async => Duration.zero;
  @override
  Widget buildViewWithOptions(VideoViewOptions options) => const SizedBox();
}

void main() {
  testWidgets(
    'loads the whole recording and seeks without replacing the player',
    (tester) async {
      final platform = FakeVideoPlatform();
      VideoPlayerPlatform.instance = platform;
      final start = DateTime.utc(2026, 11, 10);
      final requested = <DateTime>[];
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 800,
              height: 400,
              child: RecordingPlayer(
                recording: Recording(start: start, durationSeconds: 1800),
                loadUrl: () async {
                  requested.add(start);
                  return 'http://localhost/test.mp4';
                },
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      platform.events[1]!.add(
        VideoEvent(
          eventType: VideoEventType.initialized,
          duration: const Duration(minutes: 30),
          size: const Size(320, 180),
        ),
      );
      await tester.pump();
      await tester.pump();
      await tester.tap(find.byType(DropdownButton<double>));
      await tester.pumpAndSettle();
      await tester.tap(find.text('8.0×').last);
      await tester.pumpAndSettle();
      expect(platform.rates[1], 8);
      final timeline = find.byWidgetPredicate(
        (widget) =>
            widget is GestureDetector && widget.onHorizontalDragEnd != null,
      );
      await tester.tapAt(tester.getCenter(timeline));
      await tester.pump();
      expect(requested, [start]);
      expect(platform.nextId, 1);
      expect(platform.seeks.last.inSeconds, closeTo(900, 2));
      expect(platform.rates[1], 8);
      await tester.pumpWidget(const SizedBox());
    },
  );
}
