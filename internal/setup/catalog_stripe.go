package setup

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aloisdeniel/moth/internal/billing"
)

// StripeCatalog reconciles moth's DesiredCatalog into Stripe using the
// milestone-17 StripeClient (Products + recurring Prices). Unlike App Store
// Connect and Google Play, the Stripe API can do everything — honest
// automation is total (plan/17): no ManualSteps, and even the webhook
// endpoint is provisioned via the API (EnsureWebhookEndpoint).
//
// Two Stripe-isms shape the sync:
//
//   - Price immutability: a Stripe Price can never change amount, currency or
//     cadence. A drifted tier therefore gets a NEW Price on the same Product
//     and moth re-points the tier to it (write-back via ProductResult);
//     existing subscribers keep their old price — Stripe's model, surfaced in
//     the result Detail instead of pretending to edit in place.
//   - Trials are set at checkout time (subscription_data[trial_period_days] on
//     the Checkout Session, see billing.StripeTrialDays), not on the Price, so
//     the catalog sync neither pushes nor drift-checks trial periods.
//   - Product names ARE mutable: a moth display-name change renames the Stripe
//     Product in place (GetProduct + UpdateProduct), reported in the
//     ActionUpdated detail — no new Price is created for a name-only drift.
//
// Introductory offers are not pushed (Stripe models them as coupons /
// subscription phases, out of the milestone's scope).
type StripeCatalog struct {
	// BaseURL defaults to billing.StripeAPIBaseURL; tests point it at a double.
	BaseURL string
	// SecretKey is the project's sk_/rk_ secret key.
	SecretKey string
	HTTPC     billing.Doer
	Now       func() time.Time
}

// StripeWebhookEvents is the event set moth's webhook receiver consumes
// (plan/17): checkout completion plus the subscription lifecycle family.
var StripeWebhookEvents = []string{
	"checkout.session.completed",
	"customer.subscription.created",
	"customer.subscription.updated",
	"customer.subscription.deleted",
}

func (c *StripeCatalog) client() *billing.StripeClient {
	return &billing.StripeClient{BaseURL: c.BaseURL, SecretKey: c.SecretKey, HTTPC: c.HTTPC, Now: c.Now}
}

// Sync reconciles the catalog into Stripe: each tier without recorded ids gets
// a Product + recurring Price created (ids returned for write-back), each tier
// with ids is read back and diffed. Idempotent — an unchanged tier is left
// alone on a re-run. A tier-level problem (unrepresentable price) is an
// ActionFailed result, not a hard error, so the rest of the catalog still
// syncs.
func (c *StripeCatalog) Sync(ctx context.Context, cat DesiredCatalog) (*SyncResult, error) {
	res := &SyncResult{Store: billing.StoreStripe}
	client := c.client()
	for _, tier := range cat.Tiers {
		pr, err := c.syncTier(ctx, client, cat, tier)
		if err != nil {
			return nil, err
		}
		res.addProduct(pr)
	}
	return res, nil
}

func (c *StripeCatalog) syncTier(ctx context.Context, client *billing.StripeClient, cat DesiredCatalog, tier DesiredTier) (ProductResult, error) {
	pr := ProductResult{
		ProductID:     tier.ProductID,
		StoreID:       tier.StripePriceID,
		StoreParentID: tier.StripeProductID,
		Action:        ActionUnchanged,
	}
	interval, count, err := billing.StripeRecurringForPeriod(string(tier.Period))
	if err != nil {
		pr.Action = ActionFailed
		pr.Detail = err.Error()
		return pr, nil
	}
	// Stripe denominates unit_amount in the currency's own minor unit (cents
	// for USD, whole yen for JPY, thousandths for KWD): the per-currency
	// conversion is shared with the runtime webhook normalizer so the create
	// AND the drift compare below agree with what Stripe reports back.
	unitAmount, err := billing.StripeUnitAmountForMicros(tier.Price.Micros, tier.Price.Currency)
	if err != nil {
		pr.Action = ActionFailed
		pr.Detail = err.Error()
		return pr, nil
	}

	if tier.StripePriceID == "" {
		return c.provisionTier(ctx, client, cat, tier, unitAmount, interval, count, "")
	}

	current, err := client.GetPrice(ctx, tier.StripePriceID)
	if errors.Is(err, billing.ErrNotFound) {
		// The recorded price no longer exists in Stripe (deleted/archived id);
		// recreate rather than fail — the moth catalog is the desired state.
		return c.provisionTier(ctx, client, cat, tier, unitAmount, interval, count,
			fmt.Sprintf("recorded price %s no longer exists in Stripe; ", tier.StripePriceID))
	}
	if err != nil {
		return ProductResult{}, err
	}

	product, err := client.GetProduct(ctx, current.ProductID)
	if errors.Is(err, billing.ErrNotFound) {
		// The product behind the recorded price is gone (deleted in Stripe);
		// recreate the whole tier — a fresh Product, not the missing id.
		tier.StripeProductID = ""
		return c.provisionTier(ctx, client, cat, tier, unitAmount, interval, count,
			fmt.Sprintf("recorded product %s no longer exists in Stripe; ", current.ProductID))
	}
	if err != nil {
		return ProductResult{}, err
	}

	priceInSync := current.UnitAmount == unitAmount && strings.EqualFold(current.Currency, tier.Price.Currency) &&
		current.Interval == interval && normalizeIntervalCount(current.IntervalCount) == normalizeIntervalCount(count) &&
		current.Active
	nameInSync := product.Name == tier.DisplayName

	if priceInSync && nameInSync {
		pr.StoreParentID = current.ProductID
		return pr, nil
	}

	var details []string
	if !nameInSync {
		// Display-name drift. Unlike prices, Stripe product names are mutable:
		// rename the Product in place — no new resource, no re-pointing.
		if _, err := client.UpdateProduct(ctx, current.ProductID, tier.DisplayName); err != nil {
			return ProductResult{}, err
		}
		details = append(details, fmt.Sprintf("renamed product %s from %q to %q", current.ProductID, product.Name, tier.DisplayName))
	}
	if !priceInSync {
		// Price drift. Stripe prices are immutable: create a NEW Price on the
		// same Product and re-point the tier to it (the caller writes the new id
		// back); existing subscribers keep the old price — Stripe's model,
		// reported honestly instead of pretending to edit in place.
		price, err := client.CreatePrice(ctx, billing.StripePriceParams{
			ProductID:     current.ProductID,
			Currency:      tier.Price.Currency,
			UnitAmount:    unitAmount,
			Interval:      interval,
			IntervalCount: count,
			Metadata:      c.tierMetadata(cat, tier),
		})
		if err != nil {
			return ProductResult{}, err
		}
		pr.StoreID = price.ID
		details = append(details, fmt.Sprintf("price changed: created new Price %s (Stripe prices are immutable); the tier now sells at the new price, existing subscribers keep %s",
			price.ID, tier.StripePriceID))
	}
	pr.Action = ActionUpdated
	pr.StoreParentID = current.ProductID
	pr.Detail = strings.Join(details, "; ")
	return pr, nil
}

// provisionTier creates the tier's Stripe resources: the Product (reusing a
// recorded one when only the price is missing) and a recurring Price.
func (c *StripeCatalog) provisionTier(ctx context.Context, client *billing.StripeClient, cat DesiredCatalog, tier DesiredTier, unitAmount int64, interval string, count int, detailPrefix string) (ProductResult, error) {
	productID := tier.StripeProductID
	if productID == "" {
		product, err := client.CreateProduct(ctx, tier.DisplayName, c.tierMetadata(cat, tier))
		if err != nil {
			return ProductResult{}, err
		}
		productID = product.ID
	}
	price, err := client.CreatePrice(ctx, billing.StripePriceParams{
		ProductID:     productID,
		Currency:      tier.Price.Currency,
		UnitAmount:    unitAmount,
		Interval:      interval,
		IntervalCount: count,
		Metadata:      c.tierMetadata(cat, tier),
	})
	if err != nil {
		// The Product exists but the Price does not: return the product id in a
		// per-tier failure (not a hard error) so the caller's write-back records
		// it — a re-run then reuses the Product instead of provisioning a
		// duplicate (moth has no Stripe product search to find it again).
		return ProductResult{
			ProductID:     tier.ProductID,
			StoreParentID: productID,
			Action:        ActionFailed,
			Detail: detailPrefix + fmt.Sprintf(
				"created Stripe product %s but its recurring price could not be created: %v (the product id is recorded; a re-run reuses it)",
				productID, err),
		}, nil
	}
	return ProductResult{
		ProductID:     tier.ProductID,
		StoreID:       price.ID,
		StoreParentID: productID,
		Action:        ActionCreated,
		Detail:        detailPrefix + fmt.Sprintf("created Stripe product %s with recurring price %s", productID, price.ID),
	}, nil
}

// tierMetadata carries the moth identity onto the Stripe resources so they are
// traceable back to the tier and project from the Stripe dashboard.
func (c *StripeCatalog) tierMetadata(cat DesiredCatalog, tier DesiredTier) map[string]string {
	return map[string]string{
		"moth_product": tier.Reference,
		// GroupReference is "moth-<slug>" by construction (desiredCatalog).
		"moth_project": strings.TrimPrefix(cat.GroupReference, "moth-"),
	}
}

// normalizeIntervalCount treats 0 and 1 as the same cadence: Stripe defaults an
// omitted interval_count to 1.
func normalizeIntervalCount(n int) int {
	if n == 0 {
		return 1
	}
	return n
}

// EnsureWebhookEndpoint idempotently provisions moth's Stripe webhook endpoint:
// it lists the account's endpoints, returns an existing one matching url
// exactly (a real read+diff, unlike Apple's persisted-anchor idempotency), or
// creates one subscribed to StripeWebhookEvents. An existing endpoint is not
// taken at face value: one Stripe has disabled, or one missing any of moth's
// events (extra events are fine), is repaired in place via
// UpdateWebhookEndpoint — repaired reports that (created=false). The signing
// Secret is set only when created (Stripe reveals it exactly once) — the
// caller must persist it then or never; an update never returns it.
func (c *StripeCatalog) EnsureWebhookEndpoint(ctx context.Context, url string) (ep billing.StripeWebhookEndpoint, created, repaired bool, err error) {
	client := c.client()
	endpoints, err := client.ListWebhookEndpoints(ctx)
	if err != nil {
		return billing.StripeWebhookEndpoint{}, false, false, err
	}
	for _, ep := range endpoints {
		if ep.URL != url {
			continue
		}
		if ep.Status == "enabled" && hasAllEvents(ep.EnabledEvents, StripeWebhookEvents) {
			return ep, false, false, nil
		}
		updated, err := client.UpdateWebhookEndpoint(ctx, ep.ID, StripeWebhookEvents)
		if err != nil {
			return billing.StripeWebhookEndpoint{}, false, false, err
		}
		return updated, false, true, nil
	}
	ep, err = client.CreateWebhookEndpoint(ctx, url, StripeWebhookEvents)
	if err != nil {
		return billing.StripeWebhookEndpoint{}, false, false, err
	}
	return ep, true, false, nil
}

// hasAllEvents reports whether have contains every event in want (extra
// subscriptions beyond moth's set are fine).
func hasAllEvents(have, want []string) bool {
	got := make(map[string]bool, len(have))
	for _, e := range have {
		got[e] = true
	}
	for _, e := range want {
		if !got[e] {
			return false
		}
	}
	return true
}
