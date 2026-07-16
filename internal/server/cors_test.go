package server

import (
	"net/http"
	"testing"
)

// preflight sends an OPTIONS preflight for a POST from origin and returns
// the response headers.
func preflight(t *testing.T, e *testEnv, path, origin, requestHeaders string) http.Header {
	t.Helper()
	req, err := http.NewRequest(http.MethodOptions, e.url+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", requestHeaders)
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.Header
}

// TestCORSPublicSurface: the end-user API must be callable from any origin
// (a Flutter Web app is essentially never served from the moth origin) and
// the preflight must admit the metadata headers the SDK attaches to every
// call.
func TestCORSPublicSurface(t *testing.T) {
	e := newTestEnv(t, "")

	h := preflight(t, e, "/moth.auth.v1.AuthService/SignIn", "http://app.example.com",
		"content-type,x-grpc-web,x-moth-key,x-moth-platform,x-moth-sdk-version,authorization")
	if got := h.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("auth surface allow-origin: %q, want *", got)
	}
	if got := h.Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("auth surface preflight rejected the SDK metadata headers")
	}

	// ConfigService (the login screen's first call) and the pub repository
	// are public too.
	for _, path := range []string{
		"/moth.auth.v1.ConfigService/GetProjectConfig",
		"/pub/api/packages/moth_auth",
	} {
		h := preflight(t, e, path, "http://localhost:5555", "content-type")
		if got := h.Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("%s allow-origin: %q, want *", path, got)
		}
	}
}

// TestCORSAdminSurface: the cookie-authed admin API stays locked to the
// instance's own origin.
func TestCORSAdminSurface(t *testing.T) {
	e := newTestEnv(t, "")

	// Foreign origin: no CORS grant.
	h := preflight(t, e, "/moth.admin.v1.SessionService/Login", "http://evil.example.com", "content-type")
	if got := h.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("admin surface allowed foreign origin: %q", got)
	}

	// The instance's own origin (testEnv BaseURL) is allowed, with
	// credentials for the session cookie.
	h = preflight(t, e, "/moth.admin.v1.SessionService/Login", "http://localhost:8080", "content-type")
	if got := h.Get("Access-Control-Allow-Origin"); got != "http://localhost:8080" {
		t.Fatalf("admin surface allow-origin: %q", got)
	}
	if got := h.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("admin surface allow-credentials: %q", got)
	}
}
