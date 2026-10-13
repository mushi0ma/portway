import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../core/api.dart';
import '../core/session.dart';
import '../core/theme.dart';
import 'brand.dart';
import 'common.dart';

enum AuthMode { login, register, confirm }

final _emailPattern = RegExp(r'^[^\s@]+@[^\s@]+\.[^\s@]+$');

/// UI-02…04: installation, owner registration, email confirmation and login.
class AuthScreen extends StatefulWidget {
  const AuthScreen({super.key, required this.session});
  final SessionController session;
  @override
  State<AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends State<AuthScreen> {
  final _form = GlobalKey<FormState>();
  final _url = TextEditingController(
        text: const String.fromEnvironment(
          'PORTWAY_API_URL',
          defaultValue: 'https://portway.tail1443b8.ts.net',
        ),
      ),
      _email = TextEditingController(),
      _password = TextEditingController(),
      _repeat = TextEditingController(),
      _name = TextEditingController(),
      _proof = TextEditingController();
  late AuthMode _mode;
  bool _sending = false, _obscure = true, _submitted = false;
  ApiFailure? _error;
  String? _registrationKey, _sentTo;
  bool _registrationUncertain = false;

  @override
  void initState() {
    super.initState();
    // Registration first only for a freshly checked installation without an
    // owner; a cached flag from an earlier run may be stale.
    final s = widget.session;
    _mode = s.configFresh && s.installation?['setup_required'] == true
        ? AuthMode.register
        : AuthMode.login;
    _password.addListener(_onPassword);
  }

  void _onPassword() {
    if (_mode == AuthMode.register) setState(() {});
  }

  @override
  void dispose() {
    _password.removeListener(_onPassword);
    for (final c in [_url, _email, _password, _repeat, _name, _proof]) {
      c.clear();
      c.dispose();
    }
    super.dispose();
  }

  @override
  void didUpdateWidget(covariant AuthScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.session.confirmationReceived) {
      // portway://auth/confirmed is only a navigation hint: go to login.
      widget.session.confirmationReceived = false;
      _password.clear();
      _repeat.clear();
      _proof.clear();
      _mode = AuthMode.login;
      _registrationUncertain = false;
      _registrationKey = null;
      _error = null;
      _submitted = false;
    }
  }

  void _switch(AuthMode mode) {
    TextInput.finishAutofillContext(shouldSave: false);
    _password.clear();
    _repeat.clear();
    _proof.clear();
    widget.session.failure = null;
    setState(() {
      _mode = mode;
      _error = null;
      _submitted = false;
      _registrationKey = null;
      _registrationUncertain = false;
    });
  }

  Future<void> _submit() async {
    setState(() => _submitted = true);
    if (!(_form.currentState?.validate() ?? false)) return;
    FocusManager.instance.primaryFocus?.unfocus();
    final s = widget.session;
    if (s.phase == SessionPhase.installation) {
      await s.connect(_url.text);
      return;
    }
    if (_mode == AuthMode.login) {
      // The password stays in the field after a failed attempt so a typo can
      // be fixed; it is cleared when the screen is disposed.
      await s.login(_email.text, _password.text);
      return;
    }
    setState(() {
      _sending = true;
      _error = null;
    });
    try {
      await s.api!.request(
        'POST',
        '/v1/auth/register',
        body: {
          'name': _name.text.trim(),
          'email': _email.text.trim(),
          'password': _password.text,
        },
        headers: {
          'Idempotency-Key': _registrationKey ??= operationKey(),
          if (_proof.text.trim().isNotEmpty)
            'X-Setup-Proof': _proof.text.trim(),
        },
      );
      _password.clear();
      _repeat.clear();
      _proof.clear();
      _registrationKey = null;
      if (mounted) {
        setState(() {
          _sentTo = _email.text.trim();
          _mode = AuthMode.confirm;
          _registrationUncertain = false;
          _submitted = false;
        });
      }
    } on ApiFailure catch (e) {
      if (mounted) {
        setState(() {
          _error = e;
          _registrationUncertain = e.uncertain;
        });
      }
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  Future<void> _changeInstallation() async {
    final l = context.l;
    if (await confirmAction(
      context,
      l.changeInstallationTitle,
      l.changeConfirmSignedOut,
      l.continueAction,
    )) {
      _switch(AuthMode.login);
      await widget.session.changeInstallation();
    }
  }

  // --- Validators (explicit Russian messages for empty/invalid values) ----

  String? _validateUrl(String? v) {
    if ((v ?? '').trim().isEmpty) return context.l.urlRequired;
    try {
      installationUri(v!);
      return null;
    } on ApiFailure {
      return context.l.invalidUrl;
    }
  }

  String? _validateEmail(String? v) {
    final value = (v ?? '').trim();
    if (value.isEmpty) return context.l.emailRequired;
    return _emailPattern.hasMatch(value) ? null : context.l.invalidEmail;
  }

  String? _validatePassword(String? v) {
    final value = v ?? '';
    if (value.isEmpty) return context.l.passwordRequired;
    if (_mode != AuthMode.register) return null;
    if (value.length < 8) return context.l.shortPassword;
    if (!_hasLetter(value) || !_hasDigit(value)) {
      return context.l.weakPassword;
    }
    return null;
  }

  static bool _hasLetter(String v) => RegExp(r'[A-Za-zА-Яа-яЁё]').hasMatch(v);
  static bool _hasDigit(String v) => RegExp(r'\d').hasMatch(v);

  @override
  Widget build(BuildContext context) {
    final s = widget.session;
    final busy = s.busy || _sending;
    final setup = s.phase == SessionPhase.installation;
    final view = setup
        ? _connect(busy)
        : switch (_mode) {
            AuthMode.login => _login(busy),
            AuthMode.register => _register(busy),
            AuthMode.confirm => _confirm(),
          };
    final back = !setup && _mode != AuthMode.login && !busy;
    return PopScope(
      canPop: !back,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop && back) _switch(AuthMode.login);
      },
      child: Scaffold(
        appBar: back
            ? AppBar(
                leading: IconButton(
                  tooltip: MaterialLocalizations.of(context).backButtonTooltip,
                  icon: const Icon(Icons.arrow_back_rounded),
                  onPressed: () => _switch(AuthMode.login),
                ),
              )
            : null,
        body: SafeArea(
          child: Center(
            child: SingleChildScrollView(
              padding: EdgeInsets.fromLTRB(
                Space.xl,
                back ? Space.sm : Space.xxl,
                Space.xl,
                Space.xl,
              ),
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 440),
                child: Form(
                  key: _form,
                  autovalidateMode: _submitted
                      ? AutovalidateMode.onUserInteraction
                      : AutovalidateMode.disabled,
                  child: AutofillGroup(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: view,
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  List<Widget> _header(String title, String subtitle, {bool brand = true}) => [
    if (brand) ...[
      const Align(alignment: Alignment.centerLeft, child: BrandLockup()),
      const SizedBox(height: Space.xxl),
    ],
    Semantics(
      header: true,
      child: Text(title, style: context.text.headlineSmall),
    ),
    const SizedBox(height: Space.sm),
    Text(
      subtitle,
      style: context.text.bodyLarge?.copyWith(color: context.colors.muted),
    ),
    const SizedBox(height: Space.xl),
  ];

  Widget _primary(String label, bool busy, VoidCallback onPressed) =>
      FilledButton(
        onPressed: busy ? null : onPressed,
        child: busy
            ? Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  SizedBox.square(
                    dimension: 18,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      color: context.colors.disabled,
                      semanticsLabel: context.l.loading,
                    ),
                  ),
                  const SizedBox(width: Space.md),
                  Text(label),
                ],
              )
            : Text(label),
      );

  Widget _link(String prompt, String action, VoidCallback? onPressed) => Wrap(
    alignment: WrapAlignment.center,
    crossAxisAlignment: WrapCrossAlignment.center,
    children: [
      Text(
        prompt,
        style: context.text.bodyMedium?.copyWith(color: context.colors.muted),
      ),
      TextButton(onPressed: onPressed, child: Text(action)),
    ],
  );

  Widget? _sessionNotice() {
    final f = widget.session.failure;
    if (f == null || _error != null) return null;
    final l = context.l;
    return switch (f.code) {
      'SIGNED_OUT' => Notice(
        tone: Tone.accent,
        icon: Icons.logout_rounded,
        title: l.signedOut,
        message: widget.session.logoutPending
            ? l.logoutPending
            : l.signedOutHint,
      ),
      'DEVICE_REVOKED' => Notice(
        tone: Tone.warning,
        title: l.deviceRevokedTitle,
        message: l.deviceRevoked,
      ),
      'TOKEN_EXPIRED' || 'SESSION_EXPIRED' || 'INVALID_TOKEN' => Notice(
        tone: Tone.warning,
        title: l.sessionExpiredTitle,
        message: l.sessionExpired,
      ),
      'INVALID_CREDENTIALS' || 'AUTH_INVALID_CREDENTIALS' || 'AUTH_REQUIRED' =>
        Notice(title: l.credentialsTitle, message: l.credentialsError),
      _ => ErrorNotice(f),
    };
  }

  List<Widget> _connect(bool busy) {
    final l = context.l, s = widget.session;
    return [
      ..._header(l.connectTitle, l.connectHint),
      TextFormField(
        controller: _url,
        enabled: !busy,
        keyboardType: TextInputType.url,
        autocorrect: false,
        textInputAction: TextInputAction.done,
        decoration: InputDecoration(
          labelText: l.apiUrl,
          hintText: l.urlHint,
          helperText: l.urlHelper,
          prefixIcon: const Icon(Icons.link_rounded),
        ),
        validator: _validateUrl,
        onFieldSubmitted: (_) => _submit(),
      ),
      if (s.failure case final f?) ...[
        const SizedBox(height: Space.lg),
        ErrorNotice(f),
      ],
      const SizedBox(height: Space.xl),
      _primary(l.connect, busy, _submit),
      const SizedBox(height: Space.xl),
      Notice(
        tone: Tone.neutral,
        icon: Icons.laptop_mac_rounded,
        title: l.connectWhereTitle,
        message: l.connectWhere,
      ),
    ];
  }

  Widget _emailField(bool enabled) => TextFormField(
    controller: _email,
    enabled: enabled,
    keyboardType: TextInputType.emailAddress,
    autofillHints: const [AutofillHints.email],
    autocorrect: false,
    textInputAction: TextInputAction.next,
    decoration: InputDecoration(
      labelText: context.l.email,
      hintText: 'name@example.com',
      prefixIcon: const Icon(Icons.mail_outline_rounded),
    ),
    validator: _validateEmail,
  );

  Widget _passwordField(
    bool enabled, {
    bool newPassword = false,
  }) => TextFormField(
    controller: _password,
    enabled: enabled,
    obscureText: _obscure,
    enableSuggestions: false,
    autocorrect: false,
    textInputAction: newPassword ? TextInputAction.next : TextInputAction.done,
    autofillHints: [
      newPassword ? AutofillHints.newPassword : AutofillHints.password,
    ],
    decoration: InputDecoration(
      labelText: context.l.password,
      prefixIcon: const Icon(Icons.lock_outline_rounded),
      suffixIcon: IconButton(
        tooltip: _obscure ? context.l.showPassword : context.l.hidePassword,
        onPressed: () => setState(() => _obscure = !_obscure),
        icon: Icon(
          _obscure ? Icons.visibility_outlined : Icons.visibility_off_outlined,
        ),
      ),
    ),
    validator: _validatePassword,
    onFieldSubmitted: newPassword ? null : (_) => _submit(),
  );

  List<Widget> _login(bool busy) {
    final l = context.l, s = widget.session, c = context.colors;
    final notice = _sessionNotice();
    return [
      ..._header(l.loginTitle, l.loginAt(s.name)),
      _emailField(!busy),
      const SizedBox(height: Space.lg),
      _passwordField(!busy),
      if (notice != null) ...[const SizedBox(height: Space.lg), notice],
      const SizedBox(height: Space.xl),
      _primary(l.login, busy, _submit),
      const SizedBox(height: Space.sm),
      _link(
        l.noAccount,
        l.createAccount,
        busy ? null : () => _switch(AuthMode.register),
      ),
      const SizedBox(height: Space.xl),
      const Divider(),
      const SizedBox(height: Space.md),
      Row(
        children: [
          Icon(Icons.dns_outlined, size: 20, color: c.muted),
          const SizedBox(width: Space.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(s.name, style: context.text.titleSmall),
                Text(
                  s.baseUrl,
                  style: context.text.bodySmall,
                  overflow: TextOverflow.ellipsis,
                  maxLines: 2,
                ),
              ],
            ),
          ),
          TextButton(
            onPressed: busy ? null : _changeInstallation,
            child: Text(l.change),
          ),
        ],
      ),
      if (s.logoutPending && s.failure?.code != 'SIGNED_OUT') ...[
        const SizedBox(height: Space.md),
        Text(l.logoutPending, style: context.text.bodySmall),
      ],
    ];
  }

  List<Widget> _register(bool busy) {
    final l = context.l, s = widget.session, c = context.colors;
    final enabled = !busy && !_registrationUncertain;
    final value = _password.text;
    Widget rule(bool ok, String text) => Padding(
      padding: const EdgeInsets.only(top: 6),
      child: Row(
        children: [
          Icon(
            ok ? Icons.check_circle_rounded : Icons.radio_button_unchecked,
            size: 16,
            color: ok ? c.accent : c.muted,
          ),
          const SizedBox(width: Space.sm),
          Expanded(
            child: Text(
              text,
              style: context.text.bodySmall?.copyWith(
                color: ok ? c.text : c.muted,
              ),
            ),
          ),
        ],
      ),
    );
    return [
      ..._header(l.registrationTitle, l.registrationHint(s.name), brand: false),
      TextFormField(
        controller: _name,
        enabled: enabled,
        textCapitalization: TextCapitalization.words,
        textInputAction: TextInputAction.next,
        autofillHints: const [AutofillHints.name],
        decoration: InputDecoration(
          labelText: l.name,
          prefixIcon: const Icon(Icons.person_outline_rounded),
        ),
        validator: (v) => (v ?? '').trim().isEmpty
            ? l.nameRequired
            : (v!.trim().length > 100 ? l.nameTooLong : null),
      ),
      const SizedBox(height: Space.lg),
      _emailField(enabled),
      const SizedBox(height: Space.lg),
      _passwordField(enabled, newPassword: true),
      Padding(
        padding: const EdgeInsets.fromLTRB(Space.xs, Space.xs, 0, 0),
        child: Semantics(
          container: true,
          child: Column(
            children: [
              rule(value.length >= 8, l.ruleLength),
              rule(_hasLetter(value) && _hasDigit(value), l.ruleMix),
            ],
          ),
        ),
      ),
      const SizedBox(height: Space.lg),
      TextFormField(
        controller: _repeat,
        enabled: enabled,
        obscureText: _obscure,
        enableSuggestions: false,
        autocorrect: false,
        textInputAction: TextInputAction.next,
        decoration: InputDecoration(
          labelText: l.repeatPassword,
          prefixIcon: const Icon(Icons.lock_reset_rounded),
        ),
        validator: (v) => (v ?? '').isEmpty
            ? l.repeatRequired
            : v != _password.text
            ? l.passwordMismatch
            : null,
      ),
      const SizedBox(height: Space.lg),
      TextFormField(
        controller: _proof,
        enabled: enabled,
        obscureText: true,
        autocorrect: false,
        enableSuggestions: false,
        textInputAction: TextInputAction.done,
        decoration: InputDecoration(
          labelText: l.setupProof,
          helperText: l.setupProofHelper,
          prefixIcon: const Icon(Icons.key_outlined),
        ),
        validator: (v) =>
            (v ?? '').trim().isEmpty ? l.setupProofRequired : null,
        onFieldSubmitted: (_) => _submit(),
      ),
      if (_error case final e?) ...[
        const SizedBox(height: Space.lg),
        ErrorNotice(e),
      ],
      if (_registrationUncertain) ...[
        const SizedBox(height: Space.md),
        Notice(
          tone: Tone.warning,
          title: l.registrationUnknownTitle,
          message: l.registrationUnknown,
        ),
      ],
      const SizedBox(height: Space.xl),
      _primary(
        _registrationUncertain ? l.retrySame : l.register,
        busy,
        _submit,
      ),
      if (!_registrationUncertain) ...[
        const SizedBox(height: Space.sm),
        _link(
          l.haveAccount,
          l.login,
          busy ? null : () => _switch(AuthMode.login),
        ),
      ],
    ];
  }

  List<Widget> _confirm() {
    final l = context.l, c = context.colors;
    return [
      const SizedBox(height: Space.xl),
      Center(
        child: Container(
          width: 72,
          height: 72,
          decoration: BoxDecoration(
            color: c.accentContainer,
            shape: BoxShape.circle,
          ),
          child: Icon(
            Icons.mark_email_unread_outlined,
            size: 32,
            color: c.accent,
          ),
        ),
      ),
      const SizedBox(height: Space.xl),
      Semantics(
        header: true,
        child: Text(
          l.confirmationTitle,
          textAlign: TextAlign.center,
          style: context.text.headlineSmall,
        ),
      ),
      const SizedBox(height: Space.md),
      Text(
        l.confirmationHint(_sentTo ?? _email.text.trim()),
        textAlign: TextAlign.center,
        style: context.text.bodyLarge?.copyWith(color: c.muted),
      ),
      const SizedBox(height: Space.xl),
      Notice(
        tone: Tone.neutral,
        icon: Icons.info_outline_rounded,
        title: l.confirmationStepsTitle,
        message: l.confirmationSteps,
      ),
      const SizedBox(height: Space.xl),
      FilledButton(
        onPressed: () => _switch(AuthMode.login),
        child: Text(l.returnLogin),
      ),
      const SizedBox(height: Space.sm),
      TextButton(
        onPressed: () => _switch(AuthMode.register),
        child: Text(l.editRegistration),
      ),
    ];
  }
}
