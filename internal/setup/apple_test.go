package setup

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

// ascDouble simulates the App Store Connect API surface the Apple flow
// touches. keyP8 is the .p8 the fake key-creation endpoint hands out;
// nil makes that endpoint 404 (guided fallback).
type ascDouble struct {
	bundleExists  bool
	capabilityOn  bool
	keyP8         []byte
	createdBundle bool
	createdCap    bool
	createdKey    bool
}

func (d *ascDouble) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /v1/bundleIds":
			if d.bundleExists {
				w.Write([]byte(`{"data":[{"type":"bundleIds","id":"BID1","attributes":{"identifier":"com.example.demo","name":"Demo"}}]}`))
				return
			}
			w.Write([]byte(`{"data":[]}`))
		case "POST /v1/bundleIds":
			d.createdBundle, d.bundleExists = true, true
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"data":{"type":"bundleIds","id":"BID1","attributes":{"identifier":"com.example.demo","name":"Demo"}}}`))
		case "GET /v1/bundleIds/BID1/bundleIdCapabilities":
			if d.capabilityOn {
				w.Write([]byte(`{"data":[{"type":"bundleIdCapabilities","id":"c1","attributes":{"capabilityType":"APPLE_ID_AUTH"}}]}`))
				return
			}
			w.Write([]byte(`{"data":[]}`))
		case "POST /v1/bundleIdCapabilities":
			d.createdCap, d.capabilityOn = true, true
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"data":{"type":"bundleIdCapabilities","id":"c1"}}`))
		case "POST /v1/keys":
			if d.keyP8 == nil {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"errors":[{"title":"NOT_FOUND","detail":"The URL can not be found"}]}`))
				return
			}
			d.createdKey = true
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"data":{"type":"keys","id":"SIWAKEY001","attributes":{"keyId":"SIWAKEY001","privateKey":"` +
				base64.StdEncoding.EncodeToString(d.keyP8) + `"}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"errors":[{"title":"NOT_FOUND"}]}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// appleSecretIdentity is what the dry-run's minted client secret must
// assert about itself: sub = the client (Services ID or bundle ID),
// iss = the team, kid = the Sign in with Apple key.
type appleSecretIdentity struct {
	clientID, teamID, keyID string
}

// appleTokenDouble simulates Apple's token endpoint answering the dry-run:
// oauthError is the OAuth error code to return ("invalid_grant" for a
// healthy config). The double inspects the posted client_secret JWT and
// fails the test when its identity does not match `want` — the three
// identifiers are adjacent same-typed strings all the way down the call
// chain, so a transposition would otherwise go unnoticed.
func appleTokenDouble(t *testing.T, oauthError string, want appleSecretIdentity) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/token" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token request form: %v", err)
		} else {
			if got := r.PostForm.Get("client_id"); got != want.clientID {
				t.Errorf("token request client_id = %q, want %q", got, want.clientID)
			}
			checkAppleClientSecret(t, r.PostForm.Get("client_secret"), want)
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"` + oauthError + `"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// checkAppleClientSecret decodes (without verifying — the double has no
// need to) the minted ES256 client secret and asserts its header/claims
// carry the expected key, team and client identifiers.
func checkAppleClientSecret(t *testing.T, secret string, want appleSecretIdentity) {
	t.Helper()
	parts := strings.Split(secret, ".")
	if len(parts) != 3 {
		t.Errorf("client_secret is not a compact JWS: %q", secret)
		return
	}
	decode := func(part string, into any) {
		raw, err := base64.RawURLEncoding.DecodeString(part)
		if err != nil {
			t.Errorf("decode client_secret segment: %v", err)
			return
		}
		if err := json.Unmarshal(raw, into); err != nil {
			t.Errorf("parse client_secret segment: %v", err)
		}
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	var claims struct {
		Iss string `json:"iss"`
		Sub string `json:"sub"`
	}
	decode(parts[0], &header)
	decode(parts[1], &claims)
	if header.Alg != "ES256" || header.Kid != want.keyID {
		t.Errorf("client_secret header alg=%q kid=%q, want ES256/%q", header.Alg, header.Kid, want.keyID)
	}
	if claims.Iss != want.teamID {
		t.Errorf("client_secret iss = %q, want team %q", claims.Iss, want.teamID)
	}
	if claims.Sub != want.clientID {
		t.Errorf("client_secret sub = %q, want client %q", claims.Sub, want.clientID)
	}
}

func newAppleSetup(t *testing.T, fake *fakeProjects, asc *httptest.Server, apple *httptest.Server, input string, out *bytes.Buffer) *AppleSetup {
	t.Helper()
	return &AppleSetup{
		Projects:       fake,
		Prompt:         NewPrompter(strings.NewReader(input), out),
		Out:            out,
		ASC:            testASC(t, asc),
		HTTPC:          apple.Client(),
		AppleTokenBase: apple.URL,
		BaseURL:        "https://auth.example.com",
		Slug:           "demo",
		BundleID:       "com.example.demo",
		TeamID:         "TEAM123456",
	}
}

func TestAppleSetupHappyPath(t *testing.T) {
	p8, _ := testP8(t)
	double := &ascDouble{keyP8: p8}
	asc := double.server(t)
	apple := appleTokenDouble(t, "invalid_grant", appleSecretIdentity{clientID: "com.example.demo.signin", teamID: "TEAM123456", keyID: "SIWAKEY001"})
	fake := &fakeProjects{projects: []*adminv1.Project{testProject("demo")}}
	var out bytes.Buffer
	// The only guided step left is the Services ID paste.
	s := newAppleSetup(t, fake, asc, apple, "com.example.demo.signin\n", &out)

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !double.createdBundle || !double.createdCap || !double.createdKey {
		t.Fatalf("ASC calls: bundle=%v cap=%v key=%v", double.createdBundle, double.createdCap, double.createdKey)
	}
	if len(fake.updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(fake.updates))
	}
	a := fake.updates[0].Settings.Apple
	if !a.Enabled || a.TeamId != "TEAM123456" || a.KeyId != "SIWAKEY001" ||
		a.ServicesId != "com.example.demo.signin" || a.PrivateKeyP8 != string(p8) {
		t.Fatalf("unexpected apple config written: %+v", a)
	}
	if len(a.BundleIds) != 1 || a.BundleIds[0] != "com.example.demo" {
		t.Fatalf("bundle ids = %v", a.BundleIds)
	}
	if c := findCheck(t, rep, "Apple: client secret accepted (token endpoint dry-run)"); c.Status != StatusPass {
		t.Fatalf("dry-run check = %+v", c)
	}
	if rep.Failed() {
		t.Fatalf("report failed: %+v", rep.Checks)
	}
	// The guided transcript must contain the exact return URL to paste.
	if !strings.Contains(out.String(), "https://auth.example.com/oauth/apple/callback") {
		t.Fatalf("transcript missing the return URL:\n%s", out.String())
	}
}

func TestAppleSetupGuidedKeyFallback(t *testing.T) {
	// Key creation 404s (endpoint unavailable): the flow degrades to the
	// guided portal steps and reads the downloaded .p8 from disk.
	p8, _ := testP8(t)
	path := filepath.Join(t.TempDir(), "AuthKey_ABCDEF1234.p8")
	if err := os.WriteFile(path, p8, 0o600); err != nil {
		t.Fatal(err)
	}
	double := &ascDouble{} // keyP8 nil → POST /v1/keys 404s
	asc := double.server(t)
	apple := appleTokenDouble(t, "invalid_grant", appleSecretIdentity{clientID: "com.example.demo.signin", teamID: "TEAM123456", keyID: "ABCDEF1234"})
	fake := &fakeProjects{projects: []*adminv1.Project{testProject("demo")}}
	var out bytes.Buffer
	input := "ABCDEF1234\n" + path + "\ncom.example.demo.signin\n"
	s := newAppleSetup(t, fake, asc, apple, input, &out)

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a := fake.updates[0].Settings.Apple
	if a.KeyId != "ABCDEF1234" || a.PrivateKeyP8 != string(p8) {
		t.Fatalf("apple config = %+v", a)
	}
	if !strings.Contains(out.String(), "developer.apple.com/account/resources/authkeys/add") {
		t.Fatalf("transcript missing the portal URL:\n%s", out.String())
	}
	if rep.Failed() {
		t.Fatalf("report failed: %+v", rep.Checks)
	}
}

func TestAppleSetupIdempotentSecondRun(t *testing.T) {
	project := testProject("demo")
	project.Settings.Apple = &adminv1.AppleProviderConfig{
		Enabled:       true,
		ServicesId:    "com.example.demo.signin",
		TeamId:        "TEAM123456",
		KeyId:         "SIWAKEY001",
		HasPrivateKey: true,
		BundleIds:     []string{"com.example.demo"},
	}
	double := &ascDouble{bundleExists: true, capabilityOn: true, keyP8: nil}
	asc := double.server(t)
	apple := appleTokenDouble(t, "invalid_grant", appleSecretIdentity{}) // never posted to: no key material this run
	fake := &fakeProjects{projects: []*adminv1.Project{project}}
	var out bytes.Buffer
	s := newAppleSetup(t, fake, asc, apple, "", &out) // no prompts needed

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if double.createdBundle || double.createdCap || double.createdKey {
		t.Fatal("second run must not create anything console-side")
	}
	if len(fake.updates) != 0 {
		t.Fatalf("second run must not update, got %d updates", len(fake.updates))
	}
	c := findCheck(t, rep, "moth: Apple provider config")
	if c.Status != StatusPass || !strings.Contains(c.Detail, "no changes") {
		t.Fatalf("idempotency check = %+v", c)
	}
	// Without local key material the dry-run honestly warns instead of
	// pretending.
	if c := findCheck(t, rep, "Apple: client secret accepted (token endpoint dry-run)"); c.Status != StatusWarn {
		t.Fatalf("dry-run check = %+v", c)
	}
}

func TestAppleSetupDryRunDetectsRejectedKey(t *testing.T) {
	p8, _ := testP8(t)
	double := &ascDouble{keyP8: p8}
	asc := double.server(t)
	apple := appleTokenDouble(t, "invalid_client", appleSecretIdentity{clientID: "com.example.demo.signin", teamID: "TEAM123456", keyID: "SIWAKEY001"})
	fake := &fakeProjects{projects: []*adminv1.Project{testProject("demo")}}
	var out bytes.Buffer
	s := newAppleSetup(t, fake, asc, apple, "com.example.demo.signin\n", &out)

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c := findCheck(t, rep, "Apple: client secret accepted (token endpoint dry-run)")
	if c.Status != StatusFail {
		t.Fatalf("dry-run check = %+v", c)
	}
	if !rep.Failed() {
		t.Fatal("report should fail on invalid_client")
	}
}

func TestAppleSetupUnofficialAPIStub(t *testing.T) {
	s := &AppleSetup{UseUnofficialAPI: true}
	_, err := s.Run(context.Background())
	if !errors.Is(err, ErrUnofficialAPINotImplemented) {
		t.Fatalf("err = %v", err)
	}
}

func TestAppleValidators(t *testing.T) {
	tests := []struct {
		name     string
		validate func(string) (string, error)
		in       string
		ok       bool
	}{
		{"team ok", ValidateAppleTeamID, "ab12cd34ef", true}, // upcased
		{"team bad", ValidateAppleTeamID, "short", false},
		{"key ok", ValidateAppleKeyID, "ABCDEF1234", true},
		{"key bad", ValidateAppleKeyID, "with space!", false},
		{"issuer ok", ValidateASCIssuerID, "57246542-96FE-1A63-E053-0824D011072A", true},
		{"issuer bad", ValidateASCIssuerID, "not-a-uuid", false},
		{"services ok", validateServicesID, "com.example.app.signin", true},
		{"services bad", validateServicesID, "nodots", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.validate(tt.in)
			if tt.ok != (err == nil) {
				t.Fatalf("validate(%q) error = %v, want ok=%v", tt.in, err, tt.ok)
			}
		})
	}
}
