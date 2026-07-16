import '../offering.dart';
import '../purchase.dart';

/// Bridges moth's purchase flow to the native store billing SDKs.
///
/// `moth_auth` deliberately does **not** depend on `in_app_purchase` (or any
/// StoreKit / Play Billing plugin), so apps that don't sell subscriptions stay
/// light and skip that native setup — the same optional-native-adapter pattern
/// used for Google/Apple sign-in ([MothOAuthAdapter]). Apps that sell
/// subscriptions implement this interface with the plugin of their choice and
/// pass it to [MothApp] (or [MothPaywallScreen]); the SDK's example app ships a
/// ready-made `in_app_purchase` implementation to copy.
///
/// The SDK never handles money: the adapter runs the native purchase and
/// returns the resulting signed transaction / purchase token, which the SDK
/// forwards to moth for server-side validation.
abstract class MothBillingAdapter {
  /// Runs the native purchase for [product] and returns its receipt on
  /// success, or **null** when the user cancelled.
  ///
  /// Throw a [MothBillingException] for a non-cancel failure so the SDK can
  /// surface a typed result: `pending` (deferred / Ask to Buy), `alreadyOwned`
  /// (offer a restore instead), or `error`.
  Future<MothPurchaseReceipt?> purchase(MothOfferingProduct product);

  /// Re-reads the store's existing purchases on this device and returns their
  /// receipts, for `RestorePurchases` to re-link to the current user.
  Future<MothRestoreReceipts> restore();

  /// The native store products backing [offering], for display (localized
  /// prices, etc.). Optional: return an empty list when not implemented — the
  /// paywall then renders the catalog price/period from [offering].
  Future<List<MothStoreProduct>> productsFor(MothOffering offering) async =>
      const [];
}

/// A native store product's localized display fields, as read from the store
/// via a [MothBillingAdapter]. Keyed to a moth catalog product by
/// [productIdentifier].
class MothStoreProduct {
  const MothStoreProduct({
    required this.productIdentifier,
    required this.price,
    this.title = '',
    this.description = '',
  });

  /// The moth catalog product identifier this store product maps to.
  final String productIdentifier;

  /// The store's localized, formatted price string (e.g. `$9.99`).
  final String price;
  final String title;
  final String description;
}
