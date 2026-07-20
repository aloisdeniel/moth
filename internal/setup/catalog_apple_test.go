package setup

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ascCatalogDouble is an httptest stand-in for the ASC subscription surface. It
// keeps in-memory state so a second Sync run observes what the first created —
// the idempotency assertion.
type ascCatalogDouble struct {
	srv *httptest.Server

	mu          sync.Mutex
	groups      map[string]string // referenceName -> resource id
	subs        map[string]*ascSubState
	authSeen    bool
	notifyCalls int
	notify404   bool
	posts       []recordedPost
	pricePoints map[string]string // customerPrice -> price point id (available ladder)
}

type ascSubState struct {
	id         string
	gid        string
	productID  string
	name       string
	period     string
	groupLevel int
	pricePts   []string // scheduled price point ids
	locales    map[string]*ASCLocalization
}

type recordedPost struct {
	path string
	body map[string]any
}

func newASCCatalogDouble(t *testing.T) *ascCatalogDouble {
	d := &ascCatalogDouble{
		groups:      map[string]string{},
		subs:        map[string]*ascSubState{},
		pricePoints: map[string]string{"9.99": "PP_999", "4.99": "PP_499"},
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") && strings.Count(auth, ".") == 2 {
			d.mu.Lock()
			d.authSeen = true
			d.mu.Unlock()
		}
		d.mu.Lock()
		defer d.mu.Unlock()

		p := r.URL.Path
		switch {
		// List a group's subscriptions.
		case strings.HasPrefix(p, "/v1/subscriptionGroups/") && strings.HasSuffix(p, "/subscriptions"):
			gid := strings.TrimSuffix(strings.TrimPrefix(p, "/v1/subscriptionGroups/"), "/subscriptions")
			var data []map[string]any
			for _, s := range d.subs {
				if s.groupID() == gid {
					data = append(data, subResource(s))
				}
			}
			writeData(w, data)

		// List an app's subscription groups.
		case strings.HasPrefix(p, "/v1/apps/") && strings.HasSuffix(p, "/subscriptionGroups"):
			var data []map[string]any
			for ref, id := range d.groups {
				data = append(data, map[string]any{"type": "subscriptionGroups", "id": id, "attributes": map[string]any{"referenceName": ref}})
			}
			writeData(w, data)

		case p == "/v1/subscriptionGroups" && r.Method == http.MethodPost:
			body := decode(r)
			ref, _ := attr(body, "referenceName").(string)
			id := "GRP_" + ref
			d.groups[ref] = id
			writeOne(w, map[string]any{"type": "subscriptionGroups", "id": id, "attributes": map[string]any{"referenceName": ref}})

		case p == "/v1/subscriptions" && r.Method == http.MethodPost:
			body := decode(r)
			d.posts = append(d.posts, recordedPost{p, body})
			productID, _ := attr(body, "productId").(string)
			id := "SUB_" + productID
			s := &ascSubState{
				id:        id,
				productID: productID,
				name:      str(attr(body, "name")),
				period:    str(attr(body, "subscriptionPeriod")),
				locales:   map[string]*ASCLocalization{},
			}
			if gl, ok := attr(body, "groupLevel").(float64); ok {
				s.groupLevel = int(gl)
			}
			s.setGroup(rel(body, "group"))
			d.subs[id] = s
			writeOne(w, subResource(s))

		// Subscription-scoped GETs: price points, current prices, localizations.
		case strings.HasPrefix(p, "/v1/subscriptions/") && strings.Contains(p, "/pricePoints"):
			var data []map[string]any
			for price, id := range d.pricePoints {
				data = append(data, map[string]any{"type": "subscriptionPricePoints", "id": id, "attributes": map[string]any{"customerPrice": price}})
			}
			writeData(w, data)

		case strings.HasPrefix(p, "/v1/subscriptions/") && strings.HasSuffix(p, "/prices"):
			sub := d.subByID(strings.TrimSuffix(strings.TrimPrefix(p, "/v1/subscriptions/"), "/prices"))
			var data []map[string]any
			if sub != nil {
				for _, pp := range sub.pricePts {
					data = append(data, map[string]any{"type": "subscriptionPrices", "id": "PRICE_" + pp,
						"relationships": map[string]any{"subscriptionPricePoint": map[string]any{"data": map[string]any{"id": pp}}}})
				}
			}
			writeData(w, data)

		case strings.HasPrefix(p, "/v1/subscriptions/") && strings.HasSuffix(p, "/subscriptionLocalizations"):
			sub := d.subByID(strings.TrimSuffix(strings.TrimPrefix(p, "/v1/subscriptions/"), "/subscriptionLocalizations"))
			var data []map[string]any
			if sub != nil {
				for _, l := range sub.locales {
					data = append(data, map[string]any{"type": "subscriptionLocalizations", "id": l.ResourceID,
						"attributes": map[string]any{"locale": l.Locale, "name": l.Name, "description": l.Description}})
				}
			}
			writeData(w, data)

		// PATCH a subscription.
		case strings.HasPrefix(p, "/v1/subscriptions/") && r.Method == http.MethodPatch:
			id := strings.TrimPrefix(p, "/v1/subscriptions/")
			body := decode(r)
			if s := d.subs[id]; s != nil {
				s.name = str(attr(body, "name"))
				if gl, ok := attr(body, "groupLevel").(float64); ok {
					s.groupLevel = int(gl)
				}
			}
			writeOne(w, map[string]any{"type": "subscriptions", "id": id})

		case p == "/v1/subscriptionPrices" && r.Method == http.MethodPost:
			body := decode(r)
			d.posts = append(d.posts, recordedPost{p, body})
			subID := rel(body, "subscription")
			ppID := rel(body, "subscriptionPricePoint")
			if s := d.subs[subID]; s != nil {
				s.pricePts = append(s.pricePts, ppID)
			}
			writeOne(w, map[string]any{"type": "subscriptionPrices", "id": "PRICE_new"})

		case p == "/v1/subscriptionLocalizations" && r.Method == http.MethodPost:
			body := decode(r)
			d.posts = append(d.posts, recordedPost{p, body})
			subID := rel(body, "subscription")
			locale := str(attr(body, "locale"))
			if s := d.subs[subID]; s != nil {
				s.locales[locale] = &ASCLocalization{ResourceID: "LOC_" + subID + "_" + locale, Locale: locale,
					Name: str(attr(body, "name")), Description: str(attr(body, "description"))}
			}
			writeOne(w, map[string]any{"type": "subscriptionLocalizations", "id": "LOC_new"})

		case strings.HasPrefix(p, "/v1/subscriptionLocalizations/") && r.Method == http.MethodPatch:
			writeOne(w, map[string]any{"type": "subscriptionLocalizations", "id": strings.TrimPrefix(p, "/v1/subscriptionLocalizations/")})

		// App PATCH — server notification URL registration.
		case strings.HasPrefix(p, "/v1/apps/") && r.Method == http.MethodPatch:
			d.notifyCalls++
			if d.notify404 {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"errors":[{"title":"NOT_FOUND","detail":"no such endpoint"}]}`))
				return
			}
			writeOne(w, map[string]any{"type": "apps", "id": "APP1"})

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"title":"NOT_FOUND"}]}`))
		}
	})
	d.srv = httptest.NewServer(mux)
	t.Cleanup(d.srv.Close)
	return d
}

// group id is stored on the sub via setGroup/groupID.
func (s *ascSubState) setGroup(id string) { s.gid = id }
func (s *ascSubState) groupID() string    { return s.gid }

func (d *ascCatalogDouble) subByID(id string) *ascSubState { return d.subs[id] }

func subResource(s *ascSubState) map[string]any {
	return map[string]any{
		"type": "subscriptions", "id": s.id,
		"attributes": map[string]any{
			"productId": s.productID, "name": s.name,
			"subscriptionPeriod": s.period, "groupLevel": s.groupLevel,
		},
	}
}

func writeData(w http.ResponseWriter, data []map[string]any) {
	if data == nil {
		data = []map[string]any{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}
func writeOne(w http.ResponseWriter, one map[string]any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"data": one})
}
func decode(r *http.Request) map[string]any {
	b, _ := io.ReadAll(r.Body)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

// attr reads data.attributes[key] from a request body.
func attr(body map[string]any, key string) any {
	data, _ := body["data"].(map[string]any)
	a, _ := data["attributes"].(map[string]any)
	return a[key]
}

// rel reads data.relationships[name].data.id.
func rel(body map[string]any, name string) string {
	data, _ := body["data"].(map[string]any)
	rels, _ := data["relationships"].(map[string]any)
	r, _ := rels[name].(map[string]any)
	rd, _ := r["data"].(map[string]any)
	id, _ := rd["id"].(string)
	return id
}

func str(v any) string { s, _ := v.(string); return s }

func testCatalog() DesiredCatalog {
	return DesiredCatalog{
		GroupReference: "moth-main",
		Tiers: []DesiredTier{{
			ProductID: "pro_monthly", Reference: "Pro Monthly", DisplayName: "Pro",
			Description: "All features", Period: PeriodMonthly,
			Price: Money{Currency: "USD", Micros: 9_990_000}, Locale: "en-US", GroupLevel: 1,
		}},
	}
}

func newAppleCatalog(t *testing.T, d *ascCatalogDouble) *AppleCatalog {
	t.Helper()
	_, key := testP8(t)
	return &AppleCatalog{
		ASC: &ASC{
			IssuerID: "iss", KeyID: "kid", Key: key,
			BaseURL: d.srv.URL, HTTPC: d.srv.Client(),
			Now: func() time.Time { return time.Unix(1_752_000_000, 0) },
		},
		AppID:           "APP1",
		NotificationURL: "https://moth.example/billing/apple/notify/demo",
	}
}

func TestAppleCatalogSyncCreatesAndIsIdempotent(t *testing.T) {
	d := newASCCatalogDouble(t)
	a := newAppleCatalog(t, d)
	cat := testCatalog()

	res, err := a.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !d.authSeen {
		t.Fatal("ASC requests were not JWT-authenticated")
	}
	if len(res.Products) != 1 || res.Products[0].Action != ActionCreated {
		t.Fatalf("first run products = %+v", res.Products)
	}
	if res.Products[0].StoreID != "SUB_pro_monthly" {
		t.Fatalf("store id = %q", res.Products[0].StoreID)
	}
	if !res.Changed() {
		t.Fatal("first run should report changes")
	}

	// Request bodies: subscription create carried productId + period, and a
	// price point (PP_999) matching 9.99 was attached.
	var sawSubCreate, sawPrice, sawLoc bool
	for _, post := range d.posts {
		switch post.path {
		case "/v1/subscriptions":
			sawSubCreate = str(attr(post.body, "productId")) == "pro_monthly" && str(attr(post.body, "subscriptionPeriod")) == "ONE_MONTH"
		case "/v1/subscriptionPrices":
			sawPrice = rel(post.body, "subscriptionPricePoint") == "PP_999"
		case "/v1/subscriptionLocalizations":
			sawLoc = str(attr(post.body, "name")) == "Pro" && str(attr(post.body, "locale")) == "en-US"
		}
	}
	if !sawSubCreate || !sawPrice || !sawLoc {
		t.Fatalf("missing create bodies: sub=%v price=%v loc=%v", sawSubCreate, sawPrice, sawLoc)
	}

	// A new subscription surfaces the review + notification-URL manual steps.
	if !hasManual(res, "Submit pro_monthly for review") {
		t.Fatalf("missing review step: %+v", res.ManualSteps)
	}
	// Notification URL registered via API (double returns 200).
	if len(res.Notifications) != 1 || res.Notifications[0].Action != ActionCreated {
		t.Fatalf("notifications = %+v", res.Notifications)
	}

	// Second run: no product changes.
	res2, err := a.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("second Sync: %v", err)
	}
	if res2.Products[0].Action != ActionUnchanged {
		t.Fatalf("second run action = %q, want unchanged", res2.Products[0].Action)
	}
	if res2.Changed() {
		t.Fatalf("second run reported changes: %+v", res2.Products)
	}
}

func TestAppleCatalogPriceUpdateOnly(t *testing.T) {
	d := newASCCatalogDouble(t)
	a := newAppleCatalog(t, d)
	cat := testCatalog()
	if _, err := a.Sync(context.Background(), cat); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	// Change the price to 4.99 (PP_499 is on the ladder) and re-run.
	cat.Tiers[0].Price.Micros = 4_990_000
	res, err := a.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	if res.Products[0].Action != ActionUpdated {
		t.Fatalf("action = %q, want updated", res.Products[0].Action)
	}
	sub := d.subs["SUB_pro_monthly"]
	if len(sub.pricePts) != 2 || sub.pricePts[1] != "PP_499" {
		t.Fatalf("price points = %v", sub.pricePts)
	}
}

func TestAppleCatalogPriceNotOnLadderGuided(t *testing.T) {
	d := newASCCatalogDouble(t)
	a := newAppleCatalog(t, d)
	cat := testCatalog()
	cat.Tiers[0].Price.Micros = 7_770_000 // no ladder point equals 7.77
	res, err := a.Sync(context.Background(), cat)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !hasManual(res, "Set the price for pro_monthly") {
		t.Fatalf("expected guided price step, got %+v", res.ManualSteps)
	}
}

func TestAppleCatalogNotificationGuidedOn404(t *testing.T) {
	d := newASCCatalogDouble(t)
	d.notify404 = true
	a := newAppleCatalog(t, d)
	res, err := a.Sync(context.Background(), testCatalog())
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(res.Notifications) != 1 || res.Notifications[0].Action != ActionManual {
		t.Fatalf("notifications = %+v", res.Notifications)
	}
	if !hasManual(res, "Register the App Store Server Notification URL") {
		t.Fatalf("expected guided notification step, got %+v", res.ManualSteps)
	}
}

func hasManual(res *SyncResult, title string) bool {
	for _, m := range res.ManualSteps {
		if m.Title == title {
			return true
		}
	}
	return false
}
