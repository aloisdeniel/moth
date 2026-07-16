package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// AdminSession is a server-side admin browser session. The cookie value is
// never stored; only its SHA-256 hash.
type AdminSession struct {
	TokenHash string
	AdminID   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (s *Store) CreateSession(ctx context.Context, sess AdminSession) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO admin_sessions (token_hash, admin_id, created_at, expires_at)
		 VALUES (?, ?, ?, ?)`,
		sess.TokenHash, sess.AdminID, formatTime(sess.CreatedAt), formatTime(sess.ExpiresAt))
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *Store) GetSession(ctx context.Context, tokenHash string) (AdminSession, error) {
	var sess AdminSession
	var createdAt, expiresAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT token_hash, admin_id, created_at, expires_at
		 FROM admin_sessions WHERE token_hash = ?`, tokenHash,
	).Scan(&sess.TokenHash, &sess.AdminID, &createdAt, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AdminSession{}, ErrNotFound
	}
	if err != nil {
		return AdminSession{}, fmt.Errorf("scan session: %w", err)
	}
	if sess.CreatedAt, err = parseTime(createdAt); err != nil {
		return AdminSession{}, fmt.Errorf("parse session created_at: %w", err)
	}
	if sess.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return AdminSession{}, fmt.Errorf("parse session expires_at: %w", err)
	}
	return sess, nil
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM admin_sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteAdminSessionsExcept ends every session of the admin other than the
// one identified by keepTokenHash (used after a password change).
func (s *Store) DeleteAdminSessionsExcept(ctx context.Context, adminID, keepTokenHash string) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM admin_sessions WHERE admin_id = ? AND token_hash <> ?`,
		adminID, keepTokenHash); err != nil {
		return fmt.Errorf("delete admin sessions: %w", err)
	}
	return nil
}

func (s *Store) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM admin_sessions WHERE expires_at < ?`, formatTime(now)); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}
