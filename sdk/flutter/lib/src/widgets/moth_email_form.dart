import 'package:flutter/material.dart';

import 'moth_theme_scope.dart';

/// Which action the form submits.
enum MothEmailFormMode { signIn, signUp }

/// The themed email + password form used by [MothLoginScreen], exposed so
/// a custom login screen can reuse it: fields with validation (email
/// syntax, minimum password length on sign-up), autofill hints and a
/// submit button with a busy spinner. Purely presentational — [onSubmit]
/// receives the validated credentials and drives the actual RPC.
///
/// Styling comes from the ambient [Theme] (build one from a [MothTheme]
/// via `toThemeData`, or let [MothLoginScreen]/[MothThemeScope] provide
/// it); spacing follows the enclosing [MothThemeScope]'s unit.
class MothEmailForm extends StatefulWidget {
  const MothEmailForm({
    super.key,
    this.mode = MothEmailFormMode.signIn,
    required this.onSubmit,
    this.busy = false,
    this.passwordMinLength = 8,
    this.emailController,
    this.passwordController,
  });

  final MothEmailFormMode mode;

  /// Called with the trimmed email and the password once validation
  /// passes.
  final void Function(String email, String password) onSubmit;

  /// Disables the fields and shows a spinner in the submit button.
  final bool busy;

  /// Minimum password length enforced in [MothEmailFormMode.signUp] (use
  /// the project config's value).
  final int passwordMinLength;

  /// Optional external controllers, so entered text survives mode
  /// switches in a larger flow. The form creates (and owns) its own when
  /// omitted.
  final TextEditingController? emailController;
  final TextEditingController? passwordController;

  // Stable keys so app (and SDK) widget tests can target the flow.
  static const emailFieldKey = Key('moth-login-email');
  static const passwordFieldKey = Key('moth-login-password');
  static const submitButtonKey = Key('moth-login-submit');

  @override
  State<MothEmailForm> createState() => _MothEmailFormState();
}

class _MothEmailFormState extends State<MothEmailForm> {
  static final _emailPattern = RegExp(r'^[^\s@]+@[^\s@]+\.[^\s@]+$');

  final _formKey = GlobalKey<FormState>();
  TextEditingController? _ownEmail;
  TextEditingController? _ownPassword;

  TextEditingController get _email =>
      widget.emailController ?? (_ownEmail ??= TextEditingController());
  TextEditingController get _password =>
      widget.passwordController ?? (_ownPassword ??= TextEditingController());

  @override
  void dispose() {
    _ownEmail?.dispose();
    _ownPassword?.dispose();
    super.dispose();
  }

  void _submit() {
    if (!(_formKey.currentState?.validate() ?? false)) return;
    widget.onSubmit(_email.text.trim(), _password.text);
  }

  String? _validateEmail(String? value) {
    final email = value?.trim() ?? '';
    if (email.isEmpty) return 'Enter your email address';
    if (!_emailPattern.hasMatch(email)) return 'Enter a valid email address';
    return null;
  }

  String? _validatePassword(String? value) {
    final password = value ?? '';
    if (password.isEmpty) return 'Enter your password';
    if (widget.mode == MothEmailFormMode.signUp &&
        password.length < widget.passwordMinLength) {
      return 'Use at least ${widget.passwordMinLength} characters';
    }
    return null;
  }

  @override
  Widget build(BuildContext context) {
    final moth = MothThemeScope.of(context);
    final signUp = widget.mode == MothEmailFormMode.signUp;
    return Form(
      key: _formKey,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TextFormField(
            key: MothEmailForm.emailFieldKey,
            controller: _email,
            enabled: !widget.busy,
            keyboardType: TextInputType.emailAddress,
            autofillHints: const [AutofillHints.email],
            textInputAction: TextInputAction.next,
            decoration: const InputDecoration(labelText: 'Email'),
            validator: _validateEmail,
          ),
          SizedBox(height: moth.space(1.5)),
          TextFormField(
            key: MothEmailForm.passwordFieldKey,
            controller: _password,
            enabled: !widget.busy,
            obscureText: true,
            autofillHints: [
              signUp ? AutofillHints.newPassword : AutofillHints.password,
            ],
            textInputAction: TextInputAction.done,
            onFieldSubmitted: (_) => _submit(),
            decoration: const InputDecoration(labelText: 'Password'),
            validator: _validatePassword,
          ),
          SizedBox(height: moth.space(2.5)),
          FilledButton(
            key: MothEmailForm.submitButtonKey,
            onPressed: widget.busy ? null : _submit,
            child: widget.busy
                ? SizedBox.square(
                    dimension: moth.space(2.25),
                    child: const CircularProgressIndicator(strokeWidth: 2),
                  )
                : Text(signUp ? 'Create account' : 'Sign in'),
          ),
        ],
      ),
    );
  }
}
