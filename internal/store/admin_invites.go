package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// AdminInvite is a pending operator invitation. The invite token is stored
// hashed; re-inviting the same email replaces the previous invite.
type AdminInvite struct {
	ID        string
	Email     string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// CreateAdminInvite inserts the invite, replacing any pending invite for
// the same email.
func (s *Store) CreateAdminInvite(ctx context.Context, inv AdminInvite) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO admin_invites (id, email, token_hash, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (email) DO UPDATE SET
		   id         = excluded.id,
		   token_hash = excluded.token_hash,
		   expires_at = excluded.expires_at,
		   created_at = excluded.created_at`,
		inv.ID, inv.Email, inv.TokenHash, formatTime(inv.ExpiresAt), formatTime(inv.CreatedAt))
	if err != nil {
		return fmt.Errorf("create admin invite: %w", err)
	}
	return nil
}

// GetAdminInviteByTokenHash looks an invite up for redemption.
func (s *Store) GetAdminInviteByTokenHash(ctx context.Context, tokenHash string) (AdminInvite, error) {
	return scanAdminInvite(s.db.QueryRowContext(ctx,
		`SELECT id, email, token_hash, expires_at, created_at
		 FROM admin_invites WHERE token_hash = ?`, tokenHash))
}

// ListAdminInvites returns every pending invite, oldest first.
func (s *Store) ListAdminInvites(ctx context.Context) ([]AdminInvite, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, email, token_hash, expires_at, created_at
		 FROM admin_invites ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list admin invites: %w", err)
	}
	defer rows.Close()

	var invites []AdminInvite
	for rows.Next() {
		var inv AdminInvite
		var expiresAt, createdAt string
		if err := rows.Scan(&inv.ID, &inv.Email, &inv.TokenHash, &expiresAt, &createdAt); err != nil {
			return nil, fmt.Errorf("scan admin invite: %w", err)
		}
		if inv.ExpiresAt, err = parseTime(expiresAt); err != nil {
			return nil, fmt.Errorf("parse admin invite expires_at: %w", err)
		}
		if inv.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, fmt.Errorf("parse admin invite created_at: %w", err)
		}
		invites = append(invites, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list admin invites: %w", err)
	}
	return invites, nil
}

// DeleteAdminInvite removes one invite (redeemed or revoked).
func (s *Store) DeleteAdminInvite(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM admin_invites WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete admin invite: %w", err)
	}
	return requireRow(res)
}

func scanAdminInvite(row *sql.Row) (AdminInvite, error) {
	var inv AdminInvite
	var expiresAt, createdAt string
	err := row.Scan(&inv.ID, &inv.Email, &inv.TokenHash, &expiresAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AdminInvite{}, ErrNotFound
	}
	if err != nil {
		return AdminInvite{}, fmt.Errorf("scan admin invite: %w", err)
	}
	if inv.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return AdminInvite{}, fmt.Errorf("parse admin invite expires_at: %w", err)
	}
	if inv.CreatedAt, err = parseTime(createdAt); err != nil {
		return AdminInvite{}, fmt.Errorf("parse admin invite created_at: %w", err)
	}
	return inv, nil
}
