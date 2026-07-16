package store

import (
	"context"
	"fmt"
	"time"
)

// Event types, emitted by the server only (clients are never trusted to
// report events). Canonical milestone-07 names; the milestone-02 stub wrote
// "user.signed_up"/"user.signed_in", which pre-release databases may still
// contain — those legacy rows are ignored by the rollup.
const (
	EventUserSignup             = "user.signup"
	EventUserLogin              = "user.login"
	EventTokenRefresh           = "token.refresh"
	EventUserLoginFailed        = "user.login_failed"
	EventPasswordResetCompleted = "password.reset_completed"
	EventEmailVerified          = "email.verified"
	EventUserDeleted            = "user.deleted"
	EventIdentityLinked         = "identity.linked"
)

// Platforms reported by the SDK via the x-moth-platform request metadata;
// anything else (including empty) is bucketed as "other" by the rollup.
const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
	PlatformWeb     = "web"
)

// Event is one row of the analytics event stream. No PII beyond the user
// id, no IP addresses, no device ids; rows are pruned after the project's
// analytics retention window.
type Event struct {
	ID        string
	ProjectID string
	// UserID is empty for events without a subject (login failures).
	UserID string
	Type   string
	// Provider is the identity provider involved ("" when not applicable).
	Provider string
	// Platform is the SDK-reported platform ("" when none was reported).
	Platform string
	// SDKVersion is the SDK-reported version ("" when none was reported).
	SDKVersion string
	// Metadata is an optional JSON object with event-specific detail, e.g.
	// a bucketed login-failure reason ("" = none).
	Metadata  string
	CreatedAt time.Time
}

// InsertEvent writes a single event.
func (s *Store) InsertEvent(ctx context.Context, e Event) error {
	return s.InsertEvents(ctx, []Event{e})
}

// InsertEvents writes a batch of events in one transaction (the async
// event writer flushes its buffer through this).
func (s *Store) InsertEvents(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("insert events: %w", err)
	}
	for _, e := range events {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO events (id, project_id, user_id, type, provider, platform, sdk_version, metadata, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			e.ID, e.ProjectID, nullString(e.UserID), e.Type, e.Provider,
			e.Platform, e.SDKVersion, nullString(e.Metadata),
			formatTime(e.CreatedAt),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("insert event %s: %w", e.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("insert events: %w", err)
	}
	return nil
}

// ListRecentEvents returns the project's newest events (the admin activity
// feed), newest first.
func (s *Store) ListRecentEvents(ctx context.Context, projectID string, limit int) ([]Event, error) {
	// Ordering by the bare column lets SQLite walk idx_events_project_created
	// backwards and stop at LIMIT; any expression (e.g. an rtrim to fix
	// RFC3339Nano's trimmed fractional zeros) would instead materialize and
	// sort the project's whole retained event set on every feed load. The
	// cost is that within a single second a whole-second "…00Z" sorts after
	// a fractional "…00.5Z" ('Z' > '.') — imperceptible in a human-readable
	// feed. The id tie-breaker is a UUIDv7, itself time-ordered.
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, COALESCE(user_id, ''), type, provider, platform,
		        sdk_version, COALESCE(metadata, ''), created_at
		   FROM events
		  WHERE project_id = ?
		  ORDER BY created_at DESC, id DESC
		  LIMIT ?`,
		projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent events: %w", err)
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var e Event
		var createdAt string
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.UserID, &e.Type, &e.Provider,
			&e.Platform, &e.SDKVersion, &e.Metadata, &createdAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		if e.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, fmt.Errorf("parse event created_at: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list recent events: %w", err)
	}
	return events, nil
}

// DeleteEventsBefore prunes the project's raw events created before cutoff
// (the analytics retention window) and reports how many were removed.
func (s *Store) DeleteEventsBefore(ctx context.Context, projectID string, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM events WHERE project_id = ? AND created_at < ?`,
		projectID, timeBound(cutoff))
	if err != nil {
		return 0, fmt.Errorf("delete events: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete events: %w", err)
	}
	return n, nil
}
