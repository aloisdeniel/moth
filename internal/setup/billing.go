package setup

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/billing"
)

// BillingSetup drives `moth setup billing` for one project: it stores the
// project's store API credentials into moth's encrypted billing config, pushes
// moth's product catalog into App Store Connect / Google Play, wires the
// notification endpoints, and verifies each store is reachable and authenticated.
//
// It is the monetization counterpart to AppleSetup/GoogleSetup and inherits the
// same honest-automation contract: automate what the store APIs expose, fall
// back to a precise guided checklist (the AppleCatalog/GoogleCatalog ManualSteps)
// where they don't, and keep every run idempotent — the store-catalog clients
// read store state and change only deltas, so a second run reports no changes.
//
// Every external dependency is a field so tests inject doubles: the admin RPC
// clients, the App Store Connect client (ASC, catalog push), the Google token
// source + base URLs, and the base URLs the verification probes reach the store
// APIs at. Store credentials are used in-process to drive the store APIs and
// persisted only as moth's own encrypted billing config — never leaked
// platform-side.
type BillingSetup struct {
	Projects     adminv1connect.ProjectServiceClient
	Products     adminv1connect.ProductServiceClient
	BillingCreds adminv1connect.BillingCredentialsServiceClient
	Prompt       *Prompter
	Out          io.Writer
	// BaseURL is the moth instance base URL, used to build the notification
	// endpoints (/billing/apple/notifications/{slug}, /billing/google/rtdn/{slug}).
	BaseURL string

	// Slug identifies the project.
	Slug string
	// Yes skips the "push to the live stores?" confirmation (non-interactive).
	Yes bool

	// --- Apple ---
	// ASC is the App Store Connect API client (catalog push); nil skips the
	// Apple catalog push (credentials are still stored, push becomes guided).
	ASC *ASC
	// AppleAppID is the ASC app resource id the subscription group hangs off.
	AppleAppID string
	// Apple In-App-Purchase key material (stored into moth, used to verify).
	AppleIAPKeyID    string
	AppleIAPIssuerID string
	AppleBundleID    string
	AppleAppAppleID  string
	AppleIAPKey      *ecdsa.PrivateKey // parsed .p8; nil skips the Apple verify probe
	AppleIAPKeyP8    []byte            // raw .p8 to store encrypted; empty keeps the stored one
	// AppleNotificationSecret is the App Store Server Notifications shared secret
	// (stored encrypted; empty keeps the stored one).
	AppleNotificationSecret string
	// AppleServerAPIBase overrides the App Store Server API host the verify probe
	// reaches (test double). Empty uses billing's production host.
	AppleServerAPIBase string

	// --- Google ---
	// GoogleSA is the parsed Play Developer service account (catalog push +
	// verify); nil skips the Google catalog push and verify.
	GoogleSA *billing.GoogleServiceAccount
	// GoogleServiceAccountJSON is the raw SA JSON to store encrypted; empty keeps
	// the stored one.
	GoogleServiceAccountJSON []byte
	GooglePackageName        string
	// GooglePubsubTopic is the RTDN Cloud Pub/Sub topic (projects/X/topics/Y or a
	// bare topic id), stored into moth and wired to the RTDN push subscription.
	GooglePubsubTopic string
	// GoogleRTDNSecret authenticates the RTDN push webhook (stored encrypted;
	// empty keeps the stored one).
	GoogleRTDNSecret string
	// GoogleCatalogBaseURL overrides the Android Publisher host (catalog + verify;
	// test double).
	GoogleCatalogBaseURL string
	// GoogleTokenURL overrides the OAuth2 token endpoint the SA authenticates
	// against (test double). Empty uses the SA's token_uri.
	GoogleTokenURL string
	// GooglePubSubTokens is a pubsub-scoped token source for creating the RTDN
	// topic + push subscription; nil emits guided steps instead (the
	// androidpublisher-scoped billing SA cannot create Pub/Sub resources).
	GooglePubSubTokens TokenSource
	// PubSubBaseURL overrides the Cloud Pub/Sub host (test double).
	PubSubBaseURL string
	// GoogleCloudProject is the GCP project the RTDN topic/subscription live in;
	// defaults to the SA's project_id.
	GoogleCloudProject string

	// --- Stripe ---
	// Stripe enables the Stripe leg even when no secret key is provided this
	// run (credentials kept, push skipped with a warning). A non-empty
	// StripeSecretKey enables it implicitly.
	Stripe bool
	// StripeSecretKey is the project's restricted/secret key (sk_/rk_). It is
	// stored encrypted into moth's billing config AND used in-process for the
	// catalog push, webhook provisioning and verify probe — unlike ASC, the
	// same key drives provisioning and runtime. Empty keeps the stored one
	// (and skips the live Stripe calls: moth never returns stored secrets).
	StripeSecretKey string
	// StripeBaseURL overrides the Stripe API host (test double).
	StripeBaseURL string

	HTTPC billing.Doer
}

func (s *BillingSetup) defaults() {
	if s.Out == nil {
		s.Out = io.Discard
	}
	if s.HTTPC == nil {
		s.HTTPC = &http.Client{}
	}
}

func (s *BillingSetup) appleNotifyURL() string {
	return strings.TrimSuffix(s.BaseURL, "/") + "/billing/apple/notifications/" + s.Slug
}

// googleRTDNURL is the RTDN push endpoint Google delivers to. It carries the
// shared-secret ?token= query the receiver authenticates every push against
// (handleGoogleRTDN -> AuthenticatePushToken); without it Google's replayed,
// tokenless deliveries are rejected 401. The token is only appended when the
// secret is in hand this run — when it is not, an existing (correctly tokenized)
// push subscription is left untouched by EnsurePushSubscription's idempotent GET,
// and a project with no secret at all has a non-functional webhook regardless.
func (s *BillingSetup) googleRTDNURL() string {
	u := strings.TrimSuffix(s.BaseURL, "/") + "/billing/google/rtdn/" + s.Slug
	if s.GoogleRTDNSecret != "" {
		u += "?token=" + url.QueryEscape(s.GoogleRTDNSecret)
	}
	return u
}

// Run stores credentials, pushes the catalog, wires notifications and verifies
// each configured store, returning the checklist. A non-nil error aborts before
// verification (bad input, RPC failure); store-side problems surface as checks.
func (s *BillingSetup) Run(ctx context.Context) (*Report, error) {
	s.defaults()
	rep := &Report{}

	doApple := s.AppleBundleID != ""
	doGoogle := s.GooglePackageName != ""
	doStripe := s.Stripe || s.StripeSecretKey != ""
	if !doApple && !doGoogle && !doStripe {
		return nil, errors.New("nothing to configure: pass Apple (--apple-bundle-id), Google (--google-package-name) and/or Stripe (--stripe-secret-key) inputs")
	}

	project, err := findProjectBySlug(ctx, s.Projects, s.Slug)
	if err != nil {
		return nil, err
	}

	// 1. Store credentials into moth (write-only secrets; empty keeps stored).
	if err := s.storeCredentials(ctx, rep, project.Id, doApple, doGoogle, doStripe); err != nil {
		return nil, err
	}

	// 2. Build moth's desired catalog once; each store push filters to the tiers
	// carrying that store's SKU.
	products, err := s.Products.ListProducts(ctx, connect.NewRequest(&adminv1.ListProductsRequest{ProjectId: project.Id}))
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	// 3. Confirm before mutating the live stores.
	if (doApple && s.ASC != nil) || (doGoogle && s.GoogleSA != nil) || (doStripe && s.StripeSecretKey != "") {
		if err := s.confirmPush(); err != nil {
			return nil, err
		}
	}

	// 4. Push + wire + verify, per store.
	if doApple {
		s.runApple(ctx, rep, project.Id, products.Msg.Products)
	}
	if doGoogle {
		s.runGoogle(ctx, rep, project.Name, products.Msg.Products)
	}
	if doStripe {
		s.runStripe(ctx, rep, project.Id, products.Msg.Products)
	}
	return rep, nil
}

func (s *BillingSetup) confirmPush() error {
	if s.Yes {
		return nil
	}
	ok, err := s.Prompt.Confirm("Push moth's catalog into the live store(s) and wire notifications?", false)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("aborted")
	}
	return nil
}

func (s *BillingSetup) storeCredentials(ctx context.Context, rep *Report, projectID string, doApple, doGoogle, doStripe bool) error {
	req := &adminv1.UpdateBillingCredentialsRequest{ProjectId: projectID}
	if doApple {
		req.Apple = &adminv1.AppleBillingConfig{
			IapKeyId:           s.AppleIAPKeyID,
			IapIssuerId:        s.AppleIAPIssuerID,
			IapKeyP8:           string(s.AppleIAPKeyP8), // "" keeps the stored key
			BundleId:           s.AppleBundleID,
			AppAppleId:         s.AppleAppAppleID,
			NotificationSecret: s.AppleNotificationSecret,
		}
	}
	if doGoogle {
		req.Google = &adminv1.GoogleBillingConfig{
			ServiceAccountJson: string(s.GoogleServiceAccountJSON), // "" keeps the stored SA
			PackageName:        s.GooglePackageName,
			PubsubTopic:        s.GooglePubsubTopic,
			RtdnSecret:         s.GoogleRTDNSecret,
		}
	}
	if doStripe {
		req.Stripe = &adminv1.StripeBillingConfig{
			SecretKey: s.StripeSecretKey, // "" keeps the stored key
		}
	}
	if _, err := s.BillingCreds.UpdateBillingCredentials(ctx, connect.NewRequest(req)); err != nil {
		return fmt.Errorf("store billing credentials: %w", err)
	}
	rep.Pass("moth: billing credentials stored", "project "+s.Slug+" (secrets encrypted at rest)")
	return nil
}

// --- Apple ---------------------------------------------------------------

func (s *BillingSetup) runApple(ctx context.Context, rep *Report, projectID string, products []*adminv1.Product) {
	if s.ASC == nil || s.AppleAppID == "" {
		rep.Warn("Apple: catalog push", "no App Store Connect API key / app id provided — credentials stored, catalog not pushed",
			"re-run with --asc-p8/--asc-key-id/--asc-issuer-id and --apple-app-id to push the catalog")
	} else {
		cat, tiers := desiredCatalog(products, s.Slug, billing.StoreApple)
		if len(tiers) == 0 {
			rep.Skip("Apple: catalog push", "no product carries an Apple product id")
		} else {
			// Seed CurrentNotificationURL from what moth previously registered
			// (Apple exposes no read): the sync re-registers, and reports a
			// change, only when the URL actually differs — so a second run of
			// `moth setup billing` is idempotent for the notification hook too.
			ac := &AppleCatalog{
				ASC: s.ASC, AppID: s.AppleAppID,
				NotificationURL:        s.appleNotifyURL(),
				CurrentNotificationURL: s.storedNotificationURL(ctx, projectID),
			}
			res, err := ac.Sync(ctx, cat)
			if err != nil {
				rep.Fail("Apple: catalog push", err.Error(), "check the App Store Connect API key role (Admin/App Manager) and app id")
			} else {
				// Persist a freshly-registered URL so the next run recognises it
				// (Sync advances CurrentNotificationURL only on a successful
				// register, leaving it "" when the API 404s → guided fallback).
				s.persistNotificationURL(ctx, rep, projectID, ac.CurrentNotificationURL)
				reportSync(rep, "Apple", res)
			}
		}
	}
	s.verifyApple(ctx, rep)
}

// storedNotificationURL reads the App Store Server Notification URL moth last
// registered for the project (empty when none / on error), so the Apple catalog
// sync can diff against it and stay idempotent.
func (s *BillingSetup) storedNotificationURL(ctx context.Context, projectID string) string {
	resp, err := s.BillingCreds.GetBillingCredentials(ctx, connect.NewRequest(&adminv1.GetBillingCredentialsRequest{ProjectId: projectID}))
	if err != nil {
		return ""
	}
	return resp.Msg.GetApple().GetNotificationUrl()
}

// persistNotificationURL records the notification URL moth now has registered,
// but only when it changed from the stored value (a real registration this run);
// Apple has no read, so this persisted value is the sole idempotency anchor.
func (s *BillingSetup) persistNotificationURL(ctx context.Context, rep *Report, projectID, url string) {
	if url == "" || url == s.storedNotificationURL(ctx, projectID) {
		return
	}
	_, err := s.BillingCreds.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
		ProjectId: projectID,
		Apple: &adminv1.AppleBillingConfig{
			IapKeyId: s.AppleIAPKeyID, IapIssuerId: s.AppleIAPIssuerID,
			BundleId: s.AppleBundleID, AppAppleId: s.AppleAppAppleID,
			NotificationUrl: url,
		},
	}))
	if err != nil {
		rep.Warn("Apple: notification URL persisted", "could not record the registered URL: "+err.Error(),
			"re-run `moth setup billing`; the URL is still registered in App Store Connect")
	}
}

func (s *BillingSetup) verifyApple(ctx context.Context, rep *Report) {
	const name = "Apple: App Store Server API reachable"
	if s.AppleIAPKey == nil {
		rep.Warn(name, "no In-App-Purchase .p8 in hand this run, so the sandbox status read cannot be attempted",
			"re-run `moth setup billing` with --apple-iap-p8, or `moth doctor --project "+s.Slug+" --apple-iap-p8 <path>`")
		return
	}
	rep.add(appleServerAPIProbe(ctx, name, s.HTTPC, s.AppleServerAPIBase, s.AppleIAPIssuerID, s.AppleIAPKeyID, s.AppleBundleID, s.AppleIAPKey))
}

// appleServerAPIProbe reads the subscription status of a bogus transaction id
// against the App Store Server API: a 404 (billing.ErrNotFound) proves the
// In-App-Purchase key/issuer/bundle triple authenticated and the API is
// reachable (the transaction simply does not exist), while an auth error proves
// the key is wrong. Shared with `moth doctor`.
func appleServerAPIProbe(ctx context.Context, name string, httpc billing.Doer, base, issuerID, keyID, bundleID string, key *ecdsa.PrivateKey) Check {
	client := &billing.AppleClient{
		BaseURL:    base,
		SandboxURL: "", // no fallback: the probe host answers directly
		IssuerID:   issuerID,
		KeyID:      keyID,
		BundleID:   bundleID,
		Key:        key,
		HTTPC:      httpc,
	}
	_, err := client.GetAllSubscriptionStatuses(ctx, "0")
	switch {
	case err == nil:
		return Check{Name: name, Status: StatusPass, Detail: "authenticated (bundle " + bundleID + ")"}
	case errors.Is(err, billing.ErrNotFound):
		return Check{Name: name, Status: StatusPass,
			Detail: "authenticated (bundle " + bundleID + "); sandbox status read reached the API"}
	default:
		return Check{Name: name, Status: StatusFail,
			Detail:      "App Store Server API rejected the key: " + err.Error(),
			Remediation: "check the In-App-Purchase key id / issuer id / bundle id and that the .p8 is the App Store Server API key"}
	}
}

// --- Google --------------------------------------------------------------

func (s *BillingSetup) runGoogle(ctx context.Context, rep *Report, projectName string, products []*adminv1.Product) {
	var tokens *billing.GoogleTokenSource
	if s.GoogleSA != nil {
		tokens = billing.NewGoogleTokenSource(s.GoogleSA, s.GoogleTokenURL, s.HTTPC, nil)
	}
	if tokens == nil {
		rep.Warn("Google: catalog push", "no service-account JSON provided — credentials stored, catalog not pushed",
			"re-run with --google-service-account to push the catalog")
	} else {
		cat, tiers := desiredCatalog(products, s.Slug, billing.StoreGoogle)
		if len(tiers) == 0 {
			// No product carries a Google SKU yet, but the RTDN plumbing must
			// still be wired so milestone-11 renewal events reach moth (matching
			// the admin syncer, which wires RTDN whenever a topic is configured
			// regardless of tier count). The catalog push itself is skipped.
			rep.Skip("Google: catalog push", "no product carries a Google product id")
			res := &SyncResult{Store: billing.StoreGoogle}
			s.wireGoogleRTDN(ctx, res)
			reportNotifications(rep, "Google", res)
		} else {
			gc := &GoogleCatalog{BaseURL: s.GoogleCatalogBaseURL, PackageName: s.GooglePackageName, Tokens: tokens, HTTPC: s.HTTPC}
			res, err := gc.Sync(ctx, cat)
			if err != nil {
				rep.Fail("Google: catalog push", err.Error(), "check the service account has Play Developer API access to "+s.GooglePackageName)
			} else {
				s.wireGoogleRTDN(ctx, res)
				reportSync(rep, "Google", res)
			}
		}
	}
	s.verifyGoogle(ctx, rep, tokens)
}

// wireGoogleRTDN creates the RTDN topic + push subscription (when a pubsub-scoped
// token source is available) or emits the guided steps, appending to res.
func (s *BillingSetup) wireGoogleRTDN(ctx context.Context, res *SyncResult) {
	if s.GooglePubsubTopic == "" {
		return
	}
	topicID := s.GooglePubsubTopic
	if i := strings.LastIndex(topicID, "/"); i >= 0 {
		topicID = topicID[i+1:]
	}
	var ps *PubSub
	if s.GooglePubSubTokens != nil && s.GoogleCloudProject != "" {
		ps = &PubSub{BaseURL: s.PubSubBaseURL, Project: s.GoogleCloudProject, Tokens: s.GooglePubSubTokens, HTTPC: s.HTTPC}
	}
	subID := "moth-" + s.Slug + "-rtdn"
	if err := WireRTDN(ctx, ps, topicID, subID, s.googleRTDNURL(), res); err != nil {
		res.addNotification(NotificationResult{Kind: "google_rtdn_topic", Action: ActionFailed, Detail: err.Error()})
	}
}

func (s *BillingSetup) verifyGoogle(ctx context.Context, rep *Report, tokens *billing.GoogleTokenSource) {
	const name = "Google: Play Developer API reachable"
	if tokens == nil {
		rep.Warn(name, "no service-account JSON in hand this run, so the API cannot be probed",
			"re-run `moth setup billing` with --google-service-account")
		return
	}
	if _, err := tokens.Token(ctx); err != nil {
		rep.Fail(name, "service account could not obtain an access token: "+err.Error(),
			"check the service-account JSON is valid and has Play Developer API access")
		return
	}
	client := &billing.GoogleClient{BaseURL: s.GoogleCatalogBaseURL, PackageName: s.GooglePackageName, Tokens: tokens, HTTPC: s.HTTPC}
	_, _, err := client.GetSubscriptionV2(ctx, "moth-setup-probe")
	switch {
	case err == nil, errors.Is(err, billing.ErrNotFound):
		rep.Pass(name, "authenticated for "+s.GooglePackageName)
	default:
		rep.Fail(name, "Play Developer API rejected the request: "+err.Error(),
			"grant the service account Play Developer API access to "+s.GooglePackageName)
	}
}

// --- Stripe --------------------------------------------------------------

// stripeWebhookURL is moth's Stripe webhook endpoint for the project.
func (s *BillingSetup) stripeWebhookURL() string {
	return strings.TrimSuffix(s.BaseURL, "/") + "/billing/stripe/webhook/" + s.Slug
}

// runStripe pushes the catalog into Stripe (Products + recurring Prices, ids
// written back onto moth's products), provisions the webhook endpoint (signing
// secret persisted — Stripe reveals it exactly once), and verifies the key
// authenticates. Everything is automated: unlike App Store Connect, the
// Stripe API can do it all (plan/17), so there is no guided fallback — only
// the missing-key degradation, which warns and keeps stored credentials.
func (s *BillingSetup) runStripe(ctx context.Context, rep *Report, projectID string, products []*adminv1.Product) {
	if s.StripeSecretKey == "" {
		rep.Warn("Stripe: catalog push", "no secret key in hand this run (moth stores it encrypted and never returns it) — credentials kept, catalog not pushed",
			"re-run with --stripe-secret-key to push the catalog and wire the webhook")
		return
	}
	sc := &StripeCatalog{BaseURL: s.StripeBaseURL, SecretKey: s.StripeSecretKey, HTTPC: s.HTTPC}
	cat, tiers := desiredCatalog(products, s.Slug, billing.StoreStripe)
	if len(tiers) == 0 {
		rep.Skip("Stripe: catalog push", "no product maps onto Stripe (no products, or unsupported billing periods)")
	} else {
		res, err := sc.Sync(ctx, cat)
		if err != nil {
			rep.Fail("Stripe: catalog push", err.Error(), "check the secret key has write access to Products, Prices and Webhook Endpoints")
		} else {
			s.writeBackStripeIDs(ctx, rep, projectID, products, res)
			reportSync(rep, "Stripe", res)
		}
	}
	s.wireStripeWebhook(ctx, rep, projectID, sc)
	s.verifyStripe(ctx, rep, sc)
}

// writeBackStripeIDs records the Stripe ids provisioning produced onto moth's
// products (stripe_price_id / stripe_product_id), so the next sync diffs
// against them and runtime product matching resolves Stripe prices.
func (s *BillingSetup) writeBackStripeIDs(ctx context.Context, rep *Report, projectID string, products []*adminv1.Product, res *SyncResult) {
	byIdentifier := make(map[string]*adminv1.Product, len(products))
	for _, p := range products {
		byIdentifier[p.GetIdentifier()] = p
	}
	for _, pr := range res.Products {
		p, ok := byIdentifier[pr.ProductID]
		// A failed tier can still carry a created Product id (price-stage
		// failure): record it so a re-run reuses the Product instead of
		// provisioning a duplicate.
		if !ok || (pr.StoreID == "" && pr.StoreParentID == "") {
			continue
		}
		priceRecorded := pr.StoreID == "" || p.GetStripePriceId() == pr.StoreID
		productRecorded := pr.StoreParentID == "" || p.GetStripeProductId() == pr.StoreParentID
		if priceRecorded && productRecorded {
			continue // already recorded
		}
		if pr.StoreID != "" {
			p.StripePriceId = pr.StoreID
		}
		if pr.StoreParentID != "" {
			p.StripeProductId = pr.StoreParentID
		}
		_, err := s.Products.UpdateProduct(ctx, connect.NewRequest(&adminv1.UpdateProductRequest{
			ProjectId: projectID, Id: p.GetId(), Product: p,
		}))
		if err != nil {
			rep.Warn("Stripe: product ids recorded",
				"could not record price "+pr.StoreID+" / product "+pr.StoreParentID+" on "+pr.ProductID+": "+err.Error(),
				"set stripe_price_id / stripe_product_id on the product yourself (admin → Monetization, ids above) — "+
					"moth cannot look the resources up again, so a re-run would provision NEW Stripe resources, not recover these")
		}
	}
}

// wireStripeWebhook idempotently provisions the webhook endpoint and persists
// the signing secret + endpoint id into moth's billing config. Stripe reveals
// the secret only at creation: an endpoint that already exists without a
// stored secret is a warning with the honest remediation, never a silent pass.
func (s *BillingSetup) wireStripeWebhook(ctx context.Context, rep *Report, projectID string, sc *StripeCatalog) {
	const name = "Stripe: webhook endpoint"
	url := s.stripeWebhookURL()
	ep, created, repaired, err := sc.EnsureWebhookEndpoint(ctx, url)
	if err != nil {
		rep.Fail(name, err.Error(), "check the secret key can manage webhook endpoints, then re-run")
		return
	}
	// A disabled endpoint (or one missing moth events) was repaired in place;
	// surface it — the existing endpoint was NOT silently taken at face value.
	repairedNote := ""
	if repaired {
		repairedNote = "; endpoint re-enabled / events updated to the moth set"
	}
	if created {
		_, err := s.BillingCreds.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
			ProjectId: projectID,
			Stripe: &adminv1.StripeBillingConfig{
				// SecretKey "" keeps the stored one; the signing secret is
				// revealed by Stripe exactly once, right now — persist or lose.
				WebhookSecret:     ep.Secret,
				WebhookEndpointId: ep.ID,
			},
		}))
		if err != nil {
			rep.Fail(name, "endpoint "+ep.ID+" created but the signing secret could not be stored: "+err.Error(),
				"delete the endpoint in the Stripe dashboard and re-run (the secret is only revealed at creation)")
			return
		}
		rep.Pass(name, "created "+ep.ID+" → "+url+" (signing secret stored)")
		return
	}
	// Already registered: confirm moth actually holds its signing secret.
	stored, err := s.BillingCreds.GetBillingCredentials(ctx, connect.NewRequest(&adminv1.GetBillingCredentialsRequest{ProjectId: projectID}))
	if err != nil {
		rep.Warn(name, "endpoint "+ep.ID+" already registered, but the stored config could not be read: "+err.Error(), "")
		return
	}
	if !stored.Msg.GetStripe().GetHasWebhookSecret() {
		rep.Warn(name, "endpoint "+ep.ID+" already registered but moth holds no signing secret (Stripe reveals it only at creation) — webhook deliveries will be rejected"+repairedNote,
			"delete the endpoint in the Stripe dashboard and re-run, or paste its signing secret in the admin (Monetization → store credentials)")
		return
	}
	// Record the endpoint id if a manual/dashboard registration left it blank.
	if stored.Msg.GetStripe().GetWebhookEndpointId() != ep.ID {
		_, _ = s.BillingCreds.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
			ProjectId: projectID,
			Stripe:    &adminv1.StripeBillingConfig{WebhookEndpointId: ep.ID},
		}))
	}
	rep.Pass(name, "already registered ("+ep.ID+" → "+url+")"+repairedNote)
}

// verifyStripe reads a bogus price id against the Stripe API: a 404
// (billing.ErrNotFound) proves the secret key authenticated and the API is
// reachable, while an auth error proves the key is wrong — the same
// known-404-is-success semantics as appleServerAPIProbe.
func (s *BillingSetup) verifyStripe(ctx context.Context, rep *Report, sc *StripeCatalog) {
	const name = "Stripe: API reachable"
	_, err := sc.client().GetPrice(ctx, "price_moth_setup_probe")
	switch {
	case err == nil, errors.Is(err, billing.ErrNotFound):
		rep.Pass(name, "authenticated; price read reached the API")
	default:
		rep.Fail(name, "Stripe rejected the request: "+err.Error(),
			"check the secret key (sk_/rk_) is valid and has read access to Prices")
	}
}

// --- catalog mapping -----------------------------------------------------

// DesiredCatalogFromProducts builds the store DesiredCatalog from moth's admin
// product list for one store (billing.StoreApple / StoreGoogle / StoreStripe).
// It is the single mapping shared by `moth setup billing` and the admin
// MonetizationService handler — "one catalog, three faces" (plan/12): the Tiers
// slice is empty when no product targets the store.
func DesiredCatalogFromProducts(products []*adminv1.Product, slug, storeName string) DesiredCatalog {
	cat, _ := desiredCatalog(products, slug, storeName)
	return cat
}

// desiredCatalog builds the DesiredCatalog for one store from moth's products,
// mapped to DesiredTiers. For Apple and Google only products carrying that
// store's authored SKU are included; for Stripe every product is eligible —
// Stripe ids are generated by provisioning, so the tier keys on moth's own
// identifier and carries the currently-recorded Stripe ids for the diff. It
// returns the catalog and its tier slice (empty when no product targets the
// store). A product whose billing_period does not map to a shared cadence is
// dropped from the Apple/Google push (those stores only see products whose SKU
// was authored for them); for Stripe — which pushes every product — the tier is
// kept with the raw period so the sync reports an honest per-tier failure
// instead of silently omitting it.
func desiredCatalog(products []*adminv1.Product, slug, storeName string) (DesiredCatalog, []DesiredTier) {
	cat := DesiredCatalog{GroupReference: "moth-" + slug}
	for _, p := range products {
		var sku string
		switch storeName {
		case billing.StoreApple:
			sku = p.GetAppleProductId()
		case billing.StoreGoogle:
			sku = p.GetGoogleProductId()
		case billing.StoreStripe:
			sku = p.GetIdentifier()
		}
		if sku == "" {
			continue
		}
		period, ok := parseBillingPeriod(p.GetBillingPeriod())
		if !ok {
			if storeName != billing.StoreStripe {
				continue
			}
			// Stripe pushes every product, so an unmappable billing period must
			// fail loudly rather than vanish: carry the raw value through so
			// StripeCatalog.syncTier's StripeRecurringForPeriod rejection turns
			// it into an honest per-tier ActionFailed (plan/status count this
			// product; silently dropping it would let the sync claim parity).
			period = BillingPeriod(p.GetBillingPeriod())
		}
		tier := DesiredTier{
			ProductID:   sku,
			Reference:   p.GetIdentifier(),
			DisplayName: p.GetDisplayName(),
			Description: p.GetDisplayName(),
			Period:      period,
			Price:       Money{Currency: p.GetCurrency(), Micros: p.GetPriceAmountMicros()},
			Locale:      "en-US",
			GroupLevel:  int(p.GetSortOrder()) + 1,
		}
		if storeName == billing.StoreStripe {
			tier.StripePriceID = p.GetStripePriceId()
			tier.StripeProductID = p.GetStripeProductId()
		}
		if p.GetIntroPeriod() != "" {
			if ip, ok := parseBillingPeriod(p.GetIntroPeriod()); ok {
				tier.Intro = &IntroOffer{
					Period:    ip,
					FreeTrial: p.GetIntroPriceAmountMicros() == 0,
					Price:     Money{Currency: p.GetCurrency(), Micros: p.GetIntroPriceAmountMicros()},
				}
			}
		}
		cat.Tiers = append(cat.Tiers, tier)
	}
	return cat, cat.Tiers
}

// parseBillingPeriod maps a product's free-form billing_period onto the shared
// BillingPeriod cadences both stores support.
func parseBillingPeriod(s string) (BillingPeriod, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "weekly", "week", "p1w":
		return PeriodWeekly, true
	case "monthly", "month", "p1m":
		return PeriodMonthly, true
	case "two_month", "twomonth", "2month", "p2m":
		return PeriodTwoMonth, true
	case "quarterly", "quarter", "3month", "p3m":
		return PeriodQuarterly, true
	case "half_year", "halfyear", "6month", "p6m":
		return PeriodHalfYear, true
	case "yearly", "annual", "year", "p1y":
		return PeriodYearly, true
	}
	return "", false
}

// reportSync renders a store SyncResult into the checklist: one summary check
// per product action, the notification wiring, and every guided ManualStep
// (honest automation: never silently skipped).
func reportSync(rep *Report, store string, res *SyncResult) {
	created, updated, unchanged := 0, 0, 0
	for _, p := range res.Products {
		switch p.Action {
		case ActionCreated:
			created++
		case ActionUpdated:
			updated++
		case ActionUnchanged:
			unchanged++
		case ActionManual, ActionFailed:
		}
		if p.Action == ActionFailed {
			rep.Fail(store+": product "+p.ProductID, p.Detail, "re-run after resolving the error")
		}
	}
	detail := fmt.Sprintf("%d created, %d updated, %d unchanged", created, updated, unchanged)
	if res.Changed() {
		rep.Pass(store+": catalog pushed", detail)
	} else {
		rep.Pass(store+": catalog in sync", detail+" — no changes")
	}
	reportNotifications(rep, store, res)
}

// reportNotifications renders the notification wiring + guided ManualSteps of a
// SyncResult, without the per-product catalog summary. Used on its own when the
// RTDN plumbing is wired but no catalog push ran (no Google SKUs mapped yet).
func reportNotifications(rep *Report, store string, res *SyncResult) {
	for _, n := range res.Notifications {
		reportNotification(rep, store, n)
	}
	for _, m := range res.ManualSteps {
		reportManual(rep, store, m)
	}
}

func reportNotification(rep *Report, store string, n NotificationResult) {
	name := store + ": notification " + n.Kind
	switch n.Action {
	case ActionCreated, ActionUpdated:
		rep.Pass(name, "registered "+n.Endpoint)
	case ActionUnchanged:
		rep.Pass(name, "already registered")
	case ActionManual:
		rep.Warn(name, n.Detail, "see the guided step below")
	case ActionFailed:
		rep.Fail(name, n.Detail, "")
	}
}

func reportManual(rep *Report, store string, m ManualStep) {
	detail := m.Reason
	remediation := m.URL
	if len(m.Instructions) > 0 {
		remediation = strings.Join(m.Instructions, " ")
		if m.URL != "" {
			remediation = m.URL + " — " + remediation
		}
	}
	rep.Warn(store+": manual step — "+m.Title, detail, remediation)
}
