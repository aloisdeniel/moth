// Contract tests for MothStoreBilling against a fake method-channel host:
// the exact receipt payloads handed to SubmitPurchase are the contract.
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_billing/moth_billing.dart';

const monthly = MothOfferingProduct(
  identifier: 'monthly',
  appleProductId: 'app.pro.monthly',
  googleProductId: 'pro_monthly',
);
const yearly = MothOfferingProduct(
  identifier: 'yearly',
  appleProductId: 'app.pro.yearly',
  googleProductId: 'pro_yearly',
);
const webOnly = MothOfferingProduct(identifier: 'web_only');

const offering = MothOffering(
  identifier: 'default',
  isDefault: true,
  products: [monthly, yearly],
);

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  final calls = <MethodCall>[];

  /// Installs the fake host for this test.
  void host(Future<Object?> Function(MethodCall call) handler) {
    messenger.setMockMethodCallHandler(MothStoreBilling.channel, (call) {
      calls.add(call);
      return handler(call);
    });
  }

  /// Delivers a native-to-Dart call, as the platform sides do for
  /// out-of-band transaction updates, and returns the binary reply the
  /// native side would receive (success envelope = it may finish the
  /// transaction; error envelope = it must keep it alive).
  Future<ByteData?> nativeCall(MethodCall call) async {
    ByteData? reply;
    await messenger.handlePlatformMessage(
      'moth_billing',
      const StandardMethodCodec().encodeMethodCall(call),
      (data) => reply = data,
    );
    return reply;
  }

  setUp(calls.clear);
  tearDown(() {
    messenger.setMockMethodCallHandler(MothStoreBilling.channel, null);
    debugDefaultTargetPlatformOverride = null;
  });

  group('on Apple', () {
    setUp(() {
      debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
    });

    test('purchase returns the JWS receipt SubmitPurchase expects', () async {
      host((call) async {
        expect(call.method, 'purchase');
        expect(call.arguments, {'productId': 'app.pro.monthly'});
        return {'productId': 'app.pro.monthly', 'jws': 'jws-signed-tx'};
      });
      final receipt = await MothStoreBilling().purchase(monthly);
      expect(receipt, isNotNull);
      expect(receipt!.store, MothStore.apple);
      expect(receipt.productIdentifier, 'monthly');
      expect(receipt.appleJwsTransaction, 'jws-signed-tx');
      expect(receipt.googlePurchaseToken, isNull);
      expect(receipt.googleSubscriptionId, isNull);
    });

    test('restore returns the current entitlements JWS list', () async {
      host((call) async {
        expect(call.method, 'restore');
        return ['jws-1', 'jws-2'];
      });
      final restored = await MothStoreBilling().restore();
      expect(restored.store, MothStore.apple);
      expect(restored.receipts, ['jws-1', 'jws-2']);
    });

    test('productsFor maps store products back to moth identifiers', () async {
      host((call) async {
        expect(call.method, 'getProducts');
        expect(call.arguments, {
          'productIds': ['app.pro.monthly', 'app.pro.yearly'],
        });
        return [
          {
            'productId': 'app.pro.monthly',
            'price': r'$9.99',
            'currency': 'USD',
            'title': 'Pro Monthly',
            'description': 'Everything, monthly.',
          },
          // Unknown store products are dropped, not surfaced.
          {'productId': 'app.other', 'price': r'$1.00'},
        ];
      });
      final products = await MothStoreBilling().productsFor(offering);
      expect(products, hasLength(1));
      expect(products.single.productIdentifier, 'monthly');
      expect(products.single.price, r'$9.99');
      expect(products.single.title, 'Pro Monthly');
      expect(products.single.description, 'Everything, monthly.');
    });
  });

  group('on Google', () {
    setUp(() {
      debugDefaultTargetPlatformOverride = TargetPlatform.android;
    });

    test(
      'purchase returns the purchase-token receipt SubmitPurchase expects',
      () async {
        host((call) async {
          expect(call.method, 'purchase');
          expect(call.arguments, {'productId': 'pro_monthly'});
          return {'productId': 'pro_monthly', 'purchaseToken': 'token-123'};
        });
        final receipt = await MothStoreBilling().purchase(monthly);
        expect(receipt, isNotNull);
        expect(receipt!.store, MothStore.google);
        expect(receipt.productIdentifier, 'monthly');
        expect(receipt.googlePurchaseToken, 'token-123');
        expect(receipt.googleSubscriptionId, 'pro_monthly');
        expect(receipt.appleJwsTransaction, isNull);
      },
    );

    test('cancelled purchase resolves to null, not an error', () async {
      host((call) async => null);
      expect(await MothStoreBilling().purchase(monthly), isNull);
    });

    test('pending maps to MothBillingException.pending', () async {
      host((call) async {
        throw PlatformException(
          code: 'pending',
          message: 'The purchase is awaiting approval.',
        );
      });
      await expectLater(
        MothStoreBilling().purchase(monthly),
        throwsA(
          isA<MothBillingException>()
              .having((e) => e.kind, 'kind', MothPurchaseFailureKind.pending)
              .having(
                (e) => e.message,
                'message',
                'The purchase is awaiting approval.',
              ),
        ),
      );
    });

    test('already-owned maps to MothBillingException.alreadyOwned', () async {
      host((call) async {
        throw PlatformException(code: 'already-owned', message: 'Owned.');
      });
      await expectLater(
        MothStoreBilling().purchase(monthly),
        throwsA(
          isA<MothBillingException>().having(
            (e) => e.kind,
            'kind',
            MothPurchaseFailureKind.alreadyOwned,
          ),
        ),
      );
    });

    test('product-not-found maps to MothBillingException.error', () async {
      host((call) async {
        throw PlatformException(
          code: 'not-found',
          message: 'Product "pro_monthly" was not found in the store.',
        );
      });
      await expectLater(
        MothStoreBilling().purchase(monthly),
        throwsA(
          isA<MothBillingException>()
              .having((e) => e.kind, 'kind', MothPurchaseFailureKind.error)
              .having(
                (e) => e.message,
                'message',
                'Product "pro_monthly" was not found in the store.',
              ),
        ),
      );
    });

    test(
      'a tier without a store SKU fails before reaching the store',
      () async {
        host((call) async => fail('channel must not be called'));
        await expectLater(
          MothStoreBilling().purchase(webOnly),
          throwsA(
            isA<MothBillingException>().having(
              (e) => e.kind,
              'kind',
              MothPurchaseFailureKind.error,
            ),
          ),
        );
        expect(calls, isEmpty);
      },
    );

    test('restore returns the current purchase tokens', () async {
      host((call) async {
        expect(call.method, 'restore');
        return ['token-1', 'token-2'];
      });
      final restored = await MothStoreBilling().restore();
      expect(restored.store, MothStore.google);
      expect(restored.receipts, ['token-1', 'token-2']);
    });

    test('productsFor without store SKUs skips the store entirely', () async {
      host((call) async => fail('channel must not be called'));
      const bare = MothOffering(identifier: 'default', products: [webOnly]);
      expect(await MothStoreBilling().productsFor(bare), isEmpty);
      expect(calls, isEmpty);
    });

    test(
      'out-of-band updates surface as receipts on transactionUpdates',
      () async {
        host((call) async => null);
        final billing = MothStoreBilling();
        // Seed the store-id -> moth-id mapping the way an app would.
        await billing.purchase(monthly);
        final updates = billing.transactionUpdates.first;
        final reply = await nativeCall(
          const MethodCall('onTransactionUpdated', {
            'productId': 'pro_monthly',
            'purchaseToken': 'token-deferred',
          }),
        );
        final receipt = await updates;
        expect(receipt.store, MothStore.google);
        expect(receipt.productIdentifier, 'monthly');
        expect(receipt.googlePurchaseToken, 'token-deferred');
        expect(receipt.googleSubscriptionId, 'pro_monthly');
        // The consumed receipt is acknowledged with a success reply, so the
        // native side may finish the transaction.
        expect(const StandardMethodCodec().decodeEnvelope(reply!), isNull);
        billing.dispose();
      },
    );

    test('an update with no transactionUpdates listener is refused, not '
        'dropped', () async {
      host((call) async => null);
      final billing = MothStoreBilling();
      await billing.purchase(monthly);
      // Nobody listens (the adapter was never handed to MothApp): the
      // reply must be an error so the native side keeps the transaction
      // alive for redelivery instead of finishing it.
      final reply = await nativeCall(
        const MethodCall('onTransactionUpdated', {
          'productId': 'pro_monthly',
          'purchaseToken': 'token-deferred',
        }),
      );
      expect(
        () => const StandardMethodCodec().decodeEnvelope(reply!),
        throwsA(
          isA<PlatformException>().having((e) => e.code, 'code', 'unconsumed'),
        ),
      );
      billing.dispose();
    });

    test('a disposed adapter refuses updates instead of throwing into the '
        'channel as a dropped receipt', () async {
      host((call) async => null);
      final billing = MothStoreBilling();
      billing.dispose();
      final reply = await nativeCall(
        const MethodCall('onTransactionUpdated', {
          'productId': 'pro_monthly',
          'purchaseToken': 'token-renewal',
        }),
      );
      expect(
        () => const StandardMethodCodec().decodeEnvelope(reply!),
        throwsA(
          isA<PlatformException>().having((e) => e.code, 'code', 'unconsumed'),
        ),
      );
    });
  });
}
