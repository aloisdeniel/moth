package setup

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

const (
	testWebID     = "111-webweb.apps.googleusercontent.com"
	testIOSID     = "222-iosios.apps.googleusercontent.com"
	testAndroidID = "333-android.apps.googleusercontent.com"
)

// googleAuthDouble simulates Google's authorization endpoint: it answers
// per client_id according to the responses map ("ok", "not_found",
// "redirect_mismatch"), defaulting to ok. Every probe must carry the full
// authorization-request shape — real Google only reports a redirect
// mismatch when redirect_uri is actually sent, so dropping a parameter
// would make the redirect check vacuous.
func googleAuthDouble(t *testing.T, responses map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, param := range []string{"client_id", "redirect_uri", "response_type", "scope"} {
			if r.URL.Query().Get(param) == "" {
				t.Errorf("authorization probe is missing %s: %s", param, r.URL)
			}
		}
		switch responses[r.URL.Query().Get("client_id")] {
		case "not_found":
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Error 401: invalid_client\nThe OAuth client was not found."))
		case "redirect_mismatch":
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Error 400: redirect_uri_mismatch"))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("consent screen"))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newGoogleSetup(fake *fakeProjects, auth *httptest.Server, input string, out *bytes.Buffer) *GoogleSetup {
	return &GoogleSetup{
		Projects: fake,
		Prompt:   NewPrompter(strings.NewReader(input), out),
		Out:      out,
		Runner:   &fakeRunner{missing: map[string]bool{"gcloud": true, "keytool": true}},
		HTTPC:    auth.Client(),
		AuthURL:  auth.URL,
		BaseURL:  "https://auth.example.com",
		Slug:     "demo",
	}
}

func TestGoogleSetupGuidedHappyPath(t *testing.T) {
	fake := &fakeProjects{projects: []*adminv1.Project{testProject("demo")}}
	auth := googleAuthDouble(t, nil)
	var out bytes.Buffer
	// Prompted, in order: web client ID, iOS client ID, Android client
	// ID, then the web client secret.
	input := testWebID + "\n" + testIOSID + "\n" + testAndroidID + "\nGOCSPX-test-secret\n"
	s := newGoogleSetup(fake, auth, input, &out)
	s.GCPProject = "demo-project"
	s.IOSBundleID = "com.example.demo"
	s.AndroidPackage = "com.example.demo"
	s.AndroidSHA1 = strings.Repeat("AB", 20)
	s.AndroidSHA256 = strings.Repeat("CD", 32)

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(fake.updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(fake.updates))
	}
	g := fake.updates[0].Settings.Google
	if !g.Enabled || g.WebClientId != testWebID || g.IosClientId != testIOSID ||
		g.AndroidClientId != testAndroidID || g.WebClientSecret != "GOCSPX-test-secret" {
		t.Fatalf("unexpected google config written: %+v", g)
	}
	if got := fake.updates[0].UpdateMask.Paths; len(got) != 1 || got[0] != "settings" {
		t.Fatalf("update mask = %v", got)
	}
	if rep.Failed() {
		t.Fatalf("report failed: %+v", rep.Checks)
	}
	for _, name := range []string{
		"Google: web client ID resolves",
		"Google: iOS client ID resolves",
		"Google: Android client ID resolves",
	} {
		if c := findCheck(t, rep, name); c.Status != StatusPass {
			t.Fatalf("%s = %s (%s)", name, c.Status, c.Detail)
		}
	}
	// The guided transcript must contain the exact values to enter.
	transcript := out.String()
	for _, want := range []string{
		"https://console.cloud.google.com/auth/clients/create?project=demo-project",
		"https://auth.example.com/oauth/google/callback",
		"com.example.demo",
		s.AndroidSHA1,
	} {
		if !strings.Contains(transcript, want) {
			t.Fatalf("transcript missing %q:\n%s", want, transcript)
		}
	}
}

func TestGoogleSetupRejectsBadPastedClientID(t *testing.T) {
	fake := &fakeProjects{projects: []*adminv1.Project{testProject("demo")}}
	auth := googleAuthDouble(t, nil)
	var out bytes.Buffer
	// Skip iOS and Android (blank), mispaste the web client ID once, then
	// paste a valid one and skip the secret.
	input := "\n\nnot-a-client-id\n" + testWebID + "\n\n"
	s := newGoogleSetup(fake, auth, input, &out)
	s.GCPProject = "demo-project"

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "does not look like a Google OAuth client ID") {
		t.Fatalf("transcript missing the validation rejection:\n%s", out.String())
	}
	if g := fake.updates[0].Settings.Google; g.WebClientId != testWebID {
		t.Fatalf("web client id = %q", g.WebClientId)
	}
	// No secret was provided and none is stored: honest WARN.
	if c := findCheck(t, rep, "Google: web client secret"); c.Status != StatusWarn {
		t.Fatalf("web client secret check = %s", c.Status)
	}
}

func TestGoogleSetupBadFlagValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GoogleSetup)
	}{
		{"bad gcp project", func(s *GoogleSetup) { s.GCPProject = "NOT VALID" }},
		{"bad web client id", func(s *GoogleSetup) { s.GCPProject = "demo-project"; s.WebClientID = "nope" }},
		{"bad sha1", func(s *GoogleSetup) {
			s.GCPProject = "demo-project"
			s.AndroidPackage = "com.example.demo"
			s.AndroidSHA1 = "zz"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeProjects{projects: []*adminv1.Project{testProject("demo")}}
			auth := googleAuthDouble(t, nil)
			var out bytes.Buffer
			s := newGoogleSetup(fake, auth, "", &out)
			tt.mutate(s)
			if _, err := s.Run(context.Background()); err == nil {
				t.Fatal("expected an error")
			}
			if len(fake.updates) != 0 {
				t.Fatalf("no update expected, got %d", len(fake.updates))
			}
		})
	}
}

func TestGoogleSetupIdempotentSecondRun(t *testing.T) {
	project := testProject("demo")
	project.Settings.Google = &adminv1.GoogleProviderConfig{
		Enabled:            true,
		WebClientId:        testWebID,
		IosClientId:        testIOSID,
		AndroidClientId:    testAndroidID,
		HasWebClientSecret: true,
	}
	fake := &fakeProjects{projects: []*adminv1.Project{project}}
	auth := googleAuthDouble(t, nil)
	var out bytes.Buffer
	s := newGoogleSetup(fake, auth, "", &out) // no prompt input needed
	s.GCPProject = "demo-project"

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.updates) != 0 {
		t.Fatalf("second run must not update, got %d updates", len(fake.updates))
	}
	c := findCheck(t, rep, "moth: Google provider config")
	if c.Status != StatusPass || !strings.Contains(c.Detail, "no changes") {
		t.Fatalf("idempotency check = %+v", c)
	}
}

func TestGoogleSetupDetectsDeletedClient(t *testing.T) {
	project := testProject("demo")
	project.Settings.Google = &adminv1.GoogleProviderConfig{
		Enabled: true, WebClientId: testWebID, HasWebClientSecret: true,
	}
	fake := &fakeProjects{projects: []*adminv1.Project{project}}
	auth := googleAuthDouble(t, map[string]string{testWebID: "not_found"})
	var out bytes.Buffer
	s := newGoogleSetup(fake, auth, "\n\n", &out) // skip iOS and Android
	s.GCPProject = "demo-project"

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Failed() {
		t.Fatalf("expected failure for a deleted client: %+v", rep.Checks)
	}
	c := findCheck(t, rep, "Google: web client ID resolves")
	if c.Status != StatusFail {
		t.Fatalf("check = %+v", c)
	}
}

func TestGoogleSetupDetectsWebRedirectMismatch(t *testing.T) {
	project := testProject("demo")
	project.Settings.Google = &adminv1.GoogleProviderConfig{
		Enabled: true, WebClientId: testWebID, HasWebClientSecret: true,
	}
	fake := &fakeProjects{projects: []*adminv1.Project{project}}
	auth := googleAuthDouble(t, map[string]string{testWebID: "redirect_mismatch"})
	var out bytes.Buffer
	s := newGoogleSetup(fake, auth, "\n\n", &out) // skip iOS and Android
	s.GCPProject = "demo-project"

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Failed() {
		t.Fatalf("an unregistered web redirect URI must fail: %+v", rep.Checks)
	}
	c := findCheck(t, rep, "Google: web redirect URI registered")
	if c.Status != StatusFail || !strings.Contains(c.Detail, "https://auth.example.com/oauth/google/callback") {
		t.Fatalf("redirect check = %+v", c)
	}
}

func TestGoogleSetupRedirectMismatchIsMootForInstalledApps(t *testing.T) {
	// Installed-app (iOS/Android) clients register no redirect URI; a
	// redirect_uri_mismatch answer still proves the client exists.
	project := testProject("demo")
	project.Settings.Google = &adminv1.GoogleProviderConfig{
		Enabled: true, IosClientId: testIOSID, AndroidClientId: testAndroidID,
	}
	fake := &fakeProjects{projects: []*adminv1.Project{project}}
	auth := googleAuthDouble(t, map[string]string{
		testIOSID:     "redirect_mismatch",
		testAndroidID: "redirect_mismatch",
	})
	var out bytes.Buffer
	s := newGoogleSetup(fake, auth, "\n", &out) // skip the web client
	s.GCPProject = "demo-project"

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed() {
		t.Fatalf("installed-app redirect mismatch must not fail: %+v", rep.Checks)
	}
	for _, name := range []string{
		"Google: iOS client ID resolves",
		"Google: Android client ID resolves",
	} {
		if c := findCheck(t, rep, name); c.Status != StatusPass {
			t.Fatalf("%s = %+v", name, c)
		}
	}
}

func TestGoogleSetupPromptsForKeystorePassword(t *testing.T) {
	fake := &fakeProjects{projects: []*adminv1.Project{testProject("demo")}}
	auth := googleAuthDouble(t, nil)
	runner := &fakeRunner{
		missing: map[string]bool{"gcloud": true},
		output:  map[string][]byte{"keytool": []byte(keytoolOutput)},
	}
	var out bytes.Buffer
	// Prompted, in order: iOS bundle ID (blank skips iOS), keystore
	// password, Android client ID.
	input := "\nhunter2\n" + testAndroidID + "\n"
	s := newGoogleSetup(fake, auth, input, &out)
	s.Runner = runner
	s.GCPProject = "demo-project"
	s.AndroidPackage = "com.example.demo"
	s.Keystore = "/tmp/release.keystore"
	s.WebClientID = testWebID
	s.WebClientSecret = "GOCSPX-test-secret"

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed() {
		t.Fatalf("report failed: %+v", rep.Checks)
	}
	if !strings.Contains(out.String(), "Keystore password") {
		t.Fatalf("transcript missing the keystore password prompt:\n%s", out.String())
	}
	// The prompted password reaches keytool via the environment only.
	if len(runner.calls) != 1 || !strings.Contains(runner.calls[0], "-storepass:env") {
		t.Fatalf("keytool call = %v", runner.calls)
	}
	if strings.Contains(runner.calls[0], "hunter2") {
		t.Fatalf("password leaked into the argv: %v", runner.calls)
	}
	if len(runner.envs) != 1 || len(runner.envs[0]) != 1 || runner.envs[0][0] != keystorePassEnvVar+"=hunter2" {
		t.Fatalf("keytool env = %v", runner.envs)
	}
	if s.AndroidSHA1 == "" || s.AndroidSHA256 == "" {
		t.Fatalf("fingerprints not computed: %q %q", s.AndroidSHA1, s.AndroidSHA256)
	}
}

func TestReversedClientScheme(t *testing.T) {
	got := reversedClientScheme("123-abc.apps.googleusercontent.com")
	if want := "com.googleusercontent.apps.123-abc"; got != want {
		t.Fatalf("reversedClientScheme = %q, want %q", got, want)
	}
}

func TestGoogleSetupUsesGcloudWhenPresent(t *testing.T) {
	project := testProject("demo")
	project.Settings.Google = &adminv1.GoogleProviderConfig{
		Enabled: true, WebClientId: testWebID, HasWebClientSecret: true,
	}
	fake := &fakeProjects{projects: []*adminv1.Project{project}}
	auth := googleAuthDouble(t, nil)
	runner := &fakeRunner{output: map[string][]byte{"gcloud": []byte("demo-project\n")}}
	var out bytes.Buffer
	s := newGoogleSetup(fake, auth, "\n\n", &out) // skip iOS and Android
	s.Runner = runner
	s.GCPProject = "demo-project"

	rep, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if c := findCheck(t, rep, "gcloud: GCP project exists"); c.Status != StatusPass {
		t.Fatalf("gcloud check = %+v", c)
	}
	if len(runner.calls) != 1 || !strings.Contains(runner.calls[0], "projects describe demo-project") {
		t.Fatalf("gcloud calls = %v", runner.calls)
	}
}
