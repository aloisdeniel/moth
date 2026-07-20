-- Personal access tokens (moth_pat_...) authenticate the moth CLI and
-- scripts as an admin over the admin API; the plaintext is stored hashed
-- like every other secret. NULL expires_at means the token never expires;
-- revoked tokens keep their row (revoked_at set) so they stay listable
-- until pruned.
CREATE TABLE personal_access_tokens (
    id           TEXT PRIMARY KEY,
    admin_id     TEXT NOT NULL REFERENCES admins (id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    created_at   TEXT NOT NULL,
    last_used_at TEXT,
    expires_at   TEXT,
    revoked_at   TEXT
);

CREATE INDEX personal_access_tokens_admin_id ON personal_access_tokens (admin_id);
