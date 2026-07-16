package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// RefreshToken is one opaque rotating refresh token. Tokens minted by a
// sign-in and its subsequent rotations share a family_id; presenting an
// already-rotated token is theft evidence and revokes the whole family.
type RefreshToken struct {
	ID         string
	ProjectID  string
	UserID     string
	TokenHash  string // SHA-256; the plaintext is never stored
	FamilyID   string
	DeviceInfo string
	ExpiresAt  time.Time
	RotatedAt  *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

// Usable reports whether the token can still be redeemed at now.
func (rt RefreshToken) Usable(now time.Time) bool {
	return rt.RotatedAt == nil && rt.RevokedAt == nil && now.Before(rt.ExpiresAt)
}

func (s *Store) CreateRefreshToken(ctx context.Context, rt RefreshToken) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, project_id, user_id, token_hash, family_id,
		                             device_info, expires_at, rotated_at, revoked_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rt.ID, rt.ProjectID, rt.UserID, rt.TokenHash, rt.FamilyID, rt.DeviceInfo,
		formatTime(rt.ExpiresAt), formatNullTime(rt.RotatedAt), formatNullTime(rt.RevokedAt),
		formatTime(rt.CreatedAt))
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (s *Store) GetRefreshToken(ctx context.Context, projectID, tokenHash string) (RefreshToken, error) {
	rt, err := scanRefreshToken(s.db.QueryRowContext(ctx,
		`SELECT id, project_id, user_id, token_hash, family_id, device_info,
		        expires_at, rotated_at, revoked_at, created_at
		 FROM refresh_tokens WHERE project_id = ? AND token_hash = ?`, projectID, tokenHash))
	if errors.Is(err, sql.ErrNoRows) {
		return RefreshToken{}, ErrNotFound
	}
	return rt, err
}

// RotateRefreshToken marks old as rotated and inserts its successor in one
// transaction. It returns ErrNotFound when the old token was already
// rotated or revoked concurrently, so a race can never mint two successors.
func (s *Store) RotateRefreshToken(ctx context.Context, oldID string, rotatedAt time.Time, successor RefreshToken) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin rotate refresh token: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE refresh_tokens SET rotated_at = ?
		 WHERE id = ? AND rotated_at IS NULL AND revoked_at IS NULL`,
		formatTime(rotatedAt), oldID)
	if err != nil {
		return fmt.Errorf("mark refresh token rotated: %w", err)
	}
	if err := requireRow(res); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, project_id, user_id, token_hash, family_id,
		                             device_info, expires_at, rotated_at, revoked_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?)`,
		successor.ID, successor.ProjectID, successor.UserID, successor.TokenHash,
		successor.FamilyID, successor.DeviceInfo, formatTime(successor.ExpiresAt),
		formatTime(successor.CreatedAt)); err != nil {
		return fmt.Errorf("insert rotated refresh token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rotate refresh token: %w", err)
	}
	return nil
}

// RevokeRefreshToken revokes a single token by ID.
func (s *Store) RevokeRefreshToken(ctx context.Context, projectID, id string, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ?
		 WHERE project_id = ? AND id = ? AND revoked_at IS NULL`,
		formatTime(now), projectID, id)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RevokeRefreshTokenFamily revokes every token of one rotation family.
func (s *Store) RevokeRefreshTokenFamily(ctx context.Context, projectID, familyID string, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ?
		 WHERE project_id = ? AND family_id = ? AND revoked_at IS NULL`,
		formatTime(now), projectID, familyID)
	if err != nil {
		return fmt.Errorf("revoke refresh token family: %w", err)
	}
	return nil
}

// ListActiveUserRefreshTokens returns the user's usable refresh tokens at
// now — one per live device session.
func (s *Store) ListActiveUserRefreshTokens(ctx context.Context, projectID, userID string, now time.Time) ([]RefreshToken, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, user_id, token_hash, family_id, device_info,
		        expires_at, rotated_at, revoked_at, created_at
		 FROM refresh_tokens
		 WHERE project_id = ? AND user_id = ?
		   AND rotated_at IS NULL AND revoked_at IS NULL AND expires_at > ?
		 ORDER BY created_at, id`, projectID, userID, formatTime(now))
	if err != nil {
		return nil, fmt.Errorf("list active refresh tokens: %w", err)
	}
	defer rows.Close()

	var tokens []RefreshToken
	for rows.Next() {
		rt, err := scanRefreshToken(rows)
		if err != nil {
			return nil, fmt.Errorf("scan refresh token: %w", err)
		}
		tokens = append(tokens, rt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list active refresh tokens: %w", err)
	}
	return tokens, nil
}

// RevokeUserRefreshTokens revokes every refresh token of a user (all
// devices).
func (s *Store) RevokeUserRefreshTokens(ctx context.Context, projectID, userID string, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ?
		 WHERE project_id = ? AND user_id = ? AND revoked_at IS NULL`,
		formatTime(now), projectID, userID)
	if err != nil {
		return fmt.Errorf("revoke user refresh tokens: %w", err)
	}
	return nil
}

func scanRefreshToken(row rowScanner) (RefreshToken, error) {
	var rt RefreshToken
	var expiresAt, createdAt string
	var rotatedAt, revokedAt sql.NullString
	err := row.Scan(&rt.ID, &rt.ProjectID, &rt.UserID, &rt.TokenHash, &rt.FamilyID,
		&rt.DeviceInfo, &expiresAt, &rotatedAt, &revokedAt, &createdAt)
	if err != nil {
		return RefreshToken{}, err
	}
	if rt.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return RefreshToken{}, fmt.Errorf("parse refresh token expires_at: %w", err)
	}
	if rt.RotatedAt, err = parseNullTime(rotatedAt); err != nil {
		return RefreshToken{}, fmt.Errorf("parse refresh token rotated_at: %w", err)
	}
	if rt.RevokedAt, err = parseNullTime(revokedAt); err != nil {
		return RefreshToken{}, fmt.Errorf("parse refresh token revoked_at: %w", err)
	}
	if rt.CreatedAt, err = parseTime(createdAt); err != nil {
		return RefreshToken{}, fmt.Errorf("parse refresh token created_at: %w", err)
	}
	return rt, nil
}
