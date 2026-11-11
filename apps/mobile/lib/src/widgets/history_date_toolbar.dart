import 'package:flutter/material.dart';

String formatHistoryDateLabel(DateTime date, DateTime today) {
  const weekdays = ['一', '二', '三', '四', '五', '六', '日'];
  String two(int value) => value.toString().padLeft(2, '0');
  final dateOnly = DateTime(date.year, date.month, date.day);
  final todayOnly = DateTime(today.year, today.month, today.day);
  final prefix =
      '${date.year}/${two(date.month)}/${two(date.day)}（${weekdays[date.weekday - 1]}）';
  if (dateOnly == todayOnly) return '$prefix・今天';
  if (dateOnly == todayOnly.subtract(const Duration(days: 1))) {
    return '$prefix・昨天';
  }
  return prefix;
}

class HistoryDateToolbar extends StatelessWidget {
  const HistoryDateToolbar({
    super.key,
    required this.dates,
    required this.selectedDate,
    required this.onChanged,
    this.loading = false,
    this.summaryText,
    this.onRefresh,
  });

  final List<DateTime> dates;
  final DateTime selectedDate;
  final ValueChanged<DateTime> onChanged;
  final bool loading;
  final String? summaryText;
  final VoidCallback? onRefresh;

  @override
  Widget build(BuildContext context) {
    final today = dates.first;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 10),
      decoration: const BoxDecoration(
        color: Color(0xFF0D141B),
        border: Border(bottom: BorderSide(color: Color(0xFF263540))),
      ),
      child: Wrap(
        spacing: 16,
        runSpacing: 8,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          const Icon(Icons.calendar_month, color: Color(0xFF35E6A5)),
          DropdownButton<DateTime>(
            value: selectedDate,
            items: dates
                .map(
                  (date) => DropdownMenuItem(
                    value: date,
                    child: Text(formatHistoryDateLabel(date, today)),
                  ),
                )
                .toList(growable: false),
            onChanged: loading
                ? null
                : (date) {
                    if (date != null) onChanged(date);
                  },
          ),
          if (loading)
            const SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          else if (summaryText != null)
            Text(summaryText!, style: const TextStyle(color: Colors.white60)),
          const Text('可選擇最近 7 天', style: TextStyle(color: Colors.white38)),
          if (onRefresh != null)
            IconButton(
              onPressed: loading ? null : onRefresh,
              icon: const Icon(Icons.refresh),
              tooltip: '重新整理',
            ),
        ],
      ),
    );
  }
}
