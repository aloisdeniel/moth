package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Instance secret keys.
const (
	// InstanceSecretSMTPPassword is the outgoing-relay password, stored as
	// AES-GCM ciphertext under the master key instead of in plaintext inside
	// the instance_settings SMTP JSON.
	InstanceSecretSMTPPassword = "smtp_password"
)

// SetInstanceSecret upserts one instance secret. The caller passes AES-GCM
// ciphertext produced under the master key; the store never sees plaintext.
func (s *Store) SetInstanceSecret(ctx context.Context, key string, secretEnc []byte, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO instance_secrets (key, secret_enc, updated_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT (key) DO UPDATE SET secret_enc = excluded.secret_enc,
		                                 updated_at = excluded.updated_at`,
		key, secretEnc, formatTime(now))
	if err != nil {
		return fmt.Errorf("set instance secret %s: %w", key, err)
	}
	return nil
}

// GetInstanceSecret returns the ciphertext stored under key, or ErrNotFound.
func (s *Store) GetInstanceSecret(ctx context.Context, key string) ([]byte, error) {
	var secretEnc []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT secret_enc FROM instance_secrets WHERE key = ?`, key).Scan(&secretEnc)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get instance secret %s: %w", key, err)
	}
	return secretEnc, nil
}

// DeleteInstanceSecret removes key; deleting an absent one is a no-op.
func (s *Store) DeleteInstanceSecret(ctx context.Context, key string) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM instance_secrets WHERE key = ?`, key); err != nil {
		return fmt.Errorf("delete instance secret %s: %w", key, err)
	}
	return nil
}
