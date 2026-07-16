package billing

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// googleDouble is an httptest stand-in for both the OAuth token endpoint and
// the Play Developer API. It verifies the RS256 service-account assertion with
// pubKey and records call counts.
type googleDouble struct {
	srv        *httptest.Server
	pubKey     *rsa.PublicKey
	state      string
	ackState   string
	lastAuth   string
	lastPath   string
	lastMethod string

	mu         sync.Mutex
	tokenCalls int
	acked      bool
}

func newGoogleDouble(t *testing.T, pubKey *rsa.PublicKey, state string) *googleDouble {
	d := &googleDouble{pubKey: pubKey, state: state, ackState: "acknowledgementStateAcknowledged"}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("grant_type") != jwtBearerGrant {
			http.Error(w, "bad grant", http.StatusBadRequest)
			return
		}
		if err := d.verifyAssertion(r.Form.Get("assertion")); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		d.mu.Lock()
		d.tokenCalls++
		d.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "ya29.test", "expires_in": 3600, "token_type": "Bearer"})
	})
	mux.HandleFunc("/androidpublisher/", func(w http.ResponseWriter, r *http.Request) {
		d.lastAuth = r.Header.Get("Authorization")
		d.lastPath = r.URL.Path
		d.lastMethod = r.Method
		if strings.HasSuffix(r.URL.Path, ":acknowledge") {
			d.mu.Lock()
			d.acked = true
			d.mu.Unlock()
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if d.state == "__404__" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "purchaseTokenNotFound"}})
			return
		}
		resp := map[string]any{
			"subscriptionState":    d.state,
			"acknowledgementState": d.ackState,
			"lineItems": []map[string]any{{
				"productId":        "pro_monthly",
				"expiryTime":       testNow.Add(15 * 24 * time.Hour).Format(time.RFC3339),
				"autoRenewingPlan": map[string]any{"autoRenewEnabled": true},
			}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	d.srv = httptest.NewServer(mux)
	t.Cleanup(d.srv.Close)
	return d
}

func (d *googleDouble) verifyAssertion(assertion string) error {
	parts := strings.Split(assertion, ".")
	if len(parts) != 3 {
		return errors.New("assertion not a JWT")
	}
	head, _ := base64.RawURLEncoding.DecodeString(parts[0])
	var h map[string]string
	if err := json.Unmarshal(head, &h); err != nil {
		return err
	}
	if h["alg"] != "RS256" || h["kid"] != "kid-123" {
		return errors.New("bad assertion header")
	}
	body, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]any
	if err := json.Unmarshal(body, &claims); err != nil {
		return err
	}
	if claims["scope"] != androidPublisherScope || claims["iss"] != "moth@proj.iam.gserviceaccount.com" {
		return errors.New("bad assertion claims")
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	return rsa.VerifyPKCS1v15(d.pubKey, crypto.SHA256, digest[:], sig)
}

func TestGoogleSubscriptionStateMapping(t *testing.T) {
	// pubKey must match the SA key; regenerate SA per case so we hold its pub.
	cases := []struct {
		state string
		want  string
	}{
		{"SUBSCRIPTION_STATE_ACTIVE", StatusActive},
		{"SUBSCRIPTION_STATE_IN_GRACE_PERIOD", StatusInGracePeriod},
		{"SUBSCRIPTION_STATE_ON_HOLD", StatusInBillingRetry},
		{"SUBSCRIPTION_STATE_CANCELED", StatusActive},
		{"SUBSCRIPTION_STATE_EXPIRED", StatusExpired},
		{"SUBSCRIPTION_STATE_PAUSED", StatusPaused},
	}
	for _, tc := range cases {
		t.Run(tc.state, func(t *testing.T) {
			saJSON, key := testServiceAccountJSON(t, "")
			sa, err := ParseServiceAccount(saJSON)
			if err != nil {
				t.Fatalf("ParseServiceAccount: %v", err)
			}
			d := newGoogleDouble(t, &key.PublicKey, tc.state)
			ts := NewGoogleTokenSource(sa, d.srv.URL+"/token", http.DefaultClient, func() time.Time { return testNow })
			c := &GoogleClient{BaseURL: d.srv.URL, PackageName: "com.example.app", Tokens: ts, HTTPC: http.DefaultClient}

			sub, raw, err := c.GetSubscriptionV2(t.Context(), "purchase-token-xyz")
			if err != nil {
				t.Fatalf("GetSubscriptionV2: %v", err)
			}
			if sub.Status != tc.want {
				t.Fatalf("state %s -> %q, want %q", tc.state, sub.Status, tc.want)
			}
			if sub.Store != StoreGoogle || sub.StoreTransactionID != "purchase-token-xyz" || sub.ProductID != "pro_monthly" {
				t.Fatalf("sub = %+v", sub)
			}
			if !sub.AutoRenew || sub.CurrentPeriodEnd.IsZero() {
				t.Fatalf("sub renewal/expiry = %+v", sub)
			}
			if raw.SubscriptionState != tc.state {
				t.Fatalf("raw state = %q", raw.SubscriptionState)
			}
			// Request shape: correct path + bearer token.
			wantPath := "/androidpublisher/v3/applications/com.example.app/purchases/subscriptionsv2/tokens/purchase-token-xyz"
			if d.lastPath != wantPath {
				t.Fatalf("path = %q, want %q", d.lastPath, wantPath)
			}
			if d.lastAuth != "Bearer ya29.test" {
				t.Fatalf("auth = %q", d.lastAuth)
			}
		})
	}
}

func TestGoogleSubscriptionNotFound(t *testing.T) {
	saJSON, key := testServiceAccountJSON(t, "")
	sa, _ := ParseServiceAccount(saJSON)
	d := newGoogleDouble(t, &key.PublicKey, "__404__")
	ts := NewGoogleTokenSource(sa, d.srv.URL+"/token", http.DefaultClient, func() time.Time { return testNow })
	c := &GoogleClient{BaseURL: d.srv.URL, PackageName: "com.example.app", Tokens: ts, HTTPC: http.DefaultClient}
	if _, _, err := c.GetSubscriptionV2(t.Context(), "foreign-token"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestGoogleAcknowledge(t *testing.T) {
	saJSON, key := testServiceAccountJSON(t, "")
	sa, _ := ParseServiceAccount(saJSON)
	d := newGoogleDouble(t, &key.PublicKey, "SUBSCRIPTION_STATE_ACTIVE")
	ts := NewGoogleTokenSource(sa, d.srv.URL+"/token", http.DefaultClient, func() time.Time { return testNow })
	c := &GoogleClient{BaseURL: d.srv.URL, PackageName: "com.example.app", Tokens: ts, HTTPC: http.DefaultClient}
	if err := c.AcknowledgeSubscription(t.Context(), "pro_monthly", "tok-1"); err != nil {
		t.Fatalf("AcknowledgeSubscription: %v", err)
	}
	if d.lastMethod != http.MethodPost || !strings.HasSuffix(d.lastPath, ":acknowledge") {
		t.Fatalf("ack call = %s %s", d.lastMethod, d.lastPath)
	}
}

func TestGoogleTokenCaching(t *testing.T) {
	saJSON, key := testServiceAccountJSON(t, "")
	sa, _ := ParseServiceAccount(saJSON)
	d := newGoogleDouble(t, &key.PublicKey, "SUBSCRIPTION_STATE_ACTIVE")
	ts := NewGoogleTokenSource(sa, d.srv.URL+"/token", http.DefaultClient, func() time.Time { return testNow })

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := ts.Token(t.Context()); err != nil {
				t.Errorf("Token: %v", err)
			}
		}()
	}
	wg.Wait()
	d.mu.Lock()
	calls := d.tokenCalls
	d.mu.Unlock()
	if calls != 1 {
		t.Fatalf("token endpoint hit %d times, want 1 (cached)", calls)
	}
}

func TestParsePubSubPush(t *testing.T) {
	dn := DeveloperNotification{
		Version:     "1.0",
		PackageName: "com.example.app",
		SubscriptionNotification: &struct {
			Version          string `json:"version"`
			NotificationType int    `json:"notificationType"`
			PurchaseToken    string `json:"purchaseToken"`
			SubscriptionID   string `json:"subscriptionId"`
		}{Version: "1.0", NotificationType: GoogleNotifRenewed, PurchaseToken: "tok-abc", SubscriptionID: "pro_monthly"},
	}
	raw, _ := json.Marshal(dn)
	envelope := map[string]any{
		"message": map[string]any{
			"data":      base64.StdEncoding.EncodeToString(raw),
			"messageId": "msg-99",
		},
		"subscription": "projects/p/subscriptions/s",
	}
	body, _ := json.Marshal(envelope)

	got, msgID, err := ParsePubSubPush(body)
	if err != nil {
		t.Fatalf("ParsePubSubPush: %v", err)
	}
	if msgID != "msg-99" {
		t.Fatalf("messageID = %q", msgID)
	}
	if got.SubscriptionNotification == nil ||
		got.SubscriptionNotification.PurchaseToken != "tok-abc" ||
		got.SubscriptionNotification.NotificationType != GoogleNotifRenewed {
		t.Fatalf("developer notification = %+v", got.SubscriptionNotification)
	}
}

func TestParsePubSubPushMalformed(t *testing.T) {
	if _, _, err := ParsePubSubPush([]byte(`{"message":{"data":"!!!not base64"}}`)); !errors.Is(err, ErrMalformed) {
		t.Fatalf("error = %v, want ErrMalformed", err)
	}
	if _, _, err := ParsePubSubPush([]byte(`{"message":{}}`)); !errors.Is(err, ErrMalformed) {
		t.Fatalf("empty data error = %v, want ErrMalformed", err)
	}
}

func TestAuthenticatePushToken(t *testing.T) {
	if !AuthenticatePushToken("s3cr3t", "s3cr3t") {
		t.Fatal("valid token rejected")
	}
	if AuthenticatePushToken("spoofed", "s3cr3t") {
		t.Fatal("spoofed token accepted")
	}
	if AuthenticatePushToken("", "s3cr3t") {
		t.Fatal("empty token accepted")
	}
	if AuthenticatePushToken("anything", "") {
		t.Fatal("accepted against empty expected secret")
	}
}

func TestParseServiceAccountRejectsGarbage(t *testing.T) {
	if _, err := ParseServiceAccount([]byte(`{"client_email":"x"}`)); err == nil {
		t.Fatal("accepted service account with no private key")
	}
	if _, err := ParseServiceAccount([]byte(`not json`)); err == nil {
		t.Fatal("accepted non-JSON service account")
	}
}
