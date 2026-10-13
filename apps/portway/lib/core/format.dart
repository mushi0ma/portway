import 'package:intl/intl.dart';

// Russian time formatting shared by all screens. Every function takes `now`
// explicitly so widget tests and goldens are deterministic.

bool _sameDay(DateTime a, DateTime b) =>
    a.year == b.year && a.month == b.month && a.day == b.day;

/// "14:06", "завтра, 14:06", "вчера, 09:12" or "19 окт, 14:06" in local time.
String formatMoment(DateTime value, DateTime now) {
  final local = value.toLocal(), today = now.toLocal();
  final time = DateFormat.Hm('ru').format(local);
  if (_sameDay(local, today)) return time;
  if (_sameDay(local, today.add(const Duration(days: 1)))) {
    return 'завтра, $time';
  }
  if (_sameDay(local, today.subtract(const Duration(days: 1)))) {
    return 'вчера, $time';
  }
  final date = DateFormat('d MMM', 'ru').format(local).replaceAll('.', '');
  return local.year == today.year
      ? '$date, $time'
      : '${DateFormat('d MMM y', 'ru').format(local).replaceAll('.', '')}, $time';
}

/// Compact duration: "3 ч 40 мин", "12 мин", "2 дн 4 ч", "меньше минуты".
String formatSpan(Duration d) {
  if (d.isNegative) d = Duration.zero;
  if (d.inSeconds < 60) return 'меньше минуты';
  final minutes = (d.inSeconds / 60).ceil();
  // Non-breaking spaces keep a number with its unit when the line wraps.
  if (minutes < 60) return '$minutes\u00a0мин';
  final days = minutes ~/ 1440, hours = (minutes % 1440) ~/ 60;
  final rest = minutes % 60;
  if (days > 0) {
    return hours == 0 ? '$days\u00a0дн' : '$days\u00a0дн $hours\u00a0ч';
  }
  return rest == 0 ? '$hours\u00a0ч' : '$hours\u00a0ч $rest\u00a0мин';
}

/// "через 3 ч 40 мин"; past moments become "сейчас".
String formatUntil(DateTime value, DateTime now) {
  final d = value.difference(now);
  return d.inSeconds <= 0 ? 'сейчас' : 'через ${formatSpan(d)}';
}

/// "только что", "5 мин назад", "2 ч назад", otherwise the moment.
String formatAgo(DateTime value, DateTime now) {
  final d = now.difference(value);
  if (d.inSeconds < 45) return 'только что';
  if (d.inMinutes < 60) return '${d.inMinutes.clamp(1, 59)} мин назад';
  if (d.inHours < 12) return '${d.inHours} ч назад';
  return formatMoment(value, now);
}

/// Run duration: "48 с", "3 мин 05 с".
String formatElapsed(Duration d) {
  final s = d.inSeconds;
  if (s < 60) return '$s с';
  final m = s ~/ 60, rest = s % 60;
  return '$m мин ${rest.toString().padLeft(2, '0')} с';
}

String formatCount(int value) =>
    NumberFormat.decimalPattern('ru').format(value).replaceAll(' ', ' ');
