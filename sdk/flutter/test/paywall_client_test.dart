// Tests for MothClient.getPaywall / getOfferings wire behavior: the
// revision-omission (null) contract the client-side paywall cache relies on,
// and offering-tag pass-through.
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth/src/gen/moth/billing/v1/billing.pb.dart' as pb;

import 'fakes.dart';

void main() {
  late FakeMoth moth;
  late MothClient client;

  setUp(() async {
    moth = await startFakeMoth();
    client = newClient(moth);
    moth.billing.paywall = pb.Paywall(
      revisionId: 'pw-1',
      headline: 'Go Pro',
      subtitle: 'Everything unlocked.',
      benefits: ['A', 'B'],
      offering: 'premium',
      highlightedProductIdentifier: 'yearly',
      layout: pb.PaywallLayout.PAYWALL_LAYOUT_COMPACT,
    );
  });

  tearDown(() async {
    await client.dispose();
    await moth.shutdown();
  });

  test(
    'getPaywall returns the config on a fresh (empty-revision) call',
    () async {
      final paywall = await client.getPaywall();
      expect(paywall, isNotNull);
      expect(paywall!.revisionId, 'pw-1');
      expect(paywall.headline, 'Go Pro');
      expect(paywall.offering, 'premium');
      expect(paywall.layout, MothPaywallLayout.compact);
    },
  );

  test('getPaywall returns null when the known revision still matches so the '
      'cached copy is retained', () async {
    final paywall = await client.getPaywall(knownPaywallRevision: 'pw-1');
    expect(paywall, isNull);
    expect(moth.billing.lastPaywallRequest?.knownPaywallRevision, 'pw-1');
  });

  test('getOfferings forwards the requested offering tag', () async {
    await client.getOfferings(offering: 'premium');
    expect(moth.billing.lastOfferingsRequest?.offering, 'premium');
  });
}
