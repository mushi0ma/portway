import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'core/api.dart';
import 'core/session.dart';
import 'core/storage.dart';
import 'core/theme.dart';
import 'dev/fixture_server.dart';
import 'features/auth_screen.dart';
import 'features/brand.dart';
import 'features/home_screen.dart';
import 'l10n/app_localizations.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  if (fixtureMode) {
    // Explicit demo build only: in-memory storage + in-process fixtures.
    final vault = MemoryVault(), server = FixtureServer();
    runApp(
      PortwayApp(
        session: SessionController(
          vault,
          apiFactory: (uri) => ApiClient(uri, client: server.client()),
        )..restore(),
        preferences: ThemePreference(vault)..load(),
        demo: true,
      ),
    );
    return;
  }
  const vault = SecureVault();
  runApp(
    PortwayApp(
      session: SessionController(vault)..restore(),
      preferences: ThemePreference(vault)..load(),
    ),
  );
}

/// Owner's theme choice (System/Light/Dark), persisted on this device.
class ThemePreference extends ChangeNotifier {
  ThemePreference(this.vault);
  final Vault vault;
  ThemeMode mode = ThemeMode.system;
  Future<void> load() async {
    try {
      mode = switch (await vault.read('themeMode')) {
        'light' => ThemeMode.light,
        'dark' => ThemeMode.dark,
        _ => ThemeMode.system,
      };
      notifyListeners();
    } catch (_) {}
  }

  Future<void> set(ThemeMode value) async {
    mode = value;
    notifyListeners();
    try {
      await vault.write('themeMode', value.name);
    } catch (_) {}
  }
}

class PortwayApp extends StatefulWidget {
  const PortwayApp({
    super.key,
    required this.session,
    required this.preferences,
    this.minSplash = const Duration(milliseconds: 900),
    this.demo = false,
  });

  /// Fixture build: a permanent ribbon marks every screen as demo data.
  final bool demo;
  final SessionController session;
  final ThemePreference preferences;

  /// The Flutter splash stays at least this long on a cold start so the brand
  /// screen is perceptible after the native launch screen (no multi-second
  /// artificial delay; restoring the session usually takes longer anyway).
  final Duration minSplash;
  @override
  State<PortwayApp> createState() => _PortwayAppState();
}

class _PortwayAppState extends State<PortwayApp> {
  static const confirmation = MethodChannel('app.portway/auth');
  bool _splashElapsed = false;
  Timer? _splashTimer;
  @override
  void initState() {
    super.initState();
    _splashTimer = Timer(widget.minSplash, () {
      if (mounted) setState(() => _splashElapsed = true);
    });
    confirmation.setMethodCallHandler((call) async {
      if (call.method == 'confirmed') widget.session.emailConfirmed();
    });
    confirmation
        .invokeMethod<bool>('initialConfirmation')
        .then((value) {
          if (mounted && value == true) widget.session.emailConfirmed();
        })
        .catchError((Object _) {});
  }

  @override
  void dispose() {
    _splashTimer?.cancel();
    confirmation.setMethodCallHandler(null);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => ListenableBuilder(
    listenable: Listenable.merge([widget.session, widget.preferences]),
    builder: (context, _) {
      final s = widget.session, prefs = widget.preferences;
      final Widget home;
      if (s.phase == SessionPhase.restoring || !_splashElapsed) {
        home = const SplashView(key: ValueKey('splash'));
      } else if (s.phase == SessionPhase.ready) {
        home = HomeScreen(
          key: const ValueKey('home'),
          session: s,
          themeMode: prefs.mode,
          onTheme: prefs.set,
        );
      } else {
        home = AuthScreen(key: const ValueKey('auth'), session: s);
      }
      return MaterialApp(
        // A new session/installation owns a new Navigator: old routes cannot leak.
        key: ValueKey(s.epoch),
        debugShowCheckedModeBanner: false,
        onGenerateTitle: (c) => AppLocalizations.of(c)!.appTitle,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: const Locale('ru'),
        theme: portwayTheme(Brightness.light),
        darkTheme: portwayTheme(Brightness.dark),
        themeMode: prefs.mode,
        builder: widget.demo
            ? (context, child) => Banner(
                message: 'ДЕМО',
                location: BannerLocation.topEnd,
                color: const Color(0xffb3261e),
                child: child!,
              )
            : null,
        home: Builder(
          builder: (context) => AnimatedSwitcher(
            duration: MediaQuery.disableAnimationsOf(context)
                ? Duration.zero
                : const Duration(milliseconds: 280),
            switchInCurve: Curves.easeOut,
            switchOutCurve: Curves.easeIn,
            child: home,
          ),
        ),
      );
    },
  );
}
