package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// OAuth token purposes.
const (
	// OAuthTokenPurposeState links the /oauth/{provider}/start redirect to
	// its callback; tampered or replayed callbacks fail the claim.
	OAuthTokenPurposeState = "state"
	// OAuthTokenPurposeCode is the one-time code the callback hands back to
	// the app, exchanged for a token pair via ExchangeOAuthCode.
	OAuthTokenPurposeCode = "code"
	// OAuthTokenPurposeIDToken records the hash of a provider ID token
	// consumed by the native SignInWithOAuth flow; a second insert of the
	// same hash is ErrConflict, which rejects replayed tokens.
	OAuthTokenPurposeIDToken = "id_token"
)

// OAuthToken is a single-use artifact of the web-redirect OAuth fallback.
// The value handed to the browser/app is never stored, only its SHA-256
// hash.
type OAuthToken struct {
	ID        string
	ProjectID string
	Purpose   string
	TokenHash string
	Provider  string
	// UserID is set on codes (the resolved user the exchange signs in);
	// empty on states, which exist before any user is known.
	UserID string
	// RedirectURI is the app scheme target the callback redirects to,
	// validated against the project's registered redirect schemes.
	RedirectURI string
	// Payload carries flow data between legs (e.g. the OIDC nonce and PKCE
	// verifier for states, device info for codes), JSON-encoded by the
	// caller.
	Payload    string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

// CreateOAuthToken inserts a token row; ErrConflict when the hashed value
// is already recorded (an ID-token replay).
func (s *Store) CreateOAuthToken(ctx context.Context, ot OAuthToken) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO oauth_tokens (id, project_id, purpose, token_hash, provider, user_id,
		                           redirect_uri, payload, expires_at, consumed_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ot.ID, ot.ProjectID, ot.Purpose, ot.TokenHash, ot.Provider, nullString(ot.UserID),
		ot.RedirectURI, ot.Payload, formatTime(ot.ExpiresAt),
		formatNullTime(ot.ConsumedAt), formatTime(ot.CreatedAt))
	if err != nil {
		if err := conflictErr(err); errors.Is(err, ErrConflict) {
			return err
		}
		return fmt.Errorf("insert oauth token: %w", err)
	}
	return nil
}

const oauthTokenColumns = `id, project_id, purpose, token_hash, provider, user_id,
	redirect_uri, payload, expires_at, consumed_at, created_at`

// ConsumeOAuthToken atomically claims the token identified by its hashed
// value and returns it. Unknown, already-consumed and expired tokens all
// come back as ErrNotFound, so a claimed token needs no further liveness
// checks.
func (s *Store) ConsumeOAuthToken(ctx context.Context, projectID, purpose, tokenHash string, now time.Time) (OAuthToken, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OAuthToken{}, fmt.Errorf("begin consume oauth token: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE oauth_tokens SET consumed_at = ?
		 WHERE project_id = ? AND purpose = ? AND token_hash = ?
		   AND consumed_at IS NULL AND expires_at > ?`,
		formatTime(now), projectID, purpose, tokenHash, formatTime(now))
	if err != nil {
		return OAuthToken{}, fmt.Errorf("consume oauth token: %w", err)
	}
	if err := requireRow(res); err != nil {
		return OAuthToken{}, err
	}

	ot, err := scanOAuthTokenRow(tx.QueryRowContext(ctx,
		`SELECT `+oauthTokenColumns+` FROM oauth_tokens
		 WHERE project_id = ? AND token_hash = ?`, projectID, tokenHash))
	if err != nil {
		return OAuthToken{}, fmt.Errorf("scan oauth token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return OAuthToken{}, fmt.Errorf("commit consume oauth token: %w", err)
	}
	return ot, nil
}

// DeleteExpiredOAuthTokens drops every expired row across all projects
// (periodic cleanup; consumed rows expire too and are swept with them).
func (s *Store) DeleteExpiredOAuthTokens(ctx context.Context, now time.Time) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM oauth_tokens WHERE expires_at < ?`, formatTime(now)); err != nil {
		return fmt.Errorf("delete expired oauth tokens: %w", err)
	}
	return nil
}

func scanOAuthTokenRow(row rowScanner) (OAuthToken, error) {
	var ot OAuthToken
	var userID, consumedAt sql.NullString
	var expiresAt, createdAt string
	err := row.Scan(&ot.ID, &ot.ProjectID, &ot.Purpose, &ot.TokenHash, &ot.Provider,
		&userID, &ot.RedirectURI, &ot.Payload, &expiresAt, &consumedAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return OAuthToken{}, ErrNotFound
	}
	if err != nil {
		return OAuthToken{}, err
	}
	ot.UserID = userID.String
	if ot.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return OAuthToken{}, fmt.Errorf("parse oauth token expires_at: %w", err)
	}
	if ot.ConsumedAt, err = parseNullTime(consumedAt); err != nil {
		return OAuthToken{}, fmt.Errorf("parse oauth token consumed_at: %w", err)
	}
	if ot.CreatedAt, err = parseTime(createdAt); err != nil {
		return OAuthToken{}, fmt.Errorf("parse oauth token created_at: %w", err)
	}
	return ot, nil
}
