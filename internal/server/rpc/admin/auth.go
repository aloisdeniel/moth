package adminrpc

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

type adminCtxKey struct{}

type sessionHashCtxKey struct{}

// AdminFromContext returns the admin authenticated by the auth interceptor.
func AdminFromContext(ctx context.Context) (store.Admin, bool) {
	a, ok := ctx.Value(adminCtxKey{}).(store.Admin)
	return a, ok
}

// SessionHashFromContext returns the hash of the session token that
// authenticated this request, so handlers can spare the current session
// when ending the others.
func SessionHashFromContext(ctx context.Context) (string, bool) {
	h, ok := ctx.Value(sessionHashCtxKey{}).(string)
	return h, ok
}

// procedures that must work without a session.
var publicProcedures = map[string]bool{
	"/moth.admin.v1.SessionService/Login":                  true,
	"/moth.admin.v1.SessionService/Logout":                 true,
	"/moth.admin.v1.AdminAccountService/AcceptAdminInvite": true,
}

// NewAuthInterceptor validates the admin session cookie on every admin RPC
// (except Login/Logout) and injects the admin into the context.
func NewAuthInterceptor(st Store) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if publicProcedures[req.Spec().Procedure] {
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
			return next(ctx, req)
		}
	}
}

func errUnauthenticated() *connect.Error {
	return connect.NewError(connect.CodeUnauthenticated, errors.New("admin session required"))
}
