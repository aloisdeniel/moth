import 'package:flutter/widgets.dart';

import '../copy.dart';
import '../locale.dart';

/// Makes the resolved, localized [MothCopy] available to the moth widgets.
///
/// [MothApp] inserts one around the screens it owns (login and paywall),
/// fed by its [MothCopyController] so the screens re-render when the copy
/// changes — a device-language switch, or an operator's copy edit landing on
/// the next background refresh. [MothLoginScreen] and [MothPaywallScreen]
/// insert one of their own so their building blocks ([MothEmailForm],
/// [MothPurchaseButton], …) read the same copy even standalone.
///
/// The copy resolves **server override → bundled → English**: see [MothCopy].
class MothCopyScope extends InheritedWidget {
  const MothCopyScope({super.key, required this.copy, required super.child});

  final MothCopy copy;

  /// The nearest scope's copy, or null when there is none.
  static MothCopy? maybeOf(BuildContext context) =>
      context.dependOnInheritedWidgetOfExactType<MothCopyScope>()?.copy;

  /// The nearest scope's copy, or the bundled floor for the device locale when
  /// there is none (so a standalone building block is still localized).
  static MothCopy of(BuildContext context) =>
      maybeOf(context) ?? MothCopy.bundled(mothDeviceLocale());

  @override
  bool updateShouldNotify(MothCopyScope oldWidget) => copy != oldWidget.copy;
}
