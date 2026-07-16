import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:moth_auth/moth_auth.dart';

import 'main.dart' show apiBase, resolveLocalhost;

/// The signed-in surface of the example: shows the [MothScope] user (email,
/// verified badge, custom claims), calls the sample backend with the
/// authenticated http client, and exposes sign-out / delete-account.
class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  String? _backendResult;
  bool _callingBackend = false;

  Future<void> _callBackend(MothScope scope) async {
    setState(() {
      _callingBackend = true;
      _backendResult = null;
    });
    // authenticatedClient attaches a fresh (auto-refreshed) moth JWT as
    // `Authorization: Bearer ...`; the backend verifies it against the
    // project JWKS (see scripts/example_backend in the moth repository).
    final api = authenticatedClient(scope.client);
    String result;
    try {
      final url = resolveLocalhost(Uri.parse('$apiBase/api/hello'));
      final resp = await api.get(url);
      result = resp.statusCode == 200
          ? const JsonEncoder.withIndent('  ').convert(jsonDecode(resp.body))
          : 'HTTP ${resp.statusCode}: ${resp.body}';
    } on Exception catch (err) {
      result =
          '$err\n\nIs the sample backend running?\n'
          'go run ./scripts/example_backend --issuer <moth>/p/<slug>';
    } finally {
      api.close();
    }
    if (!mounted) return;
    setState(() {
      _callingBackend = false;
      _backendResult = result;
    });
  }

  @override
  Widget build(BuildContext context) {
    final scope = MothScope.of(context);
    final user = scope.user;
    if (user == null) {
      // Only visible for the frame in which sign-out flips the gate.
      return const Scaffold(body: SizedBox.shrink());
    }
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('moth example'),
        actions: [
          IconButton(
            tooltip: 'Sign out',
            icon: const Icon(Icons.logout),
            onPressed: scope.signOut,
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Signed in as', style: theme.textTheme.labelMedium),
                  const SizedBox(height: 4),
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          user.email,
                          style: theme.textTheme.titleMedium,
                        ),
                      ),
                      _VerifiedBadge(verified: user.emailVerified),
                    ],
                  ),
                  if (user.displayName != null) Text(user.displayName!),
                  const SizedBox(height: 8),
                  Text('User ID: ${user.id}', style: theme.textTheme.bodySmall),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Custom claims', style: theme.textTheme.labelMedium),
                  const SizedBox(height: 4),
                  if (user.claims.isEmpty)
                    const Text('none — set some in the moth admin')
                  else
                    for (final entry in user.claims.entries)
                      Text('${entry.key}: ${entry.value}'),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          FilledButton.icon(
            icon: const Icon(Icons.cloud),
            label: const Text('Call my backend'),
            onPressed: _callingBackend ? null : () => _callBackend(scope),
          ),
          if (_callingBackend)
            const Padding(
              padding: EdgeInsets.all(16),
              child: Center(child: CircularProgressIndicator()),
            ),
          if (_backendResult != null)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Text(
                    _backendResult!,
                    style: theme.textTheme.bodySmall!.copyWith(
                      fontFamily: 'monospace',
                    ),
                  ),
                ),
              ),
            ),
          const SizedBox(height: 24),
          OutlinedButton.icon(
            icon: const Icon(Icons.refresh),
            label: const Text('Refresh profile'),
            onPressed: scope.refreshUser,
          ),
          const SizedBox(height: 8),
          OutlinedButton.icon(
            style: OutlinedButton.styleFrom(
              foregroundColor: theme.colorScheme.error,
            ),
            icon: const Icon(Icons.delete_forever),
            label: const Text('Delete account'),
            onPressed: () => showMothDeleteAccountDialog(context),
          ),
        ],
      ),
    );
  }
}

class _VerifiedBadge extends StatelessWidget {
  const _VerifiedBadge({required this.verified});

  final bool verified;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Chip(
      visualDensity: VisualDensity.compact,
      avatar: Icon(
        verified ? Icons.verified : Icons.error_outline,
        size: 18,
        color: verified ? scheme.primary : scheme.error,
      ),
      label: Text(verified ? 'verified' : 'unverified'),
    );
  }
}
