import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:portway/core/api.dart';
import 'package:portway/core/session.dart';
import 'package:portway/core/theme.dart';
import 'package:portway/features/accounts_screen.dart';
import 'package:portway/features/auth_screen.dart';
import 'package:portway/features/codex.dart';
import 'package:portway/features/compose_screen.dart';
import 'package:portway/features/home_screen.dart';
import 'package:portway/features/profile_screen.dart';
import 'package:portway/features/run_screen.dart';
import 'package:portway/l10n/app_localizations.dart';
import 'package:portway/main.dart';

import 'fixtures.dart';
import 'support.dart';

Widget harness(
  Widget child, {
  Brightness brightness = Brightness.dark,
  double scale = 1,
}) => MaterialApp(
  debugShowCheckedModeBanner: false,
  locale: const Locale('ru'),
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  theme: portwayTheme(brightness),
  builder: (c, w) => MediaQuery(
    data: MediaQuery.of(c).copyWith(textScaler: TextScaler.linear(scale)),
    child: w!,
  ),
  home: child,
);

/// Lets fixture HTTP futures complete, then settles frames.
Future<void> settle(WidgetTester tester, {int rounds = 3}) async {
  for (var i = 0; i < rounds; i++) {
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 30)),
    );
    await tester.pump(const Duration(milliseconds: 50));
  }
  await tester.pump(const Duration(milliseconds: 400));
}

Future<void> pumpApp(
  WidgetTester tester,
  Widget widget, {
  Size size = const Size(412, 892),
}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  await tester.runAsync(() async {
    await tester.pumpWidget(widget);
    await Future<void>.delayed(const Duration(milliseconds: 60));
  });
  await settle(tester);
}

Future<void> finish(WidgetTester tester) async {
  expect(tester.takeException(), null);
  await tester.pumpWidget(const SizedBox());
}

FakeServer appServer({
  Json Function()? status,
  List<Json>? runs,
  Map<String, Object Function(http.Request)> extra = const {},
}) => FakeServer({
  'GET /v1/codex/status': (_) => status?.call() ?? statusFixture(),
  'GET /v1/codex/runs': (_) => {'items': runs ?? historyFixture()},
  'POST /v1/codex/usage/refresh': (_) =>
      ok({'command_id': 'cmd-usage'}, status: 202),
  'GET /v1/me': (_) => {
    'id': 'owner',
    'name': 'Алексей',
    'email': 'owner@example.test',
  },
  'GET /v1/devices': (_) => {
    'items': [
      {
        'id': 'fixture-device',
        'label': 'Portway Android',
        'last_seen_at': iso(fixtureNow),
        'revoked_at': null,
      },
      {
        'id': 'tablet',
        'label': 'Galaxy Tab',
        'last_seen_at': iso(fixtureNow.subtract(const Duration(days: 1))),
        'revoked_at': null,
      },
    ],
  },
  ...extra,
});

Widget home(
  SessionController s, {
  Brightness brightness = Brightness.dark,
  double scale = 1,
  ValueChanged<ThemeMode>? onTheme,
}) => harness(
  HomeScreen(
    session: s,
    themeMode: ThemeMode.dark,
    onTheme: onTheme ?? (_) {},
    now: () => fixtureNow,
  ),
  brightness: brightness,
  scale: scale,
);

/// Scrolls the visible tab's list until [target] is built and visible.
Future<void> scrollTo(WidgetTester tester, Finder target, Type within) =>
    tester.scrollUntilVisible(
      target,
      300,
      scrollable: find
          .descendant(
            of: find.byType(within),
            matching: find.byType(Scrollable),
          )
          .first,
    );

Future<void> golden(WidgetTester tester, String name) => expectLater(
  find.byType(MaterialApp),
  matchesGoldenFile('goldens/$name.png'),
);

void main() {
  setUpAll(() async {
    final icons = FontLoader('MaterialIcons')
      ..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'));
    await icons.load();
    // Exact font for goldens (OFL, test/fonts).
    final font = File('test/fonts/Roboto-Regular.ttf');
    if (font.existsSync()) {
      final bytes = ByteData.sublistView(font.readAsBytesSync());
      await (FontLoader('Roboto')..addFont(Future.value(bytes))).load();
      // Goldens have no system monospace font; reuse Roboto so code is legible.
      await (FontLoader('monospace')..addFont(Future.value(bytes))).load();
    }
  });

  testWidgets('splash shows brand, then the connect screen', (tester) async {
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final vault = MemoryVault();
    final session = SessionController(vault);
    // Pumped in the fake-async zone so the minimum splash timer is virtual.
    await tester.pumpWidget(
      PortwayApp(
        session: session,
        preferences: ThemePreference(vault),
        minSplash: const Duration(milliseconds: 600),
      ),
    );
    await tester.runAsync(session.restore);
    await tester.pump();
    expect(find.text('Portway'), findsOneWidget);
    expect(
      find.text('ИИ-агенты вашего компьютера — с телефона'),
      findsOneWidget,
    );
    await tester.pump(const Duration(milliseconds: 700));
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('Подключение к компьютеру'), findsOneWidget);
    // Validation: empty and non-HTTPS addresses get explicit messages.
    await tester.enterText(find.byType(TextFormField), '');
    await tester.tap(find.text('Подключить'));
    await tester.pump();
    expect(find.text('Введите адрес сервера'), findsOneWidget);
    await tester.enterText(find.byType(TextFormField), 'http://localhost:8080');
    await tester.pump();
    expect(
      find.text('Нужен HTTPS-адрес без пути, параметров и localhost.'),
      findsOneWidget,
    );
    await finish(tester);
  });

  for (final b in Brightness.values) {
    testWidgets('registration ${b.name}: explicit validation messages', (
      tester,
    ) async {
      final s = authenticated((_) async => ok({}))..phase = SessionPhase.login;
      s.installation!['setup_required'] = true;
      s.configFresh = true;
      await pumpApp(tester, harness(AuthScreen(session: s), brightness: b));
      expect(find.text('Регистрация владельца'), findsOneWidget);
      await tester.ensureVisible(
        find.widgetWithText(FilledButton, 'Зарегистрироваться'),
      );
      await tester.tap(find.widgetWithText(FilledButton, 'Зарегистрироваться'));
      await tester.pumpAndSettle();
      for (final m in [
        'Введите имя',
        'Введите email',
        'Введите пароль',
        'Введите пароль ещё раз',
        'Введите код настройки',
      ]) {
        expect(find.text(m), findsOneWidget, reason: m);
      }
      final fields = find.byType(TextFormField);
      await tester.enterText(fields.at(1), 'not-an-email');
      await tester.enterText(fields.at(2), 'short');
      await tester.enterText(fields.at(3), 'other');
      await tester.pumpAndSettle();
      expect(
        find.text('Некорректный email. Пример: name@example.com'),
        findsOneWidget,
      );
      expect(
        find.text('Пароль должен содержать не менее 8 символов'),
        findsOneWidget,
      );
      expect(find.text('Пароли не совпадают'), findsOneWidget);
      await tester.enterText(fields.at(2), 'longpassword');
      await tester.pumpAndSettle();
      expect(
        find.text('Добавьте хотя бы одну букву и одну цифру'),
        findsOneWidget,
      );
      await tester.drag(find.byType(Scrollable).first, const Offset(0, 600));
      await tester.pumpAndSettle();
      await golden(tester, 'register-errors-${b.name}');
      await finish(tester);
    });

    testWidgets('login ${b.name}: failed login shows a clear message', (
      tester,
    ) async {
      final s = authenticated((_) async => problem('INVALID_CREDENTIALS', 401))
        ..phase = SessionPhase.login;
      s.credentials = null;
      await pumpApp(tester, harness(AuthScreen(session: s), brightness: b));
      await tester.tap(find.widgetWithText(FilledButton, 'Войти'));
      await tester.pump();
      expect(find.text('Введите email'), findsOneWidget);
      expect(find.text('Введите пароль'), findsOneWidget);
      await tester.enterText(
        find.byType(TextFormField).at(0),
        'owner@example.test',
      );
      await tester.enterText(
        find.byType(TextFormField).at(1),
        'wrong-password',
      );
      await tester.tap(find.widgetWithText(FilledButton, 'Войти'));
      await settle(tester);
      expect(s.phase, SessionPhase.login);
      expect(
        find.text(
          'Неверный email или пароль. Проверьте данные и попробуйте снова.',
        ),
        findsOneWidget,
      );
      await golden(tester, 'login-error-${b.name}');
      // Link to registration.
      await tester.tap(find.widgetWithText(TextButton, 'Зарегистрироваться'));
      await tester.pumpAndSettle();
      expect(find.text('Регистрация владельца'), findsOneWidget);
      await finish(tester);
    });
  }

  testWidgets('auth forms stay usable at 200% text with keyboard', (
    tester,
  ) async {
    tester.view.viewInsets = const FakeViewPadding(bottom: 260);
    addTearDown(tester.view.resetViewInsets);
    final s = authenticated((_) async => ok({}))..phase = SessionPhase.login;
    await pumpApp(
      tester,
      harness(AuthScreen(session: s), scale: 2),
      size: const Size(360, 640),
    );
    await tester.ensureVisible(find.widgetWithText(FilledButton, 'Войти'));
    await tester.pump();
    expect(
      tester.getRect(find.widgetWithText(FilledButton, 'Войти')).height,
      greaterThanOrEqualTo(48),
    );
    await tester.ensureVisible(
      find.widgetWithText(TextButton, 'Зарегистрироваться'),
    );
    await tester.tap(find.widgetWithText(TextButton, 'Зарегистрироваться'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(
      find.widgetWithText(FilledButton, 'Зарегистрироваться'),
    );
    await tester.tap(find.widgetWithText(FilledButton, 'Зарегистрироваться'));
    await tester.pumpAndSettle();
    expect(find.text('Введите имя'), findsOneWidget);
    await finish(tester);
  });

  testWidgets('successful login opens Home', (tester) async {
    final vault = MemoryVault();
    final server = appServer(
      extra: {
        'POST /v1/auth/login': (_) => {
          'session': {'access_token': 'a', 'refresh_token': 'r'},
          'device_proof': 'p',
          'device_id': 'fixture-device',
          'owner': {
            'id': 'o',
            'name': 'Алексей',
            'email': 'owner@example.test',
          },
        },
      },
    );
    final s = authenticated(server.call, vault: vault)
      ..phase = SessionPhase.login;
    s.credentials = null;
    await pumpApp(
      tester,
      PortwayApp(
        session: s,
        preferences: ThemePreference(vault),
        minSplash: Duration.zero,
      ),
    );
    await tester.enterText(
      find.byType(TextFormField).at(0),
      'owner@example.test',
    );
    await tester.enterText(find.byType(TextFormField).at(1), 'portway123');
    await tester.tap(find.widgetWithText(FilledButton, 'Войти'));
    await settle(tester, rounds: 6);
    expect(s.phase, SessionPhase.ready);
    expect(find.byType(NavigationBar), findsOneWidget);
    expect(find.text('Главная'), findsWidgets);
    await finish(tester);
  });

  for (final b in Brightness.values) {
    testWidgets('home ${b.name}: usage meters and explicit exhausted state', (
      tester,
    ) async {
      final s = authenticated(appServer().call);
      await pumpApp(tester, home(s, brightness: b));
      expect(find.text('На связи'), findsOneWidget);
      expect(find.text('Лимит Codex исчерпан'), findsOneWidget);
      expect(find.textContaining('через 3\u00a0ч 40\u00a0мин'), findsWidgets);
      expect(find.text('100 %'), findsOneWidget);
      expect(find.text('Перейти на Claude Code'), findsOneWidget);
      await golden(tester, 'home-${b.name}');
      await tester.tap(find.text('Перейти на Claude Code'));
      await tester.pumpAndSettle();
      expect(find.text('Лимит Codex исчерпан'), findsNothing);
      await finish(tester);
    });
  }

  testWidgets('old API degrades to "no data yet" states', (tester) async {
    final s = authenticated(
      appServer(
        status: () => statusFixture(legacy: true),
        runs: [
          {'id': 'old', 'state': 'completed', 'created_at': iso(fixtureNow)},
        ],
      ).call,
    );
    await pumpApp(tester, home(s));
    expect(
      find.text(
        'Компьютер передаст данные об аккаунтах и лимитах после обновления сервера и моста.',
      ),
      findsOneWidget,
    );
    expect(find.text('Задача от 10:26'), findsOneWidget);
    await tester.tap(find.text('Аккаунты ИИ'));
    await settle(tester);
    expect(find.text('Аккаунтов пока нет'), findsOneWidget);
    await finish(tester);
  });

  testWidgets('navigation reaches every section; logout returns to auth', (
    tester,
  ) async {
    final s = authenticated(
      appServer(
        extra: {
          'POST /v1/auth/logout': (_) => {'device_revoked': true},
        },
      ).call,
    );
    await pumpApp(tester, home(s));
    await tester.tap(find.text('Задачи').last);
    await settle(tester);
    expect(find.text('Поиск по задачам'), findsOneWidget);
    await tester.tap(find.text('Аккаунты ИИ').last);
    await settle(tester);
    expect(find.text('Подключить аккаунт'), findsWidgets);
    await tester.tap(find.text('Профиль').last);
    await settle(tester);
    expect(find.text('owner@example.test'), findsOneWidget);
    expect(find.text('Это устройство'), findsOneWidget);
    // System back from a tab returns to Home.
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(find.text('Модели ИИ'), findsOneWidget);
    await tester.tap(find.text('Профиль').last);
    await settle(tester);
    await scrollTo(
      tester,
      find.widgetWithText(OutlinedButton, 'Выйти'),
      ProfileTab,
    );
    await tester.tap(find.widgetWithText(OutlinedButton, 'Выйти'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilledButton, 'Выйти'));
    await settle(tester);
    expect(s.phase, SessionPhase.login);
    expect(s.credentials, null);
    await finish(tester);
  });

  for (final b in Brightness.values) {
    testWidgets('profile ${b.name} golden and theme switch', (tester) async {
      ThemeMode? chosen;
      final s = authenticated(appServer().call);
      await pumpApp(tester, home(s, brightness: b, onTheme: (v) => chosen = v));
      await tester.tap(find.text('Профиль').last);
      await settle(tester);
      await golden(tester, 'profile-${b.name}');
      await scrollTo(tester, find.text('Светлая'), ProfileTab);
      await tester.tap(find.text('Светлая'));
      await tester.pump();
      expect(chosen, ThemeMode.light);
      await finish(tester);
    });
  }

  for (final b in Brightness.values) {
    testWidgets('compose ${b.name}: provider, model, effort and send body', (
      tester,
    ) async {
      final server = appServer(
        extra: {
          'POST /v1/codex/runs': (_) =>
              ok({'id': 'new-run', 'state': 'queued'}, status: 202),
          'GET /v1/codex/runs/new-run': (_) =>
              runFixture(id: 'new-run', state: 'queued', full: true),
          'GET /v1/codex/runs/new-run/events': (_) => {
            'items': [],
            'next_after': 0,
          },
        },
      );
      final s = authenticated(server.call);
      await pumpApp(tester, home(s, brightness: b));
      await tester.tap(find.text('Новая задача'));
      await settle(tester);
      // Codex is exhausted: sending is blocked with an explanation and actions.
      expect(find.text('Лимит Codex исчерпан'), findsOneWidget);
      expect(find.text('Сменить аккаунт'), findsOneWidget);
      await tester.tap(find.text('Перейти на Claude Code').first);
      await tester.pumpAndSettle();
      await golden(tester, 'compose-${b.name}');
      expect(find.text('По умолчанию'), findsWidgets);
      await tester.tap(find.text('По умолчанию').first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Claude Sonnet'));
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(TextField),
        'Объясни структуру проекта',
      );
      await tester.pump();
      await tester.tap(find.byTooltip('Отправить'));
      await settle(tester);
      final body = jsonDecode(
        server.calls('POST', '/v1/codex/runs').single.body,
      );
      expect(body, {
        'prompt': 'Объясни структуру проекта',
        'workspace': 'portway',
        'provider': 'claude',
        'model': 'sonnet',
      });
      expect(find.byType(RunScreen), findsOneWidget);
      await finish(tester);
    });
  }

  testWidgets('compose: offline blocks sending; codex effort chips', (
    tester,
  ) async {
    final s = authenticated(
      appServer(status: () => statusFixture(connected: false, exhausted: false))
          .call,
    );
    await pumpApp(tester, home(s));
    await tester.tap(find.text('Новая задача'));
    await settle(tester);
    expect(find.text('Компьютер не в сети'), findsWidgets);
    expect(find.text('Глубина рассуждений'), findsOneWidget);
    expect(find.text('Низкая'), findsOneWidget);
    await tester.enterText(find.byType(TextField), 'Вопрос');
    await tester.pump();
    final send = tester.widget<IconButton>(
      find.ancestor(
        of: find.byIcon(Icons.arrow_upward_rounded),
        matching: find.byType(IconButton),
      ),
    );
    expect(send.onPressed, null);
    await finish(tester);
  });

  testWidgets('compose: server AI_LIMIT_EXHAUSTED maps to reset time', (
    tester,
  ) async {
    final s = authenticated(
      appServer(
        status: () => statusFixture(exhausted: false),
        extra: {
          'POST /v1/codex/runs': (_) => problem(
            'AI_LIMIT_EXHAUSTED',
            429,
            details: {
              'provider': 'codex',
              'reset_at': iso(fixtureNow.add(const Duration(minutes: 50))),
            },
          ),
        },
      ).call,
    );
    await pumpApp(tester, home(s));
    await tester.tap(find.text('Новая задача'));
    await settle(tester);
    await tester.enterText(find.byType(TextField), 'Вопрос');
    await tester.pump();
    await tester.tap(find.byTooltip('Отправить'));
    await settle(tester);
    expect(find.text('Лимит Codex исчерпан'), findsOneWidget);
    expect(
      find.text('Снова доступен: 11:16 · через 50\u00a0мин'),
      findsOneWidget,
    );
    await finish(tester);
  });

  for (final b in Brightness.values) {
    testWidgets('run ${b.name}: markdown answer, meta and copy', (
      tester,
    ) async {
      final s = authenticated(
        appServer(
          extra: {
            'GET /v1/codex/runs/run-1': (_) => runFixture(full: true),
            'GET /v1/codex/runs/run-1/events': (_) => {
              'items': [
                {'sequence': 1, 'type': 'started'},
                {'sequence': 2, 'type': 'progress', 'text': 'Читаю README.md'},
                {'sequence': 3, 'type': 'completed'},
              ],
              'next_after': 3,
            },
          },
        ).call,
      );
      final repo = CodexRepository(s);
      await pumpApp(
        tester,
        harness(
          RunScreen(repository: repo, id: 'run-1', now: () => fixtureNow),
          brightness: b,
        ),
      );
      expect(find.text('Готово'), findsOneWidget);
      expect(find.textContaining('Назначение'), findsWidgets);
      expect(find.textContaining('flutter build apk --debug'), findsOneWidget);
      expect(find.text('Ответ · Codex'), findsOneWidget);
      await golden(tester, 'run-${b.name}');
      await scrollTo(tester, find.text('ID задачи'), RunScreen);
      expect(find.text('12 840 на входе · 912 на выходе'), findsOneWidget);
      await finish(tester);
    });
  }

  testWidgets('run: limit failure and active stop are explicit', (
    tester,
  ) async {
    final s = authenticated(
      appServer(
        extra: {
          'GET /v1/codex/runs/run-2': (_) => runFixture(
            id: 'run-2',
            state: 'failed',
            errorCode: 'AI_LIMIT_EXHAUSTED',
            full: true,
          ),
          'GET /v1/codex/runs/run-2/events': (_) => {
            'items': [],
            'next_after': 0,
          },
          'GET /v1/codex/runs/run-3': (_) =>
              runFixture(id: 'run-3', state: 'running', full: true),
          'GET /v1/codex/runs/run-3/events': (_) => {
            'items': [],
            'next_after': 0,
          },
        },
      ).call,
    );
    await pumpApp(
      tester,
      harness(RunScreen(repository: CodexRepository(s), id: 'run-2')),
    );
    expect(find.text('Лимит ИИ исчерпан во время выполнения'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    await pumpApp(
      tester,
      harness(RunScreen(repository: CodexRepository(s), id: 'run-3')),
    );
    expect(find.widgetWithText(OutlinedButton, 'Остановить'), findsOneWidget);
    await finish(tester);
  });

  for (final b in Brightness.values) {
    testWidgets('tasks ${b.name}: search and filters', (tester) async {
      final s = authenticated(appServer().call);
      await pumpApp(tester, home(s, brightness: b));
      await tester.tap(find.text('Задачи').last);
      await settle(tester);
      expect(find.text('Всего задач: 3'), findsOneWidget);
      await golden(tester, 'tasks-${b.name}');
      await tester.enterText(find.byType(TextField), 'readme');
      await tester.pump();
      expect(find.text('Найдено 1 из 3'), findsOneWidget);
      expect(
        find.text('Проверь README на устаревшие инструкции'),
        findsOneWidget,
      );
      await tester.tap(find.text('Сбросить').first);
      await tester.pump();
      await tester.tap(find.text('С ошибкой'));
      await tester.pump();
      expect(find.text('Найдено 1 из 3'), findsOneWidget);
      await tester.tap(find.text('Все'));
      await tester.tap(find.widgetWithText(ChoiceChip, 'Claude Code'));
      await tester.pump();
      expect(find.text('Объясни структуру проекта Portway'), findsOneWidget);
      await tester.enterText(find.byType(TextField), 'нет такого');
      await tester.pump();
      expect(find.text('Ничего не найдено'), findsOneWidget);
      await finish(tester);
    });
  }

  for (final b in Brightness.values) {
    testWidgets('accounts ${b.name}: providers, accounts and actions', (
      tester,
    ) async {
      final s = authenticated(appServer().call);
      await pumpApp(tester, home(s, brightness: b));
      await tester.tap(find.text('Аккаунты ИИ').last);
      await settle(tester);
      expect(find.text('work@example.test'), findsOneWidget);
      expect(find.text('Активный'), findsWidgets);
      expect(find.text('CLI 0.159.3'), findsOneWidget);
      await golden(tester, 'accounts-${b.name}');
      await finish(tester);
    });
  }

  testWidgets('accounts: phone login shows device code; failure explains', (
    tester,
  ) async {
    var phase = 0;
    final server = appServer(
      status: () => statusFixture(
        exhausted: false,
        login: phase == 0
            ? null
            : {
                'id': 'login-9',
                'provider': 'codex',
                'method': 'phone',
                'state': phase == 1 ? 'waiting_user' : 'failed',
                'url': 'https://auth.openai.com/codex/device',
                'user_code': 'K7QM-4XPD',
                'error_code': phase == 1 ? null : 'DEVICE_CODE_UNAVAILABLE',
                'expires_at': iso(fixtureNow.add(const Duration(minutes: 15))),
              },
      ),
      extra: {
        'POST /v1/codex/accounts/login': (_) {
          phase = 1;
          return ok({'command_id': 'c1', 'login_id': 'login-9'}, status: 202);
        },
      },
    );
    final s = authenticated(server.call);
    await pumpApp(tester, home(s));
    await tester.tap(find.text('Аккаунты ИИ').last);
    await settle(tester);
    await tester.tap(
      find.widgetWithText(OutlinedButton, 'Подключить аккаунт').first,
    );
    await tester.pumpAndSettle();
    expect(find.text('На компьютере'), findsOneWidget);
    await tester.tap(find.text('На этом телефоне'));
    await settle(tester, rounds: 5);
    expect(find.byType(AccountLoginScreen), findsOneWidget);
    expect(find.text('K7QM-4XPD'), findsOneWidget);
    expect(find.text('Открыть страницу входа'), findsOneWidget);
    expect(find.text('Вход нужно завершить до 10:41'), findsOneWidget);
    expect(
      jsonDecode(server.calls('POST', '/v1/codex/accounts/login').single.body),
      {'provider': 'codex', 'method': 'phone'},
    );
    phase = 2;
    // The 2 s poll timer may live in either the fake or the real zone.
    await tester.pump(const Duration(milliseconds: 2100));
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 2200)),
    );
    await settle(tester, rounds: 5);
    expect(
      find.textContaining('ChatGPT → Settings → Security'),
      findsOneWidget,
    );
    await finish(tester);
  });

  testWidgets('main screens survive 200% text without overflow', (
    tester,
  ) async {
    final s = authenticated(
      appServer(
        extra: {
          'GET /v1/codex/runs/run-1': (_) => runFixture(full: true),
          'GET /v1/codex/runs/run-1/events': (_) => {
            'items': [],
            'next_after': 0,
          },
        },
      ).call,
    );
    await pumpApp(tester, home(s, scale: 2), size: const Size(360, 800));
    expect(tester.takeException(), null);
    for (final tab in ['Задачи', 'Аккаунты ИИ', 'Профиль']) {
      await tester.tap(find.byTooltip(tab));
      await settle(tester);
      expect(tester.takeException(), null, reason: tab);
    }
    await tester.tap(find.byTooltip('Главная'));
    await settle(tester);
    await scrollTo(tester, find.text('Новая задача'), HomeTab);
    await tester.ensureVisible(
      find.widgetWithText(FilledButton, 'Новая задача'),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilledButton, 'Новая задача'));
    await settle(tester);
    expect(tester.takeException(), null);
    expect(find.byType(ComposeScreen), findsOneWidget);
    await tester.binding.handlePopRoute();
    await settle(tester);
    await tester.pumpWidget(const SizedBox());
    await pumpApp(
      tester,
      harness(RunScreen(repository: CodexRepository(s), id: 'run-1'), scale: 2),
      size: const Size(360, 800),
    );
    await tester.drag(find.byType(Scrollable).first, const Offset(0, -1500));
    await settle(tester);
    await finish(tester);
  });

  testWidgets('wide screen uses navigation rail', (tester) async {
    final s = authenticated(appServer().call);
    await pumpApp(tester, home(s, scale: 2), size: const Size(900, 800));
    expect(find.byType(NavigationRail), findsOneWidget);
    expect(find.byType(NavigationBar), findsNothing);
    await finish(tester);
  });
}
