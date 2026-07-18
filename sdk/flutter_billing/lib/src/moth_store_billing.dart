import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:moth_auth/moth_auth.dart';

/// [MothBillingAdapter] backed by moth's own native billing: StoreKit 2 on
/// iOS, the Play Billing Library on Android. Ships from the moth binary at
/// `/pub` in lockstep with the server, so the receipts it produces are
/// exactly what `SubmitPurchase` validates.
///
/// Auto-renewing subscriptions only. The native sides never grant anything:
/// Apple transactions are finished only after the receipt reaches Dart, and
/// Google purchases are acknowledged by the server after validation, never
/// on-device.
class MothStoreBilling implements MothBillingAdapter {
  MothStoreBilling() {
    channel.setMethodCallHandler(_onNativeCall);
  }

  /// The method channel shared with the native sides.
  @visibleForTesting
  static const MethodChannel channel = MethodChannel('moth_billing');

  final _updates = StreamController<MothPurchaseReceipt>.broadcast();

  /// moth catalog id for each store product id seen in [productsFor] /
  /// [purchase], so out-of-band receipts can be labelled with it.
  final _mothIdByStoreId = <String, String>{};

  /// Receipts for purchases that completed outside an active [purchase] call:
  /// Ask to Buy approvals, pending payments confirming, renewals delivered by
  /// the store. `MothApp` subscribes to the adapter it is given and forwards
  /// every receipt to `SubmitPurchase`; only apps driving the adapter without
  /// `MothApp` need to listen (and submit) themselves. A receipt delivered
  /// while nobody listens is refused, not dropped — the native side then
  /// leaves the transaction unfinished so the store redelivers it.
  @override
  Stream<MothPurchaseReceipt> get transactionUpdates => _updates.stream;

  /// Call when the owning widget disposes.
  void dispose() {
    _updates.close();
  }

  MothStore get _store => switch (defaultTargetPlatform) {
    TargetPlatform.iOS || TargetPlatform.macOS => MothStore.apple,
    _ => MothStore.google,
  };

  String _storeId(MothOfferingProduct p) =>
      _store == MothStore.apple ? p.appleProductId : p.googleProductId;

  @override
  Future<List<MothStoreProduct>> productsFor(MothOffering offering) async {
    final mothIds = <String, String>{
      for (final p in offering.products)
        if (_storeId(p).isNotEmpty) _storeId(p): p.identifier,
    };
    if (mothIds.isEmpty) return const [];
    _mothIdByStoreId.addAll(mothIds);
    final List<Object?>? raw;
    try {
      raw = await channel.invokeListMethod<Object?>('getProducts', {
        'productIds': mothIds.keys.toList(growable: false),
      });
    } on PlatformException catch (e) {
      throw _mapPlatformException(e);
    }
    return [
      for (final entry in raw ?? const <Object?>[])
        if (entry is Map && mothIds[entry['productId']] != null)
          MothStoreProduct(
            productIdentifier: mothIds[entry['productId']]!,
            price: entry['price'] as String? ?? '',
            title: entry['title'] as String? ?? '',
            description: entry['description'] as String? ?? '',
          ),
    ];
  }

  @override
  Future<MothPurchaseReceipt?> purchase(MothOfferingProduct product) async {
    final store = _store;
    final storeId = _storeId(product);
    if (storeId.isEmpty) {
      throw const MothBillingException.error(
        'This tier is not available on this store.',
      );
    }
    _mothIdByStoreId[storeId] = product.identifier;
    final Map<Object?, Object?>? raw;
    try {
      raw = await channel.invokeMapMethod<Object?, Object?>('purchase', {
        'productId': storeId,
      });
    } on PlatformException catch (e) {
      throw _mapPlatformException(e);
    }
    if (raw == null) return null; // the user cancelled the native flow
    return _receipt(store, raw, mothId: product.identifier);
  }

  @override
  Future<MothRestoreReceipts> restore() async {
    final store = _store;
    final List<String>? receipts;
    try {
      receipts = await channel.invokeListMethod<String>('restore');
    } on PlatformException catch (e) {
      throw _mapPlatformException(e);
    }
    return MothRestoreReceipts(store: store, receipts: receipts ?? const []);
  }

  /// The receipt for one native payload: `{productId, jws}` from Apple,
  /// `{productId, purchaseToken}` from Google — the exact shapes
  /// `SubmitPurchase` expects.
  MothPurchaseReceipt _receipt(
    MothStore store,
    Map<Object?, Object?> raw, {
    required String mothId,
  }) {
    final storeProductId = raw['productId'] as String? ?? '';
    return store == MothStore.apple
        ? MothPurchaseReceipt(
            store: MothStore.apple,
            productIdentifier: mothId,
            appleJwsTransaction: raw['jws'] as String? ?? '',
          )
        : MothPurchaseReceipt(
            store: MothStore.google,
            productIdentifier: mothId,
            googlePurchaseToken: raw['purchaseToken'] as String? ?? '',
            googleSubscriptionId: storeProductId,
          );
  }

  MothBillingException _mapPlatformException(PlatformException e) {
    final message = e.message ?? 'Purchase failed.';
    return switch (e.code) {
      'pending' => MothBillingException.pending(message),
      'already-owned' => MothBillingException.alreadyOwned(message),
      // 'unavailable', 'not-found', 'store-error' and anything unexpected are
      // plain store errors; the paywall surfaces the message.
      _ => MothBillingException.error(message),
    };
  }

  Future<Object?> _onNativeCall(MethodCall call) async {
    if (call.method == 'onTransactionUpdated' && call.arguments is Map) {
      if (_updates.isClosed || !_updates.hasListener) {
        // Refuse the receipt instead of dropping it: the error reply keeps
        // the iOS side from finishing the transaction (it replays on a later
        // launch), and on Android the unacknowledged purchase is recovered
        // by a restore or redelivered by the store.
        throw PlatformException(
          code: 'unconsumed',
          message: 'No transactionUpdates listener took the receipt.',
        );
      }
      final raw = (call.arguments as Map).cast<Object?, Object?>();
      final storeId = raw['productId'] as String? ?? '';
      _updates.add(
        _receipt(_store, raw, mothId: _mothIdByStoreId[storeId] ?? storeId),
      );
    }
    // Returning completes the call so the native side can finish() the
    // transaction knowing Dart has taken the receipt.
    return null;
  }
}
