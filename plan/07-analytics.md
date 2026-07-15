# Milestone 07 — Analytics

## Goal

Answer the questions an app owner actually asks — *is my app growing? are logins working? which sign-in method do people use?* — with zero external analytics dependency, rendered as dashboards in the admin console.

## Deliverables

### Event capture

- Formalize the `events` table stubbed in 02: `id`, `project_id`, `type`, `user_id?`, `provider?`, `platform?` (ios/android/web, from SDK user-agent header), `sdk_version?`, `created_at`, `metadata` JSON.
- Server-emitted events (authoritative, no client trust needed): `user.signup`, `user.login`, `token.refresh` (sampled), `user.login_failed` (no user_id, bucketed reason), `password.reset_completed`, `email.verified`, `user.deleted`, `identity.linked`.
- SDK sends only ambient context via gRPC request metadata (`x-moth-platform`, `x-moth-sdk-version`, attached by the milestone-05 client interceptor) — **no separate tracking calls, no PII beyond what auth already has**. moth is not a product-analytics tool.

### Aggregation

- Nightly (and on-demand) rollup job into `daily_stats` (`project_id`, `date`, `signups`, `logins`, `dau`, `failures`, per-provider counts) so dashboards never scan the raw table; raw events retained N days (project setting, default 90) then pruned by the same job.
- In-process scheduler (ticker + jitter); job runs are recorded for observability.
- DAU = distinct users with a login/refresh event that day (approximation documented in the UI tooltip).

### Admin dashboards

- **Project analytics tab**:
  - Stat tiles: total users, new users (7d, with trend vs previous 7d), DAU, login success rate.
  - Time-series charts (range picker 7/30/90d): signups per day, logins per day, DAU.
  - Breakdown donuts/bars: sign-in method share (password/Google/Apple), platform share.
  - Recent activity feed (last 50 events, humanized).
- **Instance overview** (admin home): per-project sparkline cards — this becomes the landing page.
- Charting with a lightweight embedded library (e.g. `recharts` or hand-rolled SVG via the existing stack); still no external requests from `/admin`.
- `moth.admin.v1.AnalyticsService` — `GetStats(project_id, from, to, granularity)` returning pre-aggregated series, `ListRecentEvents` for the activity feed; CSV export as an admin-cookie-authed HTTP download (`/admin/export/stats.csv`), since file downloads are a browser affordance, not an RPC.

## Key design points

- **Privacy-respecting by construction** — no IP storage in events (only coarse hashed IP in rate-limit/audit paths), no device IDs, retention capped and configurable. Sellable as a feature.
- **Rollups keep SQLite happy** — dashboards read `daily_stats` only; the events table gets one covering index `(project_id, created_at)`.
- **Failure visibility** — the login success-rate tile is an ops signal: a misconfigured Apple key after certificate expiry shows up here first. Threshold-based warning banner ("login failures elevated") on the project overview.

## Acceptance criteria

- Seed script generates 90 days of synthetic events; dashboards render correct numbers cross-checked by SQL in tests (rollup correctness is table-driven-tested, including timezone edges — days bucketed in a project-configurable timezone, default UTC).
- Rollup job is idempotent (re-running a day produces identical rows) and prunes expired raw events.
- Dashboard for a project with zero traffic renders a friendly empty state, not broken charts.
- Event writes add no measurable latency to login (async buffered writer, drop-on-overflow with a counter, never blocking auth).

## Out of scope

Funnels, cohorts, retention curves, webhooks/export streaming, alerting beyond the failure banner — post-v1.
