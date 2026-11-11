import 'dart:math' as math;

import 'package:flutter_test/flutter_test.dart';
import 'package:latlong2/latlong.dart';
import 'package:surveillance_app/src/utils/route_direction.dart';

void main() {
  test('points the start arrow east', pointStartArrowEast);
  test('points the start arrow west', pointStartArrowWest);
  test(
    'skips duplicate points before calculating direction',
    skipDuplicateStartPoints,
  );
}

void pointStartArrowEast() {
  // Arrange
  const points = [LatLng(25, 121), LatLng(25, 121.01)];

  // Act
  final angle = routeStartAngle(points);

  // Assert
  expect(angle, closeTo(0, 0.001));
}

void pointStartArrowWest() {
  // Arrange
  const points = [LatLng(25, 121), LatLng(25, 120.99)];

  // Act
  final angle = routeStartAngle(points);

  // Assert
  expect(angle.abs(), closeTo(math.pi, 0.001));
}

void skipDuplicateStartPoints() {
  // Arrange
  const points = [LatLng(25, 121), LatLng(25, 121), LatLng(25.01, 121)];

  // Act
  final angle = routeStartAngle(points);

  // Assert
  expect(angle, closeTo(-math.pi / 2, 0.001));
}
