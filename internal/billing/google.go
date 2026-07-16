package billing

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Google Play Developer API defaults, overridable for tests.
const (
	GooglePlayBaseURL     = "https://androidpublisher.googleapis.com"
	GoogleTokenURL        = "https://oauth2.googleapis.com/token"
	androidPublisherScope = "https://www.googleapis.com/auth/androidpublisher"
	// jwtBearerGrant is the RFC 7523 service-account grant type.
	jwtBearerGrant  = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	googleTokenLife = time.Hour
)

// GoogleServiceAccount is the subset of a Google service-account JSON key moth
// needs to mint an androidpublisher access token.
type GoogleServiceAccount struct {
	ClientEmail   string `json:"client_email"`
	PrivateKeyID  string `json:"private_key_id"`
	PrivateKeyPEM string `json:"private_key"`
	TokenURI      string `json:"token_uri"`

	key *rsa.PrivateKey
}

// ParseServiceAccount parses a service-account JSON key and its PEM private
// key. The parsed key is stored encrypted under the master key by the caller;
// this only turns bytes into a usable signer.
func ParseServiceAccount(data []byte) (*GoogleServiceAccount, error) {
	var sa GoogleServiceAccount
	if err := json.Unmarshal(data, &sa); err != nil {
		return nil, fmt.Errorf("billing: parse service account: %w", err)
	}
	if sa.ClientEmail == "" || sa.PrivateKeyPEM == "" {
		return nil, errors.New("billing: service account missing client_email or private_key")
	}
	block, _ := pem.Decode([]byte(sa.PrivateKeyPEM))
	if block == nil {
		return nil, errors.New("billing: service account private_key is not PEM")
	}
	key, err := parseRSAPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("billing: service account key: %w", err)
	}
	sa.key = key
	if sa.TokenURI == "" {
		sa.TokenURI = GoogleTokenURL
	}
	return &sa, nil
}

func parseRSAPrivateKey(der []byte) (*rsa.PrivateKey, error) {
	if k, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA key")
	}
	return rk, nil
}

// GoogleTokenSource exchanges a service-account assertion for an
// androidpublisher OAuth2 access token (RFC 7523 JWT-bearer grant) and caches
// it until shortly before expiry. Safe for concurrent use.
type GoogleTokenSource struct {
	sa       *GoogleServiceAccount
	tokenURL string
	httpc    Doer
	now      func() time.Time

	mu       sync.Mutex
	token    string
	expireAt time.Time
}

// NewGoogleTokenSource returns a cached token source. tokenURL defaults to the
// service account's token_uri (or GoogleTokenURL); httpc and now default to a
// timeout-bounded client and time.Now.
func NewGoogleTokenSource(sa *GoogleServiceAccount, tokenURL string, httpc Doer, now func() time.Time) *GoogleTokenSource {
	if tokenURL == "" {
		tokenURL = sa.TokenURI
	}
	if httpc == nil {
		httpc = defaultDoer()
	}
	if now == nil {
		now = time.Now
	}
	return &GoogleTokenSource{sa: sa, tokenURL: tokenURL, httpc: httpc, now: now}
}

// Token returns a valid access token, minting a fresh one when the cache is
// within a minute of expiry.
func (ts *GoogleTokenSource) Token(ctx context.Context) (string, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	now := ts.now()
	if ts.token != "" && now.Before(ts.expireAt.Add(-time.Minute)) {
		return ts.token, nil
	}
	assertion, err := ts.sign(now)
	if err != nil {
		return "", err
	}
	form := url.Values{
		"grant_type": {jwtBearerGrant},
		"assertion":  {assertion},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := ts.httpc.Do(req)
	if err != nil {
		return "", fmt.Errorf("google token: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("google token: read: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("google token: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("google token: decode: %w", err)
	}
	if tr.AccessToken == "" {
		return "", errors.New("google token: empty access_token")
	}
	ttl := time.Duration(tr.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = googleTokenLife
	}
	ts.token = tr.AccessToken
	ts.expireAt = now.Add(ttl)
	return ts.token, nil
}

// sign builds the RS256 service-account assertion JWT.
func (ts *GoogleTokenSource) sign(now time.Time) (string, error) {
	head, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": ts.sa.PrivateKeyID})
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(map[string]any{
		"iss":   ts.sa.ClientEmail,
		"scope": androidPublisherScope,
		"aud":   ts.tokenURL,
		"iat":   now.Unix(),
		"exp":   now.Add(googleTokenLife).Unix(),
	})
	if err != nil {
		return "", err
	}
	signingInput := b64url.EncodeToString(head) + "." + b64url.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, ts.sa.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("google token: sign assertion: %w", err)
	}
	return signingInput + "." + b64url.EncodeToString(sig), nil
}

// GoogleClient calls the Google Play Developer API for one project's package.
type GoogleClient struct {
	// BaseURL defaults to GooglePlayBaseURL; tests point it at a double.
	BaseURL     string
	PackageName string
	Tokens      *GoogleTokenSource
	HTTPC       Doer
}

func (c *GoogleClient) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return GooglePlayBaseURL
}

func (c *GoogleClient) httpc() Doer {
	if c.HTTPC != nil {
		return c.HTTPC
	}
	return defaultDoer()
}

// SubscriptionPurchaseV2 is the subset of the purchases.subscriptionsv2.get
// response moth reads.
type SubscriptionPurchaseV2 struct {
	SubscriptionState string `json:"subscriptionState"`
	LineItems         []struct {
		ProductID        string `json:"productId"`
		ExpiryTime       string `json:"expiryTime"` // RFC3339
		AutoRenewingPlan *struct {
			AutoRenewEnabled bool `json:"autoRenewEnabled"`
		} `json:"autoRenewingPlan"`
	} `json:"lineItems"`
	// TestPurchase is present only for license-tester / sandbox purchases.
	TestPurchase         *struct{} `json:"testPurchase"`
	AcknowledgementState string    `json:"acknowledgementState"`
}

// GetSubscriptionV2 resolves a purchaseToken to authoritative subscription
// state and returns a NormalizedSubscription. A 404 (token for another
// package/project, or an unknown token) surfaces as ErrNotFound.
func (c *GoogleClient) GetSubscriptionV2(ctx context.Context, purchaseToken string) (NormalizedSubscription, SubscriptionPurchaseV2, error) {
	path := fmt.Sprintf("/androidpublisher/v3/applications/%s/purchases/subscriptionsv2/tokens/%s",
		url.PathEscape(c.PackageName), url.PathEscape(purchaseToken))
	var out SubscriptionPurchaseV2
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return NormalizedSubscription{}, SubscriptionPurchaseV2{}, err
	}
	return normalizeGoogle(purchaseToken, &out), out, nil
}

// AcknowledgeSubscription acknowledges a purchase (purchases.subscriptions.
// acknowledge). Google auto-refunds a subscription that is not acknowledged
// within three days, so the caller acknowledges once acknowledgementState is
// "acknowledgementStatePending".
func (c *GoogleClient) AcknowledgeSubscription(ctx context.Context, subscriptionID, purchaseToken string) error {
	path := fmt.Sprintf("/androidpublisher/v3/applications/%s/purchases/subscriptions/%s/tokens/%s:acknowledge",
		url.PathEscape(c.PackageName), url.PathEscape(subscriptionID), url.PathEscape(purchaseToken))
	return c.do(ctx, http.MethodPost, path, map[string]any{}, nil)
}

func (c *GoogleClient) do(ctx context.Context, method, path string, body, out any) error {
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
		return fmt.Errorf("%w: %s", ErrNotFound, googleErrMessage(payload))
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("google play: status %d: %s", resp.StatusCode, googleErrMessage(payload))
	}
	if out != nil {
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("google play: decode response: %w", err)
		}
	}
	return nil
}

func googleErrMessage(body []byte) string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &parsed)
	return parsed.Error.Message
}

// normalizeGoogle maps a SubscriptionPurchaseV2 into the normalized model.
// State mapping (SubscriptionPurchaseV2.subscriptionState):
//
//	ACTIVE           -> active
//	IN_GRACE_PERIOD  -> in_grace_period
//	ON_HOLD          -> in_billing_retry (Google's account-hold == billing retry)
//	PAUSED           -> paused
//	CANCELED         -> active (auto-renew off, still entitled until expiry;
//	                    Google reports EXPIRED once the period ends)
//	EXPIRED / other  -> expired
func normalizeGoogle(purchaseToken string, p *SubscriptionPurchaseV2) NormalizedSubscription {
	sub := NormalizedSubscription{
		Store:              StoreGoogle,
		StoreTransactionID: purchaseToken,
		Status:             googleStatus(p.SubscriptionState),
		Environment:        EnvProduction,
	}
	if p.TestPurchase != nil {
		sub.Environment = EnvSandbox
	}
	if len(p.LineItems) > 0 {
		li := p.LineItems[0]
		sub.ProductID = li.ProductID
		sub.SubscriptionID = li.ProductID
		if li.AutoRenewingPlan != nil {
			sub.AutoRenew = li.AutoRenewingPlan.AutoRenewEnabled
		}
		if t, err := time.Parse(time.RFC3339, li.ExpiryTime); err == nil {
			sub.CurrentPeriodEnd = t.UTC()
		}
	}
	raw, _ := json.Marshal(p)
	sub.RawState = raw
	return sub
}

func googleStatus(state string) string {
	switch state {
	case "SUBSCRIPTION_STATE_ACTIVE":
		return StatusActive
	case "SUBSCRIPTION_STATE_IN_GRACE_PERIOD":
		return StatusInGracePeriod
	case "SUBSCRIPTION_STATE_ON_HOLD":
		return StatusInBillingRetry
	case "SUBSCRIPTION_STATE_PAUSED":
		return StatusPaused
	case "SUBSCRIPTION_STATE_CANCELED":
		// User turned off renewal but the paid period is still running; the
		// API reports EXPIRED once it lapses.
		return StatusActive
	default: // EXPIRED, PENDING, PENDING_PURCHASE_CANCELED, UNSPECIFIED
		return StatusExpired
	}
}

// ---- Real-time Developer Notifications (RTDN via Cloud Pub/Sub push) ----

// Google RTDN subscriptionNotification.notificationType codes.
const (
	GoogleNotifRecovered            = 1
	GoogleNotifRenewed              = 2
	GoogleNotifCanceled             = 3
	GoogleNotifPurchased            = 4
	GoogleNotifOnHold               = 5
	GoogleNotifInGracePeriod        = 6
	GoogleNotifRestarted            = 7
	GoogleNotifPriceChangeConfirmed = 8
	GoogleNotifDeferred             = 9
	GoogleNotifPaused               = 10
	GoogleNotifPauseScheduleChanged = 11
	GoogleNotifRevoked              = 12
	GoogleNotifExpired              = 13
)

// DeveloperNotification is the Play RTDN payload carried inside a Pub/Sub push.
type DeveloperNotification struct {
	Version                  string `json:"version"`
	PackageName              string `json:"packageName"`
	EventTimeMillis          string `json:"eventTimeMillis"`
	SubscriptionNotification *struct {
		Version          string `json:"version"`
		NotificationType int    `json:"notificationType"`
		PurchaseToken    string `json:"purchaseToken"`
		SubscriptionID   string `json:"subscriptionId"`
	} `json:"subscriptionNotification"`
	TestNotification *struct {
		Version string `json:"version"`
	} `json:"testNotification"`
}

// pubSubPush is the Cloud Pub/Sub push-subscription envelope.
type pubSubPush struct {
	Message struct {
		Data        string `json:"data"` // base64 std of the DeveloperNotification JSON
		MessageID   string `json:"messageId"`
		PublishTime string `json:"publishTime"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

// ParsePubSubPush parses a Pub/Sub push envelope and decodes the embedded
// DeveloperNotification. The RTDN is a nudge: the caller re-reads state via
// GetSubscriptionV2 using the returned purchaseToken. MessageID is the Pub/Sub
// message id, usable as the idempotency key.
func ParsePubSubPush(body []byte) (notif *DeveloperNotification, messageID string, err error) {
	var env pubSubPush
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, "", fmt.Errorf("%w: pubsub envelope: %v", ErrMalformed, err)
	}
	if env.Message.Data == "" {
		return nil, "", fmt.Errorf("%w: empty pubsub message data", ErrMalformed)
	}
	raw, err := base64.StdEncoding.DecodeString(env.Message.Data)
	if err != nil {
		return nil, "", fmt.Errorf("%w: pubsub data base64: %v", ErrMalformed, err)
	}
	var dn DeveloperNotification
	if err := json.Unmarshal(raw, &dn); err != nil {
		return nil, "", fmt.Errorf("%w: developer notification: %v", ErrMalformed, err)
	}
	return &dn, env.Message.MessageID, nil
}

// AuthenticatePushToken verifies a shared-secret path token guarding the RTDN
// webhook, in constant time. moth generates a random token per project and
// registers the Pub/Sub push endpoint as
// /billing/google/rtdn/{slug}?token=SECRET (or as a path segment); Google
// replays it on every push, and a request without the exact secret is dropped.
//
// This is the simple, self-contained option. The alternative — verifying the
// OIDC identity token Google signs the push with (aud = the endpoint, iss =
// accounts.google.com, email = the push service account) — needs the Google
// JWKS verifier (internal/oidc) and the configured service-account email; it
// is preferable when the endpoint is public and the operator cannot keep a URL
// secret. moth ships the shared-secret path and documents the OIDC path here.
func AuthenticatePushToken(got, want string) bool {
	if want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
