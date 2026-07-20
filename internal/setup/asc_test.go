package setup

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testASC(t *testing.T, srv *httptest.Server) *ASC {
	t.Helper()
	_, key := testP8(t)
	asc := &ASC{
		IssuerID: "57246542-96fe-1a63-e053-0824d011072a",
		KeyID:    "ASCKEY0001",
		Key:      key,
		Now:      func() time.Time { return time.Unix(1_752_000_000, 0) },
	}
	if srv != nil {
		asc.BaseURL = srv.URL
		asc.HTTPC = srv.Client()
	}
	return asc
}

func TestASCTokenShape(t *testing.T) {
	asc := testASC(t, nil)
	tok, err := asc.Token()
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT segments, got %d", len(parts))
	}
	headJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatal(err)
	}
	var head struct{ Alg, Kid, Typ string }
	if err := json.Unmarshal(headJSON, &head); err != nil {
		t.Fatal(err)
	}
	if head.Alg != "ES256" || head.Kid != "ASCKEY0001" || head.Typ != "JWT" {
		t.Fatalf("header = %+v", head)
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims struct {
		Iss string `json:"iss"`
		Aud string `json:"aud"`
		Iat int64  `json:"iat"`
		Exp int64  `json:"exp"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		t.Fatal(err)
	}
	if claims.Iss != asc.IssuerID || claims.Aud != "appstoreconnect-v1" {
		t.Fatalf("claims = %+v", claims)
	}
	if claims.Exp-claims.Iat != int64(ascTokenLifetime/time.Second) {
		t.Fatalf("lifetime = %ds", claims.Exp-claims.Iat)
	}
	if len(parts[2]) == 0 {
		t.Fatal("empty signature")
	}
}

func TestASCBundleIDCalls(t *testing.T) {
	var sawAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		sawAuth = strings.HasPrefix(auth, "Bearer ") && strings.Count(auth, ".") == 2
		switch r.Method + " " + r.URL.Path {
		case "GET /v1/bundleIds":
			// The filter matches loosely: return a near-miss too.
			w.Write([]byte(`{"data":[
				{"type":"bundleIds","id":"OTHER","attributes":{"identifier":"com.example.demo.other","name":"Other"}},
				{"type":"bundleIds","id":"BID1","attributes":{"identifier":"com.example.demo","name":"Demo"}}]}`))
		case "GET /v1/bundleIds/BID1/bundleIdCapabilities":
			w.Write([]byte(`{"data":[{"type":"bundleIdCapabilities","id":"c1","attributes":{"capabilityType":"APPLE_ID_AUTH"}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"errors":[{"title":"NOT_FOUND","detail":"missing"}]}`))
		}
	}))
	defer srv.Close()
	asc := testASC(t, srv)

	bundle, err := asc.FindBundleID(context.Background(), "com.example.demo")
	if err != nil {
		t.Fatal(err)
	}
	if bundle == nil || bundle.ResourceID != "BID1" {
		t.Fatalf("bundle = %+v", bundle)
	}
	if !sawAuth {
		t.Fatal("request was not JWT-authenticated")
	}
	has, err := asc.HasSignInWithApple(context.Background(), "BID1")
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("expected APPLE_ID_AUTH capability")
	}

	// 404s carry the ASC error payload and are recognizable.
	err = asc.EnableSignInWithApple(context.Background(), "MISSING")
	if !isASCNotFound(err) {
		t.Fatalf("expected an ASC 404, got %v", err)
	}
	if !strings.Contains(err.Error(), "NOT_FOUND") {
		t.Fatalf("error should carry the ASC title: %v", err)
	}
}
