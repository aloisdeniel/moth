---
title: Analytics
description: Per-project sign-up and sign-in analytics — the questions an app owner actually asks.
---

Every project records its own event stream and charts it in the admin
console — enough to answer *is my app growing? are logins working? which
sign-in method do people use?* — with **no external analytics dependency
and no tracking calls**. moth is an auth server, not a product-analytics
tool, and the data model reflects that.

## What's measured

The server emits events authoritatively — no client trust needed:
`user.signup`, `user.login`, `token.refresh` (sampled),
`user.login_failed` (bucketed reason, no user id),
`password.reset_completed`, `email.verified`, `user.deleted`,
`identity.linked`. The only thing the SDK contributes is ambient context
attached to normal RPCs as metadata — `platform` (ios/android/web) and SDK
version — never a separate tracking request.

## The dashboard

The project's **Analytics** tab:

- **Stat tiles** — total users, new users (7-day, with trend vs the
  previous 7 days), DAU, and login success rate.
- **Time series** (7 / 30 / 90-day range) — signups per day, logins per
  day, DAU.
- **Breakdowns** — sign-in method share (password / Google / Apple) and
  platform share.
- **Recent activity** — the last 50 events, humanized.

The **login success rate** tile doubles as an ops signal: a misconfigured
Apple key after a certificate expiry shows up here first, and the project
overview raises a warning banner when failures spike. When Apple or Google
sign-in breaks, this tile and [`moth doctor`](../../cli/reference/#moth-doctor)
are where you look.

The instance home shows a sparkline card per project — the whole portfolio
at a glance.

## How the numbers stay cheap

Dashboards never scan the raw event table. A nightly (and on-demand) job
rolls events into `daily_stats`; charts read only that. DAU is the count
of distinct users with a login or refresh that day, bucketed in a
project-configurable timezone (default UTC). Raw events are retained for a
configurable window (90 days by default), then pruned by the same job.
Event writes are async-buffered and never add latency to a login.

## Privacy by construction

- **No IP addresses** stored in events, and **no device IDs.**
- No PII beyond what auth already has.
- Retention capped and configurable; rollups are aggregate.

This is a deliberate stance, not a limitation to work around — see the
[security & threat model](../../security/#privacy-stance).

## From the terminal

```sh
moth stats get --project bird-spotter --from 2026-06-01 --to 2026-06-30 --json
```

returns the same tiles and breakdowns as JSON — see
[`moth stats get`](../../cli/reference/#moth-stats-get). A CSV export of
the time series is available from the dashboard.

## Out of scope

Funnels, cohorts, retention curves, and alerting beyond the failure banner
are intentionally absent in v1 — reach for a dedicated product-analytics
tool if you need them.
