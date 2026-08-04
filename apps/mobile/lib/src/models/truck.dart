class Truck {
  const Truck({
    required this.truckId,
    required this.plateNo,
    required this.status,
  });

  final String truckId;
  final String plateNo;
  final String status;

  factory Truck.fromJson(Map<String, dynamic> json) {
    return Truck(
      truckId: json['truckId'] as String? ?? '',
      plateNo: json['plateNo'] as String? ?? '',
      status: json['status'] as String? ?? 'offline',
    );
  }
}
