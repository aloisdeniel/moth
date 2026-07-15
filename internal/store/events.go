package store

import (
	"context"
	"fmt"
	"time"
)

// Event types written by milestone 02; real analytics land in milestone 07.
const (
	EventUserSignedUp = "user.signed_up"
	EventUserSignedIn = "user.signed_in"
	EventUserDeleted  = "user.deleted"
)

// Event is one row of the analytics event stream (stub for now).
type Event struct {
	ID        string
	ProjectID string
	UserID    string
	Type      string
	Provider  string
	Platform  string
	CreatedAt time.Time
}

func (s *Store) InsertEvent(ctx context.Context, e Event) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO events (id, project_id, user_id, type, provider, platform, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.ProjectID, nullString(e.UserID), e.Type, e.Provider, e.Platform,
		formatTime(e.CreatedAt))
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}
