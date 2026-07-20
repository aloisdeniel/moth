package store

import (
	"context"
	"fmt"
	"time"
)

// UserExport bundles a user with its provider identities for a project data
// export (the JSON `moth project export` emits).
type UserExport struct {
	User       User
	Identities []Identity
}

// UserImport is one user to bulk-load into a project. The User carries the
// (possibly foreign) password hash and its PasswordAlgo marker; Identities
// are its provider links. IDs are assigned by the caller.
type UserImport struct {
	User       User
	Identities []Identity
}

// ImportResult reports how a bulk import went.
type ImportResult struct {
	// Imported is the number of users actually inserted.
	Imported int
	// Skipped is the number of users left untouched because their email was
	// already registered in the project.
	Skipped int
}

// ExportUsers reads every user of a project with its identities, in creation
// order, for migration off moth. It streams the whole project; callers that
// need paging use ListUsersPage instead.
func (s *Store) ExportUsers(ctx context.Context, projectID string) ([]UserExport, error) {
	users, err := s.ListUsers(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return []UserExport{}, nil
	}
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	byUser, err := s.ListIdentitiesForUsers(ctx, projectID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]UserExport, len(users))
	for i, u := range users {
		out[i] = UserExport{User: u, Identities: byUser[u.ID]}
	}
	return out, nil
}

// ImportUsers bulk-inserts users (with their password hashes and identities)
// into a project in one transaction. A user whose email already exists in
// the project is skipped without aborting the import; its identities are
// skipped with it. Insertion is all-or-nothing on unexpected errors.
func (s *Store) ImportUsers(ctx context.Context, projectID string, users []UserImport, now time.Time) (ImportResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin import users: %w", err)
	}
	defer tx.Rollback()

	var res ImportResult
	for _, ui := range users {
		u := ui.User
		u.ProjectID = projectID
		if u.CreatedAt.IsZero() {
			u.CreatedAt = now
		}
		if u.UpdatedAt.IsZero() {
			u.UpdatedAt = now
		}
		if u.CustomClaims == "" {
			u.CustomClaims = "{}"
		}
		r, err := tx.ExecContext(ctx,
			`INSERT INTO users (id, project_id, email, email_verified_at, password_hash, password_algo,
			                    display_name, avatar_url, custom_claims, disabled_at, last_login_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT (project_id, email) DO NOTHING`,
			u.ID, u.ProjectID, u.Email, formatNullTime(u.EmailVerifiedAt), nullString(u.PasswordHash), u.PasswordAlgo,
			u.DisplayName, u.AvatarURL, u.CustomClaims, formatNullTime(u.DisabledAt),
			formatNullTime(u.LastLoginAt), formatTime(u.CreatedAt), formatTime(u.UpdatedAt))
		if err != nil {
			return ImportResult{}, fmt.Errorf("import user %q: %w", u.Email, err)
		}
		n, err := r.RowsAffected()
		if err != nil {
			return ImportResult{}, fmt.Errorf("import user %q: %w", u.Email, err)
		}
		if n == 0 {
			res.Skipped++
			continue
		}
		res.Imported++
		for _, id := range ui.Identities {
			id.ProjectID = projectID
			id.UserID = u.ID
			if id.CreatedAt.IsZero() {
				id.CreatedAt = now
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO identities (id, project_id, user_id, provider, provider_subject,
				                         provider_email, apple_refresh_token_enc, created_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				 ON CONFLICT (project_id, provider, provider_subject) DO NOTHING`,
				id.ID, id.ProjectID, id.UserID, id.Provider, id.ProviderSubject,
				id.ProviderEmail, id.AppleRefreshTokenEnc, formatTime(id.CreatedAt)); err != nil {
				return ImportResult{}, fmt.Errorf("import identity for %q: %w", u.Email, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return ImportResult{}, fmt.Errorf("commit import users: %w", err)
	}
	return res, nil
}
