-- Email asserted by the provider when the identity was linked (empty for
-- password identities), and, for Apple identities, the refresh token
-- obtained from the authorization-code exchange — needed to revoke Apple
-- tokens on account deletion — stored AES-GCM-encrypted under the master
-- key.
ALTER TABLE identities ADD COLUMN provider_email TEXT NOT NULL DEFAULT '';
ALTER TABLE identities ADD COLUMN apple_refresh_token_enc BLOB;

-- Per-project provider secrets (Apple .p8 private key, Google web client
-- secret), AES-GCM ciphertext under the master key. Plaintext config lives
-- in projects.settings; only secrets get a row here.
CREATE TABLE project_provider_secrets (
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    secret_enc BLOB NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (project_id, name)
);

-- Single-use artifacts of the web-redirect OAuth fallback: the state value
-- carried through the provider round-trip ("state") and the one-time code
-- the app exchanges for tokens ("code"). Values are stored SHA-256 hashed.
CREATE TABLE oauth_tokens (
    id           TEXT PRIMARY KEY,
    project_id   TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    purpose      TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    provider     TEXT NOT NULL,
    user_id      TEXT REFERENCES users (id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL DEFAULT '',
    payload      TEXT NOT NULL DEFAULT '',
    expires_at   TEXT NOT NULL,
    consumed_at  TEXT,
    created_at   TEXT NOT NULL
);

CREATE INDEX idx_oauth_tokens_expires_at ON oauth_tokens (expires_at);
