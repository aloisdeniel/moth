package adminrpc

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

type adminCtxKey struct{}

type sessionHashCtxKey struct{}

type credentialCtxKey struct{}

// CredentialType is the kind of credential that authenticated a request.
type CredentialType string

const (
	// CredentialSession is the browser session cookie (the admin SPA).
	CredentialSession CredentialType = "session"
	// CredentialPAT is a personal access token (the CLI, scripts).
	CredentialPAT CredentialType = "pat"
)

// Credential records how a request was authenticated, so actions can be
// attributed to the exact credential (the milestone-10 audit log).
type Credential struct {
	Type CredentialType
	// PATID is the personal access token id when Type is CredentialPAT.
	PATID string
	// PATExpiresAt is that token's expiry (nil: never expires), so
	// CreatePersonalAccessToken can cap PAT-minted tokens at the creating
	// token's remaining lifetime.
	PATExpiresAt *time.Time
}

// AdminFromContext returns the admin authenticated by the auth interceptor.
func AdminFromContext(ctx context.Context) (store.Admin, bool) {
	a, ok := ctx.Value(adminCtxKey{}).(store.Admin)
	return a, ok
}

// SessionHashFromContext returns the hash of the session token that
// authenticated this request, so handlers can spare the current session
// when ending the others. Absent on PAT-authenticated requests.
func SessionHashFromContext(ctx context.Context) (string, bool) {
	h, ok := ctx.Value(sessionHashCtxKey{}).(string)
	return h, ok
}

// CredentialFromContext returns which credential authenticated this
// request (cookie session or personal access token).
func CredentialFromContext(ctx context.Context) (Credential, bool) {
	c, ok := ctx.Value(credentialCtxKey{}).(Credential)
	return c, ok
}

// procedures that must work without a session.
var publicProcedures = map[string]bool{
	"/moth.admin.v1.SessionService/Login":                  true,
	"/moth.admin.v1.SessionService/Logout":                 true,
	"/moth.admin.v1.AdminAccountService/AcceptAdminInvite": true,
}

// NewAuthInterceptor authenticates every admin RPC (except the public
// procedures): either the admin session cookie (the SPA) or a personal
// access token presented as `authorization: Bearer moth_pat_...` (the
// CLI). Both paths inject the same admin identity into the context, so
// every admin handler works unchanged over either credential.
func NewAuthInterceptor(st Store) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if publicProcedures[req.Spec().Procedure] {
				return next(ctx, req)
			}
			if plain := bearerPAT(req.Header()); plain != "" {
				admin, pat, err := authenticatePAT(ctx, st, plain)
				if err != nil {
					return nil, err
				}
				ctx = context.WithValue(ctx, adminCtxKey{}, admin)
				ctx = context.WithValue(ctx, credentialCtxKey{},
					Credential{Type: CredentialPAT, PATID: pat.ID, PATExpiresAt: pat.ExpiresAt})
				return next(ctx, req)
			}
			tok, ok := sessionToken(req.Header())
			if !ok {
				return nil, errUnauthenticated()
			}
			sess, err := st.GetSession(ctx, token.Hash(tok))
			if errors.Is(err, store.ErrNotFound) {
				return nil, errUnauthenticated()
			}
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			if time.Now().After(sess.ExpiresAt) {
				return nil, errUnauthenticated()
			}
			admin, err := st.GetAdmin(ctx, sess.AdminID)
			if errors.Is(err, store.ErrNotFound) {
				return nil, errUnauthenticated()
			}
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			ctx = context.WithValue(ctx, adminCtxKey{}, admin)
			ctx = context.WithValue(ctx, sessionHashCtxKey{}, token.Hash(tok))
			ctx = context.WithValue(ctx, credentialCtxKey{}, Credential{Type: CredentialSession})
			return next(ctx, req)
		}
	}
}

// authenticatePAT resolves a personal access token to its admin and bumps
// the token's last-used time.
func authenticatePAT(ctx context.Context, st Store, plain string) (store.Admin, store.PersonalAccessToken, error) {
	pat, err := st.GetPATByHash(ctx, token.Hash(plain))
	if errors.Is(err, store.ErrNotFound) {
		return store.Admin{}, store.PersonalAccessToken{}, errBadPAT()
	}
	if err != nil {
		return store.Admin{}, store.PersonalAccessToken{}, connect.NewError(connect.CodeInternal, err)
	}
	now := time.Now()
	if !pat.Usable(now) {
		return store.Admin{}, store.PersonalAccessToken{}, errBadPAT()
	}
	admin, err := st.GetAdmin(ctx, pat.AdminID)
	if errors.Is(err, store.ErrNotFound) {
		return store.Admin{}, store.PersonalAccessToken{}, errBadPAT()
	}
	if err != nil {
		return store.Admin{}, store.PersonalAccessToken{}, connect.NewError(connect.CodeInternal, err)
	}
	if err := st.TouchPAT(ctx, pat.ID, now); err != nil {
		return store.Admin{}, store.PersonalAccessToken{}, connect.NewError(connect.CodeInternal, err)
	}
	return admin, pat, nil
}

// bearerPAT extracts a personal access token from the Authorization
// header; "" when the header is absent or carries something else.
func bearerPAT(h http.Header) string {
	auth := h.Get("Authorization")
	const scheme = "Bearer "
	if len(auth) <= len(scheme) || !strings.EqualFold(auth[:len(scheme)], scheme) {
		return ""
	}
	if tok := strings.TrimSpace(auth[len(scheme):]); strings.HasPrefix(tok, token.PATPrefix) {
		return tok
	}
	return ""
}

func errUnauthenticated() *connect.Error {
	return connect.NewError(connect.CodeUnauthenticated, errors.New("admin session required"))
}

func errBadPAT() *connect.Error {
	return connect.NewError(connect.CodeUnauthenticated,
		errors.New("invalid, expired or revoked personal access token"))
}
