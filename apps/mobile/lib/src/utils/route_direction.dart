import 'dart:math' as math;

import 'package:latlong2/latlong.dart';

double routeStartAngle(List<LatLng> points) {
  if (points.length < 2) return 0;
  final start = points.first;
  for (final next in points.skip(1)) {
    final latitudeDelta = next.latitude - start.latitude;
    final longitudeDelta = next.longitude - start.longitude;
    if (latitudeDelta.abs() < 0.0000001 && longitudeDelta.abs() < 0.0000001) {
      continue;
    }
    final meanLatitude = (start.latitude + next.latitude) / 2;
    final east = longitudeDelta * math.cos(meanLatitude * math.pi / 180);
    final north = latitudeDelta;
    return math.atan2(-north, east);
  }
  return 0;
}
