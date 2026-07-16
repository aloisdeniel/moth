package billing

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVerifyTransactionBundleMismatch(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	tok := ca.signJWS(t, sampleTxn())

	v := NewAppleVerifier(ca.rootPool(), "com.other.app", func() time.Time { return testNow })
	if _, err := v.VerifyTransaction(tok); !errors.Is(err, ErrBundleMismatch) {
		t.Fatalf("error = %v, want ErrBundleMismatch", err)
	}

	// Matching bundle verifies.
	v = NewAppleVerifier(ca.rootPool(), "com.example.app", func() time.Time { return testNow })
	txn, err := v.VerifyTransaction(tok)
	if err != nil {
		t.Fatalf("VerifyTransaction() error = %v", err)
	}
	if txn.ProductID != "pro.monthly" || txn.OriginalTransactionID != "orig-1" {
		t.Fatalf("decoded txn = %+v", txn)
	}
}

func TestVerifyNotificationNormalization(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	v := NewAppleVerifier(ca.rootPool(), "com.example.app", func() time.Time { return testNow })

	cases := []struct {
		name       string
		nType      string
		subtype    string
		autoRenew  int
		wantStatus string
		wantRenew  bool
	}{
		{"renewal", "DID_RENEW", "", 1, StatusActive, true},
		{"grace", "DID_FAIL_TO_RENEW", "GRACE_PERIOD", 1, StatusInGracePeriod, true},
		{"billingRetry", "DID_FAIL_TO_RENEW", "", 0, StatusInBillingRetry, false},
		{"refund", "REFUND", "", 0, StatusRevoked, false},
		{"revoke", "REVOKE", "", 0, StatusRevoked, false},
		{"expired", "EXPIRED", "VOLUNTARY", 0, StatusExpired, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			txnJWS := ca.signJWS(t, sampleTxn())
			renewal := map[string]any{
				"originalTransactionId": "orig-1",
				"productId":             "pro.monthly",
				"autoRenewStatus":       tc.autoRenew,
				"environment":           "Production",
			}
			renewJWS := ca.signJWS(t, renewal)
			payload := map[string]any{
				"notificationType": tc.nType,
				"subtype":          tc.subtype,
				"notificationUUID": "uuid-" + tc.name,
				"data": map[string]any{
					"bundleId":              "com.example.app",
					"environment":           "Production",
					"signedTransactionInfo": txnJWS,
					"signedRenewalInfo":     renewJWS,
				},
			}
			signed := ca.signJWS(t, payload)

			n, err := v.VerifyNotification(signed)
			if err != nil {
				t.Fatalf("VerifyNotification() error = %v", err)
			}
			if n.Store != StoreApple || n.Type != tc.nType || n.NotificationID != "uuid-"+tc.name {
				t.Fatalf("notification meta = %+v", n)
			}
			if n.Subscription.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", n.Subscription.Status, tc.wantStatus)
			}
			if n.Subscription.AutoRenew != tc.wantRenew {
				t.Fatalf("autoRenew = %v, want %v", n.Subscription.AutoRenew, tc.wantRenew)
			}
			if n.Subscription.ProductID != "pro.monthly" || n.Subscription.Environment != EnvProduction {
				t.Fatalf("subscription = %+v", n.Subscription)
			}
		})
	}
}

func TestNormalizeGracePeriodUsesGraceExpiry(t *testing.T) {
	// A subscription in a billing grace period: the last transaction's
	// ExpiresDate is in the past, but renewal info carries a future
	// gracePeriodExpiresDate. The normalized period end must be the grace expiry
	// (when access actually lapses), not the already-passed paid-period end.
	pastExpiry := testNow.Add(-24 * time.Hour)
	graceExpiry := testNow.Add(3 * 24 * time.Hour)
	txn := &JWSTransaction{
		OriginalTransactionID: "orig-1", ProductID: "pro.monthly",
		ExpiresDate: pastExpiry.UnixMilli(), Environment: "Production",
	}
	ri := &JWSRenewalInfo{
		AutoRenewStatus: 1, Environment: "Production",
		GracePeriodExpiresDate: graceExpiry.UnixMilli(),
	}
	sub := normalizeAppleSubscription(txn, ri, StatusInGracePeriod)
	if !sub.CurrentPeriodEnd.Equal(graceExpiry.UTC()) {
		t.Fatalf("grace period end = %v, want grace expiry %v", sub.CurrentPeriodEnd, graceExpiry.UTC())
	}
	if !sub.CurrentPeriodEnd.After(testNow) {
		t.Fatalf("grace-period entitlement must expire in the future, got %v", sub.CurrentPeriodEnd)
	}
	// Without a grace expiry the plain paid-period end is kept.
	ri.GracePeriodExpiresDate = 0
	sub = normalizeAppleSubscription(txn, ri, StatusInGracePeriod)
	if !sub.CurrentPeriodEnd.Equal(pastExpiry.UTC()) {
		t.Fatalf("without grace expiry, period end = %v, want paid-period end %v", sub.CurrentPeriodEnd, pastExpiry.UTC())
	}
}

func TestVerifyNotificationRejectsForeignBundle(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	v := NewAppleVerifier(ca.rootPool(), "com.example.app", func() time.Time { return testNow })
	payload := map[string]any{
		"notificationType": "DID_RENEW",
		"notificationUUID": "u1",
		"data":             map[string]any{"bundleId": "com.attacker.app", "environment": "Production"},
	}
	if _, err := v.VerifyNotification(ca.signJWS(t, payload)); !errors.Is(err, ErrBundleMismatch) {
		t.Fatalf("error = %v, want ErrBundleMismatch", err)
	}
}

// appleAPIDouble is an httptest App Store Server API returning canned status
// bodies whose signed blobs are minted by the test CA. It records the last
// Authorization header for JWT-shape assertions.
type appleAPIDouble struct {
	srv       *httptest.Server
	ca        *testCA
	lastAuth  string
	statusInt int
	notFound  bool
}

func newAppleAPIDouble(t *testing.T, ca *testCA, statusInt int) *appleAPIDouble {
	d := &appleAPIDouble{ca: ca, statusInt: statusInt}
	d.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.lastAuth = r.Header.Get("Authorization")
		if d.notFound {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"errorCode": 4040010, "errorMessage": "not found"})
			return
		}
		txnJWS := ca.signJWS(t, sampleTxn())
		renewJWS := ca.signJWS(t, map[string]any{"autoRenewStatus": 1, "environment": "Production"})
		resp := map[string]any{
			"environment": "Production",
			"bundleId":    "com.example.app",
			"data": []map[string]any{{
				"subscriptionGroupIdentifier": "grp-1",
				"lastTransactions": []map[string]any{{
					"originalTransactionId": "orig-1",
					"status":                d.statusInt,
					"signedTransactionInfo": txnJWS,
					"signedRenewalInfo":     renewJWS,
				}},
			}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(d.srv.Close)
	return d
}

func newTestAppleClient(ca *testCA, baseURL string) *AppleClient {
	return &AppleClient{
		BaseURL:    baseURL,
		SandboxURL: "",
		IssuerID:   "issuer-uuid",
		KeyID:      "KEYID123",
		BundleID:   "com.example.app",
		Key:        ca.leafKey, // any ES256 key; the double does not verify it
		HTTPC:      http.DefaultClient,
		Now:        func() time.Time { return testNow },
		Verifier:   NewAppleVerifier(ca.rootPool(), "com.example.app", func() time.Time { return testNow }),
	}
}

func TestGetAllSubscriptionStatusesMapping(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	cases := []struct {
		code int
		want string
	}{
		{1, StatusActive},
		{2, StatusExpired},
		{3, StatusInBillingRetry},
		{4, StatusInGracePeriod},
		{5, StatusRevoked},
	}
	for _, tc := range cases {
		d := newAppleAPIDouble(t, ca, tc.code)
		c := newTestAppleClient(ca, d.srv.URL)
		sub, err := c.GetAllSubscriptionStatuses(t.Context(), "orig-1")
		if err != nil {
			t.Fatalf("code %d: error = %v", tc.code, err)
		}
		if sub.Status != tc.want {
			t.Fatalf("code %d: status = %q, want %q", tc.code, sub.Status, tc.want)
		}
		if sub.StoreTransactionID != "orig-1" || sub.ProductID != "pro.monthly" {
			t.Fatalf("code %d: sub = %+v", tc.code, sub)
		}
		// Assert the request carried a Bearer ES256 JWT with the ASC shape.
		assertAppleJWTHeader(t, d.lastAuth)
	}
}

func assertAppleJWTHeader(t *testing.T, auth string) {
	t.Helper()
	tok, ok := strings.CutPrefix(auth, "Bearer ")
	if !ok {
		t.Fatalf("Authorization = %q, want Bearer token", auth)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("token is not a compact JWS: %q", tok)
	}
	head, _ := base64.RawURLEncoding.DecodeString(parts[0])
	var h map[string]string
	if err := json.Unmarshal(head, &h); err != nil {
		t.Fatalf("decode header: %v", err)
	}
	if h["alg"] != "ES256" || h["typ"] != "JWT" || h["kid"] != "KEYID123" {
		t.Fatalf("header = %v", h)
	}
	body, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]any
	if err := json.Unmarshal(body, &claims); err != nil {
		t.Fatalf("decode claims: %v", err)
	}
	if claims["aud"] != appleAPIAudience || claims["iss"] != "issuer-uuid" || claims["bid"] != "com.example.app" {
		t.Fatalf("claims = %v", claims)
	}
}

func TestGetAllSubscriptionStatusesSandboxFallback(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	// Production double 404s; sandbox double resolves.
	prod := newAppleAPIDouble(t, ca, 1)
	prod.notFound = true
	sandbox := newAppleAPIDouble(t, ca, 1)

	c := newTestAppleClient(ca, prod.srv.URL)
	c.SandboxURL = sandbox.srv.URL

	sub, err := c.GetAllSubscriptionStatuses(t.Context(), "orig-1")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if sub.Status != StatusActive {
		t.Fatalf("status = %q, want active (via sandbox fallback)", sub.Status)
	}
	if sandbox.lastAuth == "" {
		t.Fatal("sandbox host was not queried on production 404")
	}
}

func TestGetAllSubscriptionStatusesNotFound(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	d := newAppleAPIDouble(t, ca, 1)
	d.notFound = true
	c := newTestAppleClient(ca, d.srv.URL) // no sandbox fallback
	if _, err := c.GetAllSubscriptionStatuses(t.Context(), "orig-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestAppleRootsPoolShape(t *testing.T) {
	if _, ok := any(AppleRoots()).(*x509.CertPool); !ok {
		t.Fatal("AppleRoots did not return a *x509.CertPool")
	}
}
