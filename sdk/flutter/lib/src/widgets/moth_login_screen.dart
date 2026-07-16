import 'package:flutter/material.dart';

import '../client.dart';
import '../exceptions.dart';
import '../nonce.dart';
import '../project_config.dart';
import 'friendly_errors.dart';
import 'moth_scope.dart';
import 'oauth_adapter.dart';

/// Which part of the login flow is on screen.
enum _Mode { signIn, signUp, resetRequest, resetSent }

/// Batteries-included Material sign-in / sign-up / forgot-password flow.
///
/// [MothApp] shows it by default while signed out; it also works standalone
/// (pass [client], or place it under a [MothScope]). On first build it
/// fetches the project's public config and adapts: the sign-up toggle only
/// appears when public sign-up is open, password validation uses the
/// project's minimum length, and Google/Apple buttons appear per the
/// enabled providers (wired through a [MothOAuthAdapter]).
///
/// Styling is plain Material from the ambient [Theme]; the design-system
/// milestone replaces it with project-defined tokens, so avoid relying on
/// exact visuals.
class MothLoginScreen extends StatefulWidget {
  const MothLoginScreen({super.key, this.client, this.adapter, this.title});

  /// Client override for standalone use; defaults to the enclosing
  /// [MothScope]'s client.
  final MothClient? client;

  /// Adapter override for the provider buttons; defaults to the adapter
  /// passed to [MothApp].
  final MothOAuthAdapter? adapter;

  /// Headline above the form. Defaults to `'Welcome'`.
  final String? title;

  // Stable keys so app (and SDK) widget tests can target the flow.
  static const emailFieldKey = Key('moth-login-email');
  static const passwordFieldKey = Key('moth-login-password');
  static const submitButtonKey = Key('moth-login-submit');
  static const forgotPasswordKey = Key('moth-login-forgot');
  static const toggleModeKey = Key('moth-login-toggle');
  static const googleButtonKey = Key('moth-login-google');
  static const appleButtonKey = Key('moth-login-apple');
  static const errorBannerKey = Key('moth-login-error');
  static const infoBannerKey = Key('moth-login-info');
  static const sendResetKey = Key('moth-login-send-reset');
  static const backToSignInKey = Key('moth-login-back');
  static const retryConfigKey = Key('moth-login-retry-config');

  @override
  State<MothLoginScreen> createState() => _MothLoginScreenState();
}

class _MothLoginScreenState extends State<MothLoginScreen> {
  static final _emailPattern = RegExp(r'^[^\s@]+@[^\s@]+\.[^\s@]+$');

  final _formKey = GlobalKey<FormState>();
  final _email = TextEditingController();
  final _password = TextEditingController();

  MothClient? _client;
  MothProjectConfig? _config;
  bool _configFailed = false;
  _Mode _mode = _Mode.signIn;
  bool _busy = false;
  String? _error;
  String? _info;

  MothClient get client => _client!;

  MothOAuthAdapter? get _adapter =>
      widget.adapter ?? MothScope.maybeOf(context)?.oauthAdapter;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_client == null) {
      _client = widget.client ?? MothScope.maybeOf(context)?.client;
      if (_client == null) {
        throw FlutterError(
          'MothLoginScreen has no MothClient.\n'
          'Place it under MothApp, or pass the client parameter.',
        );
      }
      _fetchConfig();
    }
  }

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _fetchConfig() async {
    try {
      final config = await client.getProjectConfig();
      if (mounted) setState(() => _config = config);
    } on MothException {
      if (mounted) setState(() => _configFailed = true);
    }
  }

  void _retryConfig() {
    setState(() => _configFailed = false);
    _fetchConfig();
  }

  void _switchMode(_Mode mode) => setState(() {
    _mode = mode;
    _error = null;
    _info = null;
  });

  // ---------------------------------------------------------------- actions

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) return;
    final signUp = _mode == _Mode.signUp;
    setState(() {
      _busy = true;
      _error = null;
      _info = null;
    });
    try {
      if (signUp) {
        final result = await client.signUp(
          email: _email.text.trim(),
          password: _password.text,
        );
        if (!result.signedIn && mounted) {
          // Verification (or approval) required before the first sign-in.
          setState(() {
            _mode = _Mode.signIn;
            _info =
                'Account created — check your inbox to verify your email '
                'address, then sign in.';
          });
          _password.clear();
        }
      } else {
        await client.signIn(
          email: _email.text.trim(),
          password: _password.text,
        );
        // On success MothApp swaps this screen out; nothing more to do.
      }
    } on MothException catch (err) {
      if (mounted) setState(() => _error = friendlyMothErrorMessage(err));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _sendReset() async {
    if (!(_formKey.currentState?.validate() ?? false)) return;
    setState(() {
      _busy = true;
      _error = null;
      _info = null;
    });
    try {
      await client.requestPasswordReset(_email.text.trim());
      if (mounted) setState(() => _mode = _Mode.resetSent);
    } on MothException catch (err) {
      if (mounted) setState(() => _error = friendlyMothErrorMessage(err));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _signInWithProvider(MothOAuthProvider provider) async {
    final name = switch (provider) {
      MothOAuthProvider.google => 'Google',
      MothOAuthProvider.apple => 'Apple',
    };
    final adapter = _adapter;
    if (adapter == null) {
      setState(() {
        _error =
            'Sign-in with $name is not wired up in this app: pass a '
            'MothOAuthAdapter to MothApp or MothLoginScreen.';
        _info = null;
      });
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
      _info = null;
    });
    try {
      switch (provider) {
        case MothOAuthProvider.google:
          final credential = await adapter.getGoogleIdToken(_config!.google);
          if (credential == null) return; // cancelled
          await client.signInWithOAuth(
            provider: provider,
            idToken: credential.idToken,
          );
        case MothOAuthProvider.apple:
          final nonce = MothNonce.generate();
          final credential = await adapter.getAppleCredential(
            hashedNonce: nonce.hashed,
          );
          if (credential == null) return; // cancelled
          await client.signInWithOAuth(
            provider: provider,
            idToken: credential.idToken,
            rawNonce: nonce.raw,
            authorizationCode: credential.authorizationCode,
            givenName: credential.givenName,
            familyName: credential.familyName,
          );
      }
    } on MothException catch (err) {
      if (mounted) setState(() => _error = friendlyMothErrorMessage(err));
    } on Exception catch (err) {
      // Adapter/platform failure (missing URL scheme, entitlement, ...).
      if (mounted) setState(() => _error = 'Sign-in with $name failed: $err');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  // ------------------------------------------------------------- validators

  String? _validateEmail(String? value) {
    final email = value?.trim() ?? '';
    if (email.isEmpty) return 'Enter your email address';
    if (!_emailPattern.hasMatch(email)) return 'Enter a valid email address';
    return null;
  }

  String? _validatePassword(String? value, {required int minLength}) {
    final password = value ?? '';
    if (password.isEmpty) return 'Enter your password';
    if (_mode == _Mode.signUp && password.length < minLength) {
      return 'Use at least $minLength characters';
    }
    return null;
  }

  // ------------------------------------------------------------------ build

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final Widget content;
    if (_configFailed) {
      content = _buildConfigError(theme);
    } else if (_config == null) {
      content = const Padding(
        padding: EdgeInsets.all(32),
        child: Center(child: CircularProgressIndicator()),
      );
    } else {
      content = _buildFlow(theme, _config!);
    }
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 400),
              child: content,
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildFlow(ThemeData theme, MothProjectConfig config) {
    final subtitle = switch (_mode) {
      _Mode.signIn => 'Sign in to continue',
      _Mode.signUp => 'Create your account',
      _Mode.resetRequest => 'Reset your password',
      _Mode.resetSent => null,
    };
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          widget.title ?? 'Welcome',
          textAlign: TextAlign.center,
          style: theme.textTheme.headlineMedium,
        ),
        if (subtitle != null) ...[
          const SizedBox(height: 8),
          Text(
            subtitle,
            textAlign: TextAlign.center,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
        const SizedBox(height: 24),
        if (_error != null)
          _banner(
            key: MothLoginScreen.errorBannerKey,
            text: _error!,
            background: theme.colorScheme.errorContainer,
            foreground: theme.colorScheme.onErrorContainer,
          ),
        if (_info != null)
          _banner(
            key: MothLoginScreen.infoBannerKey,
            text: _info!,
            background: theme.colorScheme.secondaryContainer,
            foreground: theme.colorScheme.onSecondaryContainer,
          ),
        ...switch (_mode) {
          _Mode.signIn ||
          _Mode.signUp => _buildEmailPasswordForm(theme, config),
          _Mode.resetRequest => _buildResetRequest(),
          _Mode.resetSent => _buildResetSent(theme),
        },
      ],
    );
  }

  List<Widget> _buildEmailPasswordForm(
    ThemeData theme,
    MothProjectConfig config,
  ) {
    final signUp = _mode == _Mode.signUp;
    return [
      Form(
        key: _formKey,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            TextFormField(
              key: MothLoginScreen.emailFieldKey,
              controller: _email,
              enabled: !_busy,
              keyboardType: TextInputType.emailAddress,
              autofillHints: const [AutofillHints.email],
              textInputAction: TextInputAction.next,
              decoration: const InputDecoration(labelText: 'Email'),
              validator: _validateEmail,
            ),
            const SizedBox(height: 12),
            TextFormField(
              key: MothLoginScreen.passwordFieldKey,
              controller: _password,
              enabled: !_busy,
              obscureText: true,
              autofillHints: [
                signUp ? AutofillHints.newPassword : AutofillHints.password,
              ],
              textInputAction: TextInputAction.done,
              onFieldSubmitted: (_) => _submit(),
              decoration: const InputDecoration(labelText: 'Password'),
              validator: (value) =>
                  _validatePassword(value, minLength: config.passwordMinLength),
            ),
            const SizedBox(height: 20),
            FilledButton(
              key: MothLoginScreen.submitButtonKey,
              onPressed: _busy ? null : _submit,
              child: _busy
                  ? const SizedBox.square(
                      dimension: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(signUp ? 'Create account' : 'Sign in'),
            ),
          ],
        ),
      ),
      if (!signUp)
        Align(
          alignment: Alignment.centerRight,
          child: TextButton(
            key: MothLoginScreen.forgotPasswordKey,
            onPressed: _busy ? null : () => _switchMode(_Mode.resetRequest),
            child: const Text('Forgot password?'),
          ),
        ),
      if (config.signUpOpen)
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Flexible(
              child: Text(
                signUp ? 'Already have an account?' : 'No account yet?',
                style: theme.textTheme.bodyMedium,
              ),
            ),
            TextButton(
              key: MothLoginScreen.toggleModeKey,
              onPressed: _busy
                  ? null
                  : () => _switchMode(signUp ? _Mode.signIn : _Mode.signUp),
              child: Text(signUp ? 'Sign in' : 'Sign up'),
            ),
          ],
        ),
      ..._buildProviderButtons(theme, config),
    ];
  }

  List<Widget> _buildProviderButtons(
    ThemeData theme,
    MothProjectConfig config,
  ) {
    final buttons = [
      if (config.google.enabled)
        OutlinedButton.icon(
          key: MothLoginScreen.googleButtonKey,
          onPressed: _busy
              ? null
              : () => _signInWithProvider(MothOAuthProvider.google),
          icon: const Icon(Icons.g_mobiledata),
          label: const Text('Continue with Google'),
        ),
      if (config.apple.enabled)
        OutlinedButton.icon(
          key: MothLoginScreen.appleButtonKey,
          onPressed: _busy
              ? null
              : () => _signInWithProvider(MothOAuthProvider.apple),
          icon: const Icon(Icons.apple),
          label: const Text('Continue with Apple'),
        ),
    ];
    if (buttons.isEmpty) return const [];
    return [
      const SizedBox(height: 16),
      Row(
        children: [
          const Expanded(child: Divider()),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Text('or', style: theme.textTheme.bodySmall),
          ),
          const Expanded(child: Divider()),
        ],
      ),
      const SizedBox(height: 16),
      for (final (index, button) in buttons.indexed) ...[
        if (index > 0) const SizedBox(height: 12),
        button,
      ],
    ];
  }

  List<Widget> _buildResetRequest() {
    return [
      Form(
        key: _formKey,
        child: TextFormField(
          key: MothLoginScreen.emailFieldKey,
          controller: _email,
          enabled: !_busy,
          keyboardType: TextInputType.emailAddress,
          autofillHints: const [AutofillHints.email],
          textInputAction: TextInputAction.done,
          onFieldSubmitted: (_) => _sendReset(),
          decoration: const InputDecoration(labelText: 'Email'),
          validator: _validateEmail,
        ),
      ),
      const SizedBox(height: 20),
      FilledButton(
        key: MothLoginScreen.sendResetKey,
        onPressed: _busy ? null : _sendReset,
        child: _busy
            ? const SizedBox.square(
                dimension: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : const Text('Send reset link'),
      ),
      TextButton(
        key: MothLoginScreen.backToSignInKey,
        onPressed: _busy ? null : () => _switchMode(_Mode.signIn),
        child: const Text('Back to sign in'),
      ),
    ];
  }

  List<Widget> _buildResetSent(ThemeData theme) {
    return [
      Icon(
        Icons.mark_email_read_outlined,
        size: 48,
        color: theme.colorScheme.primary,
      ),
      const SizedBox(height: 16),
      Text(
        'Check your email',
        textAlign: TextAlign.center,
        style: theme.textTheme.titleLarge,
      ),
      const SizedBox(height: 8),
      Text(
        'If an account exists for ${_email.text.trim()}, a password-reset '
        'link is on its way.',
        textAlign: TextAlign.center,
        style: theme.textTheme.bodyMedium,
      ),
      const SizedBox(height: 24),
      TextButton(
        key: MothLoginScreen.backToSignInKey,
        onPressed: () => _switchMode(_Mode.signIn),
        child: const Text('Back to sign in'),
      ),
    ];
  }

  Widget _buildConfigError(ThemeData theme) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          Icons.cloud_off,
          size: 48,
          color: theme.colorScheme.onSurfaceVariant,
        ),
        const SizedBox(height: 16),
        Text(
          'Cannot reach the server',
          textAlign: TextAlign.center,
          style: theme.textTheme.titleLarge,
        ),
        const SizedBox(height: 8),
        Text(
          'Sign-in options could not be loaded. Check your connection and '
          'try again.',
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium,
        ),
        const SizedBox(height: 24),
        FilledButton(
          key: MothLoginScreen.retryConfigKey,
          onPressed: _retryConfig,
          child: const Text('Try again'),
        ),
      ],
    );
  }

  Widget _banner({
    required Key key,
    required String text,
    required Color background,
    required Color foreground,
  }) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Material(
        key: key,
        color: background,
        borderRadius: BorderRadius.circular(8),
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Text(text, style: TextStyle(color: foreground)),
        ),
      ),
    );
  }
}
