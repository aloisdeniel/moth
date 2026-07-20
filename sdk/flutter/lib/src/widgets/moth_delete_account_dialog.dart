import 'package:flutter/material.dart';

import '../exceptions.dart';
import 'friendly_errors.dart';
import 'moth_scope.dart';

/// Test/target keys for the delete-account dialog.
const mothDeletePasswordFieldKey = Key('moth-delete-password');
const mothDeleteConfirmButtonKey = Key('moth-delete-confirm');
const mothDeleteCancelButtonKey = Key('moth-delete-cancel');

/// Shows the re-authenticate-and-confirm dialog for account deletion (the
/// App Store flow) and performs [MothScope.deleteAccount] on confirm.
///
/// Password users type their password; users who only sign in with
/// Google/Apple leave the field empty — the server enforces its own
/// freshness rule and any typed error is shown inside the dialog. Returns
/// true when the account was deleted (the auth state has then already
/// flipped to signed out).
///
/// [context] must be below a [MothApp] (or [MothScope]).
Future<bool> showMothDeleteAccountDialog(BuildContext context) async {
  final scope = MothScope.of(context);
  final deleted = await showDialog<bool>(
    context: context,
    barrierDismissible: false,
    builder: (_) => _MothDeleteAccountDialog(scope: scope),
  );
  return deleted ?? false;
}

class _MothDeleteAccountDialog extends StatefulWidget {
  const _MothDeleteAccountDialog({required this.scope});

  final MothScope scope;

  @override
  State<_MothDeleteAccountDialog> createState() =>
      _MothDeleteAccountDialogState();
}

class _MothDeleteAccountDialogState extends State<_MothDeleteAccountDialog> {
  final _password = TextEditingController();
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _password.dispose();
    super.dispose();
  }

  Future<void> _delete() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.scope.deleteAccount(password: _password.text);
      if (mounted) Navigator.of(context).pop(true);
    } on MothException catch (err) {
      if (mounted) {
        setState(() {
          _busy = false;
          _error = friendlyMothErrorMessage(err);
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return AlertDialog(
      title: const Text('Delete account?'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'This permanently deletes your account and everything attached '
            'to it. This cannot be undone.',
          ),
          const SizedBox(height: 16),
          TextField(
            key: mothDeletePasswordFieldKey,
            controller: _password,
            enabled: !_busy,
            obscureText: true,
            autofillHints: const [AutofillHints.password],
            onSubmitted: (_) => _delete(),
            decoration: const InputDecoration(
              labelText: 'Password',
              helperText:
                  'Signed in with Google or Apple only? Leave this empty — '
                  'you may be asked to sign in again first.',
              helperMaxLines: 3,
            ),
          ),
          if (_error != null) ...[
            const SizedBox(height: 12),
            Text(_error!, style: TextStyle(color: theme.colorScheme.error)),
          ],
        ],
      ),
      actions: [
        TextButton(
          key: mothDeleteCancelButtonKey,
          onPressed: _busy ? null : () => Navigator.of(context).pop(false),
          child: const Text('Cancel'),
        ),
        FilledButton(
          key: mothDeleteConfirmButtonKey,
          style: FilledButton.styleFrom(
            backgroundColor: theme.colorScheme.error,
            foregroundColor: theme.colorScheme.onError,
          ),
          onPressed: _busy ? null : _delete,
          child: _busy
              ? const SizedBox.square(
                  dimension: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Text('Delete'),
        ),
      ],
    );
  }
}
