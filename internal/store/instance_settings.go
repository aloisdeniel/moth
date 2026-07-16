package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Instance setting keys.
const (
	// InstanceSettingSMTP holds the admin-configured SMTP settings as
	// JSON; it overrides the config-file SMTP section.
	InstanceSettingSMTP = "smtp"
)

// GetInstanceSetting returns the value stored under key, or ErrNotFound.
func (s *Store) GetInstanceSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx,
		`SELECT value FROM instance_settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get instance setting %s: %w", key, err)
	}
	return value, nil
}

// SetInstanceSetting stores value under key, replacing any previous value.
func (s *Store) SetInstanceSetting(ctx context.Context, key, value string, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO instance_settings (key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, formatTime(now))
	if err != nil {
		return fmt.Errorf("set instance setting %s: %w", key, err)
	}
	return nil
}

// DeleteInstanceSetting removes key; deleting a missing key is a no-op.
func (s *Store) DeleteInstanceSetting(ctx context.Context, key string) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM instance_settings WHERE key = ?`, key); err != nil {
		return fmt.Errorf("delete instance setting %s: %w", key, err)
	}
	return nil
}
