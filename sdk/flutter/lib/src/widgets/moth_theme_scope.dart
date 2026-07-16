import 'package:flutter/widgets.dart';

import '../theme.dart';

/// Makes the resolved [MothTheme] available to the moth widgets.
///
/// [MothApp] inserts one around the screens it owns (loading and
/// signed-out); [MothLoginScreen] inserts one of its own so its building
/// blocks are themed even standalone. Custom signed-out UI built from the
/// exposed blocks ([MothEmailForm], [MothProviderButtons], [MothLogo]) can
/// insert one to theme them all at once.
class MothThemeScope extends InheritedWidget {
  const MothThemeScope({super.key, required this.theme, required super.child});

  final MothTheme theme;

  /// The nearest scope's theme, or null when there is none.
  static MothTheme? maybeOf(BuildContext context) =>
      context.dependOnInheritedWidgetOfExactType<MothThemeScope>()?.theme;

  /// The nearest scope's theme, or [MothTheme.fallback].
  static MothTheme of(BuildContext context) =>
      maybeOf(context) ?? MothTheme.fallback();

  @override
  bool updateShouldNotify(MothThemeScope oldWidget) => theme != oldWidget.theme;
}
