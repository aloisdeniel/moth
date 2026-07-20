ALTER TABLE projects ADD COLUMN settings TEXT NOT NULL DEFAULT '{}';

CREATE TABLE users (
    id                TEXT PRIMARY KEY,
    project_id        TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    email             TEXT NOT NULL,
    email_verified_at TEXT,
    password_hash     TEXT,
    display_name      TEXT NOT NULL DEFAULT '',
    avatar_url        TEXT NOT NULL DEFAULT '',
    custom_claims     TEXT NOT NULL DEFAULT '{}',
    disabled_at       TEXT,
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL,
    UNIQUE (project_id, email)
);

CREATE TABLE identities (
    id               TEXT PRIMARY KEY,
    project_id       TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id          TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider         TEXT NOT NULL,
    provider_subject TEXT NOT NULL,
    created_at       TEXT NOT NULL,
    UNIQUE (project_id, provider, provider_subject)
);

CREATE INDEX idx_identities_user_id ON identities (user_id);

CREATE TABLE refresh_tokens (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    family_id   TEXT NOT NULL,
    device_info TEXT NOT NULL DEFAULT '',
    expires_at  TEXT NOT NULL,
    rotated_at  TEXT,
    revoked_at  TEXT,
    created_at  TEXT NOT NULL
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens (family_id);

CREATE TABLE email_tokens (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose     TEXT NOT NULL,
    token_hash  TEXT NOT NULL UNIQUE,
    payload     TEXT NOT NULL DEFAULT '',
    expires_at  TEXT NOT NULL,
    consumed_at TEXT,
    created_at  TEXT NOT NULL
);

CREATE INDEX idx_email_tokens_user_id ON email_tokens (user_id);

CREATE TABLE events (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    TEXT,
    type       TEXT NOT NULL,
    provider   TEXT NOT NULL DEFAULT '',
    platform   TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE INDEX idx_events_project_created ON events (project_id, created_at);
