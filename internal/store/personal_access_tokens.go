package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// PersonalAccessToken is one admin CLI/API credential (moth_pat_...). The
// plaintext is never stored; only its SHA-256 hash.
type PersonalAccessToken struct {
	ID        string
	AdminID   string
	Name      string
	TokenHash string
	CreatedAt time.Time
	// LastUsedAt is when the token last authenticated a request; nil when
	// never used.
	LastUsedAt *time.Time
	// ExpiresAt nil means the token never expires.
	ExpiresAt *time.Time
	// RevokedAt is set once the token has been revoked; the row is kept so
	// the token stays listable until pruned.
	RevokedAt *time.Time
}

// Usable reports whether the token authenticates requests at now.
func (t PersonalAccessToken) Usable(now time.Time) bool {
	return t.RevokedAt == nil && (t.ExpiresAt == nil || now.Before(*t.ExpiresAt))
}

const patColumns = `id, admin_id, name, token_hash, created_at, last_used_at, expires_at, revoked_at`

// CreatePAT inserts a personal access token.
func (s *Store) CreatePAT(ctx context.Context, t PersonalAccessToken) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO personal_access_tokens
		   (id, admin_id, name, token_hash, created_at, last_used_at, expires_at, revoked_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.AdminID, t.Name, t.TokenHash, formatTime(t.CreatedAt),
		formatNullTime(t.LastUsedAt), formatNullTime(t.ExpiresAt), formatNullTime(t.RevokedAt))
	if err != nil {
		return fmt.Errorf("create personal access token: %w", err)
	}
	return nil
}

// GetPATByHash looks a token up for authentication. The caller checks
// Usable and bumps last_used_at with TouchPAT.
func (s *Store) GetPATByHash(ctx context.Context, tokenHash string) (PersonalAccessToken, error) {
	return scanPAT(s.db.QueryRowContext(ctx,
		`SELECT `+patColumns+` FROM personal_access_tokens WHERE token_hash = ?`, tokenHash))
}

// ListPATs returns every token of the admin (revoked ones included),
// newest first.
func (s *Store) ListPATs(ctx context.Context, adminID string) ([]PersonalAccessToken, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+patColumns+` FROM personal_access_tokens
		 WHERE admin_id = ? ORDER BY created_at DESC, id DESC`, adminID)
	if err != nil {
		return nil, fmt.Errorf("list personal access tokens: %w", err)
	}
	defer rows.Close()

	var tokens []PersonalAccessToken
	for rows.Next() {
		t, err := scanPATRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list personal access tokens: %w", err)
	}
	return tokens, nil
}

// TouchPAT records that the token authenticated a request at `at`.
func (s *Store) TouchPAT(ctx context.Context, id string, at time.Time) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE personal_access_tokens SET last_used_at = ? WHERE id = ?`,
		formatTime(at), id); err != nil {
		return fmt.Errorf("touch personal access token: %w", err)
	}
	return nil
}

// RevokePAT revokes one of the admin's tokens. ErrNotFound when the token
// does not exist, belongs to another admin, or is already revoked.
func (s *Store) RevokePAT(ctx context.Context, adminID, id string, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE personal_access_tokens SET revoked_at = ?
		 WHERE id = ? AND admin_id = ? AND revoked_at IS NULL`,
		formatTime(now), id, adminID)
	if err != nil {
		return fmt.Errorf("revoke personal access token: %w", err)
	}
	return requireRow(res)
}

// DeleteExpiredPATs prunes tokens whose expiry has passed; revoked rows
// without an expiry are kept so the admin console can still show them.
func (s *Store) DeleteExpiredPATs(ctx context.Context, now time.Time) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM personal_access_tokens
		 WHERE expires_at IS NOT NULL AND expires_at < ?`, formatTime(now)); err != nil {
		return fmt.Errorf("delete expired personal access tokens: %w", err)
	}
	return nil
}

func scanPAT(row *sql.Row) (PersonalAccessToken, error) {
	t, err := scanPATRow(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return PersonalAccessToken{}, ErrNotFound
	}
	return t, err
}

func scanPATRow(scan func(...any) error) (PersonalAccessToken, error) {
	var t PersonalAccessToken
	var createdAt string
	var lastUsedAt, expiresAt, revokedAt sql.NullString
	err := scan(&t.ID, &t.AdminID, &t.Name, &t.TokenHash, &createdAt,
		&lastUsedAt, &expiresAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PersonalAccessToken{}, err
	}
	if err != nil {
		return PersonalAccessToken{}, fmt.Errorf("scan personal access token: %w", err)
	}
	if t.CreatedAt, err = parseTime(createdAt); err != nil {
		return PersonalAccessToken{}, fmt.Errorf("parse personal access token created_at: %w", err)
	}
	if t.LastUsedAt, err = parseNullTime(lastUsedAt); err != nil {
		return PersonalAccessToken{}, fmt.Errorf("parse personal access token last_used_at: %w", err)
	}
	if t.ExpiresAt, err = parseNullTime(expiresAt); err != nil {
		return PersonalAccessToken{}, fmt.Errorf("parse personal access token expires_at: %w", err)
	}
	if t.RevokedAt, err = parseNullTime(revokedAt); err != nil {
		return PersonalAccessToken{}, fmt.Errorf("parse personal access token revoked_at: %w", err)
	}
	return t, nil
}
