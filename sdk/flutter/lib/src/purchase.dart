import 'customer_info.dart';

/// The receipt a [MothBillingAdapter] produces after a successful native
/// purchase, ready for the SDK to forward to `SubmitPurchase`.
///
/// Provide exactly the field the [store] needs: [appleJwsTransaction] for
/// Apple (StoreKit 2 signed transaction JWS), or [googlePurchaseToken] +
/// [googleSubscriptionId] for Google (Play Billing purchase token and the
/// subscription product id).
class MothPurchaseReceipt {
  const MothPurchaseReceipt({
    required this.store,
    required this.productIdentifier,
    this.appleJwsTransaction,
    this.googlePurchaseToken,
    this.googleSubscriptionId,
  });

  /// The store the purchase happened on.
  final MothStore store;

  /// The moth catalog product identifier that was purchased (not the store
  /// SKU); moth maps it to the store product for validation.
  final String productIdentifier;

  final String? appleJwsTransaction;
  final String? googlePurchaseToken;
  final String? googleSubscriptionId;
}

/// The receipts a [MothBillingAdapter] returns from `restore()`: the current
/// device's store and every re-linkable purchase token / signed transaction
/// found on it.
class MothRestoreReceipts {
  const MothRestoreReceipts({required this.store, this.receipts = const []});

  final MothStore store;
  final List<String> receipts;
}

/// Why a native purchase did not complete, thrown by a [MothBillingAdapter]
/// so the SDK can surface a typed [MothPurchaseResult]. A user cancellation
/// is signalled by returning null from `purchase`, not by throwing.
class MothBillingException implements Exception {
  const MothBillingException(this.kind, [this.message = '']);

  const MothBillingException.pending([String message = ''])
    : this(MothPurchaseFailureKind.pending, message);
  const MothBillingException.alreadyOwned([String message = ''])
    : this(MothPurchaseFailureKind.alreadyOwned, message);
  const MothBillingException.error([String message = ''])
    : this(MothPurchaseFailureKind.error, message);

  final MothPurchaseFailureKind kind;
  final String message;

  @override
  String toString() => 'MothBillingException(${kind.name}): $message';
}

/// The store-side failure kinds a [MothBillingAdapter] can raise.
enum MothPurchaseFailureKind { pending, alreadyOwned, error }

/// The typed outcome of `MothScope.of(context).purchase(product)`.
///
/// ```dart
/// switch (await scope.purchase(product)) {
///   MothPurchasePurchased() => ...,   // entitlements updated, scope rebuilt
///   MothPurchasePending() => ...,     // deferred / ask-to-buy
///   MothPurchaseAlreadyOwned() => ...,
///   MothPurchaseCancelled() => ...,
///   MothPurchaseError(:final message) => ...,
/// }
/// ```
sealed class MothPurchaseResult {
  const MothPurchaseResult();

  const factory MothPurchaseResult.purchased() = MothPurchasePurchased;
  const factory MothPurchaseResult.pending() = MothPurchasePending;
  const factory MothPurchaseResult.alreadyOwned() = MothPurchaseAlreadyOwned;
  const factory MothPurchaseResult.cancelled() = MothPurchaseCancelled;
  const factory MothPurchaseResult.error(String message, {String? reason}) =
      MothPurchaseError;
}

/// The purchase completed and moth validated it; entitlements are updated.
final class MothPurchasePurchased extends MothPurchaseResult {
  const MothPurchasePurchased();
}

/// The purchase is deferred (Ask to Buy / pending payment); entitlements will
/// update once the store confirms.
final class MothPurchasePending extends MothPurchaseResult {
  const MothPurchasePending();
}

/// The user already owns this product (restore rather than buy again).
final class MothPurchaseAlreadyOwned extends MothPurchaseResult {
  const MothPurchaseAlreadyOwned();
}

/// The user cancelled the native purchase.
final class MothPurchaseCancelled extends MothPurchaseResult {
  const MothPurchaseCancelled();
}

/// The purchase failed (store error, or moth rejected the receipt). [reason]
/// carries the server `ErrorInfo` reason when the failure was server-side.
final class MothPurchaseError extends MothPurchaseResult {
  const MothPurchaseError(this.message, {this.reason});

  final String message;
  final String? reason;
}
