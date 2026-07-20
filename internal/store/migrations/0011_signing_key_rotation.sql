-- Graceful signing-key rotation. Alongside the hard reset (status flips
-- straight to "retired"), a rotate moves the previous key to status
-- "grace": it keeps appearing in the project JWKS until not_after so
-- in-flight tokens validate until they expire, then it is pruned. rotated_at
-- records when the key stopped signing; not_after is the grace expiry.
ALTER TABLE project_keys ADD COLUMN rotated_at TEXT;
ALTER TABLE project_keys ADD COLUMN not_after TEXT;

CREATE INDEX idx_project_keys_not_after ON project_keys (not_after);
