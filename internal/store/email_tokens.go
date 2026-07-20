package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Email token purposes.
const (
	// EmailTokenPurposeVerify confirms ownership of the account email.
	EmailTokenPurposeVerify = "verify"
	// EmailTokenPurposeReset authorizes a password reset.
	EmailTokenPurposeReset = "reset"
	// EmailTokenPurposeEmailChange switches the account email to the new
	// address held in Payload once that address is verified.
	EmailTokenPurposeEmailChange = "email_change"
	// EmailTokenPurposeEmailRevert restores the previous address held in
	// Payload (the link mailed to the old address after a change, valid
	// 72 h).
	EmailTokenPurposeEmailRevert = "email_revert"
)

// EmailToken is a single-use token delivered by email.
type EmailToken struct {
	ID         string
	ProjectID  string
	UserID     string
	Purpose    string
	TokenHash  string
	Payload    string // e.g. the pending new email for email_change
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

// Usable reports whether the token can still be consumed at now.
func (et EmailToken) Usable(now time.Time) bool {
	return et.ConsumedAt == nil && now.Before(et.ExpiresAt)
}

func (s *Store) CreateEmailToken(ctx context.Context, et EmailToken) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO email_tokens (id, project_id, user_id, purpose, token_hash,
		                           payload, expires_at, consumed_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		et.ID, et.ProjectID, et.UserID, et.Purpose, et.TokenHash, et.Payload,
		formatTime(et.ExpiresAt), formatNullTime(et.ConsumedAt), formatTime(et.CreatedAt))
	if err != nil {
		return fmt.Errorf("insert email token: %w", err)
	}
	return nil
}

func (s *Store) GetEmailToken(ctx context.Context, projectID, tokenHash string) (EmailToken, error) {
	var et EmailToken
	var expiresAt, createdAt string
	var consumedAt sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, user_id, purpose, token_hash, payload,
		        expires_at, consumed_at, created_at
		 FROM email_tokens WHERE project_id = ? AND token_hash = ?`, projectID, tokenHash,
	).Scan(&et.ID, &et.ProjectID, &et.UserID, &et.Purpose, &et.TokenHash, &et.Payload,
		&expiresAt, &consumedAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return EmailToken{}, ErrNotFound
	}
	if err != nil {
		return EmailToken{}, fmt.Errorf("scan email token: %w", err)
	}
	if et.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return EmailToken{}, fmt.Errorf("parse email token expires_at: %w", err)
	}
	if et.ConsumedAt, err = parseNullTime(consumedAt); err != nil {
		return EmailToken{}, fmt.Errorf("parse email token consumed_at: %w", err)
	}
	if et.CreatedAt, err = parseTime(createdAt); err != nil {
		return EmailToken{}, fmt.Errorf("parse email token created_at: %w", err)
	}
	return et, nil
}

// ConsumeEmailToken marks the token consumed exactly once; a second
// consumption (or a race) returns ErrNotFound.
func (s *Store) ConsumeEmailToken(ctx context.Context, projectID, id string, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE email_tokens SET consumed_at = ?
		 WHERE project_id = ? AND id = ? AND consumed_at IS NULL`,
		formatTime(now), projectID, id)
	if err != nil {
		return fmt.Errorf("consume email token: %w", err)
	}
	return requireRow(res)
}

// DeleteUserEmailTokens drops a user's outstanding tokens of one purpose,
// so only the most recently issued link of that kind stays valid.
func (s *Store) DeleteUserEmailTokens(ctx context.Context, projectID, userID, purpose string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM email_tokens WHERE project_id = ? AND user_id = ? AND purpose = ?`,
		projectID, userID, purpose)
	if err != nil {
		return fmt.Errorf("delete user email tokens: %w", err)
	}
	return nil
}
