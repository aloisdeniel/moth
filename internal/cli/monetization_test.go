package cli

import (
	"testing"

	"google.golang.org/protobuf/proto"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

func sampleMonetization() *adminv1.MonetizationSpec {
	return &adminv1.MonetizationSpec{
		Entitlements: []*adminv1.EntitlementSpec{
			{Identifier: "pro", DisplayName: "Pro"},
		},
		Products: []*adminv1.ProductSpec{
			{
				Identifier:        "monthly",
				DisplayName:       "Monthly",
				AppleProductId:    "app.monthly",
				GoogleProductId:   "app.monthly",
				BillingPeriod:     "monthly",
				PriceAmountMicros: 9_990_000,
				Currency:          "USD",
				Offering:          "default",
				SortOrder:         0,
				Entitlements:      []string{"pro"},
			},
		},
	}
}

// currentFromSpec builds the live catalog a dump of the given spec would have
// read back, assigning server ids — the state PlanMonetization must diff to
// zero on a second apply.
func currentFromSpec(spec *adminv1.MonetizationSpec) ([]*adminv1.Entitlement, []*adminv1.Product) {
	entByIdent := map[string]string{}
	var ents []*adminv1.Entitlement
	for _, e := range spec.GetEntitlements() {
		id := "ent-" + e.GetIdentifier()
		entByIdent[e.GetIdentifier()] = id
		ents = append(ents, &adminv1.Entitlement{
			Id: id, Identifier: e.GetIdentifier(), DisplayName: e.GetDisplayName(),
		})
	}
	var prods []*adminv1.Product
	for _, p := range spec.GetProducts() {
		var eids []string
		for _, ident := range p.GetEntitlements() {
			eids = append(eids, entByIdent[ident])
		}
		prods = append(prods, &adminv1.Product{
			Id:                     "prod-" + p.GetIdentifier(),
			Identifier:             p.GetIdentifier(),
			DisplayName:            p.GetDisplayName(),
			AppleProductId:         p.GetAppleProductId(),
			GoogleProductId:        p.GetGoogleProductId(),
			BillingPeriod:          p.GetBillingPeriod(),
			PriceAmountMicros:      p.GetPriceAmountMicros(),
			Currency:               p.GetCurrency(),
			TrialPeriod:            p.GetTrialPeriod(),
			IntroPriceAmountMicros: p.GetIntroPriceAmountMicros(),
			IntroPeriod:            p.GetIntroPeriod(),
			Offering:               p.GetOffering(),
			SortOrder:              p.GetSortOrder(),
			EntitlementIds:         eids,
		})
	}
	return ents, prods
}

func TestMonetizationSpecYAMLRoundTrip(t *testing.T) {
	spec := &adminv1.ProjectSpec{
		Name:         "Demo",
		Slug:         "demo",
		Monetization: sampleMonetization(),
	}
	data, err := SpecToYAML(spec)
	if err != nil {
		t.Fatal(err)
	}
	got, err := SpecFromYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(spec, got) {
		t.Fatalf("round trip mismatch:\nwant %v\n got %v", spec, got)
	}
}

func TestPlanMonetizationNilSpecIsNoop(t *testing.T) {
	ents, prods := currentFromSpec(sampleMonetization())
	plan := PlanMonetization(nil, ents, prods)
	if !plan.Empty() {
		t.Fatalf("nil spec must leave the catalog untouched, got %+v", plan)
	}
}

func TestPlanMonetizationIdempotent(t *testing.T) {
	spec := sampleMonetization()
	ents, prods := currentFromSpec(spec)
	plan := PlanMonetization(spec, ents, prods)
	if !plan.Empty() {
		t.Fatalf("dump re-applied must diff to zero, got %+v (%v)", plan, plan.Summary())
	}
}

func TestPlanMonetizationCreatesOnEmptyCatalog(t *testing.T) {
	spec := sampleMonetization()
	plan := PlanMonetization(spec, nil, nil)
	if got := plan.CreateEntitlements; len(got) != 1 || got[0] != "pro" {
		t.Fatalf("create entitlements = %v", got)
	}
	if got := plan.CreateProducts; len(got) != 1 || got[0] != "monthly" {
		t.Fatalf("create products = %v", got)
	}
	if len(plan.UpdateProducts)+len(plan.DeleteProducts)+len(plan.UpdateEntitlements)+len(plan.DeleteEntitlements) != 0 {
		t.Fatalf("unexpected update/delete on create: %+v", plan)
	}
}

func TestPlanMonetizationDeletesUnlisted(t *testing.T) {
	spec := &adminv1.MonetizationSpec{} // explicit empty catalog
	ents, prods := currentFromSpec(sampleMonetization())
	plan := PlanMonetization(spec, ents, prods)
	if got := plan.DeleteEntitlements; len(got) != 1 || got[0] != "pro" {
		t.Fatalf("delete entitlements = %v", got)
	}
	if got := plan.DeleteProducts; len(got) != 1 || got[0] != "monthly" {
		t.Fatalf("delete products = %v", got)
	}
}

func TestPlanMonetizationDetectsFieldChanges(t *testing.T) {
	spec := sampleMonetization()
	ents, prods := currentFromSpec(spec)

	// Change a price and an entitlement display name in the spec.
	spec.Products[0].PriceAmountMicros = 12_990_000
	spec.Entitlements[0].DisplayName = "Pro Plan"

	plan := PlanMonetization(spec, ents, prods)
	if got := plan.UpdateProducts; len(got) != 1 || got[0] != "monthly" {
		t.Fatalf("update products = %v", got)
	}
	if got := plan.UpdateEntitlements; len(got) != 1 || got[0] != "pro" {
		t.Fatalf("update entitlements = %v", got)
	}
	if !plan.CreateProductsEmpty() {
		t.Fatalf("no creates/deletes expected: %+v", plan)
	}
}

// CreateProductsEmpty is a tiny test-only helper kept local to avoid leaking
// into the package API.
func (p MonetizationPlan) CreateProductsEmpty() bool {
	return len(p.CreateProducts) == 0 && len(p.DeleteProducts) == 0 &&
		len(p.CreateEntitlements) == 0 && len(p.DeleteEntitlements) == 0
}

func TestPlanMonetizationOfferingDefaultNormalization(t *testing.T) {
	spec := sampleMonetization()
	// Spec omits the offering tag; the live product carries the explicit
	// "default" the store applies. These must be treated as equal.
	spec.Products[0].Offering = ""
	ents, prods := currentFromSpec(sampleMonetization()) // current keeps "default"
	plan := PlanMonetization(spec, ents, prods)
	if !plan.Empty() {
		t.Fatalf("empty vs default offering must be a no-op, got %+v", plan)
	}
}

func TestPlanMonetizationEntitlementGrantChange(t *testing.T) {
	spec := sampleMonetization()
	spec.Entitlements = append(spec.Entitlements, &adminv1.EntitlementSpec{Identifier: "premium", DisplayName: "Premium"})
	ents, prods := currentFromSpec(sampleMonetization())
	// current has only "pro"; add "premium" and grant it to the product.
	ents = append(ents, &adminv1.Entitlement{Id: "ent-premium", Identifier: "premium", DisplayName: "Premium"})
	spec.Products[0].Entitlements = []string{"pro", "premium"}
	plan := PlanMonetization(spec, ents, prods)
	if got := plan.UpdateProducts; len(got) != 1 || got[0] != "monthly" {
		t.Fatalf("grant change should update the product: %v", got)
	}
}
