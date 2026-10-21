import 'package:flutter_test/flutter_test.dart';
import 'package:surveillance_app/src/widgets/history_date_toolbar.dart';

void main() {
  test('formats today with weekday and today marker', formatTodayLabel);
  test(
    'formats yesterday with weekday and yesterday marker',
    formatYesterdayLabel,
  );
  test('formats older dates with weekday only', formatOlderDateLabel);
}

void formatTodayLabel() {
  // Arrange
  final today = DateTime(2026, 10, 14);

  // Act
  final label = formatHistoryDateLabel(today, today);

  // Assert
  expect(label, '2026/10/14（三）・今天');
}

void formatYesterdayLabel() {
  // Arrange
  final today = DateTime(2026, 10, 14);
  final yesterday = DateTime(2026, 10, 13);

  // Act
  final label = formatHistoryDateLabel(yesterday, today);

  // Assert
  expect(label, '2026/10/13（二）・昨天');
}

void formatOlderDateLabel() {
  // Arrange
  final today = DateTime(2026, 10, 14);
  final olderDate = DateTime(2026, 10, 12);

  // Act
  final label = formatHistoryDateLabel(olderDate, today);

  // Assert
  expect(label, '2026/10/12（一）');
}
