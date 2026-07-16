package setup

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/billing"
)

// fakeProducts serves a fixed product catalog to BillingSetup / Doctor.
type fakeProducts struct {
	adminv1connect.ProductServiceClient
	products []*adminv1.Product
}

func (f *fakeProducts) ListProducts(context.Context, *connect.Request[adminv1.ListProductsRequest]) (*connect.Response[adminv1.ListProductsResponse], error) {
	return connect.NewResponse(&adminv1.ListProductsResponse{Products: f.products}), nil
}

// fakeBillingCreds records credential writes and answers reads from an
// in-memory config so a re-read reports accurate has_* flags.
type fakeBillingCreds struct {
	adminv1connect.BillingCredentialsServiceClient
	apple   *adminv1.AppleBillingConfig
	google  *adminv1.GoogleBillingConfig
	updates int
}

func (f *fakeBillingCreds) GetBillingCredentials(context.Context, *connect.Request[adminv1.GetBillingCredentialsRequest]) (*connect.Response[adminv1.GetBillingCredentialsResponse], error) {
	a, g := f.apple, f.google
	if a == nil {
		a = &adminv1.AppleBillingConfig{}
	}
	if g == nil {
		g = &adminv1.GoogleBillingConfig{}
	}
	return connect.NewResponse(&adminv1.GetBillingCredentialsResponse{Apple: a, Google: g}), nil
}

func (f *fakeBillingCreds) UpdateBillingCredentials(_ context.Context, req *connect.Request[adminv1.UpdateBillingCredentialsRequest]) (*connect.Response[adminv1.UpdateBillingCredentialsResponse], error) {
	f.updates++
	// Reflect the write into the stored config (has_* driven by presence).
	if a := req.Msg.Apple; a != nil {
		// Mirror the server's write-only/keep-stored semantics for the pieces the
		// tests assert: an empty notification URL / p8 / secret keeps the stored
		// value, everything else is overwritten.
		prev := f.apple
		if prev == nil {
			prev = &adminv1.AppleBillingConfig{}
		}
		notifURL := a.NotificationUrl
		if notifURL == "" {
			notifURL = prev.NotificationUrl
		}
		hasKey := a.IapKeyP8 != "" || prev.HasIapKey
		hasSecret := a.NotificationSecret != "" || prev.HasNotificationSecret
		f.apple = &adminv1.AppleBillingConfig{
			IapKeyId: a.IapKeyId, IapIssuerId: a.IapIssuerId, BundleId: a.BundleId,
			AppAppleId: a.AppAppleId, HasIapKey: hasKey,
			HasNotificationSecret: hasSecret, NotificationUrl: notifURL,
		}
	}
	if g := req.Msg.Google; g != nil {
		f.google = &adminv1.GoogleBillingConfig{
			PackageName: g.PackageName, PubsubTopic: g.PubsubTopic,
			HasServiceAccount: g.ServiceAccountJson != "", HasRtdnSecret: g.RtdnSecret != "",
		}
	}
	return connect.NewResponse(&adminv1.UpdateBillingCredentialsResponse{Apple: f.apple, Google: f.google}), nil
}

// appleServerAPIDouble stands in for the App Store Server API: it answers 404
// for any subscription-status read (proving the JWT authenticated — the
// transaction simply does not exist), unless failAuth is set.
func appleServerAPIDouble(t *testing.T, failAuth bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if failAuth || !strings.HasPrefix(auth, "Bearer ") || strings.Count(auth, ".") != 2 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errorCode":401,"errorMessage":"invalid token"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorCode":4040010,"errorMessage":"Transaction id not found."}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func testProducts() []*adminv1.Product {
	return []*adminv1.Product{{
		Id: "prod1", Identifier: "monthly", DisplayName: "Pro",
		AppleProductId: "pro_monthly", GoogleProductId: "pro_monthly",
		BillingPeriod: "monthly", PriceAmountMicros: 9_990_000, Currency: "USD", SortOrder: 0,
	}}
}

func newBillingSetup(t *testing.T, asc *ascCatalogDouble, gp *googleCatalogDouble, appleAPI *httptest.Server, out *bytes.Buffer) (*BillingSetup, *fakeBillingCreds) {
	t.Helper()
	// App Store Connect has no public server-notification-URL API, so the
	// registration always degrades to a guided step (never a store change) —
	// mirror that so idempotency holds across runs.
	asc.notify404 = true
	_, iapKey := testP8(t)
	creds := &fakeBillingCreds{}
	saJSON := serviceAccountBytes(t, gp.srv.URL+"/token")
	sa, err := billing.ParseServiceAccount(saJSON)
	if err != nil {
		t.Fatal(err)
	}
	s := &BillingSetup{
		Projects:     &fakeProjects{projects: []*adminv1.Project{testProject("demo")}},
		Products:     &fakeProducts{products: testProducts()},
		BillingCreds: creds,
		Prompt:       NewPrompter(strings.NewReader(""), out),
		Out:          out,
		BaseURL:      "https://moth.example.com",
		Slug:         "demo",
		Yes:          true,
		HTTPC:        http.DefaultClient,

		AppleBundleID:    "com.example.demo",
		AppleAppAppleID:  "1234567890",
		AppleAppID:       "APP1",
		AppleIAPKeyID:    "IAPKEY0001",
		AppleIAPIssuerID: "11112222-3333-4444-5555-666677778888",
		AppleIAPKey:      iapKey,
		AppleIAPKeyP8:    []byte("p8-bytes"),
		ASC: &ASC{
			IssuerID: "iss", KeyID: "kid", Key: iapKey,
			BaseURL: asc.srv.URL, HTTPC: asc.srv.Client(),
			Now: func() time.Time { return time.Unix(1_752_000_000, 0) },
		},
		AppleServerAPIBase: appleAPI.URL,

		GooglePackageName:        "com.example.app",
		GoogleServiceAccountJSON: saJSON,
		GoogleSA:                 sa,
		GoogleCatalogBaseURL:     gp.srv.URL,
		GoogleTokenURL:           gp.srv.URL + "/token",
		GooglePubsubTopic:        "projects/demo-proj/topics/moth-rtdn",
		GoogleRTDNSecret:         "rtdn-s3cr3t",
		// A pubsub-scoped token source drives the real RTDN topic + push
		// subscription creation against the double (Cloud Pub/Sub surface).
		GooglePubSubTokens: googleTokens(t, gp.srv.URL+"/token"),
		PubSubBaseURL:      gp.srv.URL,
		GoogleCloudProject: "demo-proj",
	}
	return s, creds
}

func TestBillingSetupHappyPath(t *testing.T) {
	asc := newASCCatalogDouble(t)
	gp := newGoogleCatalogDouble(t)
	appleAPI := appleServerAPIDouble(t, false)
	var out bytes.Buffer
	s, creds := newBillingSetup(t, asc, gp, appleAPI, &out)

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if creds.updates != 1 {
		t.Fatalf("expected 1 credential write, got %d", creds.updates)
	}
	// Credentials were stored with the secrets present.
	if !creds.apple.HasIapKey || creds.apple.BundleId != "com.example.demo" {
		t.Fatalf("apple creds = %+v", creds.apple)
	}
	if !creds.google.HasServiceAccount || creds.google.PackageName != "com.example.app" {
		t.Fatalf("google creds = %+v", creds.google)
	}
	// Catalog pushed to both stores.
	assertCheck(t, rep, "Apple: catalog pushed", StatusPass)
	assertCheck(t, rep, "Google: catalog pushed", StatusPass)
	// Store APIs verified reachable + authenticated.
	assertCheck(t, rep, "Apple: App Store Server API reachable", StatusPass)
	assertCheck(t, rep, "Google: Play Developer API reachable", StatusPass)
	if rep.Failed() {
		t.Fatalf("report has failures:\n%s", reportString(rep))
	}
	if !asc.authSeen {
		t.Fatal("ASC catalog push did not authenticate")
	}
	// The RTDN push subscription must be registered with the shared-secret
	// ?token= query, or Google's tokenless deliveries are rejected 401 by the
	// webhook (handleGoogleRTDN -> AuthenticatePushToken).
	var pushEndpoint string
	for _, sub := range gp.pushSubs {
		pc, _ := sub["pushConfig"].(map[string]any)
		if pc != nil {
			pushEndpoint, _ = pc["pushEndpoint"].(string)
		}
	}
	if pushEndpoint == "" {
		t.Fatal("no RTDN push subscription was created")
	}
	if !strings.Contains(pushEndpoint, "token=rtdn-s3cr3t") {
		t.Fatalf("RTDN push endpoint %q missing the ?token= shared secret", pushEndpoint)
	}
}

// TestBillingSetupWiresRTDNWithoutGoogleProducts covers the Apple-first launch:
// a project with Google credentials + a Pub/Sub topic but no product carrying a
// Google SKU yet. The catalog push is skipped, but the RTDN topic + push
// subscription must still be wired so milestone-11 renewal events reach moth.
func TestBillingSetupWiresRTDNWithoutGoogleProducts(t *testing.T) {
	asc := newASCCatalogDouble(t)
	gp := newGoogleCatalogDouble(t)
	appleAPI := appleServerAPIDouble(t, false)
	var out bytes.Buffer
	s, _ := newBillingSetup(t, asc, gp, appleAPI, &out)
	// Products carry an Apple SKU only — no Google product id.
	s.Products = &fakeProducts{products: []*adminv1.Product{{
		Id: "prod1", Identifier: "monthly", DisplayName: "Pro",
		AppleProductId: "pro_monthly", BillingPeriod: "monthly",
		PriceAmountMicros: 9_990_000, Currency: "USD",
	}}}

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Catalog push skipped (no Google SKU) ...
	assertCheck(t, rep, "Google: catalog push", StatusSkip)
	// ... but the RTDN topic + push subscription were still created.
	if len(gp.topics) == 0 {
		t.Fatal("RTDN topic was not created despite a configured Pub/Sub topic")
	}
	if len(gp.pushSubs) == 0 {
		t.Fatal("RTDN push subscription was not created without a Google product mapped")
	}
	for _, sub := range gp.pushSubs {
		pc, _ := sub["pushConfig"].(map[string]any)
		ep, _ := pc["pushEndpoint"].(string)
		if !strings.Contains(ep, "token=rtdn-s3cr3t") {
			t.Fatalf("RTDN push endpoint %q missing the ?token= secret", ep)
		}
	}
}

func TestBillingSetupIdempotentSecondRun(t *testing.T) {
	asc := newASCCatalogDouble(t)
	gp := newGoogleCatalogDouble(t)
	appleAPI := appleServerAPIDouble(t, false)
	var out bytes.Buffer
	s, _ := newBillingSetup(t, asc, gp, appleAPI, &out)

	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	// Second run: catalog reports "in sync", no changes.
	assertCheck(t, rep, "Apple: catalog in sync", StatusPass)
	assertCheck(t, rep, "Google: catalog in sync", StatusPass)
	for _, name := range []string{"Apple: catalog pushed", "Google: catalog pushed"} {
		if hasCheck(rep, name) {
			t.Fatalf("second run should report no changes, but found %q:\n%s", name, reportString(rep))
		}
	}
}

// TestBillingSetupNotificationURLIdempotent proves the Apple server-notification
// registration is idempotent across separate CLI invocations: once moth has
// registered the URL (and persisted it, since Apple exposes no read), a second
// run recognises it and does not re-register or report a change.
func TestBillingSetupNotificationURLIdempotent(t *testing.T) {
	asc := newASCCatalogDouble(t)
	gp := newGoogleCatalogDouble(t)
	appleAPI := appleServerAPIDouble(t, false)
	var out bytes.Buffer
	s, creds := newBillingSetup(t, asc, gp, appleAPI, &out)
	asc.notify404 = false // pretend App Store Connect exposes the notification API

	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if asc.notifyCalls != 1 {
		t.Fatalf("first run should register the notification URL once, got %d", asc.notifyCalls)
	}
	if creds.apple.NotificationUrl == "" {
		t.Fatal("notification URL was not persisted after a successful registration")
	}

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if asc.notifyCalls != 1 {
		t.Fatalf("second run re-registered the notification URL (calls=%d) — not idempotent", asc.notifyCalls)
	}
	assertCheck(t, rep, "Apple: notification apple_server_notification_url", StatusPass)
	if hasCheck(rep, "Apple: catalog pushed") {
		t.Fatalf("second run should report no Apple changes:\n%s", reportString(rep))
	}
}

func TestBillingSetupRejectsBadInput(t *testing.T) {
	var out bytes.Buffer
	// No Apple bundle id and no Google package: nothing to configure.
	s := &BillingSetup{
		Projects:     &fakeProjects{projects: []*adminv1.Project{testProject("demo")}},
		Products:     &fakeProducts{},
		BillingCreds: &fakeBillingCreds{},
		Prompt:       NewPrompter(strings.NewReader(""), &out),
		Out:          &out,
		Slug:         "demo",
		Yes:          true,
	}
	if _, err := s.Run(context.Background()); err == nil {
		t.Fatal("expected an error when no store is configured")
	}

	// Unknown project slug carries the not-found code.
	s.AppleBundleID = "com.example.demo"
	s.Slug = "nope"
	_, err := s.Run(context.Background())
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown slug code = %v, want not_found", connect.CodeOf(err))
	}
}

func TestBillingSetupBadServiceAccountRejected(t *testing.T) {
	// A malformed service-account JSON fails shape validation at parse time,
	// before any store call — the CLI parses it via billing.ParseServiceAccount.
	if _, err := billing.ParseServiceAccount([]byte(`{"client_email":"x"}`)); err == nil {
		t.Fatal("expected malformed service account to be rejected")
	}
}

// assertCheck fails unless the report has a check with the given name+status.
func assertCheck(t *testing.T, rep *Report, name string, status Status) {
	t.Helper()
	for _, c := range rep.Checks {
		if c.Name == name {
			if c.Status != status {
				t.Fatalf("check %q status = %s, want %s (detail %q)", name, c.Status, status, c.Detail)
			}
			return
		}
	}
	t.Fatalf("no check named %q in:\n%s", name, reportString(rep))
}

func hasCheck(rep *Report, name string) bool {
	for _, c := range rep.Checks {
		if c.Name == name {
			return true
		}
	}
	return false
}

func reportString(rep *Report) string {
	b, _ := rep.JSON()
	return string(b)
}

// serviceAccountBytes builds a real service-account JSON with a fresh RSA key
// whose token_uri points at tokenURL, so the SA token exchange is exercised.
func serviceAccountBytes(t *testing.T, tokenURL string) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemKey := "-----BEGIN RSA PRIVATE KEY-----\n" + wrap64(base64.StdEncoding.EncodeToString(der)) + "-----END RSA PRIVATE KEY-----\n"
	saJSON, _ := json.Marshal(map[string]string{
		"type": "service_account", "client_email": "moth@proj.iam.gserviceaccount.com",
		"private_key_id": "kid-123", "private_key": pemKey, "token_uri": tokenURL,
	})
	return saJSON
}
