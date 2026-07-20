-- Append-only audit trail of admin actions and security-relevant events.
-- Rows are never updated or deleted by the application. project_id is a
-- plain column (no foreign key): audit entries must outlive the project
-- they reference. actor_type is one of "cookie", "pat" or "system"; ip is
-- coarse/hashed. before_after holds an optional JSON change summary.
CREATE TABLE audit_log (
    id           TEXT PRIMARY KEY,
    actor_type   TEXT NOT NULL,
    actor_id     TEXT NOT NULL DEFAULT '',
    actor_label  TEXT NOT NULL DEFAULT '',
    action       TEXT NOT NULL,
    target_type  TEXT NOT NULL DEFAULT '',
    target_id    TEXT NOT NULL DEFAULT '',
    project_id   TEXT,
    summary      TEXT NOT NULL DEFAULT '',
    before_after TEXT,
    ip           TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL
);

CREATE INDEX idx_audit_log_created_at ON audit_log (created_at);
CREATE INDEX idx_audit_log_project ON audit_log (project_id, created_at);
CREATE INDEX idx_audit_log_actor ON audit_log (actor_id, created_at);
