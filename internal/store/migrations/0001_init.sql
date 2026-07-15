CREATE TABLE admins (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

CREATE TABLE admin_sessions (
    token_hash TEXT PRIMARY KEY,
    admin_id   TEXT NOT NULL REFERENCES admins (id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE INDEX idx_admin_sessions_admin_id ON admin_sessions (admin_id);

CREATE TABLE projects (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    publishable_key TEXT NOT NULL UNIQUE,
    secret_key_hash TEXT NOT NULL,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE TABLE project_keys (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    kid             TEXT NOT NULL UNIQUE,
    algorithm       TEXT NOT NULL,
    public_key_pem  TEXT NOT NULL,
    private_key_enc BLOB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TEXT NOT NULL
);

CREATE INDEX idx_project_keys_project_id ON project_keys (project_id);
