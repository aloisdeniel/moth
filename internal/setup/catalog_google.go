package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/aloisdeniel/moth/internal/billing"
)

// TokenSource mints a bearer token for a Google API. billing.GoogleTokenSource
// satisfies it; the Pub/Sub wiring needs a pubsub-scoped one (see PubSub).
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// GoogleCatalog reconciles moth's DesiredCatalog into Google Play using the
// Android Publisher API, authed with the milestone-11 service-account token
// source. All calls go through an injectable Doer + BaseURL for httptest.
type GoogleCatalog struct {
	// BaseURL defaults to billing.GooglePlayBaseURL; tests point it at a double.
	BaseURL     string
	PackageName string
	Tokens      TokenSource
	HTTPC       billing.Doer
	// RegionCode is the base region whose price the base plan is created with;
	// defaults to "US".
	RegionCode string
	// BasePlanID is the base-plan id created per subscription; defaults to
	// "base" (Google requires lowercase/digits/hyphen).
	BasePlanID string
}

func (c *GoogleCatalog) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return billing.GooglePlayBaseURL
}

func (c *GoogleCatalog) httpc() billing.Doer {
	if c.HTTPC != nil {
		return c.HTTPC
	}
	return &http.Client{}
}

func (c *GoogleCatalog) region() string {
	if c.RegionCode != "" {
		return c.RegionCode
	}
	return "US"
}

func (c *GoogleCatalog) basePlanID() string {
	if c.BasePlanID != "" {
		return c.BasePlanID
	}
	return "base"
}

// errGoogleNotFound is returned by do for a 404, so callers create-on-missing.
var errGoogleNotFound = errors.New("google play: not found")

func (c *GoogleCatalog) do(ctx context.Context, method, path string, body, out any) error {
	tok, err := c.Tokens.Token(ctx)
	if err != nil {
		return err
	}
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = strings.NewReader(string(payload))
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL()+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpc().Do(req)
	if err != nil {
		return fmt.Errorf("google play: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("google play: read response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return errGoogleNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("google play: status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if out != nil && len(payload) > 0 {
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("google play: decode response: %w", err)
		}
	}
	return nil
}

// googleSubscription mirrors the Android Publisher Subscription resource subset
// moth reads/writes.
type googleSubscription struct {
	ProductID string `json:"productId"`
	Listings  []struct {
		LanguageCode string `json:"languageCode"`
		Title        string `json:"title"`
		Description  string `json:"description"`
	} `json:"listings"`
	BasePlans []struct {
		BasePlanID      string `json:"basePlanId"`
		State           string `json:"state"`
		RegionalConfigs []struct {
			RegionCode string `json:"regionCode"`
			Price      struct {
				CurrencyCode string `json:"currencyCode"`
				Units        string `json:"units"`
				Nanos        int32  `json:"nanos"`
			} `json:"price"`
		} `json:"regionalConfigs"`
	} `json:"basePlans"`
}

func (c *GoogleCatalog) subURL(productID string) string {
	return fmt.Sprintf("/androidpublisher/v3/applications/%s/subscriptions/%s",
		url.PathEscape(c.PackageName), url.PathEscape(productID))
}

// getSubscription reads a subscription; a 404 surfaces as errGoogleNotFound.
func (c *GoogleCatalog) getSubscription(ctx context.Context, productID string) (*googleSubscription, error) {
	var out googleSubscription
	if err := c.do(ctx, http.MethodGet, c.subURL(productID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// subscriptionBody builds the Subscription resource for a tier.
func (c *GoogleCatalog) subscriptionBody(t DesiredTier) (map[string]any, error) {
	period, ok := googlePeriod(t.Period)
	if !ok {
		return nil, fmt.Errorf("google play: unsupported billing period %q", t.Period)
	}
	units, nanos := moneyToUnitsNanos(t.Price)
	return map[string]any{
		"productId": t.ProductID,
		"listings": []map[string]any{{
			"languageCode": t.Locale,
			"title":        t.DisplayName,
			"description":  t.Description,
		}},
		"basePlans": []map[string]any{{
			"basePlanId": c.basePlanID(),
			"autoRenewingBasePlanType": map[string]any{
				"billingPeriodDuration": period,
			},
			"regionalConfigs": []map[string]any{{
				"regionCode":                c.region(),
				"newSubscriberAvailability": true,
				"price": map[string]any{
					"currencyCode": t.Price.Currency,
					"units":        fmt.Sprintf("%d", units),
					"nanos":        nanos,
				},
			}},
		}},
	}, nil
}

// createSubscription creates a Subscription (productId is a query param).
func (c *GoogleCatalog) createSubscription(ctx context.Context, t DesiredTier) error {
	body, err := c.subscriptionBody(t)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/androidpublisher/v3/applications/%s/subscriptions?productId=%s",
		url.PathEscape(c.PackageName), url.QueryEscape(t.ProductID))
	return c.do(ctx, http.MethodPost, path, body, nil)
}

// updateSubscription patches a Subscription (listings + base-plan config).
func (c *GoogleCatalog) updateSubscription(ctx context.Context, t DesiredTier) error {
	body, err := c.subscriptionBody(t)
	if err != nil {
		return err
	}
	path := c.subURL(t.ProductID) + "?updateMask=" + url.QueryEscape("listings,basePlans")
	return c.do(ctx, http.MethodPatch, path, body, nil)
}

// activateBasePlan moves a base plan from DRAFT to ACTIVE so it can sell.
func (c *GoogleCatalog) activateBasePlan(ctx context.Context, productID string) error {
	path := c.subURL(productID) + "/basePlans/" + url.PathEscape(c.basePlanID()) + ":activate"
	return c.do(ctx, http.MethodPost, path, map[string]any{}, nil)
}

// samePrice reports whether the existing subscription already carries the tier's
// base price + period in the base region.
func (c *GoogleCatalog) samePrice(existing *googleSubscription, t DesiredTier) bool {
	units, nanos := moneyToUnitsNanos(t.Price)
	wantUnits := fmt.Sprintf("%d", units)
	for _, bp := range existing.BasePlans {
		if bp.BasePlanID != c.basePlanID() {
			continue
		}
		for _, rc := range bp.RegionalConfigs {
			if rc.RegionCode != c.region() {
				continue
			}
			return rc.Price.CurrencyCode == t.Price.Currency &&
				rc.Price.Units == wantUnits && rc.Price.Nanos == nanos
		}
	}
	return false
}

// sameListing reports whether the base-region listing matches.
func (c *GoogleCatalog) sameListing(existing *googleSubscription, t DesiredTier) bool {
	for _, l := range existing.Listings {
		if l.LanguageCode == t.Locale {
			return l.Title == t.DisplayName && l.Description == t.Description
		}
	}
	return false
}

// Sync reconciles the catalog into Google Play: create/patch each subscription
// with its base plan + regional price, activate it, and report per-product
// actions. Idempotent — an unchanged tier is left alone on a re-run.
func (c *GoogleCatalog) Sync(ctx context.Context, cat DesiredCatalog) (*SyncResult, error) {
	res := &SyncResult{Store: billing.StoreGoogle}
	for _, tier := range cat.Tiers {
		pr := ProductResult{ProductID: tier.ProductID, StoreID: tier.ProductID, Action: ActionUnchanged}
		existing, err := c.getSubscription(ctx, tier.ProductID)
		switch {
		case errors.Is(err, errGoogleNotFound):
			if err := c.createSubscription(ctx, tier); err != nil {
				return nil, err
			}
			if err := c.activateBasePlan(ctx, tier.ProductID); err != nil {
				return nil, err
			}
			pr.Action = ActionCreated
		case err != nil:
			return nil, err
		default:
			if !c.samePrice(existing, tier) || !c.sameListing(existing, tier) {
				if err := c.updateSubscription(ctx, tier); err != nil {
					return nil, err
				}
				pr.Action = ActionUpdated
			}
		}
		res.addProduct(pr)
	}

	// Country availability beyond the base region is Play Console-only.
	res.addManual(ManualStep{
		Title:  "Confirm subscription availability",
		Reason: "the Android Publisher API sets the base region price; per-country availability is Console-only",
		URL:    "https://play.google.com/console/",
		Instructions: []string{
			"Monetize → Subscriptions: confirm each product's countries/regions.",
		},
	})
	return res, nil
}

// ---- RTDN Pub/Sub wiring ----

// PubSub is a minimal Cloud Pub/Sub Admin client for the RTDN topic + push
// subscription. Tokens MUST be pubsub-scoped in production (distinct from the
// androidpublisher billing SA); the injectable BaseURL/Doer keep it testable.
type PubSub struct {
	// BaseURL defaults to the Cloud Pub/Sub API host; tests override it.
	BaseURL string
	Project string
	Tokens  TokenSource
	HTTPC   billing.Doer
}

// PubSubBaseURL is the Cloud Pub/Sub REST host.
const PubSubBaseURL = "https://pubsub.googleapis.com"

func (p *PubSub) baseURL() string {
	if p.BaseURL != "" {
		return p.BaseURL
	}
	return PubSubBaseURL
}

func (p *PubSub) httpc() billing.Doer {
	if p.HTTPC != nil {
		return p.HTTPC
	}
	return &http.Client{}
}

func (p *PubSub) do(ctx context.Context, method, path string, body any) (int, error) {
	tok, err := p.Tokens.Token(ctx)
	if err != nil {
		return 0, err
	}
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = strings.NewReader(string(payload))
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL()+path, reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.httpc().Do(req)
	if err != nil {
		return 0, fmt.Errorf("pubsub: %w", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusNotFound {
		return resp.StatusCode, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp.StatusCode, fmt.Errorf("pubsub: status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	return resp.StatusCode, nil
}

// TopicName is the fully-qualified topic resource name.
func (p *PubSub) TopicName(topicID string) string {
	return fmt.Sprintf("projects/%s/topics/%s", p.Project, topicID)
}

// EnsureTopic creates the topic if missing (idempotent). Returns whether it was
// created.
func (p *PubSub) EnsureTopic(ctx context.Context, topicID string) (bool, error) {
	path := "/v1/" + p.TopicName(topicID)
	status, err := p.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return false, err
	}
	if status == http.StatusOK {
		return false, nil
	}
	// PUT creates the topic (Pub/Sub createTopic is an idempotent PUT).
	if _, err := p.do(ctx, http.MethodPut, path, map[string]any{}); err != nil {
		return false, err
	}
	return true, nil
}

// EnsurePushSubscription creates a push subscription delivering to pushEndpoint
// (moth's /billing/google/rtdn/{slug}?token=…). Idempotent on the id.
func (p *PubSub) EnsurePushSubscription(ctx context.Context, subID, topicID, pushEndpoint string) (bool, error) {
	path := "/v1/" + fmt.Sprintf("projects/%s/subscriptions/%s", p.Project, subID)
	status, err := p.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return false, err
	}
	if status == http.StatusOK {
		return false, nil
	}
	body := map[string]any{
		"topic":              p.TopicName(topicID),
		"pushConfig":         map[string]any{"pushEndpoint": pushEndpoint},
		"ackDeadlineSeconds": 30,
	}
	if _, err := p.do(ctx, http.MethodPut, path, body); err != nil {
		return false, err
	}
	return true, nil
}

// WireRTDN creates the RTDN topic + push subscription and appends the result
// plus the always-guided "point Play Console at the topic" step (no Android
// Publisher API for it). ps may be nil to skip API creation and emit only the
// guided steps (honest fallback when no pubsub-scoped credential is available).
func WireRTDN(ctx context.Context, ps *PubSub, topicID, subID, pushEndpoint string, res *SyncResult) error {
	topicName := ""
	if ps != nil {
		topicName = ps.TopicName(topicID)
		createdT, err := ps.EnsureTopic(ctx, topicID)
		if err != nil {
			return err
		}
		action := ActionUnchanged
		if createdT {
			action = ActionCreated
		}
		res.addNotification(NotificationResult{
			Kind: "google_rtdn_topic", Action: action, Endpoint: topicName,
		})
		createdS, err := ps.EnsurePushSubscription(ctx, subID, topicID, pushEndpoint)
		if err != nil {
			return err
		}
		action = ActionUnchanged
		if createdS {
			action = ActionCreated
		}
		res.addNotification(NotificationResult{
			Kind: "google_rtdn_push_subscription", Action: action, Endpoint: pushEndpoint,
		})
	} else {
		res.addNotification(NotificationResult{
			Kind: "google_rtdn_topic", Action: ActionManual, Endpoint: pushEndpoint,
			Detail: "no pubsub-scoped credential — create the topic/subscription by hand",
		})
	}
	res.addManual(ManualStep{
		Title:  "Point Play Console at the RTDN topic",
		Reason: "the Android Publisher API cannot set the RTDN topic; it is Console-only",
		URL:    "https://play.google.com/console/",
		Instructions: []string{
			"Monetize → Monetization setup → Real-time developer notifications.",
			"Topic name: " + topicName,
			"Then Send test notification and confirm it reaches: " + pushEndpoint,
		},
	})
	return nil
}
