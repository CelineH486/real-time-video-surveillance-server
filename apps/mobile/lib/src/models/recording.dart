class Recording {
  const Recording({
    required this.start,
    required this.durationSeconds,
    required this.url,
  });

  final DateTime start;
  final double durationSeconds;
  final String url;

  // UTC+8 is explicit: viewing from another timezone must not move midnight.
  DateTime get taiwanStart => start.toUtc().add(const Duration(hours: 8));
  DateTime get taiwanEnd => taiwanStart.add(
    Duration(
      microseconds: (durationSeconds * Duration.microsecondsPerSecond).round(),
    ),
  );

  bool get isComplete =>
      durationSeconds >= 1799 ||
      (taiwanEnd.day != taiwanStart.day &&
          taiwanEnd.hour == 0 &&
          taiwanEnd.minute == 0);

  String get timeLabel {
    String clock(DateTime value) =>
        '${value.hour.toString().padLeft(2, '0')}:${value.minute.toString().padLeft(2, '0')}';
    final end = taiwanEnd;
    final endLabel =
        end.day != taiwanStart.day && end.hour == 0 && end.minute == 0
        ? '24:00'
        : clock(end);
    return '${clock(taiwanStart)}–$endLabel';
  }

  factory Recording.fromJson(Map<String, dynamic> json) {
    return Recording(
      start: DateTime.parse(json['start'] as String),
      durationSeconds: (json['durationSeconds'] as num).toDouble(),
      url: json['url'] as String,
    );
  }
}
