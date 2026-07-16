-- Instance-wide secrets stored as AES-GCM ciphertext under the master key,
-- mirroring project_provider_secrets but not scoped to a project. The SMTP
-- relay password lives here (key = "smtp_password") instead of in plaintext
-- inside the instance_settings JSON. Plaintext never reaches the store.
CREATE TABLE instance_secrets (
    key        TEXT PRIMARY KEY,
    secret_enc BLOB NOT NULL,
    updated_at TEXT NOT NULL
);
