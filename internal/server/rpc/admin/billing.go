package adminrpc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/billing"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/store"
)

// BillingHandler implements the four moth.admin.v1 billing services:
// EntitlementService, ProductService, SubscriptionService and
// BillingCredentialsService. One struct backs all four so they share the store,
// master key and audit sink; server.New registers it under each service path.
type BillingHandler struct {
	store  Store
	master keys.MasterKey
	audit  *Auditor
	now    func() time.Time
}

var (
	_ adminv1connect.EntitlementServiceHandler        = (*BillingHandler)(nil)
	_ adminv1connect.ProductServiceHandler            = (*BillingHandler)(nil)
	_ adminv1connect.SubscriptionServiceHandler       = (*BillingHandler)(nil)
	_ adminv1connect.BillingCredentialsServiceHandler = (*BillingHandler)(nil)
)

// NewBillingHandler builds the admin billing services. now is injectable for
// tests; nil means time.Now.
func NewBillingHandler(st Store, master keys.MasterKey, auditor *Auditor, now func() time.Time) *BillingHandler {
	if now == nil {
		now = time.Now
	}
	return &BillingHandler{store: st, master: master, audit: auditor, now: now}
}

// identifierRe constrains entitlement/product identifiers to a stable,
// URL/analytics-safe shape the app depends on.
var identifierRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

func validIdentifier(id string) error {
	if !identifierRe.MatchString(id) {
		return errors.New("identifier must be 1-64 chars of lowercase letters, digits, '.', '_' or '-'")
	}
	return nil
}

func (h *BillingHandler) requireProject(ctx context.Context, projectID string) error {
	if _, err := h.store.GetProject(ctx, projectID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return connect.NewError(connect.CodeNotFound, errors.New("project not found"))
		}
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
}

func billingErr(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// --- EntitlementService ---------------------------------------------------

func (h *BillingHandler) ListEntitlements(ctx context.Context, req *connect.Request[adminv1.ListEntitlementsRequest]) (*connect.Response[adminv1.ListEntitlementsResponse], error) {
	if err := h.requireProject(ctx, req.Msg.ProjectId); err != nil {
		return nil, err
	}
	ents, err := h.store.ListEntitlements(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListEntitlementsResponse{}
	for _, e := range ents {
		resp.Entitlements = append(resp.Entitlements, entitlementProto(e))
	}
	return connect.NewResponse(resp), nil
}

func (h *BillingHandler) CreateEntitlement(ctx context.Context, req *connect.Request[adminv1.CreateEntitlementRequest]) (*connect.Response[adminv1.CreateEntitlementResponse], error) {
	if err := h.requireProject(ctx, req.Msg.ProjectId); err != nil {
		return nil, err
	}
	if err := validIdentifier(req.Msg.Identifier); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	now := h.now()
	e := store.Entitlement{
		ID:          NewID(),
		ProjectID:   req.Msg.ProjectId,
		Identifier:  req.Msg.Identifier,
		DisplayName: req.Msg.DisplayName,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.store.CreateEntitlement(ctx, e); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists,
				errors.New("an entitlement with this identifier already exists"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.audit.record(ctx, entry{Action: ActionEntitlementCreate, TargetType: "entitlement", TargetID: e.ID,
		ProjectID: e.ProjectID, Summary: fmt.Sprintf("Created entitlement %q", e.Identifier)})
	return connect.NewResponse(&adminv1.CreateEntitlementResponse{Entitlement: entitlementProto(e)}), nil
}

func (h *BillingHandler) UpdateEntitlement(ctx context.Context, req *connect.Request[adminv1.UpdateEntitlementRequest]) (*connect.Response[adminv1.UpdateEntitlementResponse], error) {
	e, err := h.store.GetEntitlement(ctx, req.Msg.ProjectId, req.Msg.Id)
	if err != nil {
		return nil, billingErr(err)
	}
	e.DisplayName = req.Msg.DisplayName
	e.UpdatedAt = h.now()
	if err := h.store.UpdateEntitlement(ctx, e); err != nil {
		return nil, billingErr(err)
	}
	h.audit.record(ctx, entry{Action: ActionEntitlementUpdate, TargetType: "entitlement", TargetID: e.ID,
		ProjectID: e.ProjectID, Summary: fmt.Sprintf("Updated entitlement %q", e.Identifier)})
	return connect.NewResponse(&adminv1.UpdateEntitlementResponse{Entitlement: entitlementProto(e)}), nil
}

func (h *BillingHandler) DeleteEntitlement(ctx context.Context, req *connect.Request[adminv1.DeleteEntitlementRequest]) (*connect.Response[adminv1.DeleteEntitlementResponse], error) {
	if err := h.store.DeleteEntitlement(ctx, req.Msg.ProjectId, req.Msg.Id); err != nil {
		return nil, billingErr(err)
	}
	h.audit.record(ctx, entry{Action: ActionEntitlementDelete, TargetType: "entitlement", TargetID: req.Msg.Id,
		ProjectID: req.Msg.ProjectId, Summary: "Deleted entitlement"})
	return connect.NewResponse(&adminv1.DeleteEntitlementResponse{}), nil
}

func entitlementProto(e store.Entitlement) *adminv1.Entitlement {
	return &adminv1.Entitlement{
		Id:          e.ID,
		Identifier:  e.Identifier,
		DisplayName: e.DisplayName,
		CreateTime:  timestamppb.New(e.CreatedAt),
		UpdateTime:  timestamppb.New(e.UpdatedAt),
	}
}

// --- ProductService -------------------------------------------------------

func (h *BillingHandler) ListProducts(ctx context.Context, req *connect.Request[adminv1.ListProductsRequest]) (*connect.Response[adminv1.ListProductsResponse], error) {
	if err := h.requireProject(ctx, req.Msg.ProjectId); err != nil {
		return nil, err
	}
	products, err := h.store.ListProducts(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListProductsResponse{}
	for _, p := range products {
		resp.Products = append(resp.Products, productProto(p))
	}
	return connect.NewResponse(resp), nil
}

func (h *BillingHandler) GetProduct(ctx context.Context, req *connect.Request[adminv1.GetProductRequest]) (*connect.Response[adminv1.GetProductResponse], error) {
	p, err := h.store.GetProduct(ctx, req.Msg.ProjectId, req.Msg.Id)
	if err != nil {
		return nil, billingErr(err)
	}
	return connect.NewResponse(&adminv1.GetProductResponse{Product: productProto(p)}), nil
}

func (h *BillingHandler) CreateProduct(ctx context.Context, req *connect.Request[adminv1.CreateProductRequest]) (*connect.Response[adminv1.CreateProductResponse], error) {
	if err := h.requireProject(ctx, req.Msg.ProjectId); err != nil {
		return nil, err
	}
	if req.Msg.Product == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("product is required"))
	}
	if err := validIdentifier(req.Msg.Product.Identifier); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	now := h.now()
	p := productFromProto(req.Msg.ProjectId, req.Msg.Product)
	p.ID = NewID()
	p.CreatedAt = now
	p.UpdatedAt = now
	if err := h.store.CreateProduct(ctx, p); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists,
				errors.New("a product with this identifier already exists"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.audit.record(ctx, entry{Action: ActionProductCreate, TargetType: "product", TargetID: p.ID,
		ProjectID: p.ProjectID, Summary: fmt.Sprintf("Created product %q", p.Identifier)})
	stored, err := h.store.GetProduct(ctx, p.ProjectID, p.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.CreateProductResponse{Product: productProto(stored)}), nil
}

func (h *BillingHandler) UpdateProduct(ctx context.Context, req *connect.Request[adminv1.UpdateProductRequest]) (*connect.Response[adminv1.UpdateProductResponse], error) {
	if req.Msg.Product == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("product is required"))
	}
	existing, err := h.store.GetProduct(ctx, req.Msg.ProjectId, req.Msg.Id)
	if err != nil {
		return nil, billingErr(err)
	}
	if err := validIdentifier(req.Msg.Product.Identifier); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	p := productFromProto(req.Msg.ProjectId, req.Msg.Product)
	p.ID = existing.ID
	p.CreatedAt = existing.CreatedAt
	p.UpdatedAt = h.now()
	if err := h.store.UpdateProduct(ctx, p); err != nil {
		return nil, billingErr(err)
	}
	h.audit.record(ctx, entry{Action: ActionProductUpdate, TargetType: "product", TargetID: p.ID,
		ProjectID: p.ProjectID, Summary: fmt.Sprintf("Updated product %q", p.Identifier)})
	stored, err := h.store.GetProduct(ctx, p.ProjectID, p.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.UpdateProductResponse{Product: productProto(stored)}), nil
}

func (h *BillingHandler) DeleteProduct(ctx context.Context, req *connect.Request[adminv1.DeleteProductRequest]) (*connect.Response[adminv1.DeleteProductResponse], error) {
	if err := h.store.DeleteProduct(ctx, req.Msg.ProjectId, req.Msg.Id); err != nil {
		return nil, billingErr(err)
	}
	h.audit.record(ctx, entry{Action: ActionProductDelete, TargetType: "product", TargetID: req.Msg.Id,
		ProjectID: req.Msg.ProjectId, Summary: "Deleted product"})
	return connect.NewResponse(&adminv1.DeleteProductResponse{}), nil
}

func productProto(p store.Product) *adminv1.Product {
	return &adminv1.Product{
		Id:                     p.ID,
		Identifier:             p.Identifier,
		DisplayName:            p.DisplayName,
		AppleProductId:         p.AppleProductID,
		GoogleProductId:        p.GoogleProductID,
		StripePriceId:          p.StripePriceID,
		StripeProductId:        p.StripeProductID,
		BillingPeriod:          p.BillingPeriod,
		PriceAmountMicros:      p.PriceAmountMicros,
		Currency:               p.Currency,
		TrialPeriod:            p.TrialPeriod,
		IntroPriceAmountMicros: p.IntroPriceAmountMicros,
		IntroPeriod:            p.IntroPeriod,
		Offering:               p.Offering,
		SortOrder:              int32(p.SortOrder),
		EntitlementIds:         p.EntitlementIDs,
		CreateTime:             timestamppb.New(p.CreatedAt),
		UpdateTime:             timestamppb.New(p.UpdatedAt),
	}
}

func productFromProto(projectID string, p *adminv1.Product) store.Product {
	return store.Product{
		ProjectID:              projectID,
		Identifier:             p.Identifier,
		DisplayName:            p.DisplayName,
		AppleProductID:         p.AppleProductId,
		GoogleProductID:        p.GoogleProductId,
		StripePriceID:          p.StripePriceId,
		StripeProductID:        p.StripeProductId,
		BillingPeriod:          p.BillingPeriod,
		PriceAmountMicros:      p.PriceAmountMicros,
		Currency:               p.Currency,
		TrialPeriod:            p.TrialPeriod,
		IntroPriceAmountMicros: p.IntroPriceAmountMicros,
		IntroPeriod:            p.IntroPeriod,
		Offering:               p.Offering,
		SortOrder:              int(p.SortOrder),
		EntitlementIDs:         p.EntitlementIds,
	}
}

// --- SubscriptionService --------------------------------------------------

func (h *BillingHandler) ListUserSubscriptions(ctx context.Context, req *connect.Request[adminv1.ListUserSubscriptionsRequest]) (*connect.Response[adminv1.ListUserSubscriptionsResponse], error) {
	if err := h.requireProject(ctx, req.Msg.ProjectId); err != nil {
		return nil, err
	}
	subs, err := h.store.ListUserSubscriptions(ctx, req.Msg.ProjectId, req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	grants, err := h.store.ListUserGrants(ctx, req.Msg.ProjectId, req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListUserSubscriptionsResponse{}
	for _, s := range subs {
		resp.Subscriptions = append(resp.Subscriptions, subscriptionProto(s))
	}
	for _, g := range grants {
		resp.Grants = append(resp.Grants, grantProto(g))
	}
	return connect.NewResponse(resp), nil
}

func (h *BillingHandler) GetUserSubscription(ctx context.Context, req *connect.Request[adminv1.GetUserSubscriptionRequest]) (*connect.Response[adminv1.GetUserSubscriptionResponse], error) {
	s, err := h.store.GetSubscription(ctx, req.Msg.ProjectId, req.Msg.Id)
	if err != nil {
		return nil, billingErr(err)
	}
	return connect.NewResponse(&adminv1.GetUserSubscriptionResponse{Subscription: subscriptionProto(s)}), nil
}

func (h *BillingHandler) GrantEntitlement(ctx context.Context, req *connect.Request[adminv1.GrantEntitlementRequest]) (*connect.Response[adminv1.GrantEntitlementResponse], error) {
	if err := h.requireProject(ctx, req.Msg.ProjectId); err != nil {
		return nil, err
	}
	// The entitlement must exist in the project (a bad id is InvalidArgument,
	// not a dangling grant).
	ent, err := h.store.GetEntitlement(ctx, req.Msg.ProjectId, req.Msg.EntitlementId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unknown entitlement"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if _, err := h.store.GetUser(ctx, req.Msg.ProjectId, req.Msg.UserId); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	grantedBy := ""
	if admin, ok := AdminFromContext(ctx); ok {
		grantedBy = admin.Email
	}
	g := store.SubscriptionGrant{
		ID:            NewID(),
		ProjectID:     req.Msg.ProjectId,
		UserID:        req.Msg.UserId,
		EntitlementID: req.Msg.EntitlementId,
		Reason:        req.Msg.Reason,
		GrantedBy:     grantedBy,
		CreatedAt:     h.now(),
	}
	if req.Msg.ExpireTime != nil {
		t := req.Msg.ExpireTime.AsTime()
		g.ExpiresAt = &t
	}
	if err := h.store.CreateSubscriptionGrant(ctx, g); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.audit.record(ctx, entry{Action: ActionGrantCreate, TargetType: "user", TargetID: g.UserID,
		ProjectID: g.ProjectID, Summary: fmt.Sprintf("Granted entitlement %q to user %s", ent.Identifier, g.UserID)})
	h.emitGrantEvent(ctx, g, store.SubscriptionEventGranted)
	return connect.NewResponse(&adminv1.GrantEntitlementResponse{Grant: grantProto(g)}), nil
}

func (h *BillingHandler) RevokeGrant(ctx context.Context, req *connect.Request[adminv1.RevokeGrantRequest]) (*connect.Response[adminv1.RevokeGrantResponse], error) {
	if err := h.store.RevokeSubscriptionGrant(ctx, req.Msg.ProjectId, req.Msg.GrantId, h.now()); err != nil {
		return nil, billingErr(err)
	}
	g, err := h.store.GetSubscriptionGrant(ctx, req.Msg.ProjectId, req.Msg.GrantId)
	if err != nil {
		return nil, billingErr(err)
	}
	h.audit.record(ctx, entry{Action: ActionGrantRevoke, TargetType: "user", TargetID: g.UserID,
		ProjectID: g.ProjectID, Summary: fmt.Sprintf("Revoked grant %s from user %s", g.ID, g.UserID)})
	h.emitGrantEvent(ctx, g, store.SubscriptionEventRevoked)
	return connect.NewResponse(&adminv1.RevokeGrantResponse{Grant: grantProto(g)}), nil
}

// emitGrantEvent records a grant/revoke into the revenue event stream (M14).
func (h *BillingHandler) emitGrantEvent(ctx context.Context, g store.SubscriptionGrant, eventType string) {
	e := store.SubscriptionEvent{
		ID:        NewID(),
		ProjectID: g.ProjectID,
		Type:      eventType,
		UserID:    g.UserID,
		CreatedAt: h.now(),
	}
	if err := h.store.InsertSubscriptionEvent(ctx, e); err != nil {
		// Best effort; the grant itself already committed and was audited.
		_ = err
	}
}

func subscriptionProto(s store.Subscription) *adminv1.Subscription {
	sub := &adminv1.Subscription{
		Id:                 s.ID,
		UserId:             s.UserID,
		Store:              adminStoreProto(s.Store),
		ProductId:          s.ProductID,
		Status:             adminStatusProto(s.Status),
		AutoRenew:          s.AutoRenew,
		Environment:        s.Environment,
		StoreTransactionId: s.StoreTransactionID,
		CreateTime:         timestamppb.New(s.CreatedAt),
		UpdateTime:         timestamppb.New(s.UpdatedAt),
	}
	if s.CurrentPeriodEnd != nil {
		sub.CurrentPeriodEnd = timestamppb.New(*s.CurrentPeriodEnd)
	}
	return sub
}

func grantProto(g store.SubscriptionGrant) *adminv1.Grant {
	grant := &adminv1.Grant{
		Id:            g.ID,
		EntitlementId: g.EntitlementID,
		Reason:        g.Reason,
		GrantedBy:     g.GrantedBy,
		CreateTime:    timestamppb.New(g.CreatedAt),
	}
	if g.ExpiresAt != nil {
		grant.ExpireTime = timestamppb.New(*g.ExpiresAt)
	}
	if g.RevokedAt != nil {
		grant.RevokeTime = timestamppb.New(*g.RevokedAt)
	}
	return grant
}

func adminStoreProto(s string) adminv1.Store {
	switch s {
	case store.SubscriptionStoreApple:
		return adminv1.Store_STORE_APPLE
	case store.SubscriptionStoreGoogle:
		return adminv1.Store_STORE_GOOGLE
	case store.SubscriptionStoreStripe:
		return adminv1.Store_STORE_STRIPE
	default:
		return adminv1.Store_STORE_UNSPECIFIED
	}
}

func adminStatusProto(s string) adminv1.SubscriptionStatus {
	switch s {
	case store.SubscriptionStatusActive:
		return adminv1.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE
	case store.SubscriptionStatusTrialing:
		return adminv1.SubscriptionStatus_SUBSCRIPTION_STATUS_TRIALING
	case store.SubscriptionStatusInGracePeriod:
		return adminv1.SubscriptionStatus_SUBSCRIPTION_STATUS_IN_GRACE_PERIOD
	case store.SubscriptionStatusInBillingRetry:
		return adminv1.SubscriptionStatus_SUBSCRIPTION_STATUS_IN_BILLING_RETRY
	case store.SubscriptionStatusPaused:
		return adminv1.SubscriptionStatus_SUBSCRIPTION_STATUS_PAUSED
	case store.SubscriptionStatusExpired:
		return adminv1.SubscriptionStatus_SUBSCRIPTION_STATUS_EXPIRED
	case store.SubscriptionStatusRevoked:
		return adminv1.SubscriptionStatus_SUBSCRIPTION_STATUS_REVOKED
	default:
		return adminv1.SubscriptionStatus_SUBSCRIPTION_STATUS_UNSPECIFIED
	}
}

// --- BillingCredentialsService --------------------------------------------

func (h *BillingHandler) GetBillingCredentials(ctx context.Context, req *connect.Request[adminv1.GetBillingCredentialsRequest]) (*connect.Response[adminv1.GetBillingCredentialsResponse], error) {
	if err := h.requireProject(ctx, req.Msg.ProjectId); err != nil {
		return nil, err
	}
	cred, err := h.store.GetBillingCredentials(ctx, req.Msg.ProjectId)
	if errors.Is(err, store.ErrNotFound) {
		// No credentials yet: return empty configs (all has_* false).
		return connect.NewResponse(&adminv1.GetBillingCredentialsResponse{
			Apple:  &adminv1.AppleBillingConfig{},
			Google: &adminv1.GoogleBillingConfig{},
			Stripe: &adminv1.StripeBillingConfig{},
		}), nil
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.GetBillingCredentialsResponse{
		Apple:  appleConfigProto(cred),
		Google: googleConfigProto(cred),
		Stripe: stripeConfigProto(cred),
	}), nil
}

func (h *BillingHandler) UpdateBillingCredentials(ctx context.Context, req *connect.Request[adminv1.UpdateBillingCredentialsRequest]) (*connect.Response[adminv1.UpdateBillingCredentialsResponse], error) {
	if err := h.requireProject(ctx, req.Msg.ProjectId); err != nil {
		return nil, err
	}
	// Seed from the stored row (if any) and merge here: the store upsert writes
	// full rows (every non-secret column), so a partial request — say a
	// Stripe-only update from `moth setup billing --stripe-secret-key` — must
	// not blank the Apple/Google configuration it does not mention. A store's
	// field-group is only overwritten when its message is present.
	cred, err := h.store.GetBillingCredentials(ctx, req.Msg.ProjectId)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	now := h.now()
	if errors.Is(err, store.ErrNotFound) {
		cred = store.BillingCredentials{CreatedAt: now}
	}
	cred.ProjectID = req.Msg.ProjectId
	cred.UpdatedAt = now

	if a := req.Msg.Apple; a != nil {
		cred.AppleIAPKeyID = a.IapKeyId
		cred.AppleIAPIssuerID = a.IapIssuerId
		cred.AppleBundleID = a.BundleId
		cred.AppleAppAppleID = a.AppAppleId
		// "" keeps the stored notification URL (write-only-ish: only the CLI
		// records it, after a successful App Store Server Notification register).
		if a.NotificationUrl != "" {
			cred.AppleNotificationURL = a.NotificationUrl
		}
		if a.IapKeyP8 != "" {
			// Validate the .p8 parses before storing it encrypted.
			if _, err := billing.ParseP8([]byte(a.IapKeyP8)); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument,
					fmt.Errorf("apple .p8 key: %w", err))
			}
			enc, err := h.master.Encrypt([]byte(a.IapKeyP8))
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			cred.AppleIAPKeyEnc = enc
		}
		if a.NotificationSecret != "" {
			enc, err := h.master.Encrypt([]byte(a.NotificationSecret))
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			cred.AppleNotificationSecretEnc = enc
		}
	}
	if g := req.Msg.Google; g != nil {
		cred.GooglePackageName = g.PackageName
		cred.GooglePubsubTopic = g.PubsubTopic
		if g.ServiceAccountJson != "" {
			if _, err := billing.ParseServiceAccount([]byte(g.ServiceAccountJson)); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument,
					fmt.Errorf("google service account: %w", err))
			}
			enc, err := h.master.Encrypt([]byte(g.ServiceAccountJson))
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			cred.GoogleServiceAccountEnc = enc
		}
		if g.RtdnSecret != "" {
			enc, err := h.master.Encrypt([]byte(g.RtdnSecret))
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			cred.GoogleRTDNSecretEnc = enc
		}
	}
	if sc := req.Msg.Stripe; sc != nil {
		// "" keeps the stored endpoint id (only recorded after a successful
		// webhook registration, mirroring AppleNotificationURL).
		if sc.WebhookEndpointId != "" {
			cred.StripeWebhookEndpointID = sc.WebhookEndpointId
		}
		if sc.SecretKey != "" {
			// Sanity-check the key shape before storing it encrypted (the
			// stripe analogue of ParseP8/ParseServiceAccount validation).
			if err := validateStripeSecretKey(sc.SecretKey); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			enc, err := h.master.Encrypt([]byte(sc.SecretKey))
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			cred.StripeSecretKeyEnc = enc
		}
		if sc.WebhookSecret != "" {
			if !strings.HasPrefix(sc.WebhookSecret, "whsec_") {
				return nil, connect.NewError(connect.CodeInvalidArgument,
					errors.New(`stripe webhook secret: must start with "whsec_"`))
			}
			enc, err := h.master.Encrypt([]byte(sc.WebhookSecret))
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			cred.StripeWebhookSecretEnc = enc
		}
	}
	if err := h.store.UpsertBillingCredentials(ctx, cred); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.audit.record(ctx, entry{Action: ActionBillingCredsUpdate, TargetType: "project", TargetID: req.Msg.ProjectId,
		ProjectID: req.Msg.ProjectId, Summary: "Updated billing credentials"})
	// Re-read so the response reports accurate has_* indicators.
	stored, err := h.store.GetBillingCredentials(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.UpdateBillingCredentialsResponse{
		Apple:  appleConfigProto(stored),
		Google: googleConfigProto(stored),
		Stripe: stripeConfigProto(stored),
	}), nil
}

// validateStripeSecretKey sanity-checks a Stripe API key shape: a standard
// secret key (sk_) or a restricted key (rk_ — recommended, plan/17), in test
// or live mode. Publishable keys (pk_) cannot call the server-side APIs.
func validateStripeSecretKey(key string) error {
	for _, prefix := range []string{"sk_test_", "sk_live_", "rk_test_", "rk_live_"} {
		if strings.HasPrefix(key, prefix) && len(key) > len(prefix) {
			return nil
		}
	}
	return errors.New(`stripe secret key: must start with "sk_test_", "sk_live_", "rk_test_" or "rk_live_"`)
}

func appleConfigProto(c store.BillingCredentials) *adminv1.AppleBillingConfig {
	return &adminv1.AppleBillingConfig{
		IapKeyId:              c.AppleIAPKeyID,
		IapIssuerId:           c.AppleIAPIssuerID,
		HasIapKey:             len(c.AppleIAPKeyEnc) > 0,
		BundleId:              c.AppleBundleID,
		AppAppleId:            c.AppleAppAppleID,
		HasNotificationSecret: len(c.AppleNotificationSecretEnc) > 0,
		NotificationUrl:       c.AppleNotificationURL,
	}
}

func googleConfigProto(c store.BillingCredentials) *adminv1.GoogleBillingConfig {
	return &adminv1.GoogleBillingConfig{
		HasServiceAccount: len(c.GoogleServiceAccountEnc) > 0,
		PackageName:       c.GooglePackageName,
		PubsubTopic:       c.GooglePubsubTopic,
		HasRtdnSecret:     len(c.GoogleRTDNSecretEnc) > 0,
	}
}

func stripeConfigProto(c store.BillingCredentials) *adminv1.StripeBillingConfig {
	return &adminv1.StripeBillingConfig{
		HasSecretKey:      len(c.StripeSecretKeyEnc) > 0,
		HasWebhookSecret:  len(c.StripeWebhookSecretEnc) > 0,
		WebhookEndpointId: c.StripeWebhookEndpointID,
	}
}
