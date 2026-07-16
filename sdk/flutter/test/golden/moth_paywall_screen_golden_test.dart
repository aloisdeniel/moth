// Golden tests for MothPaywallScreen: 3 reference themes x light/dark.
//
// Tagged `golden` and excluded from the default `flutter test` run and CI:
// rasterization differs across platforms/engine builds, so the committed
// images are only stable on the machine flavor that generated them. Run
// (or regenerate with UPDATE=1) via `make sdk-goldens`.
@Tags(['golden'])
library;

import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/billing/v1/billing.pb.dart' as pb;

import '../fakes.dart';
import '../widget_helpers.dart';
import 'moth_login_screen_golden_test.dart' show referenceThemes;

pb.Offering _offering() => pb.Offering(
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
  headline: 'Unlock Premium',
  subtitle: 'Get the full experience with a subscription.',
  benefits: [
    'Unlimited access to every feature',
    'Priority support',
    'New features first',
  ],
  highlightedProductIdentifier: 'yearly',
  layout: pb.PaywallLayout.PAYWALL_LAYOUT_TILES,
  termsUrl: 'https://example.com/terms',
  privacyUrl: 'https://example.com/privacy',
);

void main() {
  late FakeMoth moth;
  late MothClient client;

  for (final MapEntry(key: name, value: theme) in referenceThemes.entries) {
    for (final brightness in Brightness.values) {
      final mode = brightness == Brightness.dark ? 'dark' : 'light';
      testWidgets('MothPaywallScreen $name $mode', (tester) async {
        tester.view.physicalSize = const Size(390, 844);
        tester.view.devicePixelRatio = 1.0;
        addTearDown(tester.view.reset);
        tester.platformDispatcher.platformBrightnessTestValue = brightness;
        addTearDown(tester.platformDispatcher.clearPlatformBrightnessTestValue);

        moth = await runReal(tester, startFakeMoth);
        moth.billing.offering = _offering();
        moth.billing.paywall = _paywall();
        client = newClient(moth);

        await tester.pumpWidget(
          MaterialApp(
            debugShowCheckedModeBanner: false,
            home: MothPaywallScreen(client: client, theme: theme),
          ),
        );
        await pumpUntilFound(tester, find.text('Unlock Premium'));
        await tester.pump();

        await expectLater(
          find.byType(MothPaywallScreen),
          matchesGoldenFile('goldens/paywall_${name}_$mode.png'),
        );

        await settle(tester, client.dispose());
        await settle(tester, moth.shutdown());
      });
    }
  }
}
