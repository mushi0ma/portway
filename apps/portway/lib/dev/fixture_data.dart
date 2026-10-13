// Contract v2 fixture data (scratchpad contract.md, agreed 13.10.2026).
// Used by tests and by the explicit `--dart-define=PORTWAY_FIXTURES=true`
// demo build. It describes the agreed server shape, not a recorded live
// response, and is never compiled into a normal build's behaviour.

import '../core/api.dart';

class FixtureData {
  FixtureData(this.now);
  final DateTime now;

  String iso(DateTime local) => local.toUtc().toIso8601String();

  Json codexProvider({bool exhausted = true, bool loggedIn = true}) => {
    'id': 'codex',
    'name': 'Codex',
    'installed': true,
    'cli_version': '0.159.3',
    'active_account_id': 'codex-system',
    'accounts': [
      {
        'id': 'codex-system',
        'label': 'ChatGPT',
        'email': 'owner@example.test',
        'plan': 'plus',
        'logged_in': loggedIn,
        'active': true,
        'system': true,
      },
      {
        'id': 'codex-work',
        'label': 'Рабочий',
        'email': 'work@example.test',
        'plan': 'pro',
        'logged_in': true,
        'active': false,
        'system': false,
      },
    ],
    'models': [
      {
        'id': 'gpt-6.1-sol',
        'name': 'GPT-6.1-Sol',
        'description': 'Основная модель для кода',
        'default': true,
        'efforts': ['low', 'medium', 'high', 'xhigh'],
        'default_effort': 'low',
      },
      {
        'id': 'gpt-6.1-mini',
        'name': 'GPT-6.1-Mini',
        'default': false,
        'efforts': ['low', 'medium'],
      },
    ],
    'usage': {
      'status': exhausted ? 'exhausted' : 'ok',
      'checked_at': iso(now.subtract(const Duration(minutes: 2))),
      'reset_at': exhausted
          ? iso(now.add(const Duration(hours: 3, minutes: 40)))
          : null,
      'windows': [
        {
          'id': 'five_hour',
          'used_percent': exhausted ? 100 : 42,
          'window_minutes': 300,
          'resets_at': iso(now.add(const Duration(hours: 3, minutes: 40))),
        },
        {
          'id': 'weekly',
          'used_percent': 31,
          'window_minutes': 10080,
          'resets_at': iso(now.add(const Duration(days: 5, hours: 2))),
        },
      ],
    },
  };

  Json claudeProvider() => {
    'id': 'claude',
    'name': 'Claude Code',
    'installed': true,
    'cli_version': '2.1.296',
    'active_account_id': 'claude-system',
    'accounts': [
      {
        'id': 'claude-system',
        'label': 'Claude',
        'email': 'owner@example.test',
        'plan': 'claude_pro',
        'logged_in': true,
        'active': true,
        'system': true,
      },
    ],
    'models': [
      {'id': 'default', 'name': 'По умолчанию', 'default': true, 'efforts': []},
      {'id': 'opus', 'name': 'Claude Opus', 'efforts': []},
      {'id': 'sonnet', 'name': 'Claude Sonnet', 'efforts': []},
    ],
    'usage': {
      'status': 'ok',
      'checked_at': iso(now.subtract(const Duration(minutes: 1))),
      'reset_at': null,
      'windows': [
        {
          'id': 'five_hour',
          'used_percent': 4,
          'window_minutes': 300,
          'resets_at': iso(now.add(const Duration(hours: 4, minutes: 10))),
        },
        {
          'id': 'weekly',
          'used_percent': 18.5,
          'window_minutes': 10080,
          'resets_at': iso(now.add(const Duration(days: 3))),
        },
      ],
    },
  };

  Json statusFixture({
    bool connected = true,
    bool exhausted = true,
    Json? login,
    List<Json> commands = const [],
    bool legacy = false,
  }) => legacy
      ? {
          'connected': connected,
          'workspace': 'portway',
          'sandbox': 'read-only',
          'last_seen_at': iso(now.subtract(const Duration(seconds: 5))),
        }
      : {
          'connected': connected,
          'workspace': 'portway',
          'sandbox': 'read-only',
          'last_seen_at': iso(
            now.subtract(Duration(seconds: connected ? 5 : 3600)),
          ),
          'state_updated_at': iso(now.subtract(const Duration(minutes: 1))),
          'providers': [codexProvider(exhausted: exhausted), claudeProvider()],
          'login': login,
          'commands': commands,
        };

  Json runFixture({
    String id = 'run-1',
    String state = 'completed',
    String preview = 'Кратко объясни назначение проекта по README',
    String provider = 'codex',
    String? model = 'gpt-6.1-sol',
    String? errorCode,
    int minutesAgo = 20,
    bool full = false,
  }) => {
    'id': id,
    'state': state,
    'workspace': 'portway',
    'sandbox': 'read-only',
    'provider': provider,
    'model': model,
    'effort': provider == 'codex' ? 'medium' : null,
    'resolved_model': model,
    'account_label': provider == 'codex' ? 'ChatGPT' : 'Claude',
    'tokens': {'input': 12840, 'output': 912},
    'prompt_preview': preview,
    if (full) 'prompt': '$preview. Ничего не изменяй.',
    if (full && state == 'completed')
      'answer':
          '## Назначение\n\nPortway управляет **ИИ-агентами** на компьютере с телефона.\n\n'
          '- Android-клиент на Flutter\n- Go API и мост на ноутбуке\n\n'
          '```bash\nflutter build apk --debug\n```\n\nКоманда `flutter test` запускает проверки.',
    'error_code': errorCode,
    'created_at': iso(now.subtract(Duration(minutes: minutesAgo))),
    'started_at': iso(now.subtract(Duration(minutes: minutesAgo))),
    'finished_at': state == 'running' || state == 'queued'
        ? null
        : iso(now.subtract(Duration(minutes: minutesAgo - 1))),
  };

  List<Json> historyFixture() => [
    runFixture(
      id: 'run-3',
      state: 'running',
      preview: 'Найди основные точки входа приложения',
      minutesAgo: 1,
    ),
    runFixture(
      id: 'run-2',
      state: 'failed',
      preview: 'Проверь README на устаревшие инструкции',
      errorCode: 'AI_LIMIT_EXHAUSTED',
      minutesAgo: 12,
    ),
    runFixture(
      id: 'run-1',
      preview: 'Объясни структуру проекта Portway',
      provider: 'claude',
      model: 'sonnet',
      minutesAgo: 95,
    ),
  ];
}
