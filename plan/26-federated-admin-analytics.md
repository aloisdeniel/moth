# Milestone 26 — Federated Admin & Global Analytics

## Goal

Make the main instance the **single pane of glass**. Operators already do
everything from the main's admin SPA; this milestone keeps that true when the data
lives elsewhere: user lists, detail views, and management actions reach through to
the home instance transparently, and the analytics dashboards (07/14) accumulate
every instance's numbers into one global overview with a per-instance breakdown.
Reads and actions are **live proxying**; analytics are **pulled aggregates** —
still no replication of user data or raw events.

## Model

- `analytics_rollups` (main) — pulled, pre-aggregated series per
  `(instance_id, project_id, metric, bucket)`: the same daily buckets the
  milestone-07 dashboards already compute locally (signups, logins by provider,
  active users, platform split) plus the milestone-14 revenue series (MRR,
  subscriber counts by tier, store split). Rollups are idempotent upserts keyed by
  bucket — re-pulling a window is always safe. Raw `events` rows never leave their
  instance.
- `moth.cluster.v1` additions:
  - `AdminProxy`-grade RPCs are **not** added — instead the main calls the
    regional's existing `moth.admin.v1` handlers over a cluster-authenticated
    channel (`ik_` interceptor accepts the main's identity as an operator-grade
    principal, with the acting admin's identity forwarded for the audit log).
  - `PullAnalytics(project_id?, metrics, from_bucket)` → rollup rows; cursor-based
    so the main's scheduled pull (piggybacked on the 23 heartbeat cadence) stays
    incremental.

## Deliverables

### Federated user management (admin SPA on the main)

- **Project user list** becomes cluster-wide: the main fans out the list/search
  RPC to all active instances, merges and paginates; each row shows a **home
  instance** badge (region label). Search-by-email short-circuits through the
  locator (24) to a single instance query.
- **User detail** — profile, identities, sessions, devices (20), subscription
  (11), grants — is proxied live from the home instance, lag-free and
  storage-free on the main. Every management action (update claims, revoke
  sessions, revoke device, grant entitlement, delete user) executes on the home
  instance with the acting admin recorded in the home instance's audit log (10)
  as `admin@main via cluster`.
- Instance outage degrades visibly, not silently: the user list shows per-instance
  fetch status ("eu-west unreachable — 3,201 users not shown"), and a proxied
  detail view shows a clear unavailable state instead of stale data — there is no
  stale data to show, which is the point of proxying.

### Global analytics

- Project **Analytics** tab (07) and **Monetization** tab (14) on the main render
  the **sum of rollups across instances** by default, with an instance filter and
  a stacked per-instance breakdown for every existing chart. The instance
  dimension is a new group-by on charts that already exist — no new chart types.
- A new **cluster overview** on the Instances page (23): per-instance totals
  (users homed, signups this week, active users, revenue), sync freshness (last
  successful pull per instance), and config propagation lag.
- Rollup pulls are incremental and self-healing: a regional that was unreachable
  is back-filled from its last pulled bucket on reconnection; totals are marked
  "as of" the oldest instance cursor so a lagging instance can never silently
  understate a global number.

### Cluster-wide developer-backend surface

- The main's `moth.server.v1` closes the milestone-24 interim: `Introspect` stays
  local (JWKS is global), and user-targeted calls (`GetUser`, entitlement checks,
  `ListUserPushDevices`, push-device fan-out reads) are routed via the locator and
  proxied to the home instance — so a developer backend can target the main for
  everything, at one proxy hop of cost, or read `mih` from the JWT and go direct
  for latency-sensitive paths. Both patterns are documented; project secret keys
  are honored cluster-wide (propagated as hashes over the 23 fabric — reversing
  the milestone-23 exclusion now that the fan-out surface exists to justify it).

### CLI

- `moth instance status` — the cluster overview in the terminal (per 08 style).
- `moth user find <email>` gains the home-instance column; user-targeted admin
  CLI commands route via the main exactly like the SPA.

## Key design points

- **Proxy reads, pull aggregates.** Live views must be current (proxy — correct
  by construction, no sync problem); dashboards must be fast and available
  (pulled rollups — bounded, idempotent, cheap to backfill). Choosing per surface
  keeps both honest and keeps raw data where it lives.
- **The main is an operator, not a superuser backdoor.** Cluster-authenticated
  admin calls carry the acting admin identity end-to-end into the home instance's
  audit log; a regional's log is complete on its own, main included.
- **Aggregation-by-addition.** Rollups are the same aggregates 07 already
  computes for one instance; the global view is a sum over an added instance
  dimension. No cross-instance joins, no distinct-user set unions across
  instances — a user exists on exactly one instance (24), so instance sums *are*
  the global truth, which is the quiet payoff of no-replication homing.
- **Truthful degradation.** Every federated surface states what it could not
  reach rather than approximating. An operator must never mistake a partial
  number for a global one.

## Acceptance criteria

- Two-instance cluster, users homed on both: the main's user list shows all users
  with correct home badges; searching a B-homed email opens the full detail view;
  revoking one of its sessions takes effect on B (verified by the user's next
  refresh failing) and lands in B's audit log with the acting admin's identity.
- Analytics: signups generated on both instances appear in the global chart as
  the exact sum, split correctly in the per-instance breakdown; revenue (14)
  accumulates identically; re-pulling a window produces no double counting.
- Unreachable instance: user list renders with the per-instance warning and
  correct partial counts; the detail view of an affected user shows unavailable;
  global charts carry the "as of" marker; everything heals and backfills when the
  instance returns.
- Developer backend pointed at the main: `GetUser` for a B-homed user succeeds
  via proxy with the project's secret key; the same key works directly against B.
- Standalone instance: dashboards, user list, and `moth.server.v1` behave exactly
  as pre-cluster; no rollup or fan-out code paths run.

## Out of scope

Cross-instance data export/user migration tooling, real-time streaming analytics
(pull cadence is minutes, matching 07's dashboard granularity), alerting on
instance health (operators plug `/healthz` into their monitoring), and
cluster-wide rate-limit coordination (limits stay per-instance where the traffic
lands).
