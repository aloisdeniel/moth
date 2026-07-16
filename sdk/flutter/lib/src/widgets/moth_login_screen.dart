import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../client.dart';
import '../exceptions.dart';
import '../nonce.dart';
import '../project_config.dart';
import '../theme.dart';
import 'friendly_errors.dart';
import 'moth_email_form.dart';
import 'moth_logo.dart';
import 'moth_provider_buttons.dart';
import 'moth_scope.dart';
import 'moth_theme_scope.dart';
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
/// Every visual token — colors, font, spacing, corner radius, logo, legal
/// links — comes from the project's [MothTheme] as configured in the moth
/// admin. Resolution order: the [theme] parameter, else the enclosing
/// [MothThemeScope] (inserted by [MothApp], stale-while-revalidate against
/// the server), else the theme delivered with the config fetch, else the
/// default. The composable pieces ([MothEmailForm], [MothProviderButtons],
/// [MothLogo]) are exported for custom login screens.
class MothLoginScreen extends StatefulWidget {
  const MothLoginScreen({
    super.key,
    this.client,
    this.adapter,
    this.title,
    this.theme,
  });

  /// Client override for standalone use; defaults to the enclosing
  /// [MothScope]'s client.
  final MothClient? client;

  /// Adapter override for the provider buttons; defaults to the adapter
  /// passed to [MothApp].
  final MothOAuthAdapter? adapter;

  /// Headline above the form. Defaults to `'Welcome'`.
  final String? title;

  /// Theme override: wins over the server-delivered project theme.
  final MothTheme? theme;

  // Stable keys so app (and SDK) widget tests can target the flow.
  static const emailFieldKey = MothEmailForm.emailFieldKey;
  static const passwordFieldKey = MothEmailForm.passwordFieldKey;
  static const submitButtonKey = MothEmailForm.submitButtonKey;
  static const forgotPasswordKey = Key('moth-login-forgot');
  static const toggleModeKey = Key('moth-login-toggle');
  static const googleButtonKey = MothProviderButtons.googleButtonKey;
  static const appleButtonKey = MothProviderButtons.appleButtonKey;
  static const errorBannerKey = Key('moth-login-error');
  static const infoBannerKey = Key('moth-login-info');
  static const sendResetKey = Key('moth-login-send-reset');
  static const backToSignInKey = Key('moth-login-back');
  static const retryConfigKey = Key('moth-login-retry-config');
  static const logoKey = Key('moth-login-logo');
  static const termsLinkKey = Key('moth-login-terms');
  static const privacyLinkKey = Key('moth-login-privacy');

  @override
  State<MothLoginScreen> createState() => _MothLoginScreenState();
}

class _MothLoginScreenState extends State<MothLoginScreen> {
  static final _emailPattern = RegExp(r'^[^\s@]+@[^\s@]+\.[^\s@]+$');

  /// Width cap of the centered content column.
  static const _maxContentWidth = 400.0;

  final _resetFormKey = GlobalKey<FormState>();
  final _email = TextEditingController();
  final _password = TextEditingController();

  MothClient? _client;
  MothProjectConfig? _config;

  /// Theme delivered with the config fetch — the standalone fallback when
  /// neither [MothLoginScreen.theme] nor a [MothThemeScope] provides one.
  MothTheme? _serverTheme;
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
      if (mounted) {
        setState(() {
          _config = config;
          _serverTheme = config.theme;
        });
      }
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

  Future<void> _submit(String email, String password) async {
    final signUp = _mode == _Mode.signUp;
    setState(() {
      _busy = true;
      _error = null;
      _info = null;
    });
    try {
      if (signUp) {
        final result = await client.signUp(email: email, password: password);
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
        await client.signIn(email: email, password: password);
        // On success MothApp swaps this screen out; nothing more to do.
      }
    } on MothException catch (err) {
      if (mounted) setState(() => _error = friendlyMothErrorMessage(err));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _sendReset() async {
    if (!(_resetFormKey.currentState?.validate() ?? false)) return;
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

  Future<void> _openLink(String url) async {
    final uri = Uri.tryParse(url);
    if (uri == null) return;
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } on Object {
      // No browser/handler available — nothing sensible to do.
    }
  }

  // ------------------------------------------------------------- validators

  String? _validateEmail(String? value) {
    final email = value?.trim() ?? '';
    if (email.isEmpty) return 'Enter your email address';
    if (!_emailPattern.hasMatch(email)) return 'Enter a valid email address';
    return null;
  }

  // ------------------------------------------------------------------ build

  @override
  Widget build(BuildContext context) {
    final moth =
        widget.theme ??
        MothThemeScope.maybeOf(context) ??
        _serverTheme ??
        MothTheme.fallback();
    final brightness =
        MediaQuery.maybePlatformBrightnessOf(context) ?? Brightness.light;
    // The screen carries its own Theme so the project's design system
    // applies wherever it is placed — under MothApp's themed shell, the
    // developer's differently-themed app, or standalone.
    return MothThemeScope(
      theme: moth,
      child: Theme(
        data: moth.toThemeData(brightness),
        child: Builder(builder: (context) => _buildScreen(context, moth)),
      ),
    );
  }

  Widget _buildScreen(BuildContext context, MothTheme moth) {
    final theme = Theme.of(context);
    final Widget content;
    if (_configFailed) {
      content = _buildConfigError(theme, moth);
    } else if (_config == null) {
      content = Padding(
        padding: EdgeInsets.all(moth.space(4)),
        child: const Center(child: CircularProgressIndicator()),
      );
    } else {
      content = _buildFlow(theme, moth, _config!);
    }
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: EdgeInsets.all(moth.space(3)),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: _maxContentWidth),
              child: content,
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildFlow(ThemeData theme, MothTheme moth, MothProjectConfig config) {
    final subtitle = switch (_mode) {
      _Mode.signIn => 'Sign in to continue',
      _Mode.signUp => 'Create your account',
      _Mode.resetRequest => 'Reset your password',
      _Mode.resetSent => null,
    };
    final hasLogo = moth.logoLightUrl != null || moth.logoDarkUrl != null;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (hasLogo) ...[
          const MothLogo(key: MothLoginScreen.logoKey),
          SizedBox(height: moth.space(3)),
        ],
        Text(
          widget.title ?? 'Welcome',
          textAlign: TextAlign.center,
          style: theme.textTheme.headlineMedium,
        ),
        if (subtitle != null) ...[
          SizedBox(height: moth.space(1)),
          Text(
            subtitle,
            textAlign: TextAlign.center,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
        SizedBox(height: moth.space(3)),
        if (_error != null)
          _banner(
            key: MothLoginScreen.errorBannerKey,
            moth: moth,
            text: _error!,
            background: theme.colorScheme.errorContainer,
            foreground: theme.colorScheme.onErrorContainer,
          ),
        if (_info != null)
          _banner(
            key: MothLoginScreen.infoBannerKey,
            moth: moth,
            text: _info!,
            background: theme.colorScheme.secondaryContainer,
            foreground: theme.colorScheme.onSecondaryContainer,
          ),
        ...switch (_mode) {
          _Mode.signIn ||
          _Mode.signUp => _buildEmailPasswordForm(theme, moth, config),
          _Mode.resetRequest => _buildResetRequest(moth),
          _Mode.resetSent => _buildResetSent(theme, moth),
        },
        ..._buildLegalFooter(theme, moth),
      ],
    );
  }

  List<Widget> _buildEmailPasswordForm(
    ThemeData theme,
    MothTheme moth,
    MothProjectConfig config,
  ) {
    final signUp = _mode == _Mode.signUp;
    return [
      MothEmailForm(
        mode: signUp ? MothEmailFormMode.signUp : MothEmailFormMode.signIn,
        onSubmit: _submit,
        busy: _busy,
        passwordMinLength: config.passwordMinLength,
        emailController: _email,
        passwordController: _password,
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
      MothProviderButtons(
        config: config,
        busy: _busy,
        onSelected: _signInWithProvider,
      ),
    ];
  }

  List<Widget> _buildResetRequest(MothTheme moth) {
    return [
      Form(
        key: _resetFormKey,
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
      SizedBox(height: moth.space(2.5)),
      FilledButton(
        key: MothLoginScreen.sendResetKey,
        onPressed: _busy ? null : _sendReset,
        child: _busy
            ? SizedBox.square(
                dimension: moth.space(2.25),
                child: const CircularProgressIndicator(strokeWidth: 2),
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

  List<Widget> _buildResetSent(ThemeData theme, MothTheme moth) {
    return [
      Icon(
        Icons.mark_email_read_outlined,
        size: moth.space(6),
        color: theme.colorScheme.primary,
      ),
      SizedBox(height: moth.space(2)),
      Text(
        'Check your email',
        textAlign: TextAlign.center,
        style: theme.textTheme.titleLarge,
      ),
      SizedBox(height: moth.space(1)),
      Text(
        'If an account exists for ${_email.text.trim()}, a password-reset '
        'link is on its way.',
        textAlign: TextAlign.center,
        style: theme.textTheme.bodyMedium,
      ),
      SizedBox(height: moth.space(3)),
      TextButton(
        key: MothLoginScreen.backToSignInKey,
        onPressed: () => _switchMode(_Mode.signIn),
        child: const Text('Back to sign in'),
      ),
    ];
  }

  List<Widget> _buildLegalFooter(ThemeData theme, MothTheme moth) {
    final links = [
      if (moth.termsUrl != null)
        (MothLoginScreen.termsLinkKey, 'Terms of Service', moth.termsUrl!),
      if (moth.privacyUrl != null)
        (MothLoginScreen.privacyLinkKey, 'Privacy Policy', moth.privacyUrl!),
    ];
    if (links.isEmpty) return const [];
    final linkStyle = TextButton.styleFrom(
      textStyle: theme.textTheme.bodySmall,
      foregroundColor: theme.colorScheme.onSurfaceVariant,
      minimumSize: Size(0, moth.space(4)),
      padding: EdgeInsets.symmetric(horizontal: moth.space(1)),
    );
    return [
      SizedBox(height: moth.space(3)),
      Wrap(
        alignment: WrapAlignment.center,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          for (final (index, (key, label, url)) in links.indexed) ...[
            if (index > 0) Text('·', style: theme.textTheme.bodySmall),
            TextButton(
              key: key,
              style: linkStyle,
              onPressed: () => _openLink(url),
              child: Text(label),
            ),
          ],
        ],
      ),
    ];
  }

  Widget _buildConfigError(ThemeData theme, MothTheme moth) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          Icons.cloud_off,
          size: moth.space(6),
          color: theme.colorScheme.onSurfaceVariant,
        ),
        SizedBox(height: moth.space(2)),
        Text(
          'Cannot reach the server',
          textAlign: TextAlign.center,
          style: theme.textTheme.titleLarge,
        ),
        SizedBox(height: moth.space(1)),
        Text(
          'Sign-in options could not be loaded. Check your connection and '
          'try again.',
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium,
        ),
        SizedBox(height: moth.space(3)),
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
    required MothTheme moth,
    required String text,
    required Color background,
    required Color foreground,
  }) {
    return Padding(
      padding: EdgeInsets.only(bottom: moth.space(2)),
      child: Material(
        key: key,
        color: background,
        borderRadius: BorderRadius.circular(moth.cornerRadius),
        child: Padding(
          padding: EdgeInsets.all(moth.space(1.5)),
          child: Text(text, style: TextStyle(color: foreground)),
        ),
      ),
    );
  }
}
