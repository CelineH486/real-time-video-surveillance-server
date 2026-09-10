class Recording {
  const Recording({
    required this.start,
    required this.durationSeconds,
    required this.url,
  });

  final DateTime start;
  final double durationSeconds;
  final String url;

  /// Split a continuous playable span at local clock half-hour boundaries.
  /// Preserve gaps by splitting each source span independently.
  List<Recording> halfHourSegments() {
    if (!durationSeconds.isFinite || durationSeconds <= 0) return [];
    var cursor = start.toLocal();
    final end = cursor.add(
      Duration(
        microseconds: (durationSeconds * Duration.microsecondsPerSecond)
            .round(),
      ),
    );
    final result = <Recording>[];
    while (cursor.isBefore(end)) {
      final boundary = cursor
          .subtract(
            Duration(
              minutes: cursor.minute % 30,
              seconds: cursor.second,
              milliseconds: cursor.millisecond,
              microseconds: cursor.microsecond,
            ),
          )
          .add(const Duration(minutes: 30));
      final stop = boundary.isBefore(end) ? boundary : end;
      final seconds =
          stop.difference(cursor).inMicroseconds /
          Duration.microsecondsPerSecond;
      final uri = Uri.parse(url);
      result.add(
        Recording(
          start: cursor,
          durationSeconds: seconds,
          url: uri
              .replace(
                queryParameters: {
                  ...uri.queryParameters,
                  'start': cursor.toUtc().toIso8601String(),
                  'duration': seconds.toString(),
                },
              )
              .toString(),
        ),
      );
      cursor = stop;
    }
    return result;
  }

  factory Recording.fromJson(Map<String, dynamic> json) {
    return Recording(
      start: DateTime.parse(json['start'] as String),
      durationSeconds: (json['durationSeconds'] as num).toDouble(),
      url: json['url'] as String,
    );
  }
}
