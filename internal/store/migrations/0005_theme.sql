-- Per-project design-system theme (milestone 06): the current theme JSON
-- document (versioned schema, see internal/theme; '' = built-in defaults)
-- and the id of the revision it came from.
ALTER TABLE projects ADD COLUMN theme TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN theme_revision TEXT NOT NULL DEFAULT '';

-- Saved theme revisions, newest-first undo history pruned on every save to
-- the most recent entries (see store.ThemeRevisionKeep).
CREATE TABLE theme_revisions (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    theme      TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_theme_revisions_project ON theme_revisions (project_id, created_at);
