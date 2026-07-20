-- Per-project copy customization (milestone 15): the current copy-override JSON
-- document and the id of the revision it came from. The override document is a
-- BCP-47 locale tag → message key → string map; '' means the project renders
-- the bundled catalog defaults everywhere (internal/i18n). Overrides are
-- additive on top of the bundled catalog, resolved bundled-default →
-- project-default-locale-override → project-locale-override at read time.
-- Mirrors the theme (0005) and paywall (0016) storage exactly.
ALTER TABLE projects ADD COLUMN copy TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN copy_revision TEXT NOT NULL DEFAULT '';

-- Saved copy revisions, newest-first undo history pruned on every save to the
-- most recent entries (see store.CopyRevisionKeep). Mirrors theme_revisions
-- and paywall_revisions exactly.
CREATE TABLE copy_revisions (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    copy       TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_copy_revisions_project ON copy_revisions (project_id, created_at);
