class Camera {
  const Camera({
    required this.cameraId,
    required this.truckId,
    required this.name,
    required this.status,
    this.subUrl,
    this.subToken,
    this.expiresAt,
  });

  final String cameraId;
  final String truckId;
  final String name;
  final String status;
  final String? subUrl;
  final String? subToken;
  final DateTime? expiresAt;

  bool get isOnline => status == 'online';

  factory Camera.fromJson(Map<String, dynamic> json) {
    return Camera(
      cameraId: json['cameraId'] as String? ?? '',
      truckId: json['truckId'] as String? ?? '',
      name: json['name'] as String? ?? '',
      status: json['status'] as String? ?? 'offline',
      subUrl: json['subUrl'] as String?,
      subToken: json['subToken'] as String?,
      expiresAt: DateTime.tryParse(json['expiresAt'] as String? ?? ''),
    );
  }
}
