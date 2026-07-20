ALTER TABLE users ADD COLUMN last_login_at TEXT;

-- Instance-wide settings edited through the admin console (e.g. the SMTP
-- configuration), stored as JSON values per key.
CREATE TABLE instance_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Pending operator invitations; the invite token is stored hashed like
-- every other secret.
CREATE TABLE admin_invites (
    id         TEXT PRIMARY KEY,
    email      TEXT NOT NULL UNIQUE COLLATE NOCASE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
