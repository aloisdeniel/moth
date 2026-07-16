-- Per-project paywall configuration (milestone 13): the current paywall JSON
-- document (versioned schema, see internal/paywall; '' = built-in defaults)
-- and the id of the revision it came from. The paywall owns no design tokens
-- of its own — colors/typography inherit from the theme (milestone 06); this
-- config only carries copy, benefit bullets, offering/tier selection, layout
-- variant and legal links.
ALTER TABLE projects ADD COLUMN paywall TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN paywall_revision TEXT NOT NULL DEFAULT '';

-- Saved paywall revisions, newest-first undo history pruned on every save to
-- the most recent entries (see store.PaywallRevisionKeep). Mirrors
-- theme_revisions exactly.
CREATE TABLE paywall_revisions (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    paywall    TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_paywall_revisions_project ON paywall_revisions (project_id, created_at);
