package setup

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

// instanceDouble serves the plain-HTTP surface doctor probes. Handlers are
// per-path so tests break individual checks.
func instanceDouble(t *testing.T, broken map[string]bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if broken[r.URL.Path] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.URL.Path {
		case "/healthz":
			w.Write([]byte("ok"))
		case "/pub/api/packages/moth_auth":
			w.Write([]byte(`{"name":"moth_auth","versions":[{"version":"0.1.0"}]}`))
		case "/p/demo/.well-known/jwks.json":
			w.Write([]byte(`{"keys":[{"kty":"EC","crv":"P-256"}]}`))
		case "/billing/apple/notifications/demo", "/billing/google/rtdn/demo",
			"/billing/stripe/webhook/demo":
			// Wired webhook routes reject the probe's empty POST before doing any
			// work; any non-404 status means "route exists". (400 mirrors the real
			// Apple handler's "invalid notification body".)
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// googleAPIRejectsDouble mints a valid access token but rejects every Play
// Developer API call with 403 — an SA whose Play access was revoked. It drives
// googlePlayAPIProbe's "API rejected the request" FAIL branch.
func googleAPIRejectsDouble(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/token") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"ya29.test","expires_in":3600}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":403,"message":"caller does not have permission"}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newDoctor wires a doctor against the doubles with a healthy default
// configuration for project "demo".
func newDoctor(t *testing.T, srv *httptest.Server, project *adminv1.Project) (*Doctor, *fakeSettings) {
	t.Helper()
	auth := googleAuthDouble(t, nil)
	settings := &fakeSettings{baseURL: srv.URL, source: adminv1.SmtpSource_SMTP_SOURCE_CONFIG, host: "smtp.example.com"}
	return &Doctor{
		BaseURL:       srv.URL,
		HTTPC:         srv.Client(),
		Session:       &fakeSession{email: "ops@example.com", version: "dev"},
		Settings:      settings,
		Projects:      &fakeProjects{projects: []*adminv1.Project{project}},
		GoogleAuthURL: auth.URL,
	}, settings
}

func healthyProject() *adminv1.Project {
	p := testProject("demo")
	p.Settings.Google = &adminv1.GoogleProviderConfig{
		Enabled: true, WebClientId: testWebID, HasWebClientSecret: true,
	}
	p.Settings.Apple = &adminv1.AppleProviderConfig{
		Enabled:       true,
		ServicesId:    "com.example.demo.signin",
		TeamId:        "TEAM123456",
		KeyId:         "SIWAKEY001",
		HasPrivateKey: true,
		BundleIds:     []string{"com.example.demo"},
	}
	return p
}

func TestDoctorHealthyInstanceAndProject(t *testing.T) {
	srv := instanceDouble(t, nil)
	d, _ := newDoctor(t, srv, healthyProject())
	d.Slug = "demo"

	rep, err := d.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed() {
		t.Fatalf("healthy setup reported failure: %+v", rep.Checks)
	}
	wantPass := []string{
		"instance: admin API reachable and authenticated",
		"instance: base URL sanity",
		"instance: health endpoint",
		"instance: pub endpoint serves the SDK",
		"instance: outgoing email (SMTP)",
		"project: exists",
		"project: JWKS reachable",
		"project: Google web client ID resolves",
		"project: Apple sign-in",
	}
	for _, name := range wantPass {
		if c := findCheck(t, rep, name); c.Status != StatusPass {
			t.Fatalf("%s = %s (%s)", name, c.Status, c.Detail)
		}
	}
	// No local Apple key: the dry-run honestly warns.
	if c := findCheck(t, rep, "project: Apple key accepted (token endpoint dry-run)"); c.Status != StatusWarn {
		t.Fatalf("apple dry-run = %+v", c)
	}
}

func TestDoctorDetectsProblems(t *testing.T) {
	t.Run("broken jwks", func(t *testing.T) {
		srv := instanceDouble(t, map[string]bool{"/p/demo/.well-known/jwks.json": true})
		d, _ := newDoctor(t, srv, healthyProject())
		d.Slug = "demo"
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if c := findCheck(t, rep, "project: JWKS reachable"); c.Status != StatusFail {
			t.Fatalf("jwks check = %+v", c)
		}
		if !rep.Failed() {
			t.Fatal("report should fail")
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		d, _ := newDoctor(t, srv, healthyProject())
		d.Session = &fakeSession{err: connect.NewError(connect.CodeUnauthenticated, errors.New("bad token"))}
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Checks) != 1 || rep.Checks[0].Status != StatusFail {
			t.Fatalf("checks = %+v", rep.Checks)
		}
	})

	t.Run("no smtp warns", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		d, settings := newDoctor(t, srv, healthyProject())
		settings.source = adminv1.SmtpSource_SMTP_SOURCE_NONE
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if c := findCheck(t, rep, "instance: outgoing email (SMTP)"); c.Status != StatusWarn {
			t.Fatalf("smtp check = %+v", c)
		}
	})

	t.Run("base url mismatch warns", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		d, settings := newDoctor(t, srv, healthyProject())
		settings.baseURL = "https://elsewhere.example.com"
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		c := findCheck(t, rep, "instance: base URL sanity")
		if c.Status != StatusWarn || !strings.Contains(c.Detail, "elsewhere.example.com") {
			t.Fatalf("base url check = %+v", c)
		}
	})

	t.Run("deleted google client fails", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		d, _ := newDoctor(t, srv, healthyProject())
		auth := googleAuthDouble(t, map[string]string{testWebID: "not_found"})
		d.GoogleAuthURL = auth.URL
		d.Slug = "demo"
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if c := findCheck(t, rep, "project: Google web client ID resolves"); c.Status != StatusFail {
			t.Fatalf("google check = %+v", c)
		}
	})

	t.Run("unregistered web redirect URI fails", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		d, _ := newDoctor(t, srv, healthyProject())
		auth := googleAuthDouble(t, map[string]string{testWebID: "redirect_mismatch"})
		d.GoogleAuthURL = auth.URL
		d.Slug = "demo"
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		c := findCheck(t, rep, "project: Google web redirect URI registered")
		if c.Status != StatusFail || !strings.Contains(c.Detail, "/oauth/google/callback") {
			t.Fatalf("redirect check = %+v", c)
		}
		if !rep.Failed() {
			t.Fatal("report should fail")
		}
	})

	t.Run("redirect mismatch is moot for installed-app clients", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		project := healthyProject()
		project.Settings.Google.IosClientId = testIOSID
		d, _ := newDoctor(t, srv, project)
		auth := googleAuthDouble(t, map[string]string{testIOSID: "redirect_mismatch"})
		d.GoogleAuthURL = auth.URL
		d.Slug = "demo"
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if c := findCheck(t, rep, "project: Google iOS client ID resolves"); c.Status != StatusPass {
			t.Fatalf("ios check = %+v", c)
		}
		if rep.Failed() {
			t.Fatalf("installed-app redirect mismatch must not fail: %+v", rep.Checks)
		}
	})

	t.Run("incomplete apple config fails", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		project := healthyProject()
		project.Settings.Apple.HasPrivateKey = false
		d, _ := newDoctor(t, srv, project)
		d.Slug = "demo"
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		c := findCheck(t, rep, "project: Apple sign-in")
		if c.Status != StatusFail || !strings.Contains(c.Detail, "private key") {
			t.Fatalf("apple check = %+v", c)
		}
	})

	t.Run("unknown project fails", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		d, _ := newDoctor(t, srv, healthyProject())
		d.Slug = "nope"
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if c := findCheck(t, rep, "project: exists"); c.Status != StatusFail {
			t.Fatalf("exists check = %+v", c)
		}
	})
}

// TestDoctorAppleKeyDryRun covers the --apple-key path — the branch that
// fulfills the "doctor detects an expired Apple key" acceptance criterion:
// read the .p8, mint a client secret and classify Apple's answer.
func TestDoctorAppleKeyDryRun(t *testing.T) {
	p8, _ := testP8(t)
	keyPath := filepath.Join(t.TempDir(), "AuthKey_SIWAKEY001.p8")
	if err := os.WriteFile(keyPath, p8, 0o600); err != nil {
		t.Fatal(err)
	}
	const dryRunName = "project: Apple key accepted (token endpoint dry-run)"

	cases := []struct {
		name       string
		oauthError string
		want       Status
		servicesID string // "" falls back to BundleIds[0] as the client
	}{
		{"revoked key fails", "invalid_client", StatusFail, "com.example.demo.signin"},
		{"healthy key passes", "invalid_grant", StatusPass, "com.example.demo.signin"},
		{"no services ID probes the bundle ID", "invalid_grant", StatusPass, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := instanceDouble(t, nil)
			project := healthyProject()
			project.Settings.Apple.ServicesId = tc.servicesID
			d, _ := newDoctor(t, srv, project)
			d.Slug = "demo"
			d.AppleKeyPath = keyPath
			clientID := tc.servicesID
			if clientID == "" {
				clientID = "com.example.demo"
			}
			// The double asserts the minted secret carries exactly the
			// project's key/team/client triple — a transposed argument in
			// the doctor's appleTokenDryRun call would fail here.
			apple := appleTokenDouble(t, tc.oauthError, appleSecretIdentity{
				clientID: clientID, teamID: "TEAM123456", keyID: "SIWAKEY001",
			})
			d.AppleTokenBase = apple.URL

			rep, err := d.Run(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if c := findCheck(t, rep, dryRunName); c.Status != tc.want {
				t.Fatalf("dry-run = %+v, want %s", c, tc.want)
			}
		})
	}

	t.Run("unreadable key file fails", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		d, _ := newDoctor(t, srv, healthyProject())
		d.Slug = "demo"
		d.AppleKeyPath = filepath.Join(t.TempDir(), "missing.p8")
		d.AppleTokenBase = "http://127.0.0.1:0" // must never be reached
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if c := findCheck(t, rep, dryRunName); c.Status != StatusFail {
			t.Fatalf("dry-run = %+v", c)
		}
	})

	t.Run("unparseable key fails", func(t *testing.T) {
		badPath := filepath.Join(t.TempDir(), "garbage.p8")
		if err := os.WriteFile(badPath, []byte("not a key"), 0o600); err != nil {
			t.Fatal(err)
		}
		srv := instanceDouble(t, nil)
		d, _ := newDoctor(t, srv, healthyProject())
		d.Slug = "demo"
		d.AppleKeyPath = badPath
		d.AppleTokenBase = "http://127.0.0.1:0" // must never be reached
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if c := findCheck(t, rep, dryRunName); c.Status != StatusFail {
			t.Fatalf("dry-run = %+v", c)
		}
	})
}

func TestDoctorSMTPTestSend(t *testing.T) {
	srv := instanceDouble(t, nil)
	d, settings := newDoctor(t, srv, healthyProject())
	d.SMTPTestTo = "ops@example.com"
	rep, err := d.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(settings.testEmailTo) != 1 || settings.testEmailTo[0] != "ops@example.com" {
		t.Fatalf("test sends = %v", settings.testEmailTo)
	}
	if c := findCheck(t, rep, "instance: outgoing email (SMTP)"); c.Status != StatusPass {
		t.Fatalf("smtp check = %+v", c)
	}

	// A failing transport is a FAIL with remediation.
	settings.sendErr = errors.New("connection refused")
	rep, err = d.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c := findCheck(t, rep, "instance: outgoing email (SMTP)")
	if c.Status != StatusFail || c.Remediation == "" {
		t.Fatalf("smtp check = %+v", c)
	}
}

func TestDoctorJSONReport(t *testing.T) {
	srv := instanceDouble(t, map[string]bool{"/p/demo/.well-known/jwks.json": true})
	d, _ := newDoctor(t, srv, healthyProject())
	d.Slug = "demo"
	rep, err := d.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	data, err := rep.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Status Status  `json:"status"`
		Checks []Check `json:"checks"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Status != StatusFail {
		t.Fatalf("json status = %s", decoded.Status)
	}
	if len(decoded.Checks) != len(rep.Checks) {
		t.Fatalf("json has %d checks, report has %d", len(decoded.Checks), len(rep.Checks))
	}
	for _, c := range decoded.Checks {
		if c.Name == "" || c.Status == "" {
			t.Fatalf("incomplete check in JSON: %+v", c)
		}
	}
}

// billingProject returns a healthy project plus store billing credentials and a
// mapped product on the doctor.
func doctorWithBilling(t *testing.T, srv *httptest.Server, apple *adminv1.AppleBillingConfig, google *adminv1.GoogleBillingConfig, products []*adminv1.Product) *Doctor {
	t.Helper()
	d, _ := newDoctor(t, srv, healthyProject())
	d.Slug = "demo"
	d.BillingCreds = &fakeBillingCreds{apple: apple, google: google}
	d.Products = &fakeProducts{products: products}
	return d
}

func TestDoctorBilling(t *testing.T) {
	mappedProduct := []*adminv1.Product{{
		Identifier: "monthly", AppleProductId: "pro_monthly", GoogleProductId: "pro_monthly",
	}}

	t.Run("healthy configured stores pass", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		apple := &adminv1.AppleBillingConfig{
			IapKeyId: "IAPKEY0001", IapIssuerId: "iss", BundleId: "com.example.demo",
			HasIapKey: true, HasNotificationSecret: true,
		}
		google := &adminv1.GoogleBillingConfig{
			PackageName: "com.example.app", HasServiceAccount: true,
			PubsubTopic: "projects/p/topics/moth-rtdn", HasRtdnSecret: true,
		}
		d := doctorWithBilling(t, srv, apple, google, mappedProduct)
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{
			"project: Apple billing credentials",
			"project: Apple notification endpoint",
			"project: Google billing credentials",
			"project: Google RTDN endpoint",
			"project: catalog store mapping",
		} {
			if c := findCheck(t, rep, name); c.Status != StatusPass {
				t.Fatalf("%s = %s (%s)", name, c.Status, c.Detail)
			}
		}
	})

	t.Run("no credentials skips", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		d := doctorWithBilling(t, srv, &adminv1.AppleBillingConfig{}, &adminv1.GoogleBillingConfig{}, nil)
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if c := findCheck(t, rep, "project: store billing credentials"); c.Status != StatusSkip {
			t.Fatalf("expected skip, got %+v", c)
		}
	})

	t.Run("missing apple notification secret warns", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		apple := &adminv1.AppleBillingConfig{
			IapKeyId: "IAPKEY0001", IapIssuerId: "iss", BundleId: "com.example.demo", HasIapKey: true,
		}
		d := doctorWithBilling(t, srv, apple, &adminv1.GoogleBillingConfig{}, mappedProduct)
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if c := findCheck(t, rep, "project: Apple notification secret"); c.Status != StatusWarn {
			t.Fatalf("expected warn, got %+v", c)
		}
	})

	t.Run("incomplete google credentials fail", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		// Package set but no service account: incomplete.
		google := &adminv1.GoogleBillingConfig{PackageName: "com.example.app"}
		d := doctorWithBilling(t, srv, &adminv1.AppleBillingConfig{}, google, mappedProduct)
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if c := findCheck(t, rep, "project: Google billing credentials"); c.Status != StatusFail {
			t.Fatalf("expected fail, got %+v", c)
		}
		if !rep.Failed() {
			t.Fatal("report should fail")
		}
	})

	t.Run("expired apple store key fails with remediation", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		apple := &adminv1.AppleBillingConfig{
			IapKeyId: "IAPKEY0001", IapIssuerId: "iss", BundleId: "com.example.demo",
			HasIapKey: true, HasNotificationSecret: true,
		}
		d := doctorWithBilling(t, srv, apple, &adminv1.GoogleBillingConfig{}, mappedProduct)
		// Supply the .p8 so the live probe runs, and point it at a store API that
		// rejects the JWT (a rotated/expired key).
		p8, _ := testP8(t)
		keyPath := filepath.Join(t.TempDir(), "iap.p8")
		if err := os.WriteFile(keyPath, p8, 0o600); err != nil {
			t.Fatal(err)
		}
		d.AppleIAPKeyPath = keyPath
		d.AppleServerAPIBase = appleServerAPIDouble(t, true).URL

		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		c := findCheck(t, rep, "project: Apple App Store Server API reachable")
		if c.Status != StatusFail || c.Remediation == "" {
			t.Fatalf("expected FAIL+remediation for a rejected Apple key, got %+v", c)
		}
	})

	t.Run("expired google store key fails with remediation", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		google := &adminv1.GoogleBillingConfig{
			PackageName: "com.example.app", HasServiceAccount: true,
			PubsubTopic: "projects/p/topics/moth-rtdn", HasRtdnSecret: true,
		}
		d := doctorWithBilling(t, srv, &adminv1.AppleBillingConfig{}, google, mappedProduct)
		saPath := filepath.Join(t.TempDir(), "sa.json")
		if err := os.WriteFile(saPath, serviceAccountBytes(t, ""), 0o600); err != nil {
			t.Fatal(err)
		}
		d.GoogleServiceAccountPath = saPath
		// Token mints fine, but the Play Developer API rejects the request (the
		// SA lost access / the key was revoked).
		reject := googleAPIRejectsDouble(t)
		d.GoogleAPIBase = reject.URL
		d.GoogleTokenURL = reject.URL + "/token"

		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		c := findCheck(t, rep, "project: Google Play Developer API reachable")
		if c.Status != StatusFail || c.Remediation == "" {
			t.Fatalf("expected FAIL+remediation for a rejected Google key, got %+v", c)
		}
	})

	t.Run("deleted google store product fails", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		google := &adminv1.GoogleBillingConfig{
			PackageName: "com.example.app", HasServiceAccount: true,
			PubsubTopic: "projects/p/topics/moth-rtdn", HasRtdnSecret: true,
		}
		d := doctorWithBilling(t, srv, &adminv1.AppleBillingConfig{}, google, mappedProduct)
		// The Play double authenticates and answers the reachability probe, but
		// the mapped SKU "pro_monthly" was never created (deleted store-side).
		gp := newGoogleCatalogDouble(t)
		saPath := filepath.Join(t.TempDir(), "sa.json")
		if err := os.WriteFile(saPath, serviceAccountBytes(t, gp.srv.URL+"/token"), 0o600); err != nil {
			t.Fatal(err)
		}
		d.GoogleServiceAccountPath = saPath
		d.GoogleAPIBase = gp.srv.URL
		d.GoogleTokenURL = gp.srv.URL + "/token"

		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		c := findCheck(t, rep, "project: Google catalog products exist")
		if c.Status != StatusFail || !strings.Contains(c.Detail, "monthly") || c.Remediation == "" {
			t.Fatalf("expected FAIL flagging the deleted product, got %+v", c)
		}
	})

	t.Run("broken notification endpoints fail with remediation", func(t *testing.T) {
		srv := instanceDouble(t, map[string]bool{
			"/billing/apple/notifications/demo": true,
			"/billing/google/rtdn/demo":         true,
		})
		apple := &adminv1.AppleBillingConfig{
			IapKeyId: "IAPKEY0001", IapIssuerId: "iss", BundleId: "com.example.demo",
			HasIapKey: true, HasNotificationSecret: true,
		}
		google := &adminv1.GoogleBillingConfig{
			PackageName: "com.example.app", HasServiceAccount: true,
			PubsubTopic: "projects/p/topics/moth-rtdn", HasRtdnSecret: true,
		}
		d := doctorWithBilling(t, srv, apple, google, mappedProduct)
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"project: Apple notification endpoint", "project: Google RTDN endpoint"} {
			c := findCheck(t, rep, name)
			if c.Status != StatusFail || c.Remediation == "" {
				t.Fatalf("%s: expected FAIL+remediation, got %+v", name, c)
			}
		}
		if !rep.Failed() {
			t.Fatal("report should fail when the notification webhooks are missing")
		}
	})

	t.Run("unmapped product warns and json carries billing checks", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		apple := &adminv1.AppleBillingConfig{
			IapKeyId: "IAPKEY0001", IapIssuerId: "iss", BundleId: "com.example.demo",
			HasIapKey: true, HasNotificationSecret: true,
		}
		unmapped := []*adminv1.Product{{Identifier: "monthly"}} // no Apple SKU
		d := doctorWithBilling(t, srv, apple, &adminv1.GoogleBillingConfig{}, unmapped)
		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		c := findCheck(t, rep, "project: catalog store mapping")
		if c.Status != StatusWarn || !strings.Contains(c.Detail, "monthly (Apple)") {
			t.Fatalf("mapping check = %+v", c)
		}
		// --json carries the billing checks byte-for-byte in the checks array.
		data, err := rep.JSON()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "project: Apple billing credentials") {
			t.Fatalf("json missing billing checks:\n%s", data)
		}
	})
}
