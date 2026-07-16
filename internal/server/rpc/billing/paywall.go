package billingrpc

import (
	"context"
	"errors"
	"sort"

	"connectrpc.com/connect"

	billingv1 "github.com/aloisdeniel/moth/gen/moth/billing/v1"
	"github.com/aloisdeniel/moth/internal/paywall"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// DefaultPaywallRevision is the revision id reported for projects rendering
// the built-in default paywall config. A stable non-empty sentinel so the
// GetPaywall caching contract ("paywall omitted when known_paywall_revision
// matches") also works for never-customized projects: an empty
// known_paywall_revision (first call) never matches.
const DefaultPaywallRevision = "default"

// errNoProject signals the publishable-key interceptor did not populate the
// request's project (a wiring bug — the interceptor guarantees it).
var errNoProject = errors.New("no project in context")

// projectPaywall resolves a project's paywall config and the revision id
// identifying it: the stored config, or the built-in default (with
// DefaultPaywallRevision) when the project never customized anything or the
// stored document cannot be parsed. Mirrors authrpc.ProjectTheme.
func projectPaywall(p store.Project) (paywall.Config, string) {
	if p.Paywall == "" {
		return paywall.Default(), DefaultPaywallRevision
	}
	c, err := paywall.Parse([]byte(p.Paywall))
	if err != nil {
		return paywall.Default(), DefaultPaywallRevision
	}
	rev := p.PaywallRevisionID
	if rev == "" {
		rev = DefaultPaywallRevision
	}
	return c, rev
}

// offeringTag normalizes an offering id: empty selects the default offering.
func offeringTag(id string) string {
	if id == "" {
		return store.DefaultOffering
	}
	return id
}

// productOffering returns the offering a product belongs to (empty tag == the
// default offering).
func productOffering(p store.Product) string {
	if p.Offering == "" {
		return store.DefaultOffering
	}
	return p.Offering
}

// GetOfferings returns an offering's products for the paywall to display.
// Publishable-key only (no Bearer): the paywall renders before sign-in.
func (h *Handler) GetOfferings(ctx context.Context, req *connect.Request[billingv1.GetOfferingsRequest]) (*connect.Response[billingv1.GetOfferingsResponse], error) {
	project, ok := authrpc.ProjectFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errNoProject)
	}
	tag := offeringTag(req.Msg.Offering)

	products, err := h.store.ListProducts(ctx, project.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ents, err := h.store.ListEntitlements(ctx, project.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Map entitlement id -> stable identifier so the paywall labels what a tier
	// unlocks with "pro", not an opaque uuid.
	identByEnt := make(map[string]string, len(ents))
	for _, e := range ents {
		identByEnt[e.ID] = e.Identifier
	}
	// The highlighted tier comes from the paywall config (a display concern),
	// exposed here as a per-product convenience flag.
	cfg, _ := projectPaywall(project)

	var members []store.Product
	for _, p := range products {
		if productOffering(p) == tag {
			members = append(members, p)
		}
	}
	sort.SliceStable(members, func(i, j int) bool {
		if members[i].SortOrder != members[j].SortOrder {
			return members[i].SortOrder < members[j].SortOrder
		}
		return members[i].CreatedAt.Before(members[j].CreatedAt)
	})

	off := &billingv1.Offering{Identifier: tag, IsDefault: tag == store.DefaultOffering}
	for _, p := range members {
		op := &billingv1.OfferingProduct{
			Identifier:             p.Identifier,
			DisplayName:            p.DisplayName,
			AppleProductId:         p.AppleProductID,
			GoogleProductId:        p.GoogleProductID,
			BillingPeriod:          p.BillingPeriod,
			PriceAmountMicros:      p.PriceAmountMicros,
			Currency:               p.Currency,
			TrialPeriod:            p.TrialPeriod,
			IntroPriceAmountMicros: p.IntroPriceAmountMicros,
			IntroPeriod:            p.IntroPeriod,
			SortOrder:              int32(p.SortOrder),
			Highlighted:            cfg.HighlightedIdentifier != "" && cfg.HighlightedIdentifier == p.Identifier,
		}
		for _, eid := range p.EntitlementIDs {
			if ident, ok := identByEnt[eid]; ok {
				op.Entitlements = append(op.Entitlements, ident)
			}
		}
		off.Products = append(off.Products, op)
	}
	return connect.NewResponse(&billingv1.GetOfferingsResponse{Offering: off}), nil
}

// GetPaywall returns the project's public paywall config, honoring the
// stale-while-revalidate caching contract (body omitted when the client's
// known revision still matches). Publishable-key only, like GetProjectConfig.
func (h *Handler) GetPaywall(ctx context.Context, req *connect.Request[billingv1.GetPaywallRequest]) (*connect.Response[billingv1.GetPaywallResponse], error) {
	project, ok := authrpc.ProjectFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errNoProject)
	}
	cfg, rev := projectPaywall(project)
	resp := &billingv1.GetPaywallResponse{}
	if req.Msg.KnownPaywallRevision != rev {
		resp.Paywall = publicPaywall(cfg, rev)
	}
	return connect.NewResponse(resp), nil
}

// publicPaywall converts the config into the render-ready public message.
func publicPaywall(c paywall.Config, rev string) *billingv1.Paywall {
	return &billingv1.Paywall{
		RevisionId:                   rev,
		Headline:                     c.Headline,
		Subtitle:                     c.Subtitle,
		Benefits:                     c.Benefits,
		Offering:                     c.Offering,
		HighlightedProductIdentifier: c.HighlightedIdentifier,
		Layout:                       paywallLayoutProto(c.Layout),
		TermsUrl:                     c.Legal.TermsURL,
		PrivacyUrl:                   c.Legal.PrivacyURL,
	}
}

func paywallLayoutProto(layout string) billingv1.PaywallLayout {
	switch layout {
	case paywall.LayoutTiles:
		return billingv1.PaywallLayout_PAYWALL_LAYOUT_TILES
	case paywall.LayoutList:
		return billingv1.PaywallLayout_PAYWALL_LAYOUT_LIST
	case paywall.LayoutCompact:
		return billingv1.PaywallLayout_PAYWALL_LAYOUT_COMPACT
	default:
		return billingv1.PaywallLayout_PAYWALL_LAYOUT_UNSPECIFIED
	}
}
