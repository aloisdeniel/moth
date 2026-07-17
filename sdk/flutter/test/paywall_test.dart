// Widget tests for MothPaywallScreen (rendering, highlight, empty state) and
// MothApp's requiresEntitlement gating, against the in-process fake server.
import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/billing/v1/billing.pb.dart' as pb;

import 'fakes.dart';
import 'widget_helpers.dart';

pb.Offering _offeringWithTiers() => pb.Offering(
  identifier: 'default',
  isDefault: true,
  products: [
    pb.OfferingProduct(
      identifier: 'monthly',
      displayName: 'Monthly',
      appleProductId: 'com.example.monthly',
      billingPeriod: 'P1M',
      priceAmountMicros: Int64(9990000),
      currency: 'USD',
      entitlements: ['pro'],
      sortOrder: 0,
    ),
    pb.OfferingProduct(
      identifier: 'yearly',
      displayName: 'Yearly',
      appleProductId: 'com.example.yearly',
      billingPeriod: 'P1Y',
      priceAmountMicros: Int64(59990000),
      currency: 'USD',
      trialPeriod: 'P1W',
      entitlements: ['pro'],
      sortOrder: 1,
      highlighted: true,
    ),
  ],
);

pb.Paywall _paywall() => pb.Paywall(
  revisionId: 'pw-1',
  headline: 'Go Pro',
  subtitle: 'Everything unlocked.',
  benefits: ['Unlimited exports', 'Priority support'],
  highlightedProductIdentifier: 'yearly',
  layout: pb.PaywallLayout.PAYWALL_LAYOUT_TILES,
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

  Future<void> pumpStandalone(WidgetTester tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: MothPaywallScreen(client: client, adapter: FakeBillingAdapter()),
      ),
    );
  }

  testWidgets('renders the offering + config and marks the highlighted '
      'tier', (tester) async {
    moth = await runReal(tester, startFakeMoth);
    moth.billing.offering = _offeringWithTiers();
    moth.billing.paywall = _paywall();
    client = newClient(moth);

    await pumpStandalone(tester);
    await pumpUntilFound(tester, find.byKey(MothPaywallScreen.headlineKey));
    await tester.pump();

    expect(find.text('Go Pro'), findsOneWidget);
    expect(find.text('Everything unlocked.'), findsOneWidget);
    expect(find.text('Unlimited exports'), findsOneWidget);
    expect(find.text('Priority support'), findsOneWidget);
    // A card per tier.
    expect(
      find.byKey(MothPaywallScreen.tierCardKey('monthly')),
      findsOneWidget,
    );
    expect(find.byKey(MothPaywallScreen.tierCardKey('yearly')), findsOneWidget);
    // Highlight + trial badges on the yearly tier.
    expect(find.text('Most popular'), findsOneWidget);
    expect(find.text('1-week free trial'), findsOneWidget);
    // Catalog price rendered with the billing period suffix.
    expect(find.text(r'$9.99 / month'), findsOneWidget);
    expect(find.text(r'$59.99 / year'), findsOneWidget);
    expect(find.byKey(MothPaywallScreen.purchaseButtonKey), findsOneWidget);
    expect(find.byKey(MothPaywallScreen.restoreKey), findsOneWidget);

    await settle(tester, client.dispose());
    await settle(tester, moth.shutdown());
  });

  testWidgets('renders a graceful empty state when there are no products', (
    tester,
  ) async {
    moth = await runReal(tester, startFakeMoth);
    moth.billing.offering = pb.Offering(identifier: 'default', isDefault: true);
    moth.billing.paywall = _paywall();
    client = newClient(moth);

    await pumpStandalone(tester);
    await pumpUntilFound(tester, find.byKey(MothPaywallScreen.emptyStateKey));

    expect(find.byKey(MothPaywallScreen.purchaseButtonKey), findsNothing);
    expect(find.text('Nothing to purchase yet'), findsOneWidget);

    await settle(tester, client.dispose());
    await settle(tester, moth.shutdown());
  });

  testWidgets(
    'MothApp requiresEntitlement shows the paywall to a free user and the '
    'child once entitled',
    (tester) async {
      moth = await runReal(tester, startFakeMoth);
      moth.billing.offering = _offeringWithTiers();
      moth.billing.paywall = _paywall();
      client = newClient(moth);
      await runReal(
        tester,
        () => client.signIn(email: 'jane@example.com', password: 'pw'),
      );

      await tester.pumpWidget(
        MothApp(
          client: client,
          requiresEntitlement: 'pro',
          paywall: const MothPaywallScreen(),
          child: const MaterialApp(
            home: Scaffold(body: Text('SECRET CONTENT')),
          ),
        ),
      );

      // Free user: the gate resolves the offering and shows the paywall.
      await pumpUntilFound(tester, find.byKey(MothPaywallScreen.headlineKey));
      expect(find.text('SECRET CONTENT'), findsNothing);

      // The user subscribes elsewhere; a refresh flips the entitlement and the
      // gate hands them through to the child.
      moth.billing.customerInfo = _proInfo();
      await runReal(tester, () => client.getCustomerInfo());
      await pumpUntilFound(tester, find.text('SECRET CONTENT'));

      await settle(tester, client.dispose());
      await settle(tester, moth.shutdown());
    },
  );

  testWidgets(
    'gating on an entitlement no product grants never blocks the child',
    (tester) async {
      moth = await runReal(tester, startFakeMoth);
      // Products exist but none grant "vip".
      moth.billing.offering = _offeringWithTiers();
      moth.billing.paywall = _paywall();
      client = newClient(moth);
      await runReal(
        tester,
        () => client.signIn(email: 'jane@example.com', password: 'pw'),
      );

      await tester.pumpWidget(
        MothApp(
          client: client,
          requiresEntitlement: 'vip',
          paywall: const MothPaywallScreen(),
          child: const MaterialApp(
            home: Scaffold(body: Text('SECRET CONTENT')),
          ),
        ),
      );

      // Nothing sells "vip", so the gate falls through to the child.
      await pumpUntilFound(tester, find.text('SECRET CONTENT'));
      expect(find.byKey(MothPaywallScreen.headlineKey), findsNothing);

      await settle(tester, client.dispose());
      await settle(tester, moth.shutdown());
    },
  );

  testWidgets(
    'requiresEntitlement gate resolves the paywall\'s (non-default) offering, '
    'not the default one',
    (tester) async {
      moth = await runReal(tester, startFakeMoth);
      // The default offering sells nothing that grants "pro"; only the
      // "premium" offering the paywall points at does.
      moth.billing.offering = pb.Offering(
        identifier: 'default',
        isDefault: true,
      );
      moth.billing.offeringsByTag['premium'] = pb.Offering(
        identifier: 'premium',
        products: [
          pb.OfferingProduct(
            identifier: 'monthly',
            displayName: 'Monthly',
            appleProductId: 'com.example.monthly',
            billingPeriod: 'P1M',
            priceAmountMicros: Int64(9990000),
            currency: 'USD',
            entitlements: ['pro'],
          ),
        ],
      );
      moth.billing.paywall = pb.Paywall(
        revisionId: 'pw-1',
        headline: 'Go Pro',
        offering: 'premium',
        layout: pb.PaywallLayout.PAYWALL_LAYOUT_TILES,
      );
      client = newClient(moth);
      await runReal(
        tester,
        () => client.signIn(email: 'jane@example.com', password: 'pw'),
      );

      await tester.pumpWidget(
        MothApp(
          client: client,
          requiresEntitlement: 'pro',
          paywall: const MothPaywallScreen(),
          child: const MaterialApp(
            home: Scaffold(body: Text('SECRET CONTENT')),
          ),
        ),
      );

      // "pro" IS for sale (via premium), so the free user is blocked.
      await pumpUntilFound(tester, find.byKey(MothPaywallScreen.headlineKey));
      expect(find.text('SECRET CONTENT'), findsNothing);

      await settle(tester, client.dispose());
      await settle(tester, moth.shutdown());
    },
  );

  testWidgets(
    'requiresEntitlement transitions none -> active -> expired rebuild the gate',
    (tester) async {
      moth = await runReal(tester, startFakeMoth);
      moth.billing.offering = _offeringWithTiers();
      moth.billing.paywall = _paywall();
      client = newClient(moth);
      await runReal(
        tester,
        () => client.signIn(email: 'jane@example.com', password: 'pw'),
      );

      await tester.pumpWidget(
        MothApp(
          client: client,
          requiresEntitlement: 'pro',
          paywall: const MothPaywallScreen(),
          child: const MaterialApp(
            home: Scaffold(body: Text('SECRET CONTENT')),
          ),
        ),
      );

      // none: paywall.
      await pumpUntilFound(tester, find.byKey(MothPaywallScreen.headlineKey));
      expect(find.text('SECRET CONTENT'), findsNothing);

      // active: the gate hands through to the child.
      moth.billing.customerInfo = _proInfo();
      await runReal(tester, () => client.getCustomerInfo());
      await pumpUntilFound(tester, find.text('SECRET CONTENT'));

      // expired: the subscription lapses; the gate must re-hide the content
      // and show the paywall again.
      moth.billing.customerInfo = pb.CustomerInfo();
      await runReal(tester, () => client.getCustomerInfo());
      await pumpUntilFound(tester, find.byKey(MothPaywallScreen.headlineKey));
      expect(find.text('SECRET CONTENT'), findsNothing);

      await settle(tester, client.dispose());
      await settle(tester, moth.shutdown());
    },
  );

  testWidgets('tier cards show the store-localized price when the adapter '
      'provides one', (tester) async {
    moth = await runReal(tester, startFakeMoth);
    moth.billing.offering = _offeringWithTiers();
    moth.billing.paywall = _paywall();
    client = newClient(moth);
    final adapter = FakeBillingAdapter()
      ..storeProducts = const [
        MothStoreProduct(productIdentifier: 'monthly', price: '9,99 €'),
      ];

    await tester.pumpWidget(
      MaterialApp(
        home: MothPaywallScreen(client: client, adapter: adapter),
      ),
    );
    await pumpUntilFound(tester, find.byKey(MothPaywallScreen.headlineKey));
    await tester.pump();

    // Monthly: the store price wins over the catalog USD price.
    expect(find.text('9,99 € / month'), findsOneWidget);
    expect(find.text(r'$9.99 / month'), findsNothing);
    // Yearly has no store product: it falls back to the catalog price.
    expect(find.text(r'$59.99 / year'), findsOneWidget);
    expect(adapter.productsForCalls, greaterThanOrEqualTo(1));

    await settle(tester, client.dispose());
    await settle(tester, moth.shutdown());
  });

  testWidgets('compact layout shows a single tier with a period toggle', (
    tester,
  ) async {
    moth = await runReal(tester, startFakeMoth);
    moth.billing.offering = _offeringWithTiers();
    moth.billing.paywall = _paywall()
      ..layout = pb.PaywallLayout.PAYWALL_LAYOUT_COMPACT;
    client = newClient(moth);

    await pumpStandalone(tester);
    await pumpUntilFound(tester, find.byKey(MothPaywallScreen.headlineKey));
    await tester.pump();

    // Only the highlighted tier (yearly) is rendered as a card...
    expect(find.byKey(MothPaywallScreen.tierCardKey('yearly')), findsOneWidget);
    expect(find.byKey(MothPaywallScreen.tierCardKey('monthly')), findsNothing);
    // ...with a period toggle offering both tiers.
    expect(find.byType(SegmentedButton<String>), findsOneWidget);
    expect(find.text('Month'), findsOneWidget);
    expect(find.text('Year'), findsOneWidget);

    await settle(tester, client.dispose());
    await settle(tester, moth.shutdown());
  });

  testWidgets('a matching cached revision keeps the cached copy (server omits '
      'the body)', (tester) async {
    moth = await runReal(tester, startFakeMoth);
    moth.billing.offering = _offeringWithTiers();
    // The server would send this headline, but omits the body when the known
    // revision matches "pw-1".
    moth.billing.paywall = _paywall()..headline = 'SERVER HEADLINE';
    client = newClient(moth);

    final cache = MothMemoryPaywallCache();
    // Stale (outside the download-once TTL), so the screen still runs the
    // cheap revalidation round-trip against the cached revision.
    await cache.save(
      const MothPaywall(revisionId: 'pw-1', headline: 'CACHED HEADLINE'),
      fetchedAt: DateTime.now().toUtc().subtract(const Duration(hours: 2)),
    );

    await tester.pumpWidget(
      MaterialApp(
        home: MothPaywallScreen(
          client: client,
          adapter: FakeBillingAdapter(),
          paywallCache: cache,
        ),
      ),
    );
    await pumpUntilFound(tester, find.text('CACHED HEADLINE'));

    // The cached revision was echoed, and the omitted body left the cache in
    // place rather than blanking the copy.
    expect(moth.billing.lastPaywallRequest?.knownPaywallRevision, 'pw-1');
    expect(find.text('SERVER HEADLINE'), findsNothing);

    // The omitted-body match re-stamped the envelope's fetch time, so the
    // next launch stays within the download-once TTL.
    final entry = await settle(tester, cache.load());
    expect(
      DateTime.now().toUtc().difference(entry!.fetchedAt),
      lessThan(const Duration(minutes: 1)),
    );

    await settle(tester, client.dispose());
    await settle(tester, moth.shutdown());
  });

  testWidgets('download-once TTL: a fresh cached config renders with zero '
      'GetPaywall calls', (tester) async {
    moth = await runReal(tester, startFakeMoth);
    moth.billing.offering = _offeringWithTiers();
    moth.billing.paywall = _paywall()..headline = 'SERVER HEADLINE';
    client = newClient(moth);

    final cache = MothMemoryPaywallCache();
    await cache.save(
      const MothPaywall(revisionId: 'pw-1', headline: 'CACHED HEADLINE'),
      fetchedAt: DateTime.now().toUtc(),
    );

    await tester.pumpWidget(
      MaterialApp(
        home: MothPaywallScreen(
          client: client,
          adapter: FakeBillingAdapter(),
          paywallCache: cache,
        ),
      ),
    );
    await pumpUntilFound(tester, find.text('CACHED HEADLINE'));

    // Within the TTL the config is served from the cache alone — no
    // GetPaywall round-trip at all (the offering fetch is not a config RPC).
    expect(moth.billing.getPaywallCalls, 0);
    expect(find.text('SERVER HEADLINE'), findsNothing);

    await settle(tester, client.dispose());
    await settle(tester, moth.shutdown());
  });

  testWidgets('download-once TTL: an expired cached config revalidates and '
      'applies a new revision', (tester) async {
    moth = await runReal(tester, startFakeMoth);
    moth.billing.offering = _offeringWithTiers();
    moth.billing.paywall = _paywall()..headline = 'SERVER HEADLINE';
    client = newClient(moth);

    final cache = MothMemoryPaywallCache();
    await cache.save(
      const MothPaywall(revisionId: 'pw-0', headline: 'CACHED HEADLINE'),
      fetchedAt: DateTime.now().toUtc().subtract(const Duration(hours: 2)),
    );

    await tester.pumpWidget(
      MaterialApp(
        home: MothPaywallScreen(
          client: client,
          adapter: FakeBillingAdapter(),
          paywallCache: cache,
        ),
      ),
    );
    await pumpUntilFound(tester, find.text('SERVER HEADLINE'));

    // Exactly one revalidation, echoing the stale revision; the fresh body
    // replaced the cache with a new fetch time.
    expect(moth.billing.getPaywallCalls, 1);
    expect(moth.billing.lastPaywallRequest?.knownPaywallRevision, 'pw-0');
    final entry = await settle(tester, cache.load());
    expect(entry!.paywall.revisionId, 'pw-1');
    expect(
      DateTime.now().toUtc().difference(entry.fetchedAt),
      lessThan(const Duration(minutes: 1)),
    );

    await settle(tester, client.dispose());
    await settle(tester, moth.shutdown());
  });
}
