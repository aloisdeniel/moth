package billingrpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	billingv1 "github.com/aloisdeniel/moth/gen/moth/billing/v1"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

func hasEntitlement(info *billingv1.CustomerInfo, id string) *billingv1.Entitlement {
	for _, e := range info.ActiveEntitlements {
		if e.Identifier == id {
			return e
		}
	}
	return nil
}

func TestGetCustomerInfoFreeUserIsNoneNoError(t *testing.T) {
	f := newFixture(t)
	resp, err := f.h.GetCustomerInfo(f.ctx(), authReq(f, &billingv1.GetCustomerInfoRequest{}))
	if err != nil {
		t.Fatalf("GetCustomerInfo for never-paid user errored: %v", err)
	}
	if n := len(resp.Msg.CustomerInfo.ActiveEntitlements); n != 0 {
		t.Fatalf("free user should hold no entitlements, got %d", n)
	}
}

func TestSubmitPurchaseAppleGrantsEntitlement(t *testing.T) {
	f := newFixture(t)
	dbl := f.appleStatusDouble(1, appleTxn("orig-1", "com.demo.monthly", f.now.Add(30*24*time.Hour), 0, 0))
	f.setAppleCreds(dbl.URL)

	jws := f.ca.signJWS(t, appleTxn("orig-1", "com.demo.monthly", f.now.Add(30*24*time.Hour), 0, 0))
	resp, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:   billingv1.Store_STORE_APPLE,
		Receipt: &billingv1.SubmitPurchaseRequest_AppleJwsTransaction{AppleJwsTransaction: jws},
	}))
	if err != nil {
		t.Fatalf("SubmitPurchase apple: %v", err)
	}
	if e := hasEntitlement(resp.Msg.CustomerInfo, "pro"); e == nil {
		t.Fatalf("expected pro entitlement, got %+v", resp.Msg.CustomerInfo.ActiveEntitlements)
	} else if e.Source != billingv1.EntitlementSource_ENTITLEMENT_SOURCE_STORE {
		t.Errorf("source = %v, want STORE", e.Source)
	}
	// The subscription is persisted and linked to the user.
	subs, _ := f.st.ListUserSubscriptions(context.Background(), f.project.ID, f.user.ID)
	if len(subs) != 1 || subs[0].Status != store.SubscriptionStatusActive {
		t.Fatalf("subscription not stored active: %+v", subs)
	}
	// A revenue event was recorded.
	assertEventCount(t, f, store.SubscriptionEventPurchased, 1)

	// GetCustomerInfo reflects it independently.
	got, err := f.h.GetCustomerInfo(f.ctx(), authReq(f, &billingv1.GetCustomerInfoRequest{}))
	if err != nil || hasEntitlement(got.Msg.CustomerInfo, "pro") == nil {
		t.Fatalf("GetCustomerInfo did not reflect purchase: %v %+v", err, got.Msg.CustomerInfo)
	}
}

func TestSubmitPurchaseAppleTrialEmitsTrialStarted(t *testing.T) {
	f := newFixture(t)
	txn := appleTxn("orig-t", "com.demo.monthly", f.now.Add(7*24*time.Hour), 1, 0) // offerType!=0 => trial
	dbl := f.appleStatusDouble(1, txn)
	f.setAppleCreds(dbl.URL)
	_, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:   billingv1.Store_STORE_APPLE,
		Receipt: &billingv1.SubmitPurchaseRequest_AppleJwsTransaction{AppleJwsTransaction: f.ca.signJWS(t, txn)},
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertEventCount(t, f, store.SubscriptionEventTrialStarted, 1)
}

// TestSubmitPurchaseAppleStoreReportedPrice guards finding 4: revenue must be
// the store-reported transaction amount (storefront-localized), not moth's
// single catalog list price. The catalog product is 4_990_000 USD; the Apple
// transaction reports a UK buyer paying 8_990 milliunits GBP.
func TestSubmitPurchaseAppleStoreReportedPrice(t *testing.T) {
	f := newFixture(t)
	txn := appleTxn("orig-gbp", "com.demo.monthly", f.now.Add(30*24*time.Hour), 0, 0)
	txn["price"] = 8990 // milliunits => 8_990_000 micros
	txn["currency"] = "GBP"
	dbl := f.appleStatusDouble(1, txn)
	f.setAppleCreds(dbl.URL)
	if _, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:   billingv1.Store_STORE_APPLE,
		Receipt: &billingv1.SubmitPurchaseRequest_AppleJwsTransaction{AppleJwsTransaction: f.ca.signJWS(t, txn)},
	})); err != nil {
		t.Fatal(err)
	}
	rows, err := storeRawEventRows(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 event, got %+v", rows)
	}
	if rows[0].Type != store.SubscriptionEventPurchased ||
		rows[0].PriceAmountMicros != 8_990_000 || rows[0].Currency != "GBP" {
		t.Fatalf("event = %+v, want store-reported 8990000 GBP", rows[0])
	}
}

// TestSubmitPurchaseAppleTrialConversion guards findings 5/10: a trialing
// subscription that the store later reports active emits exactly one
// subscription.converted event (and no second acquisition event), feeding the
// trial-to-paid dashboard the production engine otherwise never populates.
func TestSubmitPurchaseAppleTrialConversion(t *testing.T) {
	f := newFixture(t)
	// 1) Trial start: status code 1 + offerType!=0 => trialing.
	trial := appleTxn("orig-conv", "com.demo.monthly", f.now.Add(7*24*time.Hour), 1, 0)
	dbl := f.appleStatusDouble(1, trial)
	f.setAppleCreds(dbl.URL)
	if _, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:   billingv1.Store_STORE_APPLE,
		Receipt: &billingv1.SubmitPurchaseRequest_AppleJwsTransaction{AppleJwsTransaction: f.ca.signJWS(t, trial)},
	})); err != nil {
		t.Fatal(err)
	}
	assertEventCount(t, f, store.SubscriptionEventTrialStarted, 1)

	// 2) Same original transaction now active (offerType 0). The double must
	// serve the active transaction so the authoritative re-read flips the sub.
	paid := appleTxn("orig-conv", "com.demo.monthly", f.now.Add(30*24*time.Hour), 0, 0)
	f.h.appleBaseURL = f.appleStatusDouble(1, paid).URL
	if _, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:   billingv1.Store_STORE_APPLE,
		Receipt: &billingv1.SubmitPurchaseRequest_AppleJwsTransaction{AppleJwsTransaction: f.ca.signJWS(t, paid)},
	})); err != nil {
		t.Fatal(err)
	}
	assertEventCount(t, f, store.SubscriptionEventConverted, 1)
	// No spurious second acquisition event (still exactly one trial_started, no purchased).
	assertEventCount(t, f, store.SubscriptionEventTrialStarted, 1)
	assertEventCount(t, f, store.SubscriptionEventPurchased, 0)

	// 3) A further active re-read (active -> active) must NOT re-emit converted.
	if _, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:   billingv1.Store_STORE_APPLE,
		Receipt: &billingv1.SubmitPurchaseRequest_AppleJwsTransaction{AppleJwsTransaction: f.ca.signJWS(t, paid)},
	})); err != nil {
		t.Fatal(err)
	}
	assertEventCount(t, f, store.SubscriptionEventConverted, 1)
}

func TestSubmitPurchaseAppleWrongBundleRejected(t *testing.T) {
	f := newFixture(t)
	dbl := f.appleStatusDouble(1, appleTxn("orig-x", "com.demo.monthly", f.now.Add(time.Hour), 0, 0))
	f.setAppleCreds(dbl.URL)
	// A transaction minted for another bundle id.
	bad := appleTxn("orig-x", "com.demo.monthly", f.now.Add(time.Hour), 0, 0)
	bad["bundleId"] = "com.attacker.app"
	_, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:   billingv1.Store_STORE_APPLE,
		Receipt: &billingv1.SubmitPurchaseRequest_AppleJwsTransaction{AppleJwsTransaction: f.ca.signJWS(t, bad)},
	}))
	assertReason(t, err, authrpc.ReasonInvalidReceipt)
}

func TestSubmitPurchaseAppleTamperedRejected(t *testing.T) {
	f := newFixture(t)
	dbl := f.appleStatusDouble(1, appleTxn("orig-y", "com.demo.monthly", f.now.Add(time.Hour), 0, 0))
	f.setAppleCreds(dbl.URL)
	jws := f.ca.signJWS(t, appleTxn("orig-y", "com.demo.monthly", f.now.Add(time.Hour), 0, 0))
	// Flip a byte in the payload segment.
	tampered := jws[:len(jws)-6] + "AAAAAA"
	_, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:   billingv1.Store_STORE_APPLE,
		Receipt: &billingv1.SubmitPurchaseRequest_AppleJwsTransaction{AppleJwsTransaction: tampered},
	}))
	assertReason(t, err, authrpc.ReasonInvalidReceipt)
}

func TestSubmitPurchaseNotConfigured(t *testing.T) {
	f := newFixture(t)
	_, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:   billingv1.Store_STORE_APPLE,
		Receipt: &billingv1.SubmitPurchaseRequest_AppleJwsTransaction{AppleJwsTransaction: "x.y.z"},
	}))
	assertReason(t, err, authrpc.ReasonBillingNotConfigured)
}

func TestSubmitPurchaseGoogleGrantsEntitlement(t *testing.T) {
	f := newFixture(t)
	dbl := f.googleDoubles("SUBSCRIPTION_STATE_ACTIVE", f.now.Add(30*24*time.Hour))
	f.setGoogleCreds(dbl.URL, dbl.URL+"/token", "")
	resp, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:                billingv1.Store_STORE_GOOGLE,
		Receipt:              &billingv1.SubmitPurchaseRequest_GooglePurchaseToken{GooglePurchaseToken: "ptok-1"},
		GoogleSubscriptionId: "monthly",
	}))
	if err != nil {
		t.Fatalf("SubmitPurchase google: %v", err)
	}
	if hasEntitlement(resp.Msg.CustomerInfo, "pro") == nil {
		t.Fatalf("expected pro entitlement, got %+v", resp.Msg.CustomerInfo.ActiveEntitlements)
	}
	subs, _ := f.st.ListUserSubscriptions(context.Background(), f.project.ID, f.user.ID)
	if len(subs) != 1 || subs[0].Store != store.SubscriptionStoreGoogle {
		t.Fatalf("google subscription not stored: %+v", subs)
	}
}

func TestGoogleRTDNIdempotentAndFlipsState(t *testing.T) {
	f := newFixture(t)
	// Seed an active subscription for the token via SubmitPurchase first.
	dbl := f.googleDoubles("SUBSCRIPTION_STATE_ACTIVE", f.now.Add(30*24*time.Hour))
	f.setGoogleCreds(dbl.URL, dbl.URL+"/token", "s3cret")
	if _, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:                billingv1.Store_STORE_GOOGLE,
		Receipt:              &billingv1.SubmitPurchaseRequest_GooglePurchaseToken{GooglePurchaseToken: "ptok-2"},
		GoogleSubscriptionId: "monthly",
	})); err != nil {
		t.Fatal(err)
	}
	// Now the store reports the subscription expired; an RTDN nudges a re-read.
	dbl2 := f.googleDoubles("SUBSCRIPTION_STATE_EXPIRED", f.now.Add(-time.Hour))
	f.h.googleBaseURL = dbl2.URL
	f.h.googleTokenURL = dbl2.URL + "/token"

	body := googleRTDNBody(t, 13 /* EXPIRED */, "ptok-2", "msg-1")
	if err := f.h.ProcessGoogleNotification(f.ctx(), f.project, body); err != nil {
		t.Fatalf("process rtdn: %v", err)
	}
	subs, _ := f.st.ListUserSubscriptions(context.Background(), f.project.ID, f.user.ID)
	if len(subs) != 1 || subs[0].Status != store.SubscriptionStatusExpired {
		t.Fatalf("rtdn did not flip to expired: %+v", subs)
	}
	// GetCustomerInfo now shows no entitlement.
	got, _ := f.h.GetCustomerInfo(f.ctx(), authReq(f, &billingv1.GetCustomerInfoRequest{}))
	if hasEntitlement(got.Msg.CustomerInfo, "pro") != nil {
		t.Fatalf("entitlement should be gone after expiry")
	}
	// Exactly one expiry revenue event was emitted by the first delivery.
	assertEventCount(t, f, store.SubscriptionEventExpired, 1)

	// Replaying the same message id is a no-op (idempotent) — and crucially does
	// NOT re-apply: the expiry event count stays at one. A handler that dropped
	// the dedupe guard (or re-ordered record/apply) would double-count here.
	if err := f.h.ProcessGoogleNotification(f.ctx(), f.project, body); err != nil {
		t.Fatalf("replay: %v", err)
	}
	assertEventCount(t, f, store.SubscriptionEventExpired, 1)
	// Only one store_notification recorded (replay deduped): re-insert reports
	// not-new.
	fresh, err := f.st.InsertStoreNotificationIfNew(context.Background(), store.StoreNotification{
		ID: authrpc.NewID(), ProjectID: f.project.ID, Store: store.SubscriptionStoreGoogle,
		NotificationID: "msg-1", CreatedAt: f.now})
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatalf("message id msg-1 should already be recorded (idempotency broken)")
	}
}

// appleNotif builds a signed App Store Server Notification V2 signedPayload
// carrying the given transaction, using the fixture CA.
func (f *fixture) appleNotif(t *testing.T, nType, uuid string, txn map[string]any) string {
	t.Helper()
	payload := map[string]any{
		"notificationType": nType,
		"notificationUUID": uuid,
		"data": map[string]any{
			"bundleId":              testBundleID,
			"environment":           "Production",
			"signedTransactionInfo": f.ca.signJWS(t, txn),
		},
	}
	return f.ca.signJWS(t, payload)
}

func TestAppleNotificationFlipsStateAndIdempotent(t *testing.T) {
	f := newFixture(t)
	txn := appleTxn("orig-n1", "com.demo.monthly", f.now.Add(30*24*time.Hour), 0, 0)
	dbl := f.appleStatusDouble(1, txn)
	f.setAppleCreds(dbl.URL)
	// Seed an active Apple subscription via SubmitPurchase.
	if _, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:   billingv1.Store_STORE_APPLE,
		Receipt: &billingv1.SubmitPurchaseRequest_AppleJwsTransaction{AppleJwsTransaction: f.ca.signJWS(t, txn)},
	})); err != nil {
		t.Fatal(err)
	}
	assertEventCount(t, f, store.SubscriptionEventPurchased, 1)

	// A REFUND notification arrives; the App Store Server API now reports the
	// subscription revoked (status 5). moth re-reads authoritative state, never
	// trusting the notification body.
	refundTxn := appleTxn("orig-n1", "com.demo.monthly", f.now.Add(30*24*time.Hour), 0, int(f.now.UnixMilli()))
	dbl2 := f.appleStatusDouble(5, refundTxn)
	f.h.appleBaseURL = dbl2.URL
	notif := f.appleNotif(t, "REFUND", "anotif-1", refundTxn)
	if err := f.h.ProcessAppleNotification(f.ctx(), f.project, notif); err != nil {
		t.Fatalf("process apple notification: %v", err)
	}
	subs, _ := f.st.ListUserSubscriptions(context.Background(), f.project.ID, f.user.ID)
	if len(subs) != 1 || subs[0].Status != store.SubscriptionStatusRevoked {
		t.Fatalf("refund did not flip subscription to revoked: %+v", subs)
	}
	got, _ := f.h.GetCustomerInfo(f.ctx(), authReq(f, &billingv1.GetCustomerInfoRequest{}))
	if hasEntitlement(got.Msg.CustomerInfo, "pro") != nil {
		t.Fatalf("entitlement should be gone after refund")
	}
	assertEventCount(t, f, store.SubscriptionEventRefunded, 1)

	// Replay is idempotent and does not re-apply / double-count.
	if err := f.h.ProcessAppleNotification(f.ctx(), f.project, notif); err != nil {
		t.Fatalf("replay: %v", err)
	}
	assertEventCount(t, f, store.SubscriptionEventRefunded, 1)
}

func TestGoogleRTDNStateFlipsByType(t *testing.T) {
	// Each RTDN type drives an authoritative re-read; the resolved store state
	// maps onto the subscription status (plan/11: recovered, canceled, on-hold).
	cases := []struct {
		name      string
		notifType int
		state     string
		want      string
	}{
		{"recovered", 1, "SUBSCRIPTION_STATE_ACTIVE", store.SubscriptionStatusActive},
		{"canceled", 3, "SUBSCRIPTION_STATE_CANCELED", store.SubscriptionStatusActive},
		{"onhold", 5, "SUBSCRIPTION_STATE_ON_HOLD", store.SubscriptionStatusInBillingRetry},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			dbl := f.googleDoubles("SUBSCRIPTION_STATE_ACTIVE", f.now.Add(30*24*time.Hour))
			f.setGoogleCreds(dbl.URL, dbl.URL+"/token", "s3cret")
			if _, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
				Store:                billingv1.Store_STORE_GOOGLE,
				Receipt:              &billingv1.SubmitPurchaseRequest_GooglePurchaseToken{GooglePurchaseToken: "ptok-flip"},
				GoogleSubscriptionId: "monthly",
			})); err != nil {
				t.Fatal(err)
			}
			// The RTDN nudges a re-read that now reports the new state.
			dbl2 := f.googleDoubles(c.state, f.now.Add(24*time.Hour))
			f.h.googleBaseURL = dbl2.URL
			f.h.googleTokenURL = dbl2.URL + "/token"
			body := googleRTDNBody(t, c.notifType, "ptok-flip", "msg-"+c.name)
			if err := f.h.ProcessGoogleNotification(f.ctx(), f.project, body); err != nil {
				t.Fatalf("process rtdn: %v", err)
			}
			subs, _ := f.st.ListUserSubscriptions(context.Background(), f.project.ID, f.user.ID)
			if len(subs) != 1 || subs[0].Status != c.want {
				t.Fatalf("rtdn %s: status = %+v, want %q", c.name, subs, c.want)
			}
		})
	}
}

// TestGoogleRTDNTransientFailureRedelivers proves the idempotency fix: a
// transient store-read failure during a mid-period revoke does NOT drop the
// state change. The notification row is left unprocessed, the error propagates
// (so the webhook 503s and the store redelivers), and the redelivery re-drives
// the re-read and applies the flip.
func TestGoogleRTDNTransientFailureRedelivers(t *testing.T) {
	f := newFixture(t)
	dbl := f.googleDoubles("SUBSCRIPTION_STATE_ACTIVE", f.now.Add(30*24*time.Hour))
	f.setGoogleCreds(dbl.URL, dbl.URL+"/token", "s3cret")
	if _, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:                billingv1.Store_STORE_GOOGLE,
		Receipt:              &billingv1.SubmitPurchaseRequest_GooglePurchaseToken{GooglePurchaseToken: "ptok-t"},
		GoogleSubscriptionId: "monthly",
	})); err != nil {
		t.Fatal(err)
	}

	// Point the re-read at a server that 500s (a transient store-API blip); the
	// token endpoint still works so we reach the failing GET.
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 3600})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failing.Close)
	f.h.googleBaseURL = failing.URL
	f.h.googleTokenURL = failing.URL + "/token"

	body := googleRTDNBody(t, 12 /* REVOKED */, "ptok-t", "msg-fail")
	if err := f.h.ProcessGoogleNotification(f.ctx(), f.project, body); err == nil {
		t.Fatal("transient re-read failure must propagate so the webhook can signal a retry")
	}
	// The subscription is still active (the refund was NOT silently dropped).
	subs, _ := f.st.ListUserSubscriptions(context.Background(), f.project.ID, f.user.ID)
	if len(subs) != 1 || subs[0].Status != store.SubscriptionStatusActive {
		t.Fatalf("subscription must be unchanged on transient failure: %+v", subs)
	}
	// The notification row exists but is unprocessed, so a redelivery reprocesses.
	n, err := f.st.GetStoreNotification(context.Background(), f.project.ID, store.SubscriptionStoreGoogle, "msg-fail")
	if err != nil {
		t.Fatalf("notification not recorded: %v", err)
	}
	if n.ProcessedAt != nil {
		t.Fatalf("notification wrongly marked processed after a failed apply")
	}

	// The store redelivers the SAME message id; the re-read now succeeds and
	// reports the subscription expired. The redelivery must re-apply the flip.
	recovered := f.googleDoubles("SUBSCRIPTION_STATE_EXPIRED", f.now.Add(-time.Hour))
	f.h.googleBaseURL = recovered.URL
	f.h.googleTokenURL = recovered.URL + "/token"
	if err := f.h.ProcessGoogleNotification(f.ctx(), f.project, body); err != nil {
		t.Fatalf("redelivery: %v", err)
	}
	subs, _ = f.st.ListUserSubscriptions(context.Background(), f.project.ID, f.user.ID)
	if len(subs) != 1 || subs[0].Status != store.SubscriptionStatusExpired {
		t.Fatalf("redelivery did not apply the dropped state change: %+v", subs)
	}
}

func TestSubmitPurchaseGoogleCrossProjectTokenRejected(t *testing.T) {
	f := newFixture(t)
	// The Play Developer API 404s a purchase_token that resolves under another
	// project's package / service account.
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 3600})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "not found"}})
	}))
	t.Cleanup(notFound.Close)
	f.setGoogleCreds(notFound.URL, notFound.URL+"/token", "")
	_, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:                billingv1.Store_STORE_GOOGLE,
		Receipt:              &billingv1.SubmitPurchaseRequest_GooglePurchaseToken{GooglePurchaseToken: "ptok-foreign"},
		GoogleSubscriptionId: "monthly",
	}))
	assertReason(t, err, authrpc.ReasonInvalidReceipt)
}

// TestSubmitPurchaseTransfersSubscriptionToCaller pins the intended store-mirror
// behaviour: a store transaction previously linked to another user in the same
// project transfers to the caller who now submits it (the store, source of
// truth, has granted the caller's device the purchase), last write wins —
// consistent with RestorePurchases.
func TestSubmitPurchaseTransfersSubscriptionToCaller(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	txn := appleTxn("orig-shared", "com.demo.monthly", f.now.Add(30*24*time.Hour), 0, 0)
	dbl := f.appleStatusDouble(1, txn)
	f.setAppleCreds(dbl.URL)

	// Seed the subscription owned by a DIFFERENT user in the same project.
	userA := store.User{ID: authrpc.NewID(), ProjectID: f.project.ID, Email: "a@demo.test",
		CustomClaims: "{}", CreatedAt: f.now, UpdatedAt: f.now}
	if err := f.st.CreateUser(ctx, userA); err != nil {
		t.Fatal(err)
	}
	end := f.now.Add(30 * 24 * time.Hour)
	if _, err := f.st.UpsertSubscription(ctx, store.Subscription{
		ID: authrpc.NewID(), ProjectID: f.project.ID, UserID: userA.ID, Store: store.SubscriptionStoreApple,
		ProductID: f.prodID, StoreTransactionID: "orig-shared", Status: store.SubscriptionStatusActive,
		CurrentPeriodEnd: &end, Environment: store.SubscriptionEnvironmentProduction,
		CreatedAt: f.now, UpdatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}

	// The fixture user (the caller) submits the same store identity.
	if _, err := f.h.SubmitPurchase(f.ctx(), authReq(f, &billingv1.SubmitPurchaseRequest{
		Store:   billingv1.Store_STORE_APPLE,
		Receipt: &billingv1.SubmitPurchaseRequest_AppleJwsTransaction{AppleJwsTransaction: f.ca.signJWS(t, txn)},
	})); err != nil {
		t.Fatal(err)
	}
	sub, err := f.st.GetSubscriptionByStoreID(ctx, f.project.ID, store.SubscriptionStoreApple, "orig-shared")
	if err != nil {
		t.Fatal(err)
	}
	if sub.UserID != f.user.ID {
		t.Fatalf("SubmitPurchase should transfer the subscription to the caller, got owner %s", sub.UserID)
	}
}

func TestGoogleRTDNRejectedForWrongToken(t *testing.T) {
	f := newFixture(t)
	dbl := f.googleDoubles("SUBSCRIPTION_STATE_ACTIVE", f.now.Add(time.Hour))
	f.setGoogleCreds(dbl.URL, dbl.URL+"/token", "s3cret")
	cred, _ := f.st.GetBillingCredentials(context.Background(), f.project.ID)
	if f.h.AuthenticateGooglePush(cred, "wrong") {
		t.Fatalf("wrong push token accepted")
	}
	if !f.h.AuthenticateGooglePush(cred, "s3cret") {
		t.Fatalf("correct push token rejected")
	}
}

func TestReconcileRereadsLapsedSubscription(t *testing.T) {
	f := newFixture(t)
	// Store an active subscription whose period already ended (a missed webhook).
	end := f.now.Add(-time.Hour)
	if _, err := f.st.UpsertSubscription(context.Background(), store.Subscription{
		ID: authrpc.NewID(), ProjectID: f.project.ID, UserID: f.user.ID,
		Store: store.SubscriptionStoreGoogle, ProductID: f.prodID, StoreTransactionID: "ptok-r",
		SubscriptionID: "monthly", Status: store.SubscriptionStatusActive, CurrentPeriodEnd: &end,
		Environment: store.SubscriptionEnvironmentProduction, CreatedAt: f.now, UpdatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
	dbl := f.googleDoubles("SUBSCRIPTION_STATE_EXPIRED", f.now.Add(-time.Hour))
	f.setGoogleCreds(dbl.URL, dbl.URL+"/token", "")

	if err := f.h.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	subs, _ := f.st.ListUserSubscriptions(context.Background(), f.project.ID, f.user.ID)
	if len(subs) != 1 || subs[0].Status != store.SubscriptionStatusExpired {
		t.Fatalf("reconcile did not update lapsed subscription: %+v", subs)
	}
}

// --- helpers --------------------------------------------------------------

func assertEventCount(t *testing.T, f *fixture, eventType string, want int) {
	t.Helper()
	var got int
	rows, err := storeRawEvents(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, ty := range rows {
		if ty == eventType {
			got++
		}
	}
	if got != want {
		t.Fatalf("subscription_event %q count = %d, want %d", eventType, got, want)
	}
}

func assertReason(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with reason %q, got nil", want)
	}
	if got := authrpc.ErrorReason(err); got != want {
		t.Fatalf("reason = %q, want %q (err %v)", got, want, err)
	}
}

func googleRTDNBody(t *testing.T, notifType int, purchaseToken, messageID string) []byte {
	t.Helper()
	dn := map[string]any{
		"version": "1.0", "packageName": testBundleID,
		"subscriptionNotification": map[string]any{
			"version": "1.0", "notificationType": notifType,
			"purchaseToken": purchaseToken, "subscriptionId": "monthly",
		},
	}
	raw, _ := json.Marshal(dn)
	env := map[string]any{"message": map[string]any{
		"data": base64.StdEncoding.EncodeToString(raw), "messageId": messageID,
	}}
	body, _ := json.Marshal(env)
	return body
}
