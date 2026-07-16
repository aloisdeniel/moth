-- Milestone 12: store-catalog provisioning. moth's product catalog (milestone
-- 11) is the desired state it reconciles into App Store Connect and Google
-- Play. This table is the per-product, per-store reconciliation bookkeeping the
-- sync/diff layer reads and records: what SKU moth last pushed, when, the
-- store-side revision it observed, and whether the two now agree. It never
-- holds money or renewal truth (that stays in `subscriptions`) — only the
-- catalog-sync outcome, so a drift view can be shown without re-hitting the
-- store APIs on every read.
--
-- One row per (product, store). Absence of a row means the product has never
-- been synced to that store (status is reported as `pending`). Rows cascade
-- with the product; the store SKU itself lives on products.apple_product_id /
-- google_product_id (written back by the sync), this table tracks the sync
-- lifecycle around it.
CREATE TABLE product_store_sync (
    product_id       TEXT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    store            TEXT NOT NULL,                     -- 'apple' | 'google'
    -- 'pending' (never synced) | 'in_sync' | 'drift' | 'error'.
    status           TEXT NOT NULL DEFAULT 'pending',
    -- The store SKU moth last reconciled (mirrors the value written back onto
    -- products.{apple,google}_product_id; kept here too so a failed write-back
    -- is still auditable).
    store_product_id TEXT NOT NULL DEFAULT '',
    -- Opaque store-side revision/etag observed at last sync, for cheap drift
    -- detection on the next run.
    revision         TEXT NOT NULL DEFAULT '',
    -- Last sync error (empty when status != 'error').
    error            TEXT NOT NULL DEFAULT '',
    -- When the last successful reconcile ran (NULL while pending).
    synced_at        TEXT,
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL,
    PRIMARY KEY (product_id, store)
);
