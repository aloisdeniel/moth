package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Provider secret names.
const (
	// ProviderSecretApplePrivateKey is the "Sign in with Apple" .p8 private
	// key, used to mint Apple client secrets.
	ProviderSecretApplePrivateKey = "apple_private_key"
	// ProviderSecretGoogleWebClientSecret is the Google web OAuth client
	// secret, used by the web-redirect code exchange.
	ProviderSecretGoogleWebClientSecret = "google_web_client_secret"
)

// SetProviderSecret upserts one provider secret. The caller passes AES-GCM
// ciphertext produced under the master key; the store never sees plaintext.
func (s *Store) SetProviderSecret(ctx context.Context, projectID, name string, secretEnc []byte, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO project_provider_secrets (project_id, name, secret_enc, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (project_id, name) DO UPDATE SET secret_enc = excluded.secret_enc,
		                                              updated_at = excluded.updated_at`,
		projectID, name, secretEnc, formatTime(now))
	if err != nil {
		return fmt.Errorf("set provider secret: %w", err)
	}
	return nil
}

func (s *Store) GetProviderSecret(ctx context.Context, projectID, name string) ([]byte, error) {
	var secretEnc []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT secret_enc FROM project_provider_secrets WHERE project_id = ? AND name = ?`,
		projectID, name).Scan(&secretEnc)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan provider secret: %w", err)
	}
	return secretEnc, nil
}

// DeleteProviderSecret removes the secret; deleting an absent one is a
// no-op.
func (s *Store) DeleteProviderSecret(ctx context.Context, projectID, name string) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM project_provider_secrets WHERE project_id = ? AND name = ?`,
		projectID, name); err != nil {
		return fmt.Errorf("delete provider secret: %w", err)
	}
	return nil
}
