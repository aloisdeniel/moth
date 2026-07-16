// Wires moth's purchase flow to the real store billing SDK. The moth_auth
// package deliberately has no dependency on in_app_purchase — this file is the
// glue an app copies and adapts, mirroring oauth_adapter.dart.
//
// Caveat: `in_app_purchase` surfaces StoreKit 1 receipts on iOS. moth's
// SubmitPurchase expects a StoreKit 2 signed transaction (JWS); a production
// app targeting Apple should forward a StoreKit 2 transaction (e.g. via a
// StoreKit 2 plugin). This example forwards whatever serverVerificationData the
// plugin provides so the wiring is complete on Android and illustrative on iOS.
import 'dart:async';
import 'dart:io' show Platform;

import 'package:in_app_purchase/in_app_purchase.dart';
import 'package:moth_auth/moth_auth.dart';

/// [MothBillingAdapter] backed by `in_app_purchase` (StoreKit / Play Billing).
class ExampleBillingAdapter implements MothBillingAdapter {
  ExampleBillingAdapter() {
    _sub = _iap.purchaseStream.listen(_onPurchaseUpdates, onError: (_) {});
  }

  final InAppPurchase _iap = InAppPurchase.instance;
  late final StreamSubscription<List<PurchaseDetails>> _sub;

  /// Purchases awaiting resolution, keyed by store product id.
  final _pending = <String, Completer<MothPurchaseReceipt?>>{};

  /// moth catalog id for each store product id currently being purchased.
  final _mothIdByStoreId = <String, String>{};

  /// Restored purchases collected off the stream, keyed by store product id.
  final _restored = <String, PurchaseDetails>{};

  MothStore get _store => Platform.isIOS ? MothStore.apple : MothStore.google;

  /// Call when the owning widget disposes.
  void dispose() => _sub.cancel();

  String _storeId(MothOfferingProduct p) =>
      _store == MothStore.apple ? p.appleProductId : p.googleProductId;

  @override
  Future<List<MothStoreProduct>> productsFor(MothOffering offering) async {
    if (!await _iap.isAvailable()) return const [];
    final storeIds = {
      for (final p in offering.products)
        if (_storeId(p).isNotEmpty) _storeId(p): p.identifier,
    };
    if (storeIds.isEmpty) return const [];
    final resp = await _iap.queryProductDetails(storeIds.keys.toSet());
    return [
      for (final d in resp.productDetails)
        MothStoreProduct(
          productIdentifier: storeIds[d.id] ?? d.id,
          price: d.price,
          title: d.title,
          description: d.description,
        ),
    ];
  }

  @override
  Future<MothPurchaseReceipt?> purchase(MothOfferingProduct product) async {
    if (!await _iap.isAvailable()) {
      throw const MothBillingException.error(
        'In-app purchases are unavailable on this device.',
      );
    }
    final storeId = _storeId(product);
    if (storeId.isEmpty) {
      throw const MothBillingException.error(
        'This tier is not available on this store.',
      );
    }
    final resp = await _iap.queryProductDetails({storeId});
    if (resp.productDetails.isEmpty) {
      throw MothBillingException.error(
        'Product "$storeId" was not found in the store.',
      );
    }
    final completer = Completer<MothPurchaseReceipt?>();
    _pending[storeId] = completer;
    _mothIdByStoreId[storeId] = product.identifier;
    await _iap.buyNonConsumable(
      purchaseParam: PurchaseParam(productDetails: resp.productDetails.first),
    );
    return completer.future;
  }

  @override
  Future<MothRestoreReceipts> restore() async {
    _restored.clear();
    await _iap.restorePurchases();
    // Give the platform a moment to replay restored purchases on the stream.
    await Future<void>.delayed(const Duration(seconds: 2));
    return MothRestoreReceipts(
      store: _store,
      receipts: [
        for (final p in _restored.values)
          p.verificationData.serverVerificationData,
      ],
    );
  }

  void _onPurchaseUpdates(List<PurchaseDetails> purchases) {
    for (final p in purchases) {
      switch (p.status) {
        case PurchaseStatus.pending:
          _fail(p.productID, const MothBillingException.pending());
        case PurchaseStatus.canceled:
          _succeed(p.productID, null);
        case PurchaseStatus.error:
          _fail(
            p.productID,
            MothBillingException.error(p.error?.message ?? 'Purchase failed.'),
          );
        case PurchaseStatus.purchased:
        case PurchaseStatus.restored:
          _restored[p.productID] = p;
          _succeed(p.productID, _receipt(p));
      }
      if (p.pendingCompletePurchase) {
        unawaited(_iap.completePurchase(p));
      }
    }
  }

  MothPurchaseReceipt _receipt(PurchaseDetails p) {
    final token = p.verificationData.serverVerificationData;
    final mothId = _mothIdByStoreId[p.productID] ?? p.productID;
    return _store == MothStore.apple
        ? MothPurchaseReceipt(
            store: MothStore.apple,
            productIdentifier: mothId,
            appleJwsTransaction: token,
          )
        : MothPurchaseReceipt(
            store: MothStore.google,
            productIdentifier: mothId,
            googlePurchaseToken: token,
            googleSubscriptionId: p.productID,
          );
  }

  void _succeed(String storeId, MothPurchaseReceipt? receipt) =>
      _pending.remove(storeId)?.complete(receipt);

  void _fail(String storeId, MothBillingException error) {
    final completer = _pending.remove(storeId);
    if (completer != null && !completer.isCompleted) {
      completer.completeError(error);
    }
  }
}
