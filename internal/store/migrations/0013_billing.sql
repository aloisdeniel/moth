-- Milestone 11: subscriptions & entitlements. moth mirrors the App Store /
-- Play renewal state, derives entitlements from it, and lets operators grant
-- comps. Everything is project-scoped; nothing is shared across projects.

-- Named capabilities an app gates features on (`pro`, `premium`, …). Decoupled
-- from products: which product grants an entitlement can change without an app
-- release. A project may define zero entitlements (then everyone is `none`).
CREATE TABLE entitlements (
    id           TEXT PRIMARY KEY,
    project_id   TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    identifier   TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    UNIQUE (project_id, identifier)
);

-- Subscription tiers per project. apple_product_id / google_product_id are the
-- store SKUs (either may be NULL when a tier ships on one store only). Price
-- columns are display/analytics metadata, not a source of truth. offering +
-- sort_order group products into a single paywall listing (there is no
-- separate offering table: an offering is just the set of products sharing an
-- `offering` tag, ordered by sort_order).
CREATE TABLE products (
    id                        TEXT PRIMARY KEY,
    project_id                TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    identifier                TEXT NOT NULL,
    display_name              TEXT NOT NULL DEFAULT '',
    apple_product_id          TEXT,
    google_product_id         TEXT,
    -- 'weekly' | 'monthly' | 'two_month' | 'three_month' | 'six_month' |
    -- 'yearly' | '' (unknown). Free-form; the store read is authoritative.
    billing_period            TEXT NOT NULL DEFAULT '',
    price_amount_micros       INTEGER NOT NULL DEFAULT 0,
    currency                  TEXT NOT NULL DEFAULT '',
    -- Intro/trial descriptor (display + analytics only).
    trial_period              TEXT NOT NULL DEFAULT '',
    intro_price_amount_micros INTEGER NOT NULL DEFAULT 0,
    intro_period              TEXT NOT NULL DEFAULT '',
    offering                  TEXT NOT NULL DEFAULT '',
    sort_order                INTEGER NOT NULL DEFAULT 0,
    created_at                TEXT NOT NULL,
    updated_at                TEXT NOT NULL,
    UNIQUE (project_id, identifier)
);

-- Which entitlements a product grants while active (many-to-many). Swapping
-- which tier grants `pro` is an edit here, not an app release.
CREATE TABLE product_entitlements (
    product_id     TEXT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    entitlement_id TEXT NOT NULL REFERENCES entitlements (id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, entitlement_id)
);

-- One row per (project, store, store identity) subscription. The store owns the
-- money and the renewal truth; this table is a validating mirror. Absence of a
-- row = the user is on `none`. store_transaction_id is Apple's
-- original_transaction_id or Google's purchase_token — the stable identity a
-- notification/reconcile read looks up; subscription_id holds Google's
-- product-level subscriptionId (empty for Apple). raw_state is the last store
-- payload JSON. product_id is NULL when the SKU is not mapped to a moth product.
CREATE TABLE subscriptions (
    id                   TEXT PRIMARY KEY,
    project_id           TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id              TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    store                TEXT NOT NULL,            -- 'apple' | 'google'
    product_id           TEXT REFERENCES products (id) ON DELETE SET NULL,
    store_transaction_id TEXT NOT NULL,            -- apple original_transaction_id | google purchase_token
    subscription_id      TEXT NOT NULL DEFAULT '', -- google subscriptionId; '' for apple
    -- 'active'|'trialing'|'in_grace_period'|'in_billing_retry'|'paused'|
    -- 'expired'|'revoked'.
    status               TEXT NOT NULL,
    current_period_end   TEXT,
    auto_renew           INTEGER NOT NULL DEFAULT 0,
    environment          TEXT NOT NULL DEFAULT 'production', -- 'sandbox'|'production'
    raw_state            TEXT NOT NULL DEFAULT '',
    created_at           TEXT NOT NULL,
    updated_at           TEXT NOT NULL,
    UNIQUE (project_id, store, store_transaction_id)
);

CREATE INDEX idx_subscriptions_project_user ON subscriptions (project_id, user_id);

-- Manual/promotional grants: comp a reviewer, extend a grace period, grant a
-- promo. Independent of store state; active until expires_at (NULL = forever)
-- unless revoked_at is set. granted_by is the operator credential label, for
-- the milestone-10 audit trail.
CREATE TABLE subscription_grants (
    id             TEXT PRIMARY KEY,
    project_id     TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id        TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    entitlement_id TEXT NOT NULL REFERENCES entitlements (id) ON DELETE CASCADE,
    expires_at     TEXT,
    reason         TEXT NOT NULL DEFAULT '',
    granted_by     TEXT NOT NULL DEFAULT '',
    created_at     TEXT NOT NULL,
    revoked_at     TEXT
);

CREATE INDEX idx_subscription_grants_project_user ON subscription_grants (project_id, user_id);

-- Raw App Store Server Notifications V2 / Play RTDN payloads, kept for
-- idempotency (dedupe on the store notification id) and audit. processed_at is
-- set once the notification has been applied.
CREATE TABLE store_notifications (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    store           TEXT NOT NULL,            -- 'apple' | 'google'
    notification_id TEXT NOT NULL,            -- store-provided id, for idempotency
    type            TEXT NOT NULL DEFAULT '',
    subtype         TEXT NOT NULL DEFAULT '',
    raw_payload     TEXT NOT NULL DEFAULT '',
    processed_at    TEXT,
    created_at      TEXT NOT NULL,
    UNIQUE (project_id, store, notification_id)
);

-- Per-project store API credentials. Secret material (.p8, service-account
-- JSON, webhook shared secrets) is AES-GCM ciphertext under the master key —
-- the store never sees plaintext (encryption happens in the handler layer).
-- Write-only in the admin, like every other secret; reads report presence via
-- has_* indicators derived from blob length. One row per project.
CREATE TABLE billing_credentials (
    project_id                    TEXT PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    -- Apple App Store Server API (In-App Purchase key).
    apple_iap_key_id              TEXT NOT NULL DEFAULT '',
    apple_iap_issuer_id           TEXT NOT NULL DEFAULT '',
    apple_iap_key_enc             BLOB,  -- .p8 private key
    apple_bundle_id               TEXT NOT NULL DEFAULT '',
    apple_app_apple_id            TEXT NOT NULL DEFAULT '',
    apple_notification_secret_enc BLOB,  -- webhook slug shared secret
    -- Google Play Developer API.
    google_service_account_enc    BLOB,  -- service-account JSON
    google_package_name           TEXT NOT NULL DEFAULT '',
    google_pubsub_topic           TEXT NOT NULL DEFAULT '',
    google_rtdn_secret_enc        BLOB,  -- Pub/Sub push shared secret
    created_at                    TEXT NOT NULL,
    updated_at                    TEXT NOT NULL
);

-- Append-only revenue event stream feeding milestone-14 analytics. Written
-- (emitted) by milestone 11 on purchase/renew/expire/refund/grant; rolled up
-- in 14. user_id/product_id are plain TEXT (no FK) so revenue history survives
-- user/product deletion. The covering index serves per-project time-range
-- scans.
CREATE TABLE subscription_events (
    id                  TEXT PRIMARY KEY,
    project_id          TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    type                TEXT NOT NULL,
    user_id             TEXT,
    product_id          TEXT,
    store               TEXT NOT NULL DEFAULT '',
    price_amount_micros INTEGER NOT NULL DEFAULT 0,
    currency            TEXT NOT NULL DEFAULT '',
    environment         TEXT NOT NULL DEFAULT '',
    created_at          TEXT NOT NULL
);

CREATE INDEX idx_subscription_events_project_created ON subscription_events (project_id, created_at);
