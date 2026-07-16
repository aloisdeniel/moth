package authrpc

import (
	"context"
	"errors"
	"net"
	"strings"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/internal/ratelimit"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// KeyHeader carries the project API key on auth and server RPCs.
const KeyHeader = "x-moth-key"

// NewProjectInterceptor resolves the project from the publishable key in
// x-moth-key metadata and injects it into the context. It rejects calls
// without a valid pk_ key, so handlers can rely on the project being set.
func NewProjectInterceptor(st store.ProjectStore) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			key := strings.TrimSpace(req.Header().Get(KeyHeader))
			if !strings.HasPrefix(key, token.PublishableKeyPrefix) {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					errors.New("publishable key required in "+KeyHeader+" metadata"))
			}
			project, err := st.GetProjectByPublishableKey(ctx, key)
			if errors.Is(err, store.ErrNotFound) {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					errors.New("unknown publishable key"))
			}
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			return next(WithProject(ctx, project), req)
		}
	}
}

// Default rate limits for the credential-facing RPCs (full hardening in
// milestone 10).
const (
	DefaultIPRatePerMinute      = 30
	DefaultIPBurst              = 15
	DefaultAccountRatePerMinute = 10
	DefaultAccountBurst         = 5
)

// RateLimits holds the two token-bucket limiters of the auth service.
type RateLimits struct {
	PerIP      *ratelimit.Limiter
	PerAccount *ratelimit.Limiter
}

// DefaultRateLimits returns the standard limits.
func DefaultRateLimits() RateLimits {
	return RateLimits{
		PerIP:      ratelimit.New(DefaultIPRatePerMinute, DefaultIPBurst),
		PerAccount: ratelimit.New(DefaultAccountRatePerMinute, DefaultAccountBurst),
	}
}

// throttledProcedures are the RPCs that accept credentials or trigger
// emails and therefore need brute-force / abuse protection.
var throttledProcedures = map[string]bool{
	"/moth.auth.v1.AuthService/SignUp":                   true,
	"/moth.auth.v1.AuthService/SignIn":                   true,
	"/moth.auth.v1.AuthService/SignInWithOAuth":          true,
	"/moth.auth.v1.AuthService/ExchangeOAuthCode":        true,
	"/moth.auth.v1.AuthService/RequestPasswordReset":     true,
	"/moth.auth.v1.AuthService/RequestEmailVerification": true,
}

// NewRateLimitInterceptor throttles the sensitive RPCs per client IP and,
// when the request carries an email, per account. It runs after the
// project interceptor so account keys are project-scoped.
func NewRateLimitInterceptor(limits RateLimits) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !throttledProcedures[req.Spec().Procedure] {
				return next(ctx, req)
			}
			limited := newError(connect.CodeResourceExhausted, ReasonRateLimited,
				"too many attempts, retry later")
			if ip := clientIP(req); ip != "" && !limits.PerIP.Allow(ip) {
				return nil, limited
			}
			if email := requestEmail(req.Any()); email != "" {
				key := email
				if p, ok := ProjectFromContext(ctx); ok {
					key = p.ID + "/" + email
				}
				if !limits.PerAccount.Allow(key) {
					return nil, limited
				}
			}
			return next(ctx, req)
		}
	}
}

// clientIP extracts the caller address: the first X-Forwarded-For hop when
// present (reverse-proxy deployments), otherwise the connection peer.
func clientIP(req connect.AnyRequest) string {
	if fwd := req.Header().Get("X-Forwarded-For"); fwd != "" {
		first, _, _ := strings.Cut(fwd, ",")
		return strings.TrimSpace(first)
	}
	addr := req.Peer().Addr
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// requestEmail pulls the account email out of the throttled request types.
func requestEmail(msg any) string {
	switch m := msg.(type) {
	case *authv1.SignUpRequest:
		return normalizeEmail(m.Email)
	case *authv1.SignInRequest:
		return normalizeEmail(m.Email)
	case *authv1.RequestPasswordResetRequest:
		return normalizeEmail(m.Email)
	case *authv1.RequestEmailVerificationRequest:
		return normalizeEmail(m.Email)
	}
	return ""
}
