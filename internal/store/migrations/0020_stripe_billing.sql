-- Milestone 17: Stripe billing (web). Stripe joins Apple and Google as a third
-- `store` value on the milestone-11 engine — same subscriptions table, same
-- status enum, same notifications/events streams. This migration adds the
-- Stripe SKU columns on products, the per-project Stripe credentials, the lazy
-- (project, user) → Stripe customer mapping, and the Stripe slice of the
-- milestone-14 per-store revenue split.

-- The tier's Stripe recurring Price ("price_...") and the linked Stripe
-- Product ("prod_...", written back by provisioning). Nullable like the other
-- two store SKUs: a tier may ship on any subset of the three stores.
ALTER TABLE products
    ADD COLUMN stripe_price_id TEXT;
ALTER TABLE products
    ADD COLUMN stripe_product_id TEXT;

-- Per-project Stripe credentials: the restricted secret key and the webhook
-- signing secret, AES-GCM-encrypted under the master key exactly like the
-- Apple key and Google service account. stripe_webhook_endpoint_id records the
-- webhook endpoint ("we_...") moth created via the API, so `moth setup
-- billing` stays idempotent (mirroring apple_notification_url); empty means
-- none registered.
ALTER TABLE billing_credentials
    ADD COLUMN stripe_secret_key_enc BLOB;
ALTER TABLE billing_credentials
    ADD COLUMN stripe_webhook_secret_enc BLOB;
ALTER TABLE billing_credentials
    ADD COLUMN stripe_webhook_endpoint_id TEXT NOT NULL DEFAULT '';

-- Per-(project, user) mapping to the Stripe customer, created lazily on first
-- checkout so every moth user has at most one Stripe customer per project. The
-- reverse index lets webhook processing attribute a Stripe customer back to
-- the moth user.
CREATE TABLE stripe_customers (
    project_id         TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id            TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    stripe_customer_id TEXT NOT NULL,
    created_at         TEXT NOT NULL,
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX idx_stripe_customers_stripe_id
    ON stripe_customers (project_id, stripe_customer_id);

-- Stripe slice of the milestone-14 per-store revenue split, beside
-- store_apple_revenue_micros / store_google_revenue_micros.
ALTER TABLE subscription_monthly_stats
    ADD COLUMN store_stripe_revenue_micros INTEGER NOT NULL DEFAULT 0;
