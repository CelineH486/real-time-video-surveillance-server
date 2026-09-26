const taiwanUtcOffset = Duration(hours: 8);

DateTime taiwanCalendarDate(DateTime instant) {
  final taiwan = instant.toUtc().add(taiwanUtcOffset);
  return DateTime(taiwan.year, taiwan.month, taiwan.day);
}

DateTime taiwanDayStartUtc(DateTime calendarDate) {
  return DateTime.utc(
    calendarDate.year,
    calendarDate.month,
    calendarDate.day,
  ).subtract(taiwanUtcOffset);
}
