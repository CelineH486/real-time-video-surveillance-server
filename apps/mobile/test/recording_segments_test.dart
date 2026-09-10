import 'package:flutter_test/flutter_test.dart';
import 'package:surveillance_app/src/models/recording.dart';

void main() {
  test('splits continuous recording at half hours and preserves URL token', () {
    final start = DateTime(2026, 9, 10, 22, 2, 54);
    final source = Recording(
      start: start,
      durationSeconds: 2532,
      url: 'https://example.com/content?token=example&start=old&duration=2532',
    );
    final parts = source.halfHourSegments();
    expect(parts.length, 2);
    expect(parts.first.durationSeconds, 1626);
    expect(parts.last.start, DateTime(2026, 9, 10, 22, 30));
    expect(parts.last.durationSeconds, 906);
    for (final part in parts) {
      final query = Uri.parse(part.url).queryParameters;
      expect(query['token'], 'example');
      expect(DateTime.parse(query['start']!), part.start.toUtc());
      expect(double.parse(query['duration']!), part.durationSeconds);
    }
  });

  test(
    'midnight, exact boundary, and fractional seconds preserve coverage',
    () {
      final parts = Recording(
        start: DateTime(2026, 9, 10, 23, 30),
        durationSeconds: 1800.25,
        url: 'https://example.com/content',
      ).halfHourSegments();
      expect(parts.map((part) => part.durationSeconds), [1800, 0.25]);
      expect(parts.last.start, DateTime(2026, 9, 11));
    },
  );

  test('separate spans do not fill recording gaps', () {
    final rows = [
      Recording(
        start: DateTime(2026, 9, 10, 10),
        durationSeconds: 60,
        url: 'https://example.com/content',
      ),
      Recording(
        start: DateTime(2026, 9, 10, 10, 5),
        durationSeconds: 60,
        url: 'https://example.com/content',
      ),
    ].expand((span) => span.halfHourSegments()).toList();
    expect(rows.length, 2);
    expect(rows.last.start.difference(rows.first.start).inMinutes, 5);
    expect(rows.map((row) => row.durationSeconds), [60, 60]);
  });
}
