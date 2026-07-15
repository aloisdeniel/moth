package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// User is an end user of one project. The same email in two projects is
// two unrelated accounts; every query on this table is project-scoped.
type User struct {
	ID              string
	ProjectID       string
	Email           string // stored lowercased
	EmailVerifiedAt *time.Time
	PasswordHash    string // empty for social-only accounts
	DisplayName     string
	AvatarURL       string
	CustomClaims    string // JSON object embedded in the JWT `claims` claim
	DisabledAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Verified reports whether the user's email address is verified.
func (u User) Verified() bool { return u.EmailVerifiedAt != nil }

// Disabled reports whether the user is blocked from signing in.
func (u User) Disabled() bool { return u.DisabledAt != nil }

// Identity links a user to one authentication provider.
type Identity struct {
	ID              string
	ProjectID       string
	UserID          string
	Provider        string
	ProviderSubject string
	CreatedAt       time.Time
}

// IdentityProviderPassword is the provider of email/password identities;
// its provider_subject is the user ID. Social providers land in
// milestone 04.
const IdentityProviderPassword = "password"

// ErrConflict is returned when an insert violates a uniqueness constraint
// (email already registered, identity already linked).
var ErrConflict = errors.New("conflict")

func conflictErr(err error) error {
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return ErrConflict
	}
	return err
}

// CreateUser inserts the user and its identities in one transaction.
func (s *Store) CreateUser(ctx context.Context, u User, identities ...Identity) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create user: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO users (id, project_id, email, email_verified_at, password_hash,
		                    display_name, avatar_url, custom_claims, disabled_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.ProjectID, u.Email, formatNullTime(u.EmailVerifiedAt), nullString(u.PasswordHash),
		u.DisplayName, u.AvatarURL, u.CustomClaims, formatNullTime(u.DisabledAt),
		formatTime(u.CreatedAt), formatTime(u.UpdatedAt)); err != nil {
		if err := conflictErr(err); errors.Is(err, ErrConflict) {
			return err
		}
		return fmt.Errorf("insert user: %w", err)
	}
	for _, id := range identities {
		if err := insertIdentity(ctx, tx, id); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create user: %w", err)
	}
	return nil
}

func (s *Store) CreateIdentity(ctx context.Context, id Identity) error {
	return insertIdentity(ctx, s.db, id)
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func insertIdentity(ctx context.Context, db execer, id Identity) error {
	if _, err := db.ExecContext(ctx,
		`INSERT INTO identities (id, project_id, user_id, provider, provider_subject, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id.ID, id.ProjectID, id.UserID, id.Provider, id.ProviderSubject,
		formatTime(id.CreatedAt)); err != nil {
		if err := conflictErr(err); errors.Is(err, ErrConflict) {
			return err
		}
		return fmt.Errorf("insert identity: %w", err)
	}
	return nil
}

const userColumns = `id, project_id, email, email_verified_at, password_hash,
	display_name, avatar_url, custom_claims, disabled_at, created_at, updated_at`

func (s *Store) GetUser(ctx context.Context, projectID, id string) (User, error) {
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE project_id = ? AND id = ?`, projectID, id))
}

func (s *Store) GetUserByEmail(ctx context.Context, projectID, email string) (User, error) {
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE project_id = ? AND email = ?`, projectID, email))
}

func (s *Store) ListUsers(ctx context.Context, projectID string) ([]User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE project_id = ? ORDER BY created_at, id`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// UpdateUser persists every mutable user field; callers update the struct
// returned by a Get and pass it back.
func (s *Store) UpdateUser(ctx context.Context, u User) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET email = ?, email_verified_at = ?, password_hash = ?,
		        display_name = ?, avatar_url = ?, custom_claims = ?, disabled_at = ?, updated_at = ?
		 WHERE project_id = ? AND id = ?`,
		u.Email, formatNullTime(u.EmailVerifiedAt), nullString(u.PasswordHash),
		u.DisplayName, u.AvatarURL, u.CustomClaims, formatNullTime(u.DisabledAt),
		formatTime(u.UpdatedAt), u.ProjectID, u.ID)
	if err != nil {
		if err := conflictErr(err); errors.Is(err, ErrConflict) {
			return err
		}
		return fmt.Errorf("update user: %w", err)
	}
	return requireRow(res)
}

// DeleteUser removes the user; identities, refresh tokens and email tokens
// cascade via foreign keys.
func (s *Store) DeleteUser(ctx context.Context, projectID, id string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM users WHERE project_id = ? AND id = ?`, projectID, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return requireRow(res)
}

func scanUser(row *sql.Row) (User, error) {
	u, err := scanUserRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func scanUserRow(row rowScanner) (User, error) {
	var u User
	var verifiedAt, passwordHash, disabledAt sql.NullString
	var createdAt, updatedAt string
	err := row.Scan(&u.ID, &u.ProjectID, &u.Email, &verifiedAt, &passwordHash,
		&u.DisplayName, &u.AvatarURL, &u.CustomClaims, &disabledAt, &createdAt, &updatedAt)
	if err != nil {
		return User{}, err
	}
	u.PasswordHash = passwordHash.String
	if u.EmailVerifiedAt, err = parseNullTime(verifiedAt); err != nil {
		return User{}, fmt.Errorf("parse user email_verified_at: %w", err)
	}
	if u.DisabledAt, err = parseNullTime(disabledAt); err != nil {
		return User{}, fmt.Errorf("parse user disabled_at: %w", err)
	}
	if u.CreatedAt, err = parseTime(createdAt); err != nil {
		return User{}, fmt.Errorf("parse user created_at: %w", err)
	}
	if u.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return User{}, fmt.Errorf("parse user updated_at: %w", err)
	}
	return u, nil
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func formatNullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func parseNullTime(s sql.NullString) (*time.Time, error) {
	if !s.Valid {
		return nil, nil
	}
	t, err := parseTime(s.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
