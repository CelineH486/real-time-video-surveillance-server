class StreamSession {
  const StreamSession({
    required this.truckId,
    required this.cameraId,
    required this.quality,
    required this.protocol,
    required this.url,
    required this.accessToken,
    required this.expiresAt,
  });

  final String truckId;
  final String cameraId;
  final String quality;
  final String protocol;
  final String url;
  final String accessToken;
  final DateTime expiresAt;

  factory StreamSession.fromJson(Map<String, dynamic> json) {
    return StreamSession(
      truckId: json['truckId'] as String? ?? '',
      cameraId: json['cameraId'] as String? ?? '',
      quality: json['quality'] as String? ?? '',
      protocol: json['protocol'] as String? ?? '',
      url: json['url'] as String? ?? '',
      accessToken: json['accessToken'] as String? ?? '',
      expiresAt: DateTime.parse(json['expiresAt'] as String),
    );
  }
}
