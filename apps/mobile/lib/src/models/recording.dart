class Recording {
  const Recording({
    required this.start,
    required this.durationSeconds,
    required this.url,
  });

  final DateTime start;
  final double durationSeconds;
  final String url;

  factory Recording.fromJson(Map<String, dynamic> json) {
    return Recording(
      start: DateTime.parse(json['start'] as String),
      durationSeconds: (json['durationSeconds'] as num).toDouble(),
      url: json['url'] as String,
    );
  }
}
