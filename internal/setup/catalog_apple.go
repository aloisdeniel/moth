package setup

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/aloisdeniel/moth/internal/billing"
)

// AppleCatalog reconciles moth's DesiredCatalog into App Store Connect using
// the milestone-08 ASC JWT client. Every external call goes through ASC.do, so
// the whole sync runs against an httptest double in tests.
type AppleCatalog struct {
	ASC *ASC
	// AppID is the ASC app resource id the subscription group hangs off.
	AppID string
	// Territory is the base territory whose price-point ladder the base price
	// is resolved against; defaults to "USA".
	Territory string
	// NotificationURL is moth's App Store Server Notification V2 endpoint; when
	// set, Sync attempts to register it (guided fallback on 404).
	NotificationURL string
	// CurrentNotificationURL is the URL moth has already registered (from its
	// stored config). Apple exposes no read for it, so Sync diffs against this
	// value to stay idempotent, re-registering only on a change. On a
	// successful registration Sync updates it to NotificationURL.
	CurrentNotificationURL string
}

func (a *AppleCatalog) territory() string {
	if a.Territory != "" {
		return a.Territory
	}
	return "USA"
}

// --- ASC subscription resources (extends the milestone-08 ASC client) ---

// ASCSubscriptionGroup is a subscription group resource.
type ASCSubscriptionGroup struct {
	ResourceID    string
	ReferenceName string
}

// ASCSubscription is an auto-renewable subscription resource.
type ASCSubscription struct {
	ResourceID string
	ProductID  string
	Name       string
	Period     string
	GroupLevel int
}

type ascSubResource struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes struct {
		ReferenceName      string `json:"referenceName,omitempty"`
		Name               string `json:"name,omitempty"`
		ProductID          string `json:"productId,omitempty"`
		SubscriptionPeriod string `json:"subscriptionPeriod,omitempty"`
		GroupLevel         int    `json:"groupLevel,omitempty"`
		Locale             string `json:"locale,omitempty"`
		Description        string `json:"description,omitempty"`
		CustomerPrice      string `json:"customerPrice,omitempty"`
	} `json:"attributes"`
	Relationships struct {
		SubscriptionPricePoint struct {
			Data *struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"subscriptionPricePoint"`
	} `json:"relationships,omitempty"`
}

// FindSubscriptionGroup returns the group with the given reference name, or nil.
func (c *ASC) FindSubscriptionGroup(ctx context.Context, appID, referenceName string) (*ASCSubscriptionGroup, error) {
	var out struct {
		Data []ascSubResource `json:"data"`
	}
	path := "/v1/apps/" + url.PathEscape(appID) + "/subscriptionGroups?limit=200"
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	for _, d := range out.Data {
		if d.Attributes.ReferenceName == referenceName {
			return &ASCSubscriptionGroup{ResourceID: d.ID, ReferenceName: referenceName}, nil
		}
	}
	return nil, nil
}

// CreateSubscriptionGroup creates a subscription group under an app.
func (c *ASC) CreateSubscriptionGroup(ctx context.Context, appID, referenceName string) (*ASCSubscriptionGroup, error) {
	body := map[string]any{"data": map[string]any{
		"type":       "subscriptionGroups",
		"attributes": map[string]any{"referenceName": referenceName},
		"relationships": map[string]any{
			"app": map[string]any{"data": map[string]any{"type": "apps", "id": appID}},
		},
	}}
	var out struct {
		Data ascSubResource `json:"data"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/subscriptionGroups", body, &out); err != nil {
		return nil, err
	}
	return &ASCSubscriptionGroup{ResourceID: out.Data.ID, ReferenceName: referenceName}, nil
}

// ListSubscriptions returns the subscriptions in a group.
func (c *ASC) ListSubscriptions(ctx context.Context, groupID string) ([]ASCSubscription, error) {
	var out struct {
		Data []ascSubResource `json:"data"`
	}
	path := "/v1/subscriptionGroups/" + url.PathEscape(groupID) + "/subscriptions?limit=200"
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	subs := make([]ASCSubscription, 0, len(out.Data))
	for _, d := range out.Data {
		subs = append(subs, ASCSubscription{
			ResourceID: d.ID,
			ProductID:  d.Attributes.ProductID,
			Name:       d.Attributes.Name,
			Period:     d.Attributes.SubscriptionPeriod,
			GroupLevel: d.Attributes.GroupLevel,
		})
	}
	return subs, nil
}

// CreateSubscription creates an auto-renewable subscription in a group.
func (c *ASC) CreateSubscription(ctx context.Context, groupID string, t DesiredTier) (*ASCSubscription, error) {
	period, ok := applePeriod(t.Period)
	if !ok {
		return nil, fmt.Errorf("app store connect: unsupported billing period %q", t.Period)
	}
	body := map[string]any{"data": map[string]any{
		"type": "subscriptions",
		"attributes": map[string]any{
			"name":               t.Reference,
			"productId":          t.ProductID,
			"subscriptionPeriod": period,
			"groupLevel":         t.GroupLevel,
		},
		"relationships": map[string]any{
			"group": map[string]any{"data": map[string]any{"type": "subscriptionGroups", "id": groupID}},
		},
	}}
	var out struct {
		Data ascSubResource `json:"data"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/subscriptions", body, &out); err != nil {
		return nil, err
	}
	return &ASCSubscription{ResourceID: out.Data.ID, ProductID: t.ProductID, Name: t.Reference, Period: period, GroupLevel: t.GroupLevel}, nil
}

// UpdateSubscription patches a subscription's reference name / group level.
func (c *ASC) UpdateSubscription(ctx context.Context, subID, name string, groupLevel int) error {
	body := map[string]any{"data": map[string]any{
		"type":       "subscriptions",
		"id":         subID,
		"attributes": map[string]any{"name": name, "groupLevel": groupLevel},
	}}
	return c.do(ctx, http.MethodPatch, "/v1/subscriptions/"+url.PathEscape(subID), body, nil)
}

// FindPricePoint resolves the price-point resource id whose customerPrice
// equals want in the given territory, or "" when the ladder has no exact match
// (the caller then emits a guided price-schedule step).
func (c *ASC) FindPricePoint(ctx context.Context, subID, territory, want string) (string, error) {
	var out struct {
		Data []ascSubResource `json:"data"`
	}
	path := "/v1/subscriptions/" + url.PathEscape(subID) + "/pricePoints?filter%5Bterritory%5D=" + url.QueryEscape(territory) + "&limit=200"
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return "", err
	}
	for _, d := range out.Data {
		if d.Attributes.CustomerPrice == want {
			return d.ID, nil
		}
	}
	return "", nil
}

// CurrentPricePointIDs returns the price-point ids currently scheduled on a
// subscription, used to decide whether the base price already matches.
func (c *ASC) CurrentPricePointIDs(ctx context.Context, subID string) ([]string, error) {
	var out struct {
		Data []ascSubResource `json:"data"`
	}
	path := "/v1/subscriptions/" + url.PathEscape(subID) + "/prices?limit=200"
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	var ids []string
	for _, d := range out.Data {
		if d.Relationships.SubscriptionPricePoint.Data != nil {
			ids = append(ids, d.Relationships.SubscriptionPricePoint.Data.ID)
		}
	}
	return ids, nil
}

// CreateSubscriptionPrice attaches a price point to a subscription (base
// territory, effective immediately).
func (c *ASC) CreateSubscriptionPrice(ctx context.Context, subID, pricePointID string) error {
	body := map[string]any{"data": map[string]any{
		"type":       "subscriptionPrices",
		"attributes": map[string]any{"preserveCurrentPrice": false},
		"relationships": map[string]any{
			"subscription":           map[string]any{"data": map[string]any{"type": "subscriptions", "id": subID}},
			"subscriptionPricePoint": map[string]any{"data": map[string]any{"type": "subscriptionPricePoints", "id": pricePointID}},
		},
	}}
	return c.do(ctx, http.MethodPost, "/v1/subscriptionPrices", body, nil)
}

// ASCLocalization is a subscription localization resource.
type ASCLocalization struct {
	ResourceID  string
	Locale      string
	Name        string
	Description string
}

// FindLocalization returns the localization for a locale, or nil.
func (c *ASC) FindLocalization(ctx context.Context, subID, locale string) (*ASCLocalization, error) {
	var out struct {
		Data []ascSubResource `json:"data"`
	}
	path := "/v1/subscriptions/" + url.PathEscape(subID) + "/subscriptionLocalizations?limit=200"
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	for _, d := range out.Data {
		if d.Attributes.Locale == locale {
			return &ASCLocalization{ResourceID: d.ID, Locale: locale, Name: d.Attributes.Name, Description: d.Attributes.Description}, nil
		}
	}
	return nil, nil
}

// CreateLocalization creates a subscription localization.
func (c *ASC) CreateLocalization(ctx context.Context, subID string, t DesiredTier) error {
	body := map[string]any{"data": map[string]any{
		"type": "subscriptionLocalizations",
		"attributes": map[string]any{
			"locale":      t.Locale,
			"name":        t.DisplayName,
			"description": t.Description,
		},
		"relationships": map[string]any{
			"subscription": map[string]any{"data": map[string]any{"type": "subscriptions", "id": subID}},
		},
	}}
	return c.do(ctx, http.MethodPost, "/v1/subscriptionLocalizations", body, nil)
}

// UpdateLocalization patches a localization's name / description.
func (c *ASC) UpdateLocalization(ctx context.Context, locID, name, description string) error {
	body := map[string]any{"data": map[string]any{
		"type":       "subscriptionLocalizations",
		"id":         locID,
		"attributes": map[string]any{"name": name, "description": description},
	}}
	return c.do(ctx, http.MethodPatch, "/v1/subscriptionLocalizations/"+url.PathEscape(locID), body, nil)
}

// RegisterServerNotificationURL attempts to set the App Store Server
// Notification V2 URL for the app. There is no stable public ASC endpoint for
// this; a 404 (isASCNotFound) means the caller degrades to the guided step.
func (c *ASC) RegisterServerNotificationURL(ctx context.Context, appID, notifyURL string) error {
	body := map[string]any{"data": map[string]any{
		"type":       "apps",
		"id":         appID,
		"attributes": map[string]any{"appStoreServerNotificationUrl": notifyURL},
	}}
	return c.do(ctx, http.MethodPatch, "/v1/apps/"+url.PathEscape(appID), body, nil)
}

// appleCustomerPrice formats Money as Apple's customerPrice string (2 decimals).
func appleCustomerPrice(m Money) string {
	return strconv.FormatFloat(float64(m.Micros)/1_000_000, 'f', 2, 64)
}

// Sync reconciles the catalog into App Store Connect. It reads existing state,
// diffs, and creates/updates only what changed, so a second run reports no
// changes. Steps Apple's API cannot perform are returned as ManualSteps.
func (a *AppleCatalog) Sync(ctx context.Context, cat DesiredCatalog) (*SyncResult, error) {
	res := &SyncResult{Store: billing.StoreApple}

	group, err := a.ASC.FindSubscriptionGroup(ctx, a.AppID, cat.GroupReference)
	if err != nil {
		return nil, err
	}
	if group == nil {
		if group, err = a.ASC.CreateSubscriptionGroup(ctx, a.AppID, cat.GroupReference); err != nil {
			return nil, err
		}
	}

	existing, err := a.ASC.ListSubscriptions(ctx, group.ResourceID)
	if err != nil {
		return nil, err
	}
	byProduct := make(map[string]ASCSubscription, len(existing))
	for _, s := range existing {
		byProduct[s.ProductID] = s
	}

	for _, tier := range cat.Tiers {
		pr := ProductResult{ProductID: tier.ProductID, Action: ActionUnchanged}
		sub, ok := byProduct[tier.ProductID]
		if !ok {
			created, err := a.ASC.CreateSubscription(ctx, group.ResourceID, tier)
			if err != nil {
				return nil, err
			}
			sub = *created
			pr.Action = ActionCreated
			// A brand-new subscription must be submitted for review before it
			// can sell — Apple exposes no API for that.
			res.addManual(a.reviewStep(tier))
		} else if sub.Name != tier.Reference || sub.GroupLevel != tier.GroupLevel {
			if err := a.ASC.UpdateSubscription(ctx, sub.ResourceID, tier.Reference, tier.GroupLevel); err != nil {
				return nil, err
			}
			pr.Action = ActionUpdated
		}
		pr.StoreID = sub.ResourceID

		// Price: resolve the base-territory price point, attach if not already
		// scheduled. No exact ladder match -> guided price step.
		changedPrice, err := a.reconcilePrice(ctx, sub.ResourceID, tier, res)
		if err != nil {
			return nil, err
		}
		if changedPrice && pr.Action == ActionUnchanged {
			pr.Action = ActionUpdated
		}

		// Localization.
		changedLoc, err := a.reconcileLocalization(ctx, sub.ResourceID, tier)
		if err != nil {
			return nil, err
		}
		if changedLoc && pr.Action == ActionUnchanged {
			pr.Action = ActionUpdated
		}

		res.addProduct(pr)
	}

	// Availability across territories is Console-only.
	res.addManual(ManualStep{
		Title:  "Set subscription availability",
		Reason: "the ASC API does not manage per-territory availability",
		URL:    "https://appstoreconnect.apple.com/apps/" + a.AppID + "/distribution/subscriptions",
		Instructions: []string{
			"Open the app's Subscriptions, confirm each tier's availability includes your markets.",
		},
	})

	a.reconcileNotification(ctx, res)
	return res, nil
}

func (a *AppleCatalog) reconcilePrice(ctx context.Context, subID string, tier DesiredTier, res *SyncResult) (bool, error) {
	want := appleCustomerPrice(tier.Price)
	pointID, err := a.ASC.FindPricePoint(ctx, subID, a.territory(), want)
	if err != nil {
		return false, err
	}
	if pointID == "" {
		res.addManual(ManualStep{
			Title:  "Set the price for " + tier.ProductID,
			Reason: "Apple prices are fixed price points; no ladder point equals " + want + " " + tier.Price.Currency + " in " + a.territory(),
			URL:    "https://appstoreconnect.apple.com/apps/" + a.AppID + "/distribution/subscriptions",
			Instructions: []string{
				"Open " + tier.Reference + " → Subscription Prices, pick the closest available price point.",
			},
		})
		return false, nil
	}
	current, err := a.ASC.CurrentPricePointIDs(ctx, subID)
	if err != nil {
		return false, err
	}
	for _, id := range current {
		if id == pointID {
			return false, nil // already priced
		}
	}
	if err := a.ASC.CreateSubscriptionPrice(ctx, subID, pointID); err != nil {
		return false, err
	}
	return true, nil
}

func (a *AppleCatalog) reconcileLocalization(ctx context.Context, subID string, tier DesiredTier) (bool, error) {
	if tier.Locale == "" {
		return false, nil
	}
	loc, err := a.ASC.FindLocalization(ctx, subID, tier.Locale)
	if err != nil {
		return false, err
	}
	if loc == nil {
		return true, a.ASC.CreateLocalization(ctx, subID, tier)
	}
	if loc.Name != tier.DisplayName || loc.Description != tier.Description {
		return true, a.ASC.UpdateLocalization(ctx, loc.ResourceID, tier.DisplayName, tier.Description)
	}
	return false, nil
}

func (a *AppleCatalog) reconcileNotification(ctx context.Context, res *SyncResult) {
	if a.NotificationURL == "" {
		return
	}
	if a.NotificationURL == a.CurrentNotificationURL {
		res.addNotification(NotificationResult{
			Kind: "apple_server_notification_url", Action: ActionUnchanged,
			Endpoint: a.NotificationURL, Detail: "already registered",
		})
		return
	}
	err := a.ASC.RegisterServerNotificationURL(ctx, a.AppID, a.NotificationURL)
	switch {
	case err == nil:
		action := ActionCreated
		if a.CurrentNotificationURL != "" {
			action = ActionUpdated
		}
		a.CurrentNotificationURL = a.NotificationURL
		res.addNotification(NotificationResult{
			Kind: "apple_server_notification_url", Action: action,
			Endpoint: a.NotificationURL, Detail: "registered via ASC API",
		})
	case isASCNotFound(err):
		res.addNotification(NotificationResult{
			Kind: "apple_server_notification_url", Action: ActionManual,
			Endpoint: a.NotificationURL, Detail: "no ASC API — set it in the console",
		})
		res.addManual(ManualStep{
			Title:  "Register the App Store Server Notification URL",
			Reason: "the ASC API does not expose the server-notification URL",
			URL:    "https://appstoreconnect.apple.com/apps/" + a.AppID + "/distribution/api",
			Instructions: []string{
				"App Information → App Store Server Notifications → Production Server URL:",
				"  " + a.NotificationURL,
			},
		})
	default:
		res.addNotification(NotificationResult{
			Kind: "apple_server_notification_url", Action: ActionFailed,
			Endpoint: a.NotificationURL, Detail: err.Error(),
		})
	}
}

func (a *AppleCatalog) reviewStep(tier DesiredTier) ManualStep {
	return ManualStep{
		Title:  "Submit " + tier.ProductID + " for review",
		Reason: "a new subscription must be reviewed before it can sell; the ASC API cannot submit it",
		URL:    "https://appstoreconnect.apple.com/apps/" + a.AppID + "/distribution/subscriptions",
		Instructions: []string{
			"Open " + tier.Reference + ", add a review screenshot + notes, then Submit for Review.",
		},
	}
}
