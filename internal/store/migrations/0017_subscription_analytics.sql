-- Milestone 14: subscription revenue analytics. Extends the milestone-07
-- rollup pipeline with the milestone-11 subscription_events stream. Dashboards
-- read ONLY these pre-aggregated tables, never the raw event stream — same
-- stance as daily_stats.

-- Per-project, per-month, per-currency revenue + subscriber rollup. The
-- headline is revenue_micros (store-reported gross, per currency — never a
-- blended/FX total). period is 'YYYY-MM' in the project's rollup timezone. A
-- month can hold several rows, one per currency seen; the PK keeps them
-- coexisting. store_*_revenue_micros split the same net revenue by store for
-- the per-store breakdown. Amounts are net of refunds (refunded events
-- subtract from the month they land in).
CREATE TABLE subscription_monthly_stats (
    project_id                  TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    period                      TEXT NOT NULL,            -- 'YYYY-MM' in the rollup timezone
    currency                    TEXT NOT NULL DEFAULT '', -- ISO 4217; '' when the event carried none
    -- Net store-reported gross: SUM(purchased + renewed) - SUM(refunded).
    revenue_micros              INTEGER NOT NULL DEFAULT 0,
    -- Distinct users with an active-subscription event this month (purchased,
    -- renewed or trial_started) — the same "seen active in the window"
    -- approximation daily_stats.dau uses, not a point-in-time stock.
    active_subscribers          INTEGER NOT NULL DEFAULT 0,
    new_subscribers             INTEGER NOT NULL DEFAULT 0, -- count(subscription.purchased)
    renewals                    INTEGER NOT NULL DEFAULT 0, -- count(subscription.renewed)
    churned                     INTEGER NOT NULL DEFAULT 0, -- count(expired + revoked)
    trials_started              INTEGER NOT NULL DEFAULT 0, -- count(subscription.trial_started)
    trials_converted            INTEGER NOT NULL DEFAULT 0, -- count(subscription.converted)
    store_apple_revenue_micros  INTEGER NOT NULL DEFAULT 0,
    store_google_revenue_micros INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (project_id, period, currency)
);

-- Per-tier (product) slice of the same month/currency rollup — the revenue and
-- subscriber breakdown by product. product_id is '' for events with no mapped
-- moth product. Kept in its own table so the monthly PK stays
-- (project_id, period, currency); dashboards join by (period, currency).
-- NOTE: active_subscribers here is the per-currency distinct-user count and is
-- NOT additive across currencies — the currency-agnostic active count for the
-- headline/series/tier dashboards lives in subscription_period_active below.
CREATE TABLE subscription_tier_stats (
    project_id         TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    period             TEXT NOT NULL,
    currency           TEXT NOT NULL DEFAULT '',
    product_id         TEXT NOT NULL DEFAULT '', -- '' = events with no mapped product
    revenue_micros     INTEGER NOT NULL DEFAULT 0,
    new_subscribers    INTEGER NOT NULL DEFAULT 0,
    active_subscribers INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (project_id, period, currency, product_id)
);

-- Currency-agnostic active-subscriber counts per month. active_subscribers is a
-- COUNT(DISTINCT user_id) over the active-type events in the month and is NOT
-- additive across currencies, so it cannot be summed from the per-currency
-- monthly/tier rows (a user transacting in two currencies would be counted
-- twice). This table stores the correct distinct count once per (period,
-- product_id): product_id = '' is the all-products month total (the headline
-- and series figure), and a non-empty product_id is that tier's distinct count
-- (the per-tier breakdown). The '' total is computed independently of the tier
-- rows — a user active on two tiers is one distinct user in the '' total.
CREATE TABLE subscription_period_active (
    project_id         TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    period             TEXT NOT NULL,
    product_id         TEXT NOT NULL DEFAULT '', -- '' = all products (month total)
    active_subscribers INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (project_id, period, product_id)
);
