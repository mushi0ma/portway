// Test wrappers around the shared contract v2 fixture data.

import 'package:portway/core/api.dart';
import 'package:portway/dev/fixture_data.dart';

/// Fixed local "now" for deterministic formatting and goldens.
final fixtureNow = DateTime(2026, 10, 13, 10, 26);
final _data = FixtureData(fixtureNow);

String iso(DateTime local) => _data.iso(local);
Json codexProvider({bool exhausted = true, bool loggedIn = true}) =>
    _data.codexProvider(exhausted: exhausted, loggedIn: loggedIn);
Json claudeProvider() => _data.claudeProvider();
Json statusFixture({
  bool connected = true,
  bool exhausted = true,
  Json? login,
  List<Json> commands = const [],
  bool legacy = false,
}) => _data.statusFixture(
  connected: connected,
  exhausted: exhausted,
  login: login,
  commands: commands,
  legacy: legacy,
);
Json runFixture({
  String id = 'run-1',
  String state = 'completed',
  String preview = 'Кратко объясни назначение проекта по README',
  String provider = 'codex',
  String? model = 'gpt-6.1-sol',
  String? errorCode,
  int minutesAgo = 20,
  bool full = false,
}) => _data.runFixture(
  id: id,
  state: state,
  preview: preview,
  provider: provider,
  model: model,
  errorCode: errorCode,
  minutesAgo: minutesAgo,
  full: full,
);
List<Json> historyFixture() => _data.historyFixture();
