import 'package:flutter/material.dart';

import '../client.dart';
import '../project_config.dart';
import 'moth_theme_scope.dart';

/// The themed social sign-in buttons used by [MothLoginScreen], exposed so
/// a custom login screen can reuse them: one button per provider the
/// project config enables (nothing at all when none are), preceded by an
/// "or" divider by default. [onSelected] receives the tapped provider and
/// drives the actual flow.
///
/// Styling comes from the ambient [Theme]; spacing follows the enclosing
/// [MothThemeScope]'s unit.
class MothProviderButtons extends StatelessWidget {
  const MothProviderButtons({
    super.key,
    required this.config,
    required this.onSelected,
    this.busy = false,
    this.showDivider = true,
  });

  /// The project's public config; only enabled providers get a button.
  final MothProjectConfig config;

  final void Function(MothOAuthProvider provider) onSelected;

  /// Disables the buttons.
  final bool busy;

  /// Renders the "or" divider above the buttons (turn off when the
  /// buttons stand alone rather than under an email form).
  final bool showDivider;

  // Stable keys so app (and SDK) widget tests can target the flow.
  static const googleButtonKey = Key('moth-login-google');
  static const appleButtonKey = Key('moth-login-apple');

  @override
  Widget build(BuildContext context) {
    final moth = MothThemeScope.of(context);
    final theme = Theme.of(context);
    final buttons = [
      if (config.google.enabled)
        OutlinedButton.icon(
          key: googleButtonKey,
          onPressed: busy ? null : () => onSelected(MothOAuthProvider.google),
          icon: const Icon(Icons.g_mobiledata),
          label: const Text('Continue with Google'),
        ),
      if (config.apple.enabled)
        OutlinedButton.icon(
          key: appleButtonKey,
          onPressed: busy ? null : () => onSelected(MothOAuthProvider.apple),
          icon: const Icon(Icons.apple),
          label: const Text('Continue with Apple'),
        ),
    ];
    if (buttons.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (showDivider) ...[
          SizedBox(height: moth.space(2)),
          Row(
            children: [
              const Expanded(child: Divider()),
              Padding(
                padding: EdgeInsets.symmetric(horizontal: moth.space(1.5)),
                child: Text('or', style: theme.textTheme.bodySmall),
              ),
              const Expanded(child: Divider()),
            ],
          ),
          SizedBox(height: moth.space(2)),
        ],
        for (final (index, button) in buttons.indexed) ...[
          if (index > 0) SizedBox(height: moth.space(1.5)),
          button,
        ],
      ],
    );
  }
}
