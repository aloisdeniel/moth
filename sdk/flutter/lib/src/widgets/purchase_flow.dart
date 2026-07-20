import '../client.dart';
import '../customer_info.dart';
import '../exceptions.dart';
import '../offering.dart';
import '../purchase.dart';
import 'billing_adapter.dart';

/// Runs the full purchase flow for [product]: the native store purchase via
/// [adapter], then `SubmitPurchase` to moth for validation. Shared by
/// `MothScope.purchase` and [MothPaywallScreen] so both behave identically.
///
/// Never throws for an expected outcome — cancel, pending, already-owned, and
/// store/validation errors all come back as a typed [MothPurchaseResult].
Future<MothPurchaseResult> runMothPurchase(
  MothClient client,
  MothBillingAdapter adapter,
  MothOfferingProduct product,
) async {
  final MothPurchaseReceipt? receipt;
  try {
    receipt = await adapter.purchase(product);
  } on MothBillingException catch (err) {
    return switch (err.kind) {
      MothPurchaseFailureKind.pending => const MothPurchasePending(),
      MothPurchaseFailureKind.alreadyOwned => const MothPurchaseAlreadyOwned(),
      MothPurchaseFailureKind.error => MothPurchaseError(
        err.message.isEmpty
            ? 'The purchase could not be completed.'
            : err.message,
      ),
    };
  } on Object catch (err) {
    return MothPurchaseError('The purchase could not be completed: $err');
  }
  if (receipt == null) return const MothPurchaseCancelled();
  return submitMothReceipt(client, receipt);
}

/// Validates [receipt] with `SubmitPurchase`: the server half of
/// [runMothPurchase], also run on its own by `MothApp` for receipts that
/// complete out of band ([MothBillingAdapter.transactionUpdates] — Ask to Buy
/// approvals, pending payments confirming, renewals). Never throws — a
/// validation failure comes back as a typed [MothPurchaseError].
Future<MothPurchaseResult> submitMothReceipt(
  MothClient client,
  MothPurchaseReceipt receipt,
) async {
  try {
    await client.submitPurchase(
      store: receipt.store,
      productIdentifier: receipt.productIdentifier,
      appleJwsTransaction: receipt.appleJwsTransaction,
      googlePurchaseToken: receipt.googlePurchaseToken,
      googleSubscriptionId: receipt.googleSubscriptionId,
    );
    return const MothPurchasePurchased();
  } on MothException catch (err) {
    return MothPurchaseError(err.message, reason: err.reason);
  } on Object catch (err) {
    // Symmetric with the adapter call in runMothPurchase: a non-MothException
    // must never escape (e.g. a StateError from accessToken() when the
    // session was cleared under the store dialog). The store already charged
    // and returned a receipt, so surface it as an error the caller can
    // retry/report rather than stranding the paywall busy and silently
    // dropping the receipt.
    return MothPurchaseError('The purchase could not be completed: $err');
  }
}

/// Reads the device's existing store purchases via [adapter] and re-links them
/// to the current user through `RestorePurchases`. Shared by
/// `MothScope.restorePurchases` and [MothPaywallScreen].
Future<MothCustomerInfo> runMothRestore(
  MothClient client,
  MothBillingAdapter adapter,
) async {
  final restored = await adapter.restore();
  return client.restorePurchases(
    store: restored.store,
    receipts: restored.receipts,
  );
}
