package serverapi

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	serverv1 "github.com/aloisdeniel/moth/gen/moth/server/v1"
	"github.com/aloisdeniel/moth/gen/moth/server/v1/serverv1connect"
	"github.com/aloisdeniel/moth/internal/entitlements"
	"github.com/aloisdeniel/moth/internal/store"
)

// EntitlementStore is what the developer-backend entitlement service reads.
type EntitlementStore interface {
	store.UserStore
	store.EntitlementStore
	store.ProductStore
	store.SubscriptionStore
	store.SubscriptionGrantStore
}

var _ serverv1connect.EntitlementServiceHandler = (*EntitlementHandler)(nil)

// EntitlementHandler implements moth.server.v1.EntitlementService: it hands the
// developer's own backend the same derived entitlement set the client sees, so
// server-side feature gating never has to trust the client.
type EntitlementHandler struct {
	store EntitlementStore
	now   func() time.Time
}

// NewEntitlementHandler builds the service. now is injectable for tests; nil
// means time.Now.
func NewEntitlementHandler(st EntitlementStore, now func() time.Time) *EntitlementHandler {
	if now == nil {
		now = time.Now
	}
	return &EntitlementHandler{store: st, now: now}
}

// GetUserEntitlements returns the user's currently-held entitlements. A user
// with no subscription and no grant returns an empty set (the free `none`
// state), never an error.
func (h *EntitlementHandler) GetUserEntitlements(ctx context.Context, req *connect.Request[serverv1.GetUserEntitlementsRequest]) (*connect.Response[serverv1.GetUserEntitlementsResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	// Confirm the user exists so a bad id is NotFound, not a silent empty set.
	if _, err := h.store.GetUser(ctx, proj.ID, req.Msg.UserId); err != nil {
		return nil, userErr(err)
	}
	ents, err := h.store.ListEntitlements(ctx, proj.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	products, err := h.store.ListProducts(ctx, proj.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	subs, err := h.store.ListUserSubscriptions(ctx, proj.ID, req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	grants, err := h.store.ListUserGrants(ctx, proj.ID, req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	derived := entitlements.Derive(h.now(), ents, products, subs, grants)

	resp := &serverv1.GetUserEntitlementsResponse{}
	for _, e := range derived {
		pe := &serverv1.Entitlement{
			Identifier:        e.Identifier,
			Source:            e.Source,
			ProductIdentifier: e.ProductIdentifier,
			IsSandbox:         e.IsSandbox,
		}
		if !e.ExpireTime.IsZero() {
			pe.ExpireTime = timestamppb.New(e.ExpireTime)
		}
		resp.Entitlements = append(resp.Entitlements, pe)
	}
	return connect.NewResponse(resp), nil
}
