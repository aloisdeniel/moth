// Package billing validates App Store and Google Play subscription receipts
// and parses the two stores' server notifications, mapping both into one
// normalized model. It is deliberately self-contained: no database, no proto,
// no server wiring. Callers hand it project store credentials / receipts and
// get back a NormalizedSubscription or NormalizedNotification; the caller owns
// persistence and entitlement derivation.
//
// The store is the source of truth; moth is a validating mirror. Apple
// StoreKit 2 signed transactions and App Store Server Notifications V2 are
// verified locally against Apple's root CA via their x5c certificate chain
// (NOT a JWKS — see jws.go), and authoritative renewal state comes from the
// App Store Server API. Google Play state comes from the Play Developer API
// (purchases.subscriptionsv2.get); RTDN pushes are nudges to re-read, never
// trusted for state. Every outbound client takes an injectable HTTP Doer,
// overridable base/token URLs, and an injectable clock, so the whole engine
// is testable against httptest doubles with no network — mirroring
// internal/oidc.
package billing

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// Doer is the subset of *http.Client the package needs; injectable so tests
// and the server package can point it at doubles. Mirrors oidc.Doer.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

func defaultDoer() Doer { return &http.Client{Timeout: 15 * time.Second} }

// Store identifies which store a subscription originates from.
const (
	StoreApple  = "apple"
	StoreGoogle = "google"
)

// Environment distinguishes sandbox/test receipts from live ones. Stored so a
// project never conflates a tester's sandbox subscription with production.
const (
	EnvSandbox    = "sandbox"
	EnvProduction = "production"
)

// Subscription status enum, minimal and mapped identically from both stores
// (plan/11 §Model). These string constants are the contract the server agent
// maps onto its subscription_status column and entitlement-derivation matrix:
//
//	active, trialing, in_grace_period, in_billing_retry -> entitlement GRANTED
//	                                                        (grace/retry keep
//	                                                        access per store
//	                                                        policy)
//	paused, expired, revoked                            -> NOT granted
const (
	StatusActive         = "active"
	StatusTrialing       = "trialing"
	StatusInGracePeriod  = "in_grace_period"
	StatusInBillingRetry = "in_billing_retry" // Google "on hold"
	StatusPaused         = "paused"
	StatusExpired        = "expired"
	StatusRevoked        = "revoked"
)

// Validation / verification errors. Callers map ErrMalformed and the verify
// failures to connect.CodeInvalidArgument, and a store 404 to
// connect.CodeNotFound (see ErrNotFound below).
var (
	ErrMalformed        = errors.New("billing: malformed token")
	ErrInvalidSignature = errors.New("billing: invalid signature")
	ErrUntrustedChain   = errors.New("billing: certificate chain not trusted")
	ErrBundleMismatch   = errors.New("billing: bundle id mismatch")
	ErrNotFound         = errors.New("billing: not found")
)

// NormalizedSubscription is the store-agnostic subscription state both stores
// map into. The server agent persists it to the subscriptions table and
// derives entitlements from Status. It is intentionally flat and free of any
// store SDK type.
type NormalizedSubscription struct {
	// Store is StoreApple or StoreGoogle.
	Store string
	// ProductID is the store product identifier (Apple productId / Google
	// line-item productId). Maps to a moth product tier downstream.
	ProductID string
	// StoreTransactionID is the stable store identity moth keys the
	// subscription on: originalTransactionId (Apple) or purchaseToken
	// (Google).
	StoreTransactionID string
	// SubscriptionID is the Google base-plan/subscription id, or the Apple
	// subscriptionGroupIdentifier. Empty when the store does not supply one.
	SubscriptionID string
	// Status is one of the status constants above.
	Status string
	// CurrentPeriodEnd is when the paid period ends (renewal or expiry).
	// Zero when the store did not supply an expiry.
	CurrentPeriodEnd time.Time
	// AutoRenew reflects the store's renewal flag at read time.
	AutoRenew bool
	// Environment is EnvSandbox or EnvProduction.
	Environment string
	// RawState is the verified store JSON (decoded transaction / purchase),
	// persisted verbatim for audit and reconciliation.
	RawState json.RawMessage
}

// NormalizedNotification is the store-agnostic shape of a store notification.
// Type/Subtype carry the store's own notification vocabulary (for the audit
// trail); NotificationID is the store's unique id used to dedupe replays. The
// embedded Subscription is best-effort from the notification body — the caller
// MUST re-read authoritative state from the store API before trusting it for
// entitlement changes (plan/11: "the notification is a nudge, not a payload
// to trust").
type NormalizedNotification struct {
	Store          string
	Type           string
	Subtype        string
	NotificationID string
	Subscription   NormalizedSubscription
	Raw            json.RawMessage
}

// appleMillisToTime converts Apple's "milliseconds since the Unix epoch"
// timestamps to a UTC time. Zero in -> zero out.
func appleMillisToTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}
