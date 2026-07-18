package authrpc

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/internal/events"
	"github.com/aloisdeniel/moth/internal/ratelimit"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// KeyHeader carries the project API key on auth and server RPCs.
const KeyHeader = "x-moth-key"

// NewProjectInterceptor resolves the project from the publishable key in
// x-moth-key metadata and injects it into the context. It rejects calls
// without a valid pk_ key, so handlers can rely on the project being set.
// It also captures the SDK's ambient analytics context (x-moth-platform,
// x-moth-sdk-version) so the handlers' event emissions can carry it.
func NewProjectInterceptor(st store.ProjectStore) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			ctx = events.WithClientInfo(ctx, events.ClientInfoFromHeader(req.Header()))
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

// throttledProcedures are the RPCs that accept credentials or trigger
// emails and therefore need brute-force / abuse protection.
var throttledProcedures = map[string]bool{
	"/moth.auth.v1.AuthService/SignUp":                   true,
	"/moth.auth.v1.AuthService/SignIn":                   true,
	"/moth.auth.v1.AuthService/SignInWithOAuth":          true,
	"/moth.auth.v1.AuthService/ExchangeOAuthCode":        true,
	"/moth.auth.v1.AuthService/RequestPasswordReset":     true,
	"/moth.auth.v1.AuthService/RequestEmailVerification": true,
	// Milestone 11 — SubmitPurchase does an outbound store round-trip per call,
	// so it is throttled per IP and per project (no account bucket: the request
	// carries no email).
	"/moth.billing.v1.BillingService/SubmitPurchase": true,
	// Milestone 17 — both Stripe RPCs do an outbound Stripe round-trip per call
	// (checkout may even create a Stripe customer), so they carry the same
	// per-IP / per-project throttle as SubmitPurchase.
	"/moth.billing.v1.BillingService/CreateCheckoutSession":      true,
	"/moth.billing.v1.BillingService/CreateBillingPortalSession": true,
	// Milestone 20 — RegisterDevice is a per-launch credential write from every
	// client, so it carries the per-IP / per-project throttle (no account
	// bucket: the request carries no email).
	"/moth.push.v1.PushService/RegisterDevice": true,
}

// NewRateLimitInterceptor throttles the sensitive RPCs against the shared,
// persistent limiter: per client IP, per project, and — when the request
// carries an email — per account. It runs after the project interceptor so
// the account and project buckets are project-scoped. Over-limit calls fail
// with CodeResourceExhausted carrying the RATE_LIMITED reason and a
// google.rpc.RetryInfo detail.
//
// These are credential-facing RPCs, so a limiter storage error fails CLOSED:
// if the throttle cannot be evaluated (a locked/full/corrupt rate_limits
// table) the call is denied with RATE_LIMITED rather than let through
// unthrottled, which would otherwise re-open unlimited brute force during
// exactly the window the database is stressed.
func NewRateLimitInterceptor(limiter *ratelimit.Limiter, log *slog.Logger) connect.UnaryInterceptorFunc {
	if log == nil {
		log = slog.Default()
	}
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if limiter == nil || !throttledProcedures[req.Spec().Procedure] {
				return next(ctx, req)
			}
			check := func(d ratelimit.Decision, err error) error {
				if err != nil {
					log.ErrorContext(ctx, "rate limit check failed; denying request",
						"procedure", req.Spec().Procedure, "error", err.Error())
					return rateLimitError(0)
				}
				if !d.Allowed {
					return rateLimitError(d.RetryAfter)
				}
				return nil
			}
			if ip := limiter.ClientIP(req.Peer().Addr, req.Header().Get("X-Forwarded-For")); ip != "" {
				if err := check(limiter.IP(ctx, ip)); err != nil {
					return nil, err
				}
			}
			projectID := ""
			if p, ok := ProjectFromContext(ctx); ok {
				projectID = p.ID
				if err := check(limiter.Project(ctx, projectID)); err != nil {
					return nil, err
				}
			}
			if email := requestEmail(req.Any()); email != "" {
				if err := check(limiter.Account(ctx, projectID, email)); err != nil {
					return nil, err
				}
			}
			return next(ctx, req)
		}
	}
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
