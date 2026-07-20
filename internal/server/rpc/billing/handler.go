// Package billingrpc implements moth.billing.v1.BillingService — the
// publishable-key + Bearer client API for subscriptions and entitlements — and
// hosts the shared store-validation plumbing the webhook and reconciliation
// sweep reuse. The store is the source of truth; every state change here is the
// result of a verified signed receipt or an authoritative store read, never a
// client's say-so. `none` (no entitlements) is a first-class, always-valid
// state: GetCustomerInfo never errors for a free user.
package billingrpc

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	billingv1 "github.com/aloisdeniel/moth/gen/moth/billing/v1"
	"github.com/aloisdeniel/moth/gen/moth/billing/v1/billingv1connect"
	"github.com/aloisdeniel/moth/internal/billing"
	"github.com/aloisdeniel/moth/internal/entitlements"
	"github.com/aloisdeniel/moth/internal/keys"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// Store is everything the billing service, webhook and reconciliation need from
// persistence. User authentication is delegated to the auth handler.
type Store interface {
	store.ProjectStore
	store.EntitlementStore
	store.ProductStore
	store.SubscriptionStore
	store.SubscriptionGrantStore
	store.StoreNotificationStore
	store.BillingCredentialStore
	store.StripeCustomerStore
	store.SubscriptionEventStore
	store.CopyStore
	// UserStore backs webhook user attribution: a Stripe checkout event names a
	// moth user id that must exist in the project before a subscription row is
	// created for it (RPC paths authenticate the user via the auth handler
	// instead).
	store.UserStore
}

var _ billingv1connect.BillingServiceHandler = (*Handler)(nil)

// Handler implements moth.billing.v1.BillingService and carries the shared
// store-client plumbing.
type Handler struct {
	store  Store
	master keys.MasterKey
	auth   *authrpc.Handler // Bearer user authentication (shared with auth.v1)
	log    *slog.Logger
	now    func() time.Time
	httpc  billing.Doer

	// Endpoint + trust overrides. Zero values mean the real stores and Apple's
	// embedded roots; tests point them at httptest doubles and a test CA.
	appleBaseURL    string
	appleSandboxURL string
	appleRoots      *x509.CertPool
	googleBaseURL   string
	googleTokenURL  string
	stripeBaseURL   string
}

// Options configures the billing handler.
type Options struct {
	Store  Store
	Master keys.MasterKey
	Auth   *authrpc.Handler
	Logger *slog.Logger
	Now    func() time.Time
	// HTTPClient performs outbound store calls; nil falls back to a
	// timeout-bounded default.
	HTTPClient billing.Doer
	// The following are test-only overrides.
	AppleBaseURL    string
	AppleSandboxURL string
	AppleRoots      *x509.CertPool
	GoogleBaseURL   string
	GoogleTokenURL  string
	StripeBaseURL   string
}

// New builds the billing handler.
func New(o Options) *Handler {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.HTTPClient == nil {
		o.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Handler{
		store:           o.Store,
		master:          o.Master,
		auth:            o.Auth,
		log:             o.Logger,
		now:             o.Now,
		httpc:           o.HTTPClient,
		appleBaseURL:    o.AppleBaseURL,
		appleSandboxURL: o.AppleSandboxURL,
		appleRoots:      o.AppleRoots,
		googleBaseURL:   o.GoogleBaseURL,
		googleTokenURL:  o.GoogleTokenURL,
		stripeBaseURL:   o.StripeBaseURL,
	}
}

// --- store clients --------------------------------------------------------

// appleVerifier builds a verifier over the configured roots (Apple's real roots
// unless overridden) bound to the project's bundle id.
func (h *Handler) appleVerifier(cred store.BillingCredentials) *billing.AppleVerifier {
	return billing.NewAppleVerifier(h.appleRoots, cred.AppleBundleID, h.now)
}

// appleClient builds an App Store Server API client from the project's
// decrypted In-App-Purchase key. It returns errNotConfigured when Apple billing
// is not set up for the project.
func (h *Handler) appleClient(cred store.BillingCredentials) (*billing.AppleClient, error) {
	if len(cred.AppleIAPKeyEnc) == 0 || cred.AppleIAPKeyID == "" || cred.AppleIAPIssuerID == "" || cred.AppleBundleID == "" {
		return nil, errNotConfigured
	}
	p8, err := h.master.Decrypt(cred.AppleIAPKeyEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt apple iap key: %w", err)
	}
	key, err := billing.ParseP8(p8)
	if err != nil {
		return nil, fmt.Errorf("parse apple iap key: %w", err)
	}
	return &billing.AppleClient{
		BaseURL:    h.appleBaseURL,
		SandboxURL: h.appleSandboxURL,
		IssuerID:   cred.AppleIAPIssuerID,
		KeyID:      cred.AppleIAPKeyID,
		BundleID:   cred.AppleBundleID,
		Key:        key,
		HTTPC:      h.httpc,
		Now:        h.now,
		Verifier:   h.appleVerifier(cred),
	}, nil
}

// googleClient builds a Play Developer API client from the project's decrypted
// service account. Returns errNotConfigured when Google billing is not set up.
func (h *Handler) googleClient(cred store.BillingCredentials) (*billing.GoogleClient, error) {
	if len(cred.GoogleServiceAccountEnc) == 0 || cred.GooglePackageName == "" {
		return nil, errNotConfigured
	}
	saJSON, err := h.master.Decrypt(cred.GoogleServiceAccountEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt google service account: %w", err)
	}
	sa, err := billing.ParseServiceAccount(saJSON)
	if err != nil {
		return nil, fmt.Errorf("parse google service account: %w", err)
	}
	tokens := billing.NewGoogleTokenSource(sa, h.googleTokenURL, h.httpc, h.now)
	return &billing.GoogleClient{
		BaseURL:     h.googleBaseURL,
		PackageName: cred.GooglePackageName,
		Tokens:      tokens,
		HTTPC:       h.httpc,
	}, nil
}

// stripeClient builds a Stripe REST API client from the project's decrypted
// secret key. Returns errNotConfigured when Stripe billing is not set up —
// checked on field presence, not row presence, so a project with only Apple or
// Google credentials still gets BILLING_NOT_CONFIGURED from the checkout RPCs.
func (h *Handler) stripeClient(cred store.BillingCredentials) (*billing.StripeClient, error) {
	if len(cred.StripeSecretKeyEnc) == 0 {
		return nil, errNotConfigured
	}
	key, err := h.master.Decrypt(cred.StripeSecretKeyEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt stripe secret key: %w", err)
	}
	return &billing.StripeClient{
		BaseURL:   h.stripeBaseURL,
		SecretKey: string(key),
		HTTPC:     h.httpc,
		Now:       h.now,
	}, nil
}

// errNotConfigured signals a project has no store credentials for the requested
// store.
var errNotConfigured = errors.New("billing: store not configured for project")

// --- customer info --------------------------------------------------------

// customerInfo loads the user's catalog, subscriptions and grants, derives the
// held entitlements, and builds the proto CustomerInfo. It never fails for a
// free user: an empty entitlement set is the valid `none` state.
func (h *Handler) customerInfo(ctx context.Context, projectID, userID string) (*billingv1.CustomerInfo, error) {
	ents, err := h.store.ListEntitlements(ctx, projectID)
	if err != nil {
		return nil, err
	}
	products, err := h.store.ListProducts(ctx, projectID)
	if err != nil {
		return nil, err
	}
	subs, err := h.store.ListUserSubscriptions(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	grants, err := h.store.ListUserGrants(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	derived := entitlements.Derive(h.now(), ents, products, subs, grants)

	prodIdentByID := make(map[string]string, len(products))
	for _, p := range products {
		prodIdentByID[p.ID] = p.Identifier
	}

	info := &billingv1.CustomerInfo{}
	for _, e := range derived {
		pe := &billingv1.Entitlement{
			Identifier:        e.Identifier,
			Source:            mapSourceToProto(e.Source),
			ProductIdentifier: e.ProductIdentifier,
		}
		if !e.ExpireTime.IsZero() {
			pe.ExpireTime = timestamppb.New(e.ExpireTime)
		}
		info.ActiveEntitlements = append(info.ActiveEntitlements, pe)
	}
	for _, s := range subs {
		as := &billingv1.ActiveSubscription{
			ProductIdentifier: prodIdentByID[s.ProductID],
			Store:             mapStoreToProto(s.Store),
			Status:            mapStatusToProto(s.Status),
			AutoRenew:         s.AutoRenew,
			IsSandbox:         s.Environment == store.SubscriptionEnvironmentSandbox,
		}
		if s.CurrentPeriodEnd != nil {
			as.CurrentPeriodEnd = timestamppb.New(*s.CurrentPeriodEnd)
		}
		info.Subscriptions = append(info.Subscriptions, as)
	}
	return info, nil
}

// --- enum mappers ---------------------------------------------------------

func mapSourceToProto(src string) billingv1.EntitlementSource {
	switch src {
	case entitlements.SourceStore:
		return billingv1.EntitlementSource_ENTITLEMENT_SOURCE_STORE
	case entitlements.SourceGrant:
		return billingv1.EntitlementSource_ENTITLEMENT_SOURCE_GRANT
	default:
		return billingv1.EntitlementSource_ENTITLEMENT_SOURCE_NONE
	}
}

func mapStoreToProto(s string) billingv1.Store {
	switch s {
	case store.SubscriptionStoreApple:
		return billingv1.Store_STORE_APPLE
	case store.SubscriptionStoreGoogle:
		return billingv1.Store_STORE_GOOGLE
	case store.SubscriptionStoreStripe:
		return billingv1.Store_STORE_STRIPE
	default:
		return billingv1.Store_STORE_UNSPECIFIED
	}
}

func mapStatusToProto(s string) billingv1.SubscriptionStatus {
	switch s {
	case store.SubscriptionStatusActive:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE
	case store.SubscriptionStatusTrialing:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_TRIALING
	case store.SubscriptionStatusInGracePeriod:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_IN_GRACE_PERIOD
	case store.SubscriptionStatusInBillingRetry:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_IN_BILLING_RETRY
	case store.SubscriptionStatusPaused:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_PAUSED
	case store.SubscriptionStatusExpired:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_EXPIRED
	case store.SubscriptionStatusRevoked:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_REVOKED
	default:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_UNSPECIFIED
	}
}

// GetCustomerInfo returns the signed-in user's entitlements and subscriptions.
func (h *Handler) GetCustomerInfo(ctx context.Context, req *connect.Request[billingv1.GetCustomerInfoRequest]) (*connect.Response[billingv1.GetCustomerInfoResponse], error) {
	project, user, err := h.auth.AuthenticateUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	info, err := h.customerInfo(ctx, project.ID, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&billingv1.GetCustomerInfoResponse{CustomerInfo: info}), nil
}
