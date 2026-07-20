-- Milestone 07: formalize the analytics event stream and its rollups.

-- Ambient SDK context (x-moth-sdk-version request metadata) and optional
-- structured event detail (e.g. a bucketed login-failure reason) as a JSON
-- object, NULL when absent. The existing provider/platform columns keep
-- their milestone-02 shape ('' = unknown), and the covering index
-- idx_events_project_created (project_id, created_at) from migration 0002
-- already serves every analytics query.
ALTER TABLE events ADD COLUMN sdk_version TEXT NOT NULL DEFAULT '';
ALTER TABLE events ADD COLUMN metadata TEXT;

-- Pre-aggregated per-project daily counters, written by the rollup job so
-- dashboards never scan the raw events table. date is the calendar day
-- (YYYY-MM-DD) in the project's rollup timezone. The provider and platform
-- columns break down that day's logins.
CREATE TABLE daily_stats (
    project_id       TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    date             TEXT NOT NULL,
    signups          INTEGER NOT NULL DEFAULT 0,
    logins           INTEGER NOT NULL DEFAULT 0,
    dau              INTEGER NOT NULL DEFAULT 0,
    failures         INTEGER NOT NULL DEFAULT 0,
    logins_password  INTEGER NOT NULL DEFAULT 0,
    logins_google    INTEGER NOT NULL DEFAULT 0,
    logins_apple     INTEGER NOT NULL DEFAULT 0,
    platform_ios     INTEGER NOT NULL DEFAULT 0,
    platform_android INTEGER NOT NULL DEFAULT 0,
    platform_web     INTEGER NOT NULL DEFAULT 0,
    platform_other   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (project_id, date)
);

-- One row per rollup job run (nightly or on-demand), for observability.
-- error is NULL on success.
CREATE TABLE rollup_runs (
    id             TEXT PRIMARY KEY,
    started_at     TEXT NOT NULL,
    finished_at    TEXT NOT NULL,
    days_processed INTEGER NOT NULL DEFAULT 0,
    events_pruned  INTEGER NOT NULL DEFAULT 0,
    error          TEXT
);
