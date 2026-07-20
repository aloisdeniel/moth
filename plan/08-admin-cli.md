# Milestone 08 — Admin CLI & One-Command Provider Setup

## Goal

Everything the admin SPA can do, scriptable from a terminal — create and configure projects without opening a browser — plus the headline command: `moth setup google` / `moth setup apple`, which configure the corresponding cloud consoles for a project in a single call. This is the automation surface for CI, dotfiles, and developers who live in the terminal.

## Deliverables

### CLI client mode (same binary)

- The `moth` binary doubles as a remote client, kubectl-style: named contexts (server URL + credential) stored in `~/.config/moth/config.toml`, `--context`/`MOTH_CONTEXT` to switch. No second binary to install.
- **Personal access tokens** (server-side addition): `moth_pat_...` tokens, stored hashed, created/listed/revoked in the admin SPA (and via `moth admin token` locally on the server host); accepted by the existing admin auth interceptor as `authorization: Bearer` metadata alongside cookie sessions. Every PAT-authenticated action carries the token's identity for the audit log (10).
- `moth login <url>` — prompts for a PAT (or mints one via a one-time browser approval flow) and writes the context.
- Command groups mirroring the `moth.admin.v1` services — thin cobra wrappers over the same generated gRPC client, so CLI and SPA can never diverge in capability:
  - `moth project create|list|get|update|delete`, `moth project keys show|reset-signing|regenerate-secret`
  - `moth user list|get|create|invite|disable|delete`, `moth user claims set`, `moth user sessions revoke`
  - `moth stats get`, `moth project export|import`, instance settings incl. SMTP test.
- Scripting ergonomics: `--json` output for every command, meaningful exit codes, `--yes` for non-interactive destructive ops, secrets printed only with an explicit `--show-secret`.
- **Declarative mode**: `moth project apply -f moth.yaml` — idempotent create-or-update of a project's full desired state (settings, providers, theme, legal links); `moth project dump` emits the current state as that file. Gives teams reviewable, versionable auth config (poor-man's Terraform; a real provider stays in the parking lot).

### `moth setup google --project <slug>` (one-command Google configuration)

Inputs: GCP project ID, iOS bundle ID, Android package name + SHA-1/SHA-256 fingerprints (or a keystore path to compute them).

- Automates everything Google's official APIs expose (authenticating via `gcloud` application-default credentials), and falls back to a **guided flow** where OAuth client creation isn't API-accessible: opens the exact console URL, states precisely what to click and paste back, and validates each pasted value's shape before accepting it. First implementation task is a capability spike documenting which steps automate cleanly — the plan commits to the UX (one command, no unexplained console visits), not to over-promised API coverage.
- Writes the resulting iOS/Android/web client IDs into the moth project's provider config (same RPC the admin SPA uses).
- Ends with verification: client IDs resolve against Google's discovery/tokeninfo endpoints, redirect URI registered for the web flow matches moth's, and the project's `GetProjectConfig` now advertises Google. Prints a colored checklist.

### `moth setup apple --project <slug>` (one-command Apple configuration)

Inputs: an App Store Connect API key (issuer ID, key ID, `.p8` path), app bundle ID.

- Via the official App Store Connect API: verify/create the Bundle ID, enable the Sign in with Apple capability, create the Sign in with Apple key (downloads the `.p8` — Apple allows this exactly once; the CLI immediately uploads it into moth's encrypted provider config).
- Services ID + return-URL registration has no official API (fastlane's spaceship precedent uses the unofficial portal API): default to the guided flow with exact values to paste (Services ID string, return URL), with an `--unofficial-api` opt-in flag evaluated during the spike.
- Ends with verification: moth generates a client secret from the stored key and performs a dry-run against Apple's token endpoint, and checks the web-flow return URL. Prints the checklist.

### Shared behavior

- Both setup commands are **idempotent and re-runnable**: they diff current console/moth state against desired state and only change what's needed — safe to run after a certificate expiry or key rotation.
- `moth doctor --project <slug>` — full config health check usable any time: JWKS reachable, SMTP test send, provider configs verified (Apple key validity/expiry, Google client IDs resolve), pub endpoint serves the SDK, base-URL/TLS sanity. The support answer to "login stopped working".

### Agent skill (`moth skill`)

Coding agents (Claude Code and anything else that reads the Agent Skills format) are increasingly the ones wiring auth into apps — give them a first-class artifact instead of hoping they scrape docs:

- `moth skill export [--project <slug>] [--dir .claude/skills/moth]` — writes a skill directory (`SKILL.md` with name/description frontmatter + `references/`) teaching an agent both halves of moth:
  - **Integrating moth into a specific app**: add the served `moth_auth` dependency, wrap `MothApp`, read `MothScope`, call the developer's backend with the token, verify JWTs server-side (JWKS + introspection), platform setup steps for Google/Apple. With `--project` and a configured context, the skill is **interpolated with that project's real values** (endpoint, publishable key, JWKS URL, enabled providers) — the agent equivalent of the milestone-03 setup-instructions page.
  - **Administering moth via the CLI**: the command groups, `--json` output contracts for programmatic parsing, `moth project apply` for declarative changes, `moth setup google|apple` orchestration, `moth doctor` for diagnosis — written so an agent can operate an instance end to end without a browser.
- Content is **assembled from the milestone-09 docs tree and the generated CLI reference at release time** and embedded in the binary — the skill can never drift from the docs or the actual command surface. Written agent-first: terse, exact commands, expected outputs, failure modes and their fixes.
- `moth skill export --format claude|generic` covers Claude Code conventions by default with a plain-markdown fallback for other frameworks.

### Docs & CI

- CLI reference generated from cobra command definitions into the docs content tree (published by the milestone-09 website and embedded at `/docs` in 10).
- Setup-instructions page (03) gains the CLI path: `moth login ... && moth setup google ... && moth setup apple ...` next to the manual walkthrough.
- Integration tests run CLI commands against a spawned server; setup commands tested against recorded API fixtures plus a manual checklist with real Google/Apple accounts.

## Key design points

- **One domain layer, three faces** — SPA, CLI, and `moth.server.v1` all sit on the same services; the CLI adds no new business logic, only the PAT credential type and the provider-console orchestration.
- **Honest automation** — where Apple/Google offer no API, the command never silently degrades: it says what it automated, what it needs pasted, and verifies the result. The contract is "one command and it's *verified* working", not "no human input ever".
- **Store nothing platform-side** — ASC keys and gcloud credentials are used in-process for the setup call and never persisted by moth or the CLI config.

## Acceptance criteria

- From a clean terminal against a fresh instance: `moth login` → `moth project create demo` → `moth setup google` → `moth setup apple` → the milestone-05 example app signs in with email, Google, and Apple — any remaining manual console steps enumerated by the CLI itself, none discovered by surprise.
- `moth project apply -f` twice in a row: second run reports zero changes (idempotency test).
- Revoking a PAT in the SPA immediately fails the CLI's next call; PAT actions appear attributed in the audit log once 10 lands.
- `moth doctor` detects and clearly reports: bad SMTP credentials, expired Apple key, deleted Google client ID.
- Every command passes `--json` golden-output tests; CLI docs build in CI.
- Agent-skill validation: the exported skill passes format linting, and a coding agent given only the skill and a fresh Flutter project completes the integration task (add dependency → wrap app → sign in against a dev instance) in a scripted evaluation run in CI (plus a manual admin-task spot check).

## Out of scope

Terraform/Pulumi provider (parking lot), Android/iOS project file modification (the SDK setup docs cover it), Google Play Console / App Store Connect app-record creation — moth configures auth, not app distribution.
