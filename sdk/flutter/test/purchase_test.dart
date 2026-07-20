// Tests for the purchase flow (adapter -> SubmitPurchase -> entitlements),
// exercising every typed outcome against the fake adapter + fake server.
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/billing/v1/billing.pb.dart' as pb;
import 'package:moth_auth/src/widgets/purchase_flow.dart';

import 'fakes.dart';

const _product = MothOfferingProduct(
  identifier: 'monthly',
  displayName: 'Monthly',
  appleProductId: 'com.example.monthly',
  entitlements: ['pro'],
);

pb.CustomerInfo _proInfo() => pb.CustomerInfo(
  activeEntitlements: [
    pb.Entitlement(
      identifier: 'pro',
      source: pb.EntitlementSource.ENTITLEMENT_SOURCE_STORE,
    ),
  ],
);

void main() {
  late FakeMoth moth;
  late MothClient client;
  late FakeBillingAdapter adapter;

  setUp(() async {
    moth = await startFakeMoth();
    client = newClient(moth);
    adapter = FakeBillingAdapter();
    await client.signIn(email: 'jane@example.com', password: 'pw');
  });

  tearDown(() async {
    await client.dispose();
    await moth.shutdown();
  });

  test('a successful purchase validates and flips the entitlement', () async {
    moth.billing.customerInfoAfterPurchase = _proInfo();

    final result = await runMothPurchase(client, adapter, _product);

    expect(result, isA<MothPurchasePurchased>());
    expect(adapter.lastProduct?.identifier, 'monthly');
    // The receipt was forwarded to SubmitPurchase with the Apple oneof.
    expect(moth.billing.lastSubmit?.store, pb.Store.STORE_APPLE);
    expect(moth.billing.lastSubmit?.productIdentifier, 'monthly');
    expect(moth.billing.lastSubmit?.appleJwsTransaction, 'jws-monthly');
    expect(client.currentCustomerInfo.hasEntitlement('pro'), isTrue);
  });

  test('a cancelled purchase never reaches the server', () async {
    adapter.cancel = true;

    final result = await runMothPurchase(client, adapter, _product);

    expect(result, isA<MothPurchaseCancelled>());
    expect(moth.billing.lastSubmit, isNull);
    expect(client.currentCustomerInfo.hasEntitlement('pro'), isFalse);
  });

  test('a deferred (pending) purchase surfaces MothPurchasePending', () async {
    adapter.throwOnPurchase = const MothBillingException.pending();

    final result = await runMothPurchase(client, adapter, _product);

    expect(result, isA<MothPurchasePending>());
    expect(moth.billing.lastSubmit, isNull);
  });

  test('an already-owned product surfaces MothPurchaseAlreadyOwned', () async {
    adapter.throwOnPurchase = const MothBillingException.alreadyOwned();

    final result = await runMothPurchase(client, adapter, _product);

    expect(result, isA<MothPurchaseAlreadyOwned>());
    // The already-owned short-circuit must never forward a receipt.
    expect(moth.billing.lastSubmit, isNull);
  });

  test('a purchase while signed out maps to MothPurchaseError, not an '
      'escaping StateError', () async {
    // The store can hand back a valid receipt after the moth session was
    // cleared (e.g. a background refresh failed under the store dialog).
    // submitPurchase then throws a plain StateError from accessToken(); it
    // must be caught and surfaced as a typed error, never escape.
    await client.signOut();

    final result = await runMothPurchase(client, adapter, _product);

    expect(result, isA<MothPurchaseError>());
    expect(adapter.purchaseCalls, 1);
    // The native purchase succeeded but the receipt could not be submitted.
    expect(moth.billing.lastSubmit, isNull);
  });

  test('a store-side failure surfaces MothPurchaseError', () async {
    adapter.throwOnPurchase = const MothBillingException.error('store down');

    final result = await runMothPurchase(client, adapter, _product);

    expect(result, isA<MothPurchaseError>());
    expect((result as MothPurchaseError).message, 'store down');
  });

  test('a server-rejected receipt surfaces MothPurchaseError with the '
      'reason', () async {
    moth.billing.purchaseError = mothError(
      3 /* invalid argument */,
      'INVALID_RECEIPT',
      'receipt rejected',
    );

    final result = await runMothPurchase(client, adapter, _product);

    expect(result, isA<MothPurchaseError>());
    expect((result as MothPurchaseError).reason, 'INVALID_RECEIPT');
    expect(client.currentCustomerInfo.hasEntitlement('pro'), isFalse);
  });

  test('restore re-links receipts and updates the entitlements', () async {
    moth.billing.customerInfo = _proInfo();

    final info = await runMothRestore(client, adapter);

    expect(adapter.restoreCalls, 1);
    expect(moth.billing.lastRestore?.store, pb.Store.STORE_APPLE);
    expect(moth.billing.lastRestore?.receipts, ['restore-jws']);
    expect(info.hasEntitlement('pro'), isTrue);
    expect(client.currentCustomerInfo.hasEntitlement('pro'), isTrue);
  });
}
