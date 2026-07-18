import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moth_auth/moth_auth.dart';
import 'package:moth_auth_example/home_screen.dart';
import 'package:moth_auth_example/main.dart';

void main() {
  testWidgets('explains the missing publishable key on a bare run', (
    tester,
  ) async {
    // Tests run without --dart-define, so the app renders the setup hint
    // instead of MothApp.
    await tester.pumpWidget(const ExampleApp());
    expect(find.text('No publishable key'), findsOneWidget);
    expect(find.textContaining('MOTH_PUBLISHABLE_KEY'), findsOneWidget);
  });

  testWidgets('home screen renders the MothScope user', (tester) async {
    final client = MothClient(
      MothConfig(
        endpoint: Uri.parse('http://localhost:1'), // never dialed
        publishableKey: 'pk_test',
      ),
      tokenStore: InMemoryTokenStore(),
    );
    addTearDown(client.dispose);
    const user = MothUser(
      id: 'user-1',
      email: 'jane@example.com',
      emailVerified: true,
      displayName: 'Jane',
      claims: {'role': 'admin'},
    );
    await tester.pumpWidget(
      MothScope(
        client: client,
        state: const MothSignedIn(user),
        child: const MaterialApp(home: HomeScreen()),
      ),
    );
    expect(find.text('jane@example.com'), findsOneWidget);
    expect(find.text('verified'), findsOneWidget);
    expect(find.text('role: admin'), findsOneWidget);
    // The subscription card reads the (free by default) entitlement state.
    expect(find.text('Free tier'), findsOneWidget);
    // No push controller in this bare scope, so the push card reports the
    // unavailable state instead of a toggle.
    await tester.scrollUntilVisible(
      find.textContaining('Unavailable'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    expect(find.textContaining('push is disabled'), findsOneWidget);
    // "Call my backend" sits below the fold of the lazy ListView.
    await tester.scrollUntilVisible(
      find.text('Call my backend'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    expect(find.text('Call my backend'), findsOneWidget);
    // "Delete account" sits below the fold of the lazy ListView.
    await tester.scrollUntilVisible(
      find.text('Delete account'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    expect(find.text('Delete account'), findsOneWidget);
  });
}
