import 'package:flutter_test/flutter_test.dart';
import 'package:surveillance_app/src/utils/taiwan_time.dart';

void main() {
  test('maps an instant to its Taiwan calendar date', () {
    expect(
      taiwanCalendarDate(DateTime.utc(2026, 9, 17, 16, 30)),
      DateTime(2026, 9, 18),
    );
  });

  test('builds Taiwan midnight as an explicit UTC instant', () {
    expect(
      taiwanDayStartUtc(DateTime(2026, 9, 18)),
      DateTime.utc(2026, 9, 17, 16),
    );
  });
}
