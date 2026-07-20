# Milestone 23 — Instance Fabric (enrollment & config propagation)

## Goal

Let one moth deployment span **several geographic instances** while every instance
stays exactly what moth has always been: a single binary with its own SQLite file.
One instance is the **main** — the control plane, where the admin lives and projects
are defined. Any number of **regional** instances enroll with the main and receive
the project catalog so they can serve auth traffic locally. **Nothing is
replicated**: no user rows, no sessions, no events move between instances — only
project *configuration* flows (main → regional) and public signing keys flow back
(regional → main) so JWTs minted anywhere verify everywhere.

This milestone is the plumbing: instance identity, enrollment, the
instance-to-instance RPC surface, config propagation, and the aggregated JWKS.
Routing users to a home instance is milestone 24; the SDKs follow in 25; the
federated admin and global analytics in 26.

## Model

- `instances` (main) — the registry of enrolled regional instances: `id` (UUIDv7),
  `name`, `region` (free-form label, e.g. `eu-west`), `base_url` (public origin the
  SDKs and browsers reach it at), `instance_key_hash` (SHA-256 of the `ik_`
  credential, like `sk_` keys), `status` (`pending` | `active` | `disabled`),
  `enrolled_at`, `last_seen_at`, and the moth `version` it last reported. The main
  itself appears as a built-in row (`is_main`), so "the set of instances" is one
  list everywhere.
- `instance_identity` (regional) — a single-row table: the instance's own id, the
  main's `base_url`, and the `ik_` credential (hashed is impossible here — the
  regional must present it — so it is AES-GCM-encrypted under the regional's own
  master key, like project private keys).
- `project_replicas` (regional) — the propagated project catalog: for each project,
  the config a regional needs to serve auth: slug, publishable key, provider
  credentials, theme/copy/paywall config blobs (`moth.projectconfig.v1`, already
  serialized protos — they propagate byte-for-byte), and a `config_revision` for
  cheap sync. **Not** propagated: secret keys (`sk_` targets one instance's data —
  milestone 26 revisits the developer-backend story), users, or anything user-owned.
- `instance_project_keys` (both sides) — each instance generates its **own** ES256
  keypair per project under its **own** master key; private keys never cross the
  wire. The public halves propagate to every instance, so each one can serve the
  **aggregated JWKS** — the union of all instances' public keys for a project —
  at the existing `/p/{slug}/.well-known/jwks.json`. `kid` carries the instance id
  prefix, so rotation and revocation stay per-instance.

## Deliverables

### Enrollment

- Admin (main): **Instances** page — add an instance (name + region) → the main
  mints an enrollment token (`it_`, single-use, short-lived, shown once, hashed at
  rest like everything else) and shows the join command.
- Regional: `moth serve --join https://main.example.com --enroll-token it_…`
  (equivalently `MOTH_MAIN_URL` / `MOTH_ENROLL_TOKEN`, or a `[main]` section in
  `moth.toml`). On first boot the instance calls `Enroll` on the main, trades the
  enrollment token for its permanent `ik_` credential + instance id, stores them in
  `instance_identity`, and pulls the initial project catalog. Subsequent boots
  reconnect with the `ik_` key; `--join` flags are then ignored.
- `moth instance list|add|disable|remove` in the milestone-08 CLI, driving the same
  admin RPCs.

### Instance-to-instance surface (`moth.cluster.v1`)

A new proto package, served by both roles, authenticated by a new `ik_` interceptor
(same shape as the `sk_` one):

- `Enroll(enroll_token, name?, base_url, version)` → instance id + `ik_` key
  (main-only; the one RPC authenticated by enrollment token instead).
- `SyncProjects(known_revisions)` → changed project configs + tombstones for
  deleted projects (regional pulls; long-poll so config edits land in seconds
  without a push fabric).
- `PublishProjectKeys(project_id, public_keys)` / `ListProjectKeys` — public-key
  exchange feeding the aggregated JWKS on every instance.
- `Heartbeat(version, stats)` — regional → main, feeds `last_seen_at` and the
  Instances page health column; piggybacks the pull-trigger for `SyncProjects`.

`buf` conventions as everywhere: new package `proto/moth/cluster/`, `make proto`,
committed `gen/`.

### Config propagation semantics

- The main is the **only writer** of project configuration; regionals treat
  `project_replicas` as a cache with a revision cursor — the same
  revision-and-merge model the SDKs already use for theme/copy (05/15), one level
  up.
- Propagation is **eventually consistent and read-only**: a regional serves the
  config it has; a stale theme is a cosmetic lag, not a correctness problem.
  Deleting a project on the main tombstones it everywhere (regional refuses new
  auth for it immediately on sync; local user rows are handled in 24's lifecycle).
- SMTP stays **per-instance local config** (`[smtp]` in each instance's
  `moth.toml`) — mail should egress near where it is triggered, and email tokens
  are local rows (24).

### Verification unchanged for developers

- The aggregated JWKS means the developer's backend keeps pointing at **one** JWKS
  URL (any instance serves the same set) and verifies tokens minted by any
  instance offline — per-project isolation is untouched because keys remain
  per-project, now per-project-per-instance.
- gRPC reflection, healthz, and version endpoints on regionals behave exactly like
  a standalone instance; `/healthz` gains a `cluster` block (role, main
  reachability, config sync lag).

## Key design points

- **Configuration flows down, public keys flow up, data never moves.** That single
  sentence is the whole replication policy. Every instance remains an independent,
  individually backed-up SQLite deployment; losing the main degrades enrollment,
  config edits, and the 26 global views — never token minting or verification on
  regionals.
- **Regionals are not read replicas.** They are peers that happen to receive their
  project catalog from the main. There is no failover between instances and no
  shared storage; an instance going down takes its homed users offline until it
  returns (the same availability story a single moth always had, now per region).
- **`ik_` is a first-class key family.** Same hygiene as `pk_`/`sk_`: prefixed,
  random via `internal/token`, hashed at rest on the verifying side, shown once,
  revocable per instance from the Instances page (disable = regional keeps serving
  local users but stops syncing and is refused by the main).
- **Standalone stays the default.** A moth with no `[main]` config and no enrolled
  instances behaves byte-for-byte as today; the fabric is dormant until the first
  enrollment. No flag day, no migration for existing deployments.

## Acceptance criteria

- Enroll a regional against a dev main: enrollment token round-trip, `ik_` stored
  encrypted, instance appears `active` with heartbeat freshness on the Instances
  page; a reused or expired enrollment token is refused.
- Create/edit/delete a project on the main → the regional's `project_replicas`
  converges (long-poll, seconds); the regional serves the project's hosted pages
  and public config with the propagated theme; a deleted project is refused.
- Both instances serve an identical aggregated JWKS for a project; a token signed
  by either instance verifies against the JWKS fetched from the other; rotating
  one instance's project key changes only that instance's `kid`s.
- Secret keys and user rows demonstrably never appear in `moth.cluster.v1`
  traffic (asserted in handler tests, not just by omission).
- A disabled instance's `ik_` calls are refused; re-enabling restores sync without
  re-enrollment. A standalone instance runs with zero cluster overhead.
- `make cross` still passes; the fabric adds no external dependencies.

## Out of scope

User routing and the home-instance model (24), SDK awareness (25), admin fan-out
and analytics accumulation (26), instance-to-instance TLS management (operators
terminate TLS as they already do for public traffic), automatic geo-placement or
latency measurement (the operator names regions; clients pick in 25), and any form
of data replication, failover, or shared storage — permanently, not just this
milestone.
