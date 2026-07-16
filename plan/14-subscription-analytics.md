# Milestone 14 — Subscription Analytics (admin)

## Goal

Answer the monetization questions an app owner actually asks — *how much revenue did this
app make this month? how many active subscribers? are people churning?* — rendered as
dashboards on the project's analytics tab, with zero external analytics dependency. This
extends the milestone-07 pipeline with the subscription event stream from milestone 11; the
headline number, per the request, is **total revenue per month**.

## Deliverables

### Event capture & rollup

- Formalize the `subscription_events` stream emitted by the milestone-11 engine (store
  validation + notification handling): `type` (`trial_started`, `subscription_started`,
  `renewed`, `converted_from_trial`, `cancelled`, `expired`, `refunded`, `grace_entered`,
  `billing_retry`), `user_id`, `product_id`, `store`, `price_amount` + `currency` (store-
  reported, per event), `environment`, `created_at`. Sandbox events are excluded from
  production dashboards by default.
- Extend the milestone-07 nightly (and on-demand) rollup into `subscription_daily_stats` /
  `subscription_monthly_stats`: revenue, active subscribers, new / renewed / churned,
  trials started and converted, per tier and per store — **currency-aware** (aggregated per
  currency; no invented FX). Rollups keep SQLite happy and dashboards read the aggregates,
  never the raw stream.

### Admin dashboards (project analytics tab)

- **Revenue**: total revenue per month (the headline), a monthly revenue time series, and a
  per-tier / per-store breakdown — shown per currency (a primary currency selectable, or a
  per-currency stack), honest that the figure is store-reported gross unless the store
  provides proceeds.
- **Subscribers**: active subscribers (with trend vs the previous period), new vs churned,
  trial-to-paid conversion, and a per-tier share. A small "subscriptions" stat-tile row
  beside the existing auth tiles.
- **Health signal**: an elevated-churn / failed-renewal banner, mirroring the milestone-07
  login-failure banner — a misconfigured store key or a spike in billing failures shows up
  here first.
- Charting reuses the milestone-07 hand-rolled SVG components and the design-system tokens;
  still no external requests from `/admin`. `moth.admin.v1.AnalyticsService` gains subscription
  series + breakdown methods; CSV export of the revenue rollup as an admin-authed HTTP download.

## Key design points

- **Honest money** — dashboards show store-reported list/gross amounts and are explicit that
  net-of-store-commission and tax are not modeled; per-currency, never a fabricated blended
  total. Credibility over a prettier single number.
- **Rollups, not scans** — like milestone 07, dashboards read pre-aggregated tables; the raw
  subscription-event table gets one covering index and the same retention/pruning policy.
- **Privacy-respecting by construction** — revenue and counts only; no per-user financial PII
  beyond what the subscription record already holds, consistent with the milestone-07 stance.
- **Idempotent** — re-running a month's rollup produces identical rows; refunds and
  clawbacks adjust the period they belong to.

## Acceptance criteria

- A seed script generates a few months of synthetic subscription events (starts, renewals,
  trials, cancellations, refunds across tiers/stores/currencies); dashboards render correct
  numbers cross-checked by SQL in tests, including month boundaries in the project timezone.
- Total revenue per month matches the sum of the underlying events per currency; a refund
  reduces the month it applies to.
- The rollup is idempotent (re-running a month is a no-op) and respects retention.
- A project with no subscriptions renders a friendly empty state, not broken charts.
- Subscription-event writes add no measurable latency to the validation/notification path
  (async buffered writer, reusing the milestone-07 machinery).

## Out of scope

Cohort/retention curves, LTV modeling, forecast/MRR projections beyond a current-month figure,
net-revenue (post-commission/tax) accounting, funnel attribution from paywall impressions,
and webhook/export streaming of revenue data — all post-v1.1. This milestone closes the
Monetization phase; the phase ships as a coordinated **v1.1** release (binary + SDK + docs),
following the milestone-10 release pipeline.
