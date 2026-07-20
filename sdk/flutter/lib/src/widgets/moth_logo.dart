import 'package:flutter/material.dart';

import '../theme.dart';
import 'moth_theme_scope.dart';

/// The project's logo as configured in the moth admin: picks the light or
/// dark variant to match the ambient brightness and renders nothing when
/// the project has no logo (so it can be dropped into any layout
/// unconditionally).
///
/// The theme resolves from [theme], else the enclosing [MothThemeScope],
/// else the default (which has no logo).
class MothLogo extends StatelessWidget {
  const MothLogo({super.key, this.theme, this.height});

  /// Theme override for standalone use.
  final MothTheme? theme;

  /// Rendered height; defaults to 8 spacing units.
  final double? height;

  @override
  Widget build(BuildContext context) {
    final moth = theme ?? MothThemeScope.of(context);
    final dark = Theme.of(context).brightness == Brightness.dark;
    final url = dark
        ? (moth.logoDarkUrl ?? moth.logoLightUrl)
        : moth.logoLightUrl;
    if (url == null) return const SizedBox.shrink();
    final logoHeight = height ?? moth.space(8);
    return Image.network(
      url,
      height: logoHeight,
      fit: BoxFit.contain,
      semanticLabel: 'Logo',
      // While loading (and if the download fails) hold the height so the
      // layout doesn't jump.
      errorBuilder: (_, _, _) => SizedBox(height: logoHeight),
      frameBuilder: (_, child, frame, _) =>
          frame == null ? SizedBox(height: logoHeight) : child,
    );
  }
}
