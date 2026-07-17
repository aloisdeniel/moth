package billingrpc

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	billingv1 "github.com/aloisdeniel/moth/gen/moth/billing/v1"
	"github.com/aloisdeniel/moth/internal/paywall"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// ctxWith returns a context scoped to p, as the publishable-key interceptor
// would, so the paywall/offerings handlers resolve the right project.
func ctxWith(p store.Project) context.Context {
	return authrpc.WithProject(context.Background(), p)
}

// customizePaywall installs cfg as f.project's paywall and returns the
// project reloaded with its new PaywallRevisionID, plus that id.
func (f *fixture) customizePaywall(cfg paywall.Config) (store.Project, string) {
	f.t.Helper()
	raw, err := paywall.Encode(cfg)
	if err != nil {
		f.t.Fatal(err)
	}
	rev := store.PaywallRevision{
		ID: authrpc.NewID(), ProjectID: f.project.ID,
		Paywall: raw, CreatedAt: f.now,
	}
	if err := f.st.SetProjectPaywall(f.ctx(), rev, f.project.PaywallRevisionID); err != nil {
		f.t.Fatal(err)
	}
	p, err := f.st.GetProject(context.Background(), f.project.ID)
	if err != nil {
		f.t.Fatal(err)
	}
	return p, rev.ID
}

func TestGetPaywallDefaultProjectReturnsBodyWithSentinel(t *testing.T) {
	f := newFixture(t)
	// A first-launch client sends an empty known revision.
	resp, err := f.h.GetPaywall(f.ctx(), connect.NewRequest(&billingv1.GetPaywallRequest{}))
	if err != nil {
		t.Fatalf("GetPaywall: %v", err)
	}
	if resp.Msg.Paywall == nil {
		t.Fatal("default project must return a paywall body on the first call")
	}
	if got := resp.Msg.Paywall.RevisionId; got != DefaultPaywallRevision {
		t.Errorf("revision = %q, want %q", got, DefaultPaywallRevision)
	}
	if got := resp.Msg.Paywall.Headline; got != paywall.Default().Headline {
		t.Errorf("headline = %q, want default", got)
	}
}

func TestGetPaywallOmitsBodyWhenRevisionMatches(t *testing.T) {
	f := newFixture(t)
	// The client already holds the default revision: the body is omitted.
	resp, err := f.h.GetPaywall(f.ctx(), connect.NewRequest(&billingv1.GetPaywallRequest{
		KnownPaywallRevision: DefaultPaywallRevision,
	}))
	if err != nil {
		t.Fatalf("GetPaywall: %v", err)
	}
	if resp.Msg.Paywall != nil {
		t.Fatalf("body must be omitted when the revision matches, got %+v", resp.Msg.Paywall)
	}
}

func TestGetPaywallCustomizedProjectRevisionRoundTrip(t *testing.T) {
	f := newFixture(t)
	cfg := paywall.Default()
	cfg.Headline = "Custom headline"
	cfg.Offering = "premium"
	cfg.HighlightedIdentifier = "monthly"
	cfg.Layout = paywall.LayoutCompact
	p, revID := f.customizePaywall(cfg)
	ctx := ctxWith(p)

	// Empty known revision (a never-synced client) gets the full custom body.
	resp, err := f.h.GetPaywall(ctx, connect.NewRequest(&billingv1.GetPaywallRequest{}))
	if err != nil {
		t.Fatalf("GetPaywall: %v", err)
	}
	if resp.Msg.Paywall == nil {
		t.Fatal("expected a paywall body for a customized project")
	}
	if got := resp.Msg.Paywall.RevisionId; got != revID {
		t.Errorf("revision = %q, want %q", got, revID)
	}
	if got := resp.Msg.Paywall.Headline; got != "Custom headline" {
		t.Errorf("headline = %q, want %q", got, "Custom headline")
	}
	if got := resp.Msg.Paywall.Offering; got != "premium" {
		t.Errorf("offering = %q, want premium", got)
	}
	if resp.Msg.Paywall.Layout != billingv1.PaywallLayout_PAYWALL_LAYOUT_COMPACT {
		t.Errorf("layout = %v, want COMPACT", resp.Msg.Paywall.Layout)
	}

	// A client that already holds this revision gets the body omitted.
	resp2, err := f.h.GetPaywall(ctx, connect.NewRequest(&billingv1.GetPaywallRequest{
		KnownPaywallRevision: revID,
	}))
	if err != nil {
		t.Fatalf("GetPaywall(known): %v", err)
	}
	if resp2.Msg.Paywall != nil {
		t.Fatalf("body must be omitted when the custom revision matches, got %+v", resp2.Msg.Paywall)
	}
}

func TestGetOfferingsDefaultReturnsProductsAndEntitlements(t *testing.T) {
	f := newFixture(t)
	// Highlight the fixture's "monthly" tier via the paywall config.
	cfg := paywall.Default()
	cfg.HighlightedIdentifier = "monthly"
	p, _ := f.customizePaywall(cfg)

	resp, err := f.h.GetOfferings(ctxWith(p), connect.NewRequest(&billingv1.GetOfferingsRequest{}))
	if err != nil {
		t.Fatalf("GetOfferings: %v", err)
	}
	off := resp.Msg.Offering
	if off == nil || !off.IsDefault || off.Identifier != store.DefaultOffering {
		t.Fatalf("expected the default offering, got %+v", off)
	}
	if len(off.Products) != 1 {
		t.Fatalf("want 1 product, got %d", len(off.Products))
	}
	prod := off.Products[0]
	if prod.Identifier != "monthly" {
		t.Errorf("identifier = %q, want monthly", prod.Identifier)
	}
	if !prod.Highlighted {
		t.Error("monthly should be highlighted by the paywall config")
	}
	if len(prod.Entitlements) != 1 || prod.Entitlements[0] != "pro" {
		t.Errorf("entitlements = %v, want [pro]", prod.Entitlements)
	}
}

func TestGetOfferingsNonDefaultTagIsIsolated(t *testing.T) {
	f := newFixture(t)
	// A product in a non-default offering "premium".
	prem := store.Product{
		ID: authrpc.NewID(), ProjectID: f.project.ID, Identifier: "prem-monthly",
		DisplayName: "Premium Monthly", Offering: "premium",
		EntitlementIDs: []string{f.entPro.ID}, CreatedAt: f.now, UpdatedAt: f.now,
	}
	if err := f.st.CreateProduct(context.Background(), prem); err != nil {
		t.Fatal(err)
	}

	// The premium offering returns only its own product...
	resp, err := f.h.GetOfferings(f.ctx(), connect.NewRequest(&billingv1.GetOfferingsRequest{Offering: "premium"}))
	if err != nil {
		t.Fatalf("GetOfferings(premium): %v", err)
	}
	if off := resp.Msg.Offering; off.Identifier != "premium" || off.IsDefault ||
		len(off.Products) != 1 || off.Products[0].Identifier != "prem-monthly" {
		t.Fatalf("premium offering = %+v", off)
	}

	// ...and the default offering does not include it.
	def, err := f.h.GetOfferings(f.ctx(), connect.NewRequest(&billingv1.GetOfferingsRequest{}))
	if err != nil {
		t.Fatalf("GetOfferings(default): %v", err)
	}
	for _, p := range def.Msg.Offering.Products {
		if p.Identifier == "prem-monthly" {
			t.Fatal("premium product leaked into the default offering")
		}
	}
}
