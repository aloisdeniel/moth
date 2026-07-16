/// Flutter SDK for moth — the one-binary authentication server.
///
/// Wrap your app in [MothApp], read auth state with [MothScope.of], and use
/// [MothClient] directly for everything beyond the built-in flow.
library;

export 'src/auth_state.dart';
export 'src/client.dart';
export 'src/config.dart';
export 'src/customer_info.dart';
export 'src/entitlement_cache.dart' hide defaultEntitlementCache;
export 'src/exceptions.dart';
export 'src/http_client.dart';
export 'src/nonce.dart';
export 'src/offering.dart';
export 'src/paywall_cache.dart' hide defaultPaywallCache;
export 'src/project_config.dart';
export 'src/purchase.dart';
export 'src/subscription_controller.dart';
export 'src/theme.dart';
export 'src/theme_cache.dart' hide defaultThemeCache;
export 'src/theme_controller.dart';
export 'src/theme_fonts.dart';
export 'src/token_store.dart';
export 'src/user.dart';
export 'src/version.dart';
export 'src/widgets/billing_adapter.dart';
export 'src/widgets/friendly_errors.dart';
export 'src/widgets/moth_app.dart';
export 'src/widgets/moth_delete_account_dialog.dart';
export 'src/widgets/moth_email_form.dart';
export 'src/widgets/moth_login_screen.dart';
export 'src/widgets/moth_logo.dart';
export 'src/widgets/moth_paywall_screen.dart';
export 'src/widgets/moth_provider_buttons.dart';
export 'src/widgets/moth_scope.dart';
export 'src/widgets/moth_theme_scope.dart';
export 'src/widgets/oauth_adapter.dart';
