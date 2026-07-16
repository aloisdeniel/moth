// Package events captures the server-emitted analytics events of
// milestone 07. Handlers build events with the typed constructors and hand
// them to a Writer, whose Emit never blocks the calling RPC: events are
// buffered in a bounded channel and batch-inserted by a background
// goroutine, with drop-on-overflow rather than back-pressure.
//
// The package is self-contained: it defines its own narrow store interface
// (BatchInserter) and does not import the rest of the server.
package events

import (
	"context"
	"encoding/json"
	"time"
)

// Event types emitted by the auth handlers. Authoritative — clients never
// send events, only ambient context in request metadata.
const (
	TypeUserSignup             = "user.signup"
	TypeUserLogin              = "user.login"
	TypeTokenRefresh           = "token.refresh"
	TypeUserLoginFailed        = "user.login_failed"
	TypePasswordResetCompleted = "password.reset_completed"
	TypeEmailVerified          = "email.verified"
	TypeUserDeleted            = "user.deleted"
	TypeIdentityLinked         = "identity.linked"
)

// FailureReason buckets why a login failed. Only these coarse buckets are
// stored — never the raw error, never an identifier of the account.
type FailureReason string

const (
	ReasonInvalidCredentials FailureReason = "invalid_credentials"
	ReasonDisabled           FailureReason = "disabled"
	ReasonProviderDisabled   FailureReason = "provider_disabled"
	ReasonOther              FailureReason = "other"
)

// bucket maps any FailureReason onto the known enum, defaulting to other,
// so arbitrary strings can never leak into metadata.
func (r FailureReason) bucket() FailureReason {
	switch r {
	case ReasonInvalidCredentials, ReasonDisabled, ReasonProviderDisabled:
		return r
	default:
		return ReasonOther
	}
}

// Event is one analytics event, mirroring the events table: who did what,
// on which platform, when. UserID, Provider, Platform, SDKVersion and
// Metadata are all optional.
type Event struct {
	Type       string
	ProjectID  string
	UserID     string
	Provider   string
	Platform   string
	SDKVersion string
	// Metadata is a small bag of extra fields (e.g. the login-failure
	// reason). It is JSON-encoded at write time via MetadataJSON.
	Metadata  map[string]string
	CreatedAt time.Time
}

// MetadataJSON returns the metadata encoded as a JSON object, or "" when
// there is none. Encoding failures cannot happen for map[string]string.
func (e Event) MetadataJSON() string {
	if len(e.Metadata) == 0 {
		return ""
	}
	b, err := json.Marshal(e.Metadata)
	if err != nil {
		return "" // unreachable for map[string]string
	}
	return string(b)
}

// newEvent stamps the common fields, pulling platform and SDK version from
// the client info stashed in ctx by the auth interceptor (see headers.go).
func newEvent(ctx context.Context, typ, projectID string) Event {
	info := ClientInfoFromContext(ctx)
	return Event{
		Type:       typ,
		ProjectID:  projectID,
		Platform:   info.Platform,
		SDKVersion: info.SDKVersion,
		CreatedAt:  time.Now().UTC(),
	}
}

// Signup records a new account. Provider is "" for email/password.
func Signup(ctx context.Context, projectID, userID, provider string) Event {
	e := newEvent(ctx, TypeUserSignup, projectID)
	e.UserID = userID
	e.Provider = provider
	return e
}

// Login records a successful sign-in.
func Login(ctx context.Context, projectID, userID, provider string) Event {
	e := newEvent(ctx, TypeUserLogin, projectID)
	e.UserID = userID
	e.Provider = provider
	return e
}

// TokenRefresh records a refresh-token exchange. Writers keep each user's
// first refresh of the day and sample the rest (see Config.SampleRates):
// the type exists to approximate DAU — which needs every active user, not
// every refresh.
func TokenRefresh(ctx context.Context, projectID, userID string) Event {
	e := newEvent(ctx, TypeTokenRefresh, projectID)
	e.UserID = userID
	return e
}

// LoginFailed records a failed sign-in. By construction it carries no user
// ID — only the coarse reason bucket in metadata — so failure analytics can
// never identify an account.
func LoginFailed(ctx context.Context, projectID, provider string, reason FailureReason) Event {
	e := newEvent(ctx, TypeUserLoginFailed, projectID)
	e.Provider = provider
	e.Metadata = map[string]string{"reason": string(reason.bucket())}
	return e
}

// PasswordResetCompleted records a finished password reset.
func PasswordResetCompleted(ctx context.Context, projectID, userID string) Event {
	e := newEvent(ctx, TypePasswordResetCompleted, projectID)
	e.UserID = userID
	return e
}

// EmailVerified records a confirmed email address.
func EmailVerified(ctx context.Context, projectID, userID string) Event {
	e := newEvent(ctx, TypeEmailVerified, projectID)
	e.UserID = userID
	return e
}

// UserDeleted records an account deletion.
func UserDeleted(ctx context.Context, projectID, userID string) Event {
	e := newEvent(ctx, TypeUserDeleted, projectID)
	e.UserID = userID
	return e
}

// IdentityLinked records a social identity being attached to an existing
// account.
func IdentityLinked(ctx context.Context, projectID, userID, provider string) Event {
	e := newEvent(ctx, TypeIdentityLinked, projectID)
	e.UserID = userID
	e.Provider = provider
	return e
}
