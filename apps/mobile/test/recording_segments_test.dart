import 'package:flutter_test/flutter_test.dart';
import 'package:surveillance_app/src/models/recording.dart';

void main() {
  test('uses the physical recording start instead of clock half-hours', () {
    final recording = Recording(
      start: DateTime.parse('2026-09-16T03:24:00Z'),
      durationSeconds: 1800,
    );

    expect(recording.timeLabel, '11:24–11:54');
    expect(recording.isComplete, isTrue);
  });

  test('shows Taiwan midnight as 24:00', () {
    final recording = Recording(
      start: DateTime.parse('2026-09-16T15:44:00Z'),
      durationSeconds: 16 * 60 + 1,
    );

    expect(recording.timeLabel, '23:44–24:00');
    expect(recording.isComplete, isTrue);
  });

  test('keeps a short interrupted recording marked partial', () {
    final recording = Recording(
      start: DateTime.parse('2026-09-16T03:24:00Z'),
      durationSeconds: 95,
    );

    expect(recording.timeLabel, '11:24–11:25');
    expect(recording.isComplete, isFalse);
  });
}
