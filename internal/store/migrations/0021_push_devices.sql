-- Milestone 20: push device registry. One row per (project, user) push
-- registration: which push service reaches the device (`target`), the raw
-- credential the developer's sender needs back (`token`, stored plaintext —
-- it cannot be hashed like sk_ keys), and the client-generated installation
-- id (`device_id`) that lets one physical device replace its own row instead
-- of accumulating. Rows are revoked (revoked_at + revoke_reason), never
-- deleted, so invalidation is auditable and idempotent.
CREATE TABLE push_devices (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    target        TEXT NOT NULL,                    -- 'apns' | 'fcm' | 'webpush'
    token         TEXT NOT NULL,                    -- APNs token | FCM token | serialized Web Push subscription
    device_id     TEXT NOT NULL,                    -- client-generated stable installation id
    permission    TEXT NOT NULL DEFAULT 'unknown',  -- 'granted' | 'provisional' | 'denied' | 'unknown'
    -- Display metadata for the admin view and sender-side locale targeting.
    platform      TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT '',
    os_version    TEXT NOT NULL DEFAULT '',
    app_version   TEXT NOT NULL DEFAULT '',
    locale        TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    last_seen_at  TEXT NOT NULL,                    -- refreshed on every register/heartbeat
    revoked_at    TEXT,
    revoke_reason TEXT                              -- 'signed_out' | 'reported_invalid' | 'stale' | 'replaced' | 'admin'
);

-- Uniqueness holds among *active* rows only (partial indexes): a credential
-- belongs to at most one live registration per project, and one installation
-- has at most one live row per project. Revoked rows keep their token/device
-- for audit without blocking re-registration.
CREATE UNIQUE INDEX idx_push_devices_active_token
    ON push_devices (project_id, target, token) WHERE revoked_at IS NULL;
CREATE UNIQUE INDEX idx_push_devices_active_device
    ON push_devices (project_id, device_id) WHERE revoked_at IS NULL;

-- List queries: a user's devices, and the staleness sweep over active rows.
CREATE INDEX idx_push_devices_project_user ON push_devices (project_id, user_id);
CREATE INDEX idx_push_devices_active_last_seen
    ON push_devices (last_seen_at) WHERE revoked_at IS NULL;
