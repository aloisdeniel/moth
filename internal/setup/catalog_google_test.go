package setup

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aloisdeniel/moth/internal/billing"
)

// googleCatalogDouble stands in for the OAuth token endpoint + Android
// Publisher subscriptions surface + Cloud Pub/Sub, holding in-memory state so a
// second Sync run sees the first run's writes (idempotency).
type googleCatalogDouble struct {
	srv *httptest.Server

	mu         sync.Mutex
	subs       map[string]map[string]any // productId -> subscription resource
	activated  map[string]bool
	topics     map[string]bool
	pushSubs   map[string]map[string]any
	lastAuth   string
	posts      []recordedGoogPost
	tokenCalls int
}

type recordedGoogPost struct {
	method string
	path   string
	body   map[string]any
}

func newGoogleCatalogDouble(t *testing.T) *googleCatalogDouble {
	d := &googleCatalogDouble{
		subs: map[string]map[string]any{}, activated: map[string]bool{},
		topics: map[string]bool{}, pushSubs: map[string]map[string]any{},
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		d.tokenCalls++
		d.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "ya29.test", "expires_in": 3600})
	})

	mux.HandleFunc("/androidpublisher/", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.lastAuth = r.Header.Get("Authorization")
		p := r.URL.Path

		if strings.HasSuffix(p, ":activate") {
			productID := subIDFromActivate(p)
			d.activated[productID] = true
			_, _ = w.Write([]byte(`{}`))
			return
		}
		productID := lastSeg(p)
		switch r.Method {
		case http.MethodGet:
			sub, ok := d.subs[productID]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "not found"}})
				return
			}
			_ = json.NewEncoder(w).Encode(sub)
		case http.MethodPost:
			body := decode(r)
			d.posts = append(d.posts, recordedGoogPost{r.Method, p, body})
			id, _ := body["productId"].(string)
			// Reconstruct the resource in the shape the Android Publisher API
			// returns rather than echoing moth's write body verbatim, so the
			// idempotency assertion proves moth's read/diff against a Play-shaped
			// resource (with computed fields like base-plan state) — not just a
			// round-trip of moth's own JSON.
			res := playSubResource(body)
			d.subs[id] = res
			_ = json.NewEncoder(w).Encode(res)
		case http.MethodPatch:
			body := decode(r)
			d.posts = append(d.posts, recordedGoogPost{r.Method, p, body})
			res := playSubResource(body)
			d.subs[productID] = res
			_ = json.NewEncoder(w).Encode(res)
		}
	})

	mux.HandleFunc("/v1/", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		name := strings.TrimPrefix(r.URL.Path, "/v1/")
		isTopic := strings.Contains(name, "/topics/")
		switch r.Method {
		case http.MethodGet:
			exists := d.topics[name]
			if !isTopic {
				_, exists = d.pushSubs[name]
			}
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{}`))
				return
			}
			_, _ = w.Write([]byte(`{"name":"` + name + `"}`))
		case http.MethodPut:
			body := decode(r)
			d.posts = append(d.posts, recordedGoogPost{r.Method, r.URL.Path, body})
			if isTopic {
				d.topics[name] = true
			} else {
				d.pushSubs[name] = body
			}
			_, _ = w.Write([]byte(`{"name":"` + name + `"}`))
		}
	})

	d.srv = httptest.NewServer(mux)
	t.Cleanup(d.srv.Close)
	return d
}

// playSubResource rebuilds an Android Publisher Subscription resource from
// moth's write body into the shape the real GET returns: it drops moth's
// write-only wrappers (autoRenewingBasePlanType, newSubscriberAvailability) and
// adds the server-computed base-plan `state`, so a read-back is an independent
// Play-shaped resource, not an echo of moth's request bytes.
func playSubResource(body map[string]any) map[string]any {
	toSlice := func(v any) []any { s, _ := v.([]any); return s }
	var listings []any
	for _, l := range toSlice(body["listings"]) {
		lm, _ := l.(map[string]any)
		listings = append(listings, map[string]any{
			"languageCode": lm["languageCode"],
			"title":        lm["title"],
			"description":  lm["description"],
		})
	}
	var basePlans []any
	for _, bp := range toSlice(body["basePlans"]) {
		bm, _ := bp.(map[string]any)
		var rcs []any
		for _, rc := range toSlice(bm["regionalConfigs"]) {
			rcm, _ := rc.(map[string]any)
			price, _ := rcm["price"].(map[string]any)
			rcs = append(rcs, map[string]any{
				"regionCode": rcm["regionCode"],
				"price": map[string]any{
					"currencyCode": price["currencyCode"],
					"units":        price["units"],
					"nanos":        price["nanos"],
				},
			})
		}
		basePlans = append(basePlans, map[string]any{
			"basePlanId":      bm["basePlanId"],
			"state":           "ACTIVE", // Play computes this; moth's write omits it
			"regionalConfigs": rcs,
		})
	}
	return map[string]any{
		"productId": body["productId"],
		"listings":  listings,
		"basePlans": basePlans,
	}
}

func lastSeg(p string) string {
	i := strings.LastIndex(p, "/")
	return p[i+1:]
}

func subIDFromActivate(p string) string {
	// .../subscriptions/{id}/basePlans/{bp}:activate
	parts := strings.Split(p, "/")
	for i, s := range parts {
		if s == "subscriptions" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// googleTokens builds a real billing.GoogleTokenSource pointed at the double's
// token endpoint (exercises the milestone-11 service-account exchange).
func googleTokens(t *testing.T, tokenURL string) *billing.GoogleTokenSource {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemStr := "-----BEGIN RSA PRIVATE KEY-----\n" + wrap64(base64.StdEncoding.EncodeToString(der)) + "-----END RSA PRIVATE KEY-----\n"
	saJSON, _ := json.Marshal(map[string]string{
		"type": "service_account", "client_email": "moth@proj.iam.gserviceaccount.com",
		"private_key_id": "kid-123", "private_key": pemStr, "token_uri": tokenURL,
	})
	sa, err := billing.ParseServiceAccount(saJSON)
	if err != nil {
		t.Fatalf("ParseServiceAccount: %v", err)
	}
	return billing.NewGoogleTokenSource(sa, tokenURL, http.DefaultClient, func() time.Time { return time.Unix(1_752_000_000, 0) })
}

func wrap64(s string) string {
	var b strings.Builder
	for len(s) > 64 {
		b.WriteString(s[:64] + "\n")
		s = s[64:]
	}
	b.WriteString(s + "\n")
	return b.String()
}

func newGoogleCatalog(t *testing.T, d *googleCatalogDouble) *GoogleCatalog {
	return &GoogleCatalog{
		BaseURL: d.srv.URL, PackageName: "com.example.app",
		Tokens: googleTokens(t, d.srv.URL+"/token"), HTTPC: http.DefaultClient,
	}
}

func TestGoogleCatalogSyncCreatesAndIsIdempotent(t *testing.T) {
	d := newGoogleCatalogDouble(t)
	c := newGoogleCatalog(t, d)
	cat := testCatalog()

	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if d.lastAuth != "Bearer ya29.test" {
		t.Fatalf("auth = %q (service-account token exchange failed)", d.lastAuth)
	}
	if len(res.Products) != 1 || res.Products[0].Action != ActionCreated {
		t.Fatalf("products = %+v", res.Products)
	}
	if res.Products[0].StoreID != "pro_monthly" {
		t.Fatalf("store id = %q", res.Products[0].StoreID)
	}
	if !d.activated["pro_monthly"] {
		t.Fatal("base plan was not activated")
	}

	// Create body: productId, base-plan period P1M, and price 9 units 990000000 nanos.
	var create map[string]any
	for _, post := range d.posts {
		if post.method == http.MethodPost && strings.HasSuffix(post.path, "/subscriptions") {
			create = post.body
		}
	}
	if create == nil {
		t.Fatal("no subscription create recorded")
	}
	bp := create["basePlans"].([]any)[0].(map[string]any)
	if bp["autoRenewingBasePlanType"].(map[string]any)["billingPeriodDuration"] != "P1M" {
		t.Fatalf("period = %+v", bp)
	}
	price := bp["regionalConfigs"].([]any)[0].(map[string]any)["price"].(map[string]any)
	if price["units"] != "9" || price["nanos"].(float64) != 990000000 || price["currencyCode"] != "USD" {
		t.Fatalf("price = %+v", price)
	}

	// Second run: unchanged.
	res2, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	if res2.Products[0].Action != ActionUnchanged || res2.Changed() {
		t.Fatalf("second run = %+v changed=%v", res2.Products, res2.Changed())
	}
}

func TestGoogleCatalogPriceUpdateOnly(t *testing.T) {
	d := newGoogleCatalogDouble(t)
	c := newGoogleCatalog(t, d)
	cat := testCatalog()
	if _, err := c.Sync(context.Background(), cat); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	cat.Tiers[0].Price.Micros = 4_990_000
	res, err := c.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	if res.Products[0].Action != ActionUpdated {
		t.Fatalf("action = %q, want updated", res.Products[0].Action)
	}
	var sawPatch bool
	for _, post := range d.posts {
		if post.method == http.MethodPatch {
			sawPatch = true
		}
	}
	if !sawPatch {
		t.Fatal("expected a PATCH on price change")
	}
}

func TestGoogleRTDNWiring(t *testing.T) {
	d := newGoogleCatalogDouble(t)
	ps := &PubSub{
		BaseURL: d.srv.URL, Project: "demo-proj",
		Tokens: googleTokens(t, d.srv.URL+"/token"), HTTPC: http.DefaultClient,
	}
	res := &SyncResult{Store: billing.StoreGoogle}
	endpoint := "https://moth.example/billing/google/rtdn/demo?token=s3cr3t"
	if err := WireRTDN(context.Background(), ps, "moth-rtdn", "moth-rtdn-push", endpoint, res); err != nil {
		t.Fatalf("WireRTDN: %v", err)
	}

	// Topic + push subscription created; request shapes correct.
	var putTopic, putSub map[string]any
	for _, post := range d.posts {
		if post.method != http.MethodPut {
			continue
		}
		if strings.Contains(post.path, "/topics/") {
			putTopic = post.body
		} else if strings.Contains(post.path, "/subscriptions/") {
			putSub = post.body
		}
	}
	if putTopic == nil || putSub == nil {
		t.Fatalf("missing topic/sub PUTs: topic=%v sub=%v", putTopic != nil, putSub != nil)
	}
	if putSub["topic"] != "projects/demo-proj/topics/moth-rtdn" {
		t.Fatalf("push sub topic = %v", putSub["topic"])
	}
	if putSub["pushConfig"].(map[string]any)["pushEndpoint"] != endpoint {
		t.Fatalf("push endpoint = %v", putSub["pushConfig"])
	}
	// The un-automatable Console-pointing step is always returned.
	if !hasManual(res, "Point Play Console at the RTDN topic") {
		t.Fatalf("missing guided RTDN step: %+v", res.ManualSteps)
	}

	// Idempotent: second wiring creates nothing new.
	res2 := &SyncResult{Store: billing.StoreGoogle}
	if err := WireRTDN(context.Background(), ps, "moth-rtdn", "moth-rtdn-push", endpoint, res2); err != nil {
		t.Fatalf("WireRTDN 2: %v", err)
	}
	if res2.Changed() {
		t.Fatalf("second RTDN wiring reported changes: %+v", res2.Notifications)
	}
}

func TestGoogleRTDNGuidedWithoutCredential(t *testing.T) {
	res := &SyncResult{Store: billing.StoreGoogle}
	endpoint := "https://moth.example/billing/google/rtdn/demo?token=s3cr3t"
	if err := WireRTDN(context.Background(), nil, "moth-rtdn", "moth-rtdn-push", endpoint, res); err != nil {
		t.Fatalf("WireRTDN: %v", err)
	}
	if len(res.Notifications) != 1 || res.Notifications[0].Action != ActionManual {
		t.Fatalf("notifications = %+v", res.Notifications)
	}
	if !hasManual(res, "Point Play Console at the RTDN topic") {
		t.Fatalf("missing guided RTDN step: %+v", res.ManualSteps)
	}
}
