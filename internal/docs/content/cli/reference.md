# cli/reference

moth — authentication for your mobile apps in one binary

This tree (`docs/`) is the documentation content the milestone-09 website publishes; the generated reference below always matches the released binary's command surface.

## moth admin

Manage admin accounts of the local instance

## moth admin create

Create an admin account (or reset its password if the email exists)

```
moth admin create [flags]
```

Flags:

```
      --addr string       listen address (default ":8080")
      --base-url string   public base URL of this instance (default "http://localhost:8080")
      --config string     config file (default moth.toml if present)
      --data-dir string   data directory (database, keys, uploads) (default "./data")
      --email string      admin email address (required)
      --password string   password (generated and printed if omitted)
```

## moth admin token

Manage personal access tokens of the local instance

## moth admin token create

Mint a personal access token for an admin (printed exactly once)

```
moth admin token create [flags]
```

Flags:

```
      --addr string           listen address (default ":8080")
      --base-url string       public base URL of this instance (default "http://localhost:8080")
      --config string         config file (default moth.toml if present)
      --data-dir string       data directory (database, keys, uploads) (default "./data")
      --email string          admin email address (required)
      --expires-in-days int   days until expiry (0: never expires)
      --json                  print machine-readable JSON
      --name string           token label (default "cli")
```

## moth admin token list

List an admin's personal access tokens, newest first

```
moth admin token list [flags]
```

Flags:

```
      --addr string       listen address (default ":8080")
      --base-url string   public base URL of this instance (default "http://localhost:8080")
      --config string     config file (default moth.toml if present)
      --data-dir string   data directory (database, keys, uploads) (default "./data")
      --email string      admin email address (required)
      --json              print machine-readable JSON
```

## moth admin token revoke

Revoke one of an admin's personal access tokens

```
moth admin token revoke [flags]
```

Flags:

```
      --addr string       listen address (default ":8080")
      --base-url string   public base URL of this instance (default "http://localhost:8080")
      --config string     config file (default moth.toml if present)
      --data-dir string   data directory (database, keys, uploads) (default "./data")
      --email string      admin email address (required)
      --id string         token id (required)
      --json              print machine-readable JSON
```

## moth backup

Write a single gzip-compressed tar archive containing an
online-consistent snapshot of the SQLite database (taken with VACUUM INTO, so
it is safe to run against a live server) plus the uploads and key material.

The archive is self-contained; restore it with "moth restore".

Cron example — a daily backup at 03:30, keeping the last 14 days:

  30 3 * * *  moth backup --data-dir /var/lib/moth --to /backups/moth-$(date +\%F).tar.gz && find /backups -name 'moth-*.tar.gz' -mtime +14 -delete

For unattended backups without cron, set backup_dir (and optionally
backup_interval) in the config so "moth serve" writes them itself.

```
moth backup [flags]
```

Flags:

```
      --addr string       listen address (default ":8080")
      --base-url string   public base URL of this instance (default "http://localhost:8080")
      --config string     config file (default moth.toml if present)
      --data-dir string   data directory (database, keys, uploads) (default "./data")
      --to string         archive path (default moth-backup-<timestamp>.tar.gz in the working directory)
```

## moth doctor

Runs the support checklist for "login stopped working": admin API
reachability, base-URL/TLS sanity, health and pub endpoints, SMTP
configuration (with a real test send via --smtp-to), and — with --project
— the project's JWKS plus its Google/Apple provider configuration,
verified against the providers' live endpoints.

```
moth doctor [flags]
```

Flags:

```
      --apple-iap-p8 string             path to the project's App Store Server API In-App-Purchase .p8, enabling the billing store probe
      --apple-key string                path to the project's Sign in with Apple .p8, enabling the Apple token-endpoint dry-run
      --context string                  named context from the CLI config (default: MOTH_CONTEXT, then current-context)
      --google-service-account string   path to the project's Play Developer API service-account JSON, enabling the billing store probe
      --json                            print machine-readable JSON
      --project string                  project slug to check provider config for
      --smtp-to string                  send a real test email to this address
      --stripe-secret-key string        the project's Stripe secret key (sk_/rk_), enabling the Stripe API probe
```

## moth instance

Instance-wide settings of a moth server (remote)

Flags:

```
      --context string   named context from the CLI config (default: MOTH_CONTEXT, then current-context)
      --json             print machine-readable JSON
```

## moth instance get

Show base URL, version and effective SMTP settings

```
moth instance get
```

## moth instance smtp

Configure and test outgoing email

## moth instance smtp clear

Drop the stored SMTP configuration (falls back to the config file, then console)

```
moth instance smtp clear [flags]
```

Flags:

```
      --yes   skip the confirmation prompt
```

## moth instance smtp set

Set stores the SMTP relay configuration in the database. An empty
--password keeps the currently stored one; use 'moth instance smtp clear'
to drop the stored configuration and fall back to the config file.

```
moth instance smtp set [flags]
```

Flags:

```
      --from string       sender address (required)
      --host string       SMTP host (required)
      --password string   SMTP password (empty keeps the stored one)
      --port int32        SMTP port (default 587)
      --username string   SMTP username
```

## moth instance smtp test

Send a probe email through the effective transport

```
moth instance smtp test [flags]
```

Flags:

```
      --to string   recipient address (required)
```

## moth login

Login prompts for a personal access token (create one in the admin
console under Account, or with 'moth admin token create' on the server
host), validates it against the server and saves the pair as a named
context in the CLI config file. The new context becomes the current one.

When stdin is not a terminal the token is read from it directly, so
scripts can pipe it in: echo "$MOTH_PAT" | moth login https://... --name ci

```
moth login <url> [flags]
```

Flags:

```
      --json          print machine-readable JSON
      --name string   context name (default: the server host)
```

## moth project

Manage the projects of a moth server (remote)

Flags:

```
      --context string   named context from the CLI config (default: MOTH_CONTEXT, then current-context)
      --json             print machine-readable JSON
```

## moth project apply

Apply reads a ProjectSpec YAML (see 'moth project dump'), diffs it
against the live project identified by its slug, and applies only what
differs: it creates the project when the slug is free, updates the name
and settings otherwise, and installs (or resets) the theme. Running the
same spec twice reports zero changes.

Unset numeric settings, an empty timezone, absent redirect_schemes and
redirect_origins lists and absent google/apple sections keep the
server's current values,
so partial specs keep unrelated fields untouched. Booleans are the
exception: proto3 cannot distinguish an omitted boolean from false, so a
partial spec that omits e.g. require_email_verification applies it as
false — the plan lists every settings field it is about to change; start
from 'moth project dump' to be safe. Write-only provider secrets present
in the spec are (re)written on every apply.

```
moth project apply -f <spec.yaml> [flags]
```

Flags:

```
  -f, --file string   spec YAML file (required)
      --yes           apply without the confirmation prompt
```

## moth project create

Create a project

```
moth project create <name> [flags]
```

Flags:

```
      --show-secret   print the server-to-server secret key
      --slug string   explicit slug (default: derived from the name)
```

## moth project delete

Delete a project and all its users, keys and tokens

```
moth project delete <slug|id> [flags]
```

Flags:

```
      --yes   skip the confirmation prompt
```

## moth project dump

Dump writes the project's full desired state — name, slug, settings and
theme — as YAML on stdout, the exact document 'moth project apply -f'
consumes. Write-only provider secrets never appear (only their has_*
presence flags); applying a dump keeps the stored secrets.

The slug is optional when the server hosts exactly one project.

```
moth project dump [slug|id]
```

## moth project export

Export writes the project's user accounts — email, display name,
verification/disabled state, custom claims, provider identities and the
encoded password hash — as one JSON document, the input of
'moth project import'.

Password hashes travel with the users (a native argon2id hash, tagged
"argon2id"), so migrating between moth instances keeps everyone signed in
without a reset. Social identities also re-link automatically on the
user's next social sign-in. Project configuration is a separate concern:
see 'moth project dump'.

```
moth project export <slug|id> [flags]
```

Flags:

```
  -o, --output string   output file (default: stdout)
```

## moth project get

Show one project

```
moth project get <slug|id>
```

## moth project import

Import creates the document's users in the target project, restoring
display name, avatar, email verification, disabled state, custom claims,
provider identities and the encoded password hash. A user whose email
already exists in the project is skipped, so re-running an import is safe.

Foreign password hashes (bcrypt, scrypt, argon2, pbkdf2 — tagged per user
in the document's password_algorithm field) are accepted: each is
verified with its original algorithm on the user's first sign-in and then
transparently rehashed to argon2id, so teams can migrate from another
auth system without forcing a password reset.

```
moth project import <slug|id> -f <export.json> [flags]
```

Flags:

```
  -f, --file string   export JSON file (required)
      --yes           skip the confirmation prompt
```

## moth project init

Init walks the milestone-22 wizard in the terminal: platforms, sign-in
(email/password defaults, Google/Apple with credentials entered in-flow or
deferred to 'moth setup google|apple'), monetization (the first entitlement
and its tiers; store credentials always deferred to 'moth setup billing')
and push notifications. Nothing is written until the final confirmation —
abandoning at any prompt creates nothing.

After creation the pk_/sk_ keys are printed exactly once, followed by the
derived setup checklist (whatever was deferred) and a ready-to-commit
'moth project apply' spec of what was just built.

Interactive only: with piped stdin it refuses and points at the scriptable
'moth project create' and 'moth project apply' instead.

```
moth project init [flags]
```

Flags:

```
      --spec-out string   write the emitted 'moth project apply' spec YAML to this file instead of printing it
```

## moth project keys

Manage a project's signing and secret keys

## moth project keys regenerate-secret

Replace the server-to-server secret key (the old one stops working immediately)

```
moth project keys regenerate-secret <slug|id> [flags]
```

Flags:

```
      --show-secret   print the new secret key (required: it is shown exactly once)
      --yes           skip the confirmation prompt
```

## moth project keys reset-signing

Replace the signing keypair (invalidates every issued token; all users must sign in again)

```
moth project keys reset-signing <slug|id> [flags]
```

Flags:

```
      --yes   skip the confirmation prompt
```

## moth project keys rotate

Rotate mints a fresh ES256 signing key that signs new tokens from now
on, while the previous public key stays in the project JWKS for a grace
period so in-flight access tokens keep validating and no user is signed
out. Expired grace keys are pruned automatically.

Unlike 'moth project keys reset-signing', rotate never invalidates
existing sessions. Pass --grace-seconds to override the default grace
(the project's access-token TTL plus a clock-skew margin).

```
moth project keys rotate <slug|id> [flags]
```

Flags:

```
      --grace-seconds int32   seconds the previous key stays in the JWKS (0 = access-token TTL + clock skew)
```

## moth project keys show

Show the active token-signing key and JWKS/issuer values

```
moth project keys show <slug|id>
```

## moth project list

List projects

```
moth project list
```

## moth project update

Update a project (settings are edited declaratively: see 'moth project apply')

```
moth project update <slug|id> [flags]
```

Flags:

```
      --name string   new display name
```

## moth restore

Extract a "moth backup" archive into the data directory, recreating
the database, uploads and keys.

For safety the restore refuses to write into a non-empty data directory unless
--force is given, so an accidental restore cannot clobber a running instance.
Stop "moth serve" before restoring over an existing data directory.

```
moth restore <archive> [flags]
```

Flags:

```
      --addr string       listen address (default ":8080")
      --base-url string   public base URL of this instance (default "http://localhost:8080")
      --config string     config file (default moth.toml if present)
      --data-dir string   data directory (database, keys, uploads) (default "./data")
      --force             overwrite a non-empty data directory
```

## moth serve

Start the moth server

```
moth serve [flags]
```

Flags:

```
      --acme-domain string         comma-separated hostname(s) to obtain a Let's Encrypt certificate for; enables built-in HTTPS on :443 and http-01 on :80
      --addr string                listen address (default ":8080")
      --backup-dir string          directory for scheduled automatic backups (empty disables)
      --backup-interval duration   interval between scheduled backups (default 24h0m0s)
      --base-url string            public base URL of this instance (default "http://localhost:8080")
      --config string              config file (default moth.toml if present)
      --data-dir string            data directory (database, keys, uploads) (default "./data")
      --log-format string          log handler: text or json (default "text")
      --reflection                 enable gRPC server reflection in release builds
      --trusted-proxies string     comma-separated CIDRs/IPs whose X-Forwarded-For is trusted for client-IP rate limiting
```

## moth setup

Configure sign-in providers for a project (automated where APIs exist, guided where they don't)

Flags:

```
      --context string   named context from the CLI config (default: MOTH_CONTEXT, then current-context)
      --json             print machine-readable JSON
```

## moth setup apple

Configures Sign in with Apple end to end, authenticated with an App
Store Connect API key (used in-process only, never stored): verifies or
creates the bundle ID, enables the Sign in with Apple capability, creates
the Sign in with Apple key (Apple serves the .p8 exactly once — it is
immediately stored, encrypted, in the moth project), walks through the
Services ID registration (which has no official API), and dry-runs a
minted client secret against Apple's token endpoint.

Idempotent: re-running diffs the current configuration and only changes
what is needed.

```
moth setup apple [flags]
```

Flags:

```
      --bundle-id string     app bundle ID
      --issuer-id string     App Store Connect API issuer ID
      --key-id string        App Store Connect API key ID
      --p8 string            path to the App Store Connect API .p8 key
      --project string       project slug (required)
      --rotate-key           create a fresh Sign in with Apple key even when one is stored
      --services-id string   Services ID for the web-redirect flow
      --team-id string       Apple Developer Team ID
      --unofficial-api       reserved: drive the unofficial developer-portal API for Services IDs (evaluated and deliberately not implemented)
```

## moth setup billing

Configures a project's store monetization end to end: it stores the
Apple App Store Server API, Google Play Developer API and Stripe
credentials into moth's encrypted billing config, pushes moth's product
catalog into App Store Connect, Google Play and Stripe (automated where
the store APIs allow it, guided with exact values where they don't),
wires the notification/webhook endpoints, and verifies each store is
reachable and authenticated.

The App Store Connect API key (--asc-*) drives the Apple catalog push and
is used in-process only, never stored. The In-App-Purchase key (--apple-
iap-*) and the Google service account are stored encrypted for the
milestone-11 billing engine. The Stripe secret key (--stripe-secret-key,
a restricted key is recommended) is stored encrypted AND drives the
Stripe leg in-process: it provisions a Product + recurring Price per
tier (writing the ids back onto moth's products), creates the webhook
endpoint via the API and stores the returned signing secret. Idempotent:
re-running diffs the current store state and changes only what is needed.

```
moth setup billing [flags]
```

Flags:

```
      --apple-app-apple-id string              the app's numeric App Store id
      --apple-app-id string                    App Store Connect app resource id (for the catalog push)
      --apple-bundle-id string                 app bundle id (enables Apple; blank skips Apple)
      --apple-iap-issuer-id string             App Store Server API issuer id
      --apple-iap-key-id string                App Store Server API In-App-Purchase key id
      --apple-iap-p8 string                    path to the App Store Server API In-App-Purchase .p8 (stored encrypted)
      --apple-notification-secret string       App Store Server Notifications shared secret (stored encrypted)
      --asc-issuer-id string                   App Store Connect API issuer id (catalog push; not stored)
      --asc-key-id string                      App Store Connect API key id (catalog push; not stored)
      --asc-p8 string                          path to the App Store Connect API .p8 (catalog push; not stored)
      --google-cloud-project string            GCP project the RTDN topic lives in (with --google-pubsub-service-account)
      --google-package-name string             Android application id (enables Google; blank skips Google)
      --google-pubsub-service-account string   path to a pubsub-scoped SA JSON to create the RTDN topic/subscription (else guided)
      --google-pubsub-topic string             Cloud Pub/Sub topic for RTDN (projects/<p>/topics/<t> or a bare topic id)
      --google-rtdn-secret string              RTDN push webhook shared secret (stored encrypted)
      --google-service-account string          path to the Play Developer API service-account JSON (stored encrypted)
      --project string                         project slug (required)
      --stripe                                 enable Stripe (prompts for the secret key when --stripe-secret-key is omitted)
      --stripe-secret-key string               Stripe restricted/secret key (sk_/rk_; stored encrypted, also drives the catalog push + webhook creation; prefer omitting it: the command prompts without echo)
      --yes                                    push to the live stores without the confirmation prompt
```

## moth setup google

Configures Sign in with Google end to end: verifies the GCP project
(via gcloud when installed), computes Android signing fingerprints (via
keytool or pasted values), walks through creating the OAuth clients in the
Google console (client creation has no public API), writes the client IDs
into the moth project, and verifies each one against Google's endpoints.

Idempotent: re-running diffs the current configuration and only changes
what is needed.

```
moth setup google [flags]
```

Flags:

```
      --android-client-id string   existing Android OAuth client ID (skips the guided step)
      --android-package string     Android application ID (blank skips Android)
      --android-sha1 string        Android signing certificate SHA-1 fingerprint
      --android-sha256 string      Android signing certificate SHA-256 fingerprint
      --gcp-project string         GCP project ID hosting the OAuth clients
      --ios-bundle-id string       iOS app bundle ID (blank skips iOS)
      --ios-client-id string       existing iOS OAuth client ID (skips the guided step)
      --keystore string            keystore path to compute the fingerprints with keytool
      --keystore-pass string       keystore password (prefer omitting it: the command prompts without echo, keeping the password out of shell history)
      --project string             project slug (required)
      --web-client-id string       existing web OAuth client ID (skips the guided step)
      --web-client-secret string   web OAuth client secret (stored encrypted; blank keeps the stored one)
```

## moth skill

Agent skill teaching coding agents to integrate and administer moth

Flags:

```
      --context string   named context from the CLI config (default: MOTH_CONTEXT, then current-context)
      --json             print machine-readable JSON
```

## moth skill export

Write the embedded moth agent skill to a directory: a SKILL.md with
name/description frontmatter plus references/ teaching an agent to
integrate the moth_auth Flutter SDK into an app and to administer a moth
server through this CLI.

With --project (and a configured context, see 'moth login') every snippet
is interpolated with that project's real values — endpoint, publishable
key, JWKS URL, enabled providers — the agent equivalent of the project's
setup-instructions page. Without --project the skill carries documented
placeholders instead and no server is contacted.

--format claude (default) follows Claude Code conventions; --format
generic writes plain markdown (README.md, no frontmatter) for other agent
frameworks. Exports are idempotent: re-running overwrites the files in
place, so regenerating after a config change is safe.

```
moth skill export [flags]
```

Flags:

```
      --dir string       directory to write the skill into (default ".claude/skills/moth")
      --format string    skill flavor: claude (Claude Code conventions) or generic (plain markdown) (default "claude")
      --project string   project slug (or id) to interpolate real values for; requires a configured context
```

## moth stats

Project analytics (remote)

Flags:

```
      --context string   named context from the CLI config (default: MOTH_CONTEXT, then current-context)
      --json             print machine-readable JSON
```

## moth stats get

Show a project's stat tiles and breakdowns

```
moth stats get [flags]
```

Flags:

```
      --from string      first day, YYYY-MM-DD (default: 30 days ago)
      --project string   project slug or id (required)
      --to string        last day, YYYY-MM-DD (default: today)
```

## moth token

Manage your personal access tokens (remote)

Flags:

```
      --context string   named context from the CLI config (default: MOTH_CONTEXT, then current-context)
      --json             print machine-readable JSON
```

## moth token create

Mint a personal access token (the value is shown exactly once)

```
moth token create <name> [flags]
```

Flags:

```
      --expires-in-days int32   days until expiry (0: never expires)
```

## moth token list

List your personal access tokens, newest first

```
moth token list
```

## moth token revoke

Revoke a personal access token (its next use fails immediately)

```
moth token revoke <id> [flags]
```

Flags:

```
      --yes   skip the confirmation prompt
```

## moth user

Manage a project's end users (remote)

Flags:

```
      --context string   named context from the CLI config (default: MOTH_CONTEXT, then current-context)
      --json             print machine-readable JSON
      --project string   project slug or id (required)
```

## moth user claims

Manage a user's custom JWT claims

## moth user claims set

Replace the user's custom claims with a JSON object ('{}' clears them)

```
moth user claims set <id|email> <json>
```

## moth user create

Create a user (with a password, or with an invite email)

```
moth user create <email> [flags]
```

Flags:

```
      --display-name string   display name
      --invite                send a set-password invite email
      --password string       initial password (omit with --invite to let the user choose one)
      --verified              mark the email address as already verified
```

## moth user delete

Permanently delete a user, their identities and sessions

```
moth user delete <id|email> [flags]
```

Flags:

```
      --yes   skip the confirmation prompt
```

## moth user disable

Block sign-in and revoke the user's sessions

```
moth user disable <id|email>
```

## moth user enable

Re-enable a disabled user

```
moth user enable <id|email>
```

## moth user get

Show one user with identities and active sessions

```
moth user get <id|email>
```

## moth user invite

Create a user and email them a set-password invite

```
moth user invite <email> [flags]
```

Flags:

```
      --display-name string   display name
```

## moth user list

List users, newest first

```
moth user list [flags]
```

Flags:

```
      --page-size int32   users per page (server default 50, max 200)
      --query string      substring filter on email and display name
```

## moth user sessions

Manage a user's device sessions

## moth user sessions revoke

Revoke every session of the user (all devices sign out)

```
moth user sessions revoke <id|email> [flags]
```

Flags:

```
      --yes   skip the confirmation prompt
```

## moth version

Print the moth version

```
moth version
```
