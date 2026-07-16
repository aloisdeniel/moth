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
	LastLoginAt     *time.Time
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
	// ProviderEmail is the email the provider asserted when the identity
	// was linked; empty for password identities.
	ProviderEmail string
	// AppleRefreshTokenEnc is the Apple refresh token from the
	// authorization-code exchange, AES-GCM-encrypted under the master key
	// by the caller; nil for every other provider. Needed to revoke Apple
	// tokens when the account is deleted.
	AppleRefreshTokenEnc []byte
	CreatedAt            time.Time
}

// Identity providers.
const (
	// IdentityProviderPassword is the provider of email/password
	// identities; its provider_subject is the user ID.
	IdentityProviderPassword = "password"
	// IdentityProviderGoogle is Sign in with Google; provider_subject is
	// the Google `sub` claim.
	IdentityProviderGoogle = "google"
	// IdentityProviderApple is Sign in with Apple; provider_subject is the
	// Apple `sub` claim.
	IdentityProviderApple = "apple"
)

// ErrConflict is returned when an insert violates a uniqueness constraint
// (email already registered, identity already linked) or a
// compare-and-swap loses to a concurrent write (SetProjectTheme).
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
		                    display_name, avatar_url, custom_claims, disabled_at, last_login_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.ProjectID, u.Email, formatNullTime(u.EmailVerifiedAt), nullString(u.PasswordHash),
		u.DisplayName, u.AvatarURL, u.CustomClaims, formatNullTime(u.DisabledAt),
		formatNullTime(u.LastLoginAt), formatTime(u.CreatedAt), formatTime(u.UpdatedAt)); err != nil {
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
		`INSERT INTO identities (id, project_id, user_id, provider, provider_subject,
		                         provider_email, apple_refresh_token_enc, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id.ID, id.ProjectID, id.UserID, id.Provider, id.ProviderSubject,
		id.ProviderEmail, id.AppleRefreshTokenEnc, formatTime(id.CreatedAt)); err != nil {
		if err := conflictErr(err); errors.Is(err, ErrConflict) {
			return err
		}
		return fmt.Errorf("insert identity: %w", err)
	}
	return nil
}

const userColumns = `id, project_id, email, email_verified_at, password_hash,
	display_name, avatar_url, custom_claims, disabled_at, last_login_at, created_at, updated_at`

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

// SetUserLastLogin records a successful sign-in without touching
// updated_at, so concurrent profile edits cannot race with it.
func (s *Store) SetUserLastLogin(ctx context.Context, projectID, id string, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = ? WHERE project_id = ? AND id = ?`,
		formatTime(at), projectID, id)
	if err != nil {
		return fmt.Errorf("set user last login: %w", err)
	}
	return nil
}

// UserPage selects one page of a project's users, newest first (IDs are
// UUIDv7, so ID order is creation order). AfterID is the last ID of the
// previous page; empty means the first page. Query is a case-insensitive
// substring filter on email and display name.
type UserPage struct {
	Query   string
	AfterID string
	Limit   int
}

// likePattern escapes LIKE wildcards in q and wraps it in %...%.
func likePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

func (s *Store) ListUsersPage(ctx context.Context, projectID string, page UserPage) ([]User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE project_id = ?`
	args := []any{projectID}
	if page.Query != "" {
		q += ` AND (email LIKE ? ESCAPE '\' OR display_name LIKE ? ESCAPE '\')`
		p := likePattern(page.Query)
		args = append(args, p, p)
	}
	if page.AfterID != "" {
		q += ` AND id < ?`
		args = append(args, page.AfterID)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, page.Limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list users page: %w", err)
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
		return nil, fmt.Errorf("list users page: %w", err)
	}
	return users, nil
}

// CountUsers counts a project's users matching query ("" matches all).
func (s *Store) CountUsers(ctx context.Context, projectID, query string) (int, error) {
	q := `SELECT COUNT(*) FROM users WHERE project_id = ?`
	args := []any{projectID}
	if query != "" {
		q += ` AND (email LIKE ? ESCAPE '\' OR display_name LIKE ? ESCAPE '\')`
		p := likePattern(query)
		args = append(args, p, p)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

// CountUsersByProject returns the user count of every project.
func (s *Store) CountUsersByProject(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT project_id, COUNT(*) FROM users GROUP BY project_id`)
	if err != nil {
		return nil, fmt.Errorf("count users by project: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var projectID string
		var n int
		if err := rows.Scan(&projectID, &n); err != nil {
			return nil, fmt.Errorf("scan user count: %w", err)
		}
		counts[projectID] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("count users by project: %w", err)
	}
	return counts, nil
}

const identityColumns = `id, project_id, user_id, provider, provider_subject,
	provider_email, apple_refresh_token_enc, created_at`

func scanIdentityRow(row rowScanner) (Identity, error) {
	var id Identity
	var createdAt string
	if err := row.Scan(&id.ID, &id.ProjectID, &id.UserID, &id.Provider,
		&id.ProviderSubject, &id.ProviderEmail, &id.AppleRefreshTokenEnc, &createdAt); err != nil {
		return Identity{}, err
	}
	var err error
	if id.CreatedAt, err = parseTime(createdAt); err != nil {
		return Identity{}, fmt.Errorf("parse identity created_at: %w", err)
	}
	return id, nil
}

// GetIdentity resolves one provider identity by its provider-issued
// subject; the first step of social sign-in.
func (s *Store) GetIdentity(ctx context.Context, projectID, provider, subject string) (Identity, error) {
	id, err := scanIdentityRow(s.db.QueryRowContext(ctx,
		`SELECT `+identityColumns+` FROM identities
		 WHERE project_id = ? AND provider = ? AND provider_subject = ?`,
		projectID, provider, subject))
	if errors.Is(err, sql.ErrNoRows) {
		return Identity{}, ErrNotFound
	}
	if err != nil {
		return Identity{}, fmt.Errorf("scan identity: %w", err)
	}
	return id, nil
}

// ListUserIdentities returns one user's identities in link order.
func (s *Store) ListUserIdentities(ctx context.Context, projectID, userID string) ([]Identity, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+identityColumns+` FROM identities
		 WHERE project_id = ? AND user_id = ? ORDER BY created_at, id`, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("list user identities: %w", err)
	}
	defer rows.Close()

	var ids []Identity
	for rows.Next() {
		id, err := scanIdentityRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan identity: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list user identities: %w", err)
	}
	return ids, nil
}

// DeleteUserIdentities removes the user's identities of one provider
// (UnlinkIdentity); ErrNotFound when none existed. Callers enforce the
// "never leave zero login methods" rule before calling.
func (s *Store) DeleteUserIdentities(ctx context.Context, projectID, userID, provider string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM identities WHERE project_id = ? AND user_id = ? AND provider = ?`,
		projectID, userID, provider)
	if err != nil {
		return fmt.Errorf("delete user identities: %w", err)
	}
	return requireRow(res)
}

// SetIdentityAppleRefreshToken stores (or, with nil, clears) the encrypted
// Apple refresh token of one identity.
func (s *Store) SetIdentityAppleRefreshToken(ctx context.Context, projectID, id string, tokenEnc []byte) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE identities SET apple_refresh_token_enc = ? WHERE project_id = ? AND id = ?`,
		tokenEnc, projectID, id)
	if err != nil {
		return fmt.Errorf("set identity apple refresh token: %w", err)
	}
	return requireRow(res)
}

// SetIdentityProviderEmail keeps the provider-asserted email current on
// repeat sign-ins.
func (s *Store) SetIdentityProviderEmail(ctx context.Context, projectID, id, email string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE identities SET provider_email = ? WHERE project_id = ? AND id = ?`,
		email, projectID, id)
	if err != nil {
		return fmt.Errorf("set identity provider email: %w", err)
	}
	return requireRow(res)
}

// ListIdentitiesForUsers returns the identities of the given users, keyed
// by user ID. Used to render provider badges on a page of users.
func (s *Store) ListIdentitiesForUsers(ctx context.Context, projectID string, userIDs []string) (map[string][]Identity, error) {
	if len(userIDs) == 0 {
		return map[string][]Identity{}, nil
	}
	placeholders := strings.Repeat("?,", len(userIDs)-1) + "?"
	args := make([]any, 0, len(userIDs)+1)
	args = append(args, projectID)
	for _, id := range userIDs {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+identityColumns+` FROM identities
		 WHERE project_id = ? AND user_id IN (`+placeholders+`)
		 ORDER BY created_at, id`, args...)
	if err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}
	defer rows.Close()

	byUser := make(map[string][]Identity)
	for rows.Next() {
		id, err := scanIdentityRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan identity: %w", err)
		}
		byUser[id.UserID] = append(byUser[id.UserID], id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}
	return byUser, nil
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
	var verifiedAt, passwordHash, disabledAt, lastLoginAt sql.NullString
	var createdAt, updatedAt string
	err := row.Scan(&u.ID, &u.ProjectID, &u.Email, &verifiedAt, &passwordHash,
		&u.DisplayName, &u.AvatarURL, &u.CustomClaims, &disabledAt, &lastLoginAt, &createdAt, &updatedAt)
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
	if u.LastLoginAt, err = parseNullTime(lastLoginAt); err != nil {
		return User{}, fmt.Errorf("parse user last_login_at: %w", err)
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
