class TruckLocation {
  const TruckLocation({
    required this.truckId,
    required this.latitude,
    required this.longitude,
    required this.speedKmh,
    required this.fixQuality,
    required this.recordedAt,
    required this.receivedAt,
    this.altitudeM,
    this.headingDegrees,
    this.accuracyM,
    this.satellites,
    this.stoppedSince,
    this.stoppedSeconds,
  });

  final String truckId;
  final double latitude;
  final double longitude;
  final double? altitudeM;
  final double speedKmh;
  final double? headingDegrees;
  final double? accuracyM;
  final int? satellites;
  final int fixQuality;
  final DateTime recordedAt;
  final DateTime receivedAt;
  final DateTime? stoppedSince;
  final int? stoppedSeconds;

  factory TruckLocation.fromJson(Map<String, dynamic> json) {
    return TruckLocation(
      truckId: json['truckId'] as String,
      latitude: (json['latitude'] as num).toDouble(),
      longitude: (json['longitude'] as num).toDouble(),
      altitudeM: (json['altitudeM'] as num?)?.toDouble(),
      speedKmh: (json['speedKmh'] as num).toDouble(),
      headingDegrees: (json['headingDegrees'] as num?)?.toDouble(),
      accuracyM: (json['accuracyM'] as num?)?.toDouble(),
      satellites: (json['satellites'] as num?)?.toInt(),
      fixQuality: (json['fixQuality'] as num).toInt(),
      recordedAt: DateTime.parse(json['recordedAt'] as String),
      receivedAt: DateTime.parse(json['receivedAt'] as String),
      stoppedSince: json['stoppedSince'] == null
          ? null
          : DateTime.parse(json['stoppedSince'] as String),
      stoppedSeconds: (json['stoppedSeconds'] as num?)?.toInt(),
    );
  }
}
