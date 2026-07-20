// Tests for MothSubscriptionController's stale-while-revalidate flow and the
// client's entitlement transitions, against the in-process fake billing
// server.
import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/billing/v1/billing.pb.dart' as pb;

import 'fakes.dart';

pb.CustomerInfo _proInfo() => pb.CustomerInfo(
  activeEntitlements: [
    pb.Entitlement(
      identifier: 'pro',
      source: pb.EntitlementSource.ENTITLEMENT_SOURCE_STORE,
      productIdentifier: 'monthly',
    ),
  ],
  subscriptions: [
    pb.ActiveSubscription(
      productIdentifier: 'monthly',
      store: pb.Store.STORE_APPLE,
      status: pb.SubscriptionStatus.SUBSCRIPTION_STATUS_ACTIVE,
      autoRenew: true,
    ),
  ],
);

Future<void> _waitUntil(bool Function() cond, {String? reason}) async {
  final deadline = DateTime.now().add(const Duration(seconds: 5));
  while (!cond()) {
    if (DateTime.now().isAfter(deadline)) {
      fail('timed out waiting for ${reason ?? 'condition'}');
    }
    await Future<void>.delayed(const Duration(milliseconds: 10));
  }
}

void main() {
  late FakeMoth moth;
  late MothClient client;

  setUp(() async {
    moth = await startFakeMoth();
    client = newClient(moth);
  });

  tearDown(() async {
    await client.dispose();
    await moth.shutdown();
  });

  Future<void> signIn() =>
      client.signIn(email: 'jane@example.com', password: 'pw');

  test('a free / never-paid user gets a valid empty CustomerInfo, published '
      'from the server response', () async {
    await signIn();
    // The server reports no entitlements but a lapsed subscription — a valid
    // "free" state whose subscriptions list is non-empty, so a controller that
    // actually published the server result differs from the initial empty
    // default (which has no subscriptions).
    moth.billing.customerInfo = pb.CustomerInfo(
      subscriptions: [
        pb.ActiveSubscription(
          productIdentifier: 'monthly',
          store: pb.Store.STORE_APPLE,
          status: pb.SubscriptionStatus.SUBSCRIPTION_STATUS_EXPIRED,
        ),
      ],
    );
    final controller = MothSubscriptionController(
      client: client,
      cache: MothMemoryEntitlementCache(),
    );
    addTearDown(controller.dispose);
    await controller.start();
    await _waitUntil(
      () => controller.value.subscriptions.isNotEmpty,
      reason: 'server CustomerInfo published',
    );
    expect(controller.value.activeEntitlements, isEmpty);
    expect(controller.value.hasEntitlement('pro'), isFalse);
    expect(moth.billing.getCustomerInfoCalls, greaterThanOrEqualTo(1));
  });

  test('client entitlement transitions none -> active -> expired', () async {
    await signIn();
    expect(client.currentCustomerInfo.hasEntitlement('pro'), isFalse);

    moth.billing.customerInfo = _proInfo();
    final active = await client.getCustomerInfo();
    expect(active.hasEntitlement('pro'), isTrue);
    expect(client.currentCustomerInfo.hasEntitlement('pro'), isTrue);
    final ent = active.entitlement('pro')!;
    expect(ent.source, MothEntitlementSource.store);
    expect(ent.productIdentifier, 'monthly');
    expect(active.subscriptions.single.store, MothStore.apple);

    // Subscription lapses: the server now reports the free tier.
    moth.billing.customerInfo = pb.CustomerInfo();
    final expired = await client.getCustomerInfo();
    expect(expired.hasEntitlement('pro'), isFalse);
    expect(client.currentCustomerInfo.activeEntitlements, isEmpty);
  });

  test('stale-while-revalidate: cached snapshot renders before the refresh, '
      'then the server truth replaces it', () async {
    await signIn();
    final cache = MothMemoryEntitlementCache();
    // The user was pro on the last launch.
    await cache.save('user-1', MothCustomerInfo.fromProto(_proInfo()));
    // The server now reports the subscription lapsed, and holds the response.
    moth.billing.customerInfo = pb.CustomerInfo();
    moth.billing.getCustomerInfoGate = Completer<void>();

    final controller = MothSubscriptionController(client: client, cache: cache);
    addTearDown(controller.dispose);

    await controller.start();
    // Cached pro renders while the refresh is gated.
    await _waitUntil(
      () => controller.value.hasEntitlement('pro'),
      reason: 'cached pro',
    );

    moth.billing.getCustomerInfoGate!.complete();
    // ...then the server's fresh (free) state lands.
    await _waitUntil(
      () => !controller.value.hasEntitlement('pro'),
      reason: 'server free',
    );
    // The fresh state was written back to the cache.
    expect((await cache.load('user-1'))!.hasEntitlement('pro'), isFalse);
  });

  test('signing out resets the subscription state to free', () async {
    await signIn();
    moth.billing.customerInfo = _proInfo();
    await client.getCustomerInfo();
    expect(client.currentCustomerInfo.hasEntitlement('pro'), isTrue);

    final controller = MothSubscriptionController(
      client: client,
      cache: MothMemoryEntitlementCache(),
    );
    addTearDown(controller.dispose);
    await controller.start();
    await _waitUntil(() => controller.value.hasEntitlement('pro'));

    await client.signOut();
    await _waitUntil(() => !controller.value.hasEntitlement('pro'));
    expect(client.currentCustomerInfo.activeEntitlements, isEmpty);
  });

  test('signing out does not overwrite the outgoing user\'s cached '
      'entitlements with the free reset', () async {
    await signIn();
    moth.billing.customerInfo = _proInfo();
    await client.getCustomerInfo();

    final cache = MothMemoryEntitlementCache();
    final controller = MothSubscriptionController(client: client, cache: cache);
    addTearDown(controller.dispose);
    await controller.start();
    // The pro snapshot is persisted for the signed-in user.
    await _waitUntil(
      () => controller.value.hasEntitlement('pro'),
      reason: 'pro published',
    );
    // The controller persists asynchronously; wait for the cache write.
    final deadline = DateTime.now().add(const Duration(seconds: 5));
    while (!((await cache.load('user-1'))?.hasEntitlement('pro') ?? false)) {
      if (DateTime.now().isAfter(deadline)) fail('timed out caching pro');
      await Future<void>.delayed(const Duration(milliseconds: 10));
    }

    await client.signOut();
    await _waitUntil(() => !controller.value.hasEntitlement('pro'));

    // The sign-out free reset must NOT have clobbered the user's cache, so a
    // returning subscriber gates instantly on next sign-in.
    final cached = await cache.load('user-1');
    expect(cached, isNotNull);
    expect(cached!.hasEntitlement('pro'), isTrue);
  });

  test('entitlementsChanged reflects the disk-cached snapshot before the '
      'server refresh lands (offline launch)', () async {
    await signIn();
    final cache = MothMemoryEntitlementCache();
    await cache.save('user-1', MothCustomerInfo.fromProto(_proInfo()));
    // The background refresh fails (offline): the cached snapshot must remain
    // visible on the client's stream, not be masked by the free default.
    moth.billing.nextError = mothError(14, 'UNAVAILABLE', 'offline');

    final controller = MothSubscriptionController(client: client, cache: cache);
    addTearDown(controller.dispose);
    await controller.start();

    await _waitUntil(
      () => client.currentCustomerInfo.hasEntitlement('pro'),
      reason: 'client primed from cache',
    );
    // A late (non-widget) subscriber replays the cached pro snapshot.
    final replayed = await client.customerInfoChanges.first;
    expect(replayed.hasEntitlement('pro'), isTrue);
  });
}
