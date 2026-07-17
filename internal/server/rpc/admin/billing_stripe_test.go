package adminrpc

import (
	"bytes"
	"context"
	"testing"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

func TestUpdateBillingCredentialsStripe(t *testing.T) {
	h, st, master, project := newBillingTestHandler(t)
	ctx := context.Background()

	resp, err := h.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
		ProjectId: project.ID,
		Stripe: &adminv1.StripeBillingConfig{
			SecretKey:         "rk_test_abc123",
			WebhookSecret:     "whsec_shh",
			WebhookEndpointId: "we_1",
		},
	}))
	if err != nil {
		t.Fatalf("UpdateBillingCredentials: %v", err)
	}
	// The response echoes no secrets, only has_* indicators + the endpoint id.
	sc := resp.Msg.Stripe
	if !sc.GetHasSecretKey() || !sc.GetHasWebhookSecret() || sc.GetWebhookEndpointId() != "we_1" {
		t.Fatalf("stripe config = %+v", sc)
	}
	if sc.GetSecretKey() != "" || sc.GetWebhookSecret() != "" {
		t.Fatalf("secrets echoed back: %+v", sc)
	}

	// At rest: ciphertext only, round-tripping through the master key.
	cred, err := st.GetBillingCredentials(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(cred.StripeSecretKeyEnc, []byte("rk_test_")) ||
		bytes.Contains(cred.StripeWebhookSecretEnc, []byte("whsec_")) {
		t.Fatal("stripe secrets stored in cleartext")
	}
	if dec, err := master.Decrypt(cred.StripeSecretKeyEnc); err != nil || string(dec) != "rk_test_abc123" {
		t.Fatalf("secret key does not round-trip: %v", err)
	}
	if dec, err := master.Decrypt(cred.StripeWebhookSecretEnc); err != nil || string(dec) != "whsec_shh" {
		t.Fatalf("webhook secret does not round-trip: %v", err)
	}
	if cred.StripeWebhookEndpointID != "we_1" {
		t.Fatalf("endpoint id = %q", cred.StripeWebhookEndpointID)
	}

	// Empty secrets + endpoint id on a later update keep the stored values
	// (write-only semantics, endpoint id mirrors AppleNotificationURL).
	resp2, err := h.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
		ProjectId: project.ID,
		Stripe:    &adminv1.StripeBillingConfig{},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp2.Msg.Stripe.GetHasSecretKey() || !resp2.Msg.Stripe.GetHasWebhookSecret() ||
		resp2.Msg.Stripe.GetWebhookEndpointId() != "we_1" {
		t.Fatalf("empty stripe block should keep stored values: %+v", resp2.Msg.Stripe)
	}
	cred2, _ := st.GetBillingCredentials(ctx, project.ID)
	if !bytes.Equal(cred2.StripeSecretKeyEnc, cred.StripeSecretKeyEnc) ||
		!bytes.Equal(cred2.StripeWebhookSecretEnc, cred.StripeWebhookSecretEnc) {
		t.Fatal("empty secrets should keep the stored ciphertext")
	}

	// GetBillingCredentials reports the stripe block too.
	got, err := h.GetBillingCredentials(ctx, connect.NewRequest(&adminv1.GetBillingCredentialsRequest{ProjectId: project.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Msg.Stripe.GetHasSecretKey() || got.Msg.Stripe.GetWebhookEndpointId() != "we_1" {
		t.Fatalf("get stripe config = %+v", got.Msg.Stripe)
	}
}

// TestUpdateBillingCredentialsPartialUpdateKeepsOtherStores: the request is
// sectioned per store and the store upsert writes full rows, so the handler
// must merge with the stored row — a Stripe-only update (exactly what
// `moth setup billing --stripe-secret-key` sends) must not blank the
// Apple/Google configuration and silently break their webhook processing.
func TestUpdateBillingCredentialsPartialUpdateKeepsOtherStores(t *testing.T) {
	h, st, _, project := newBillingTestHandler(t)
	ctx := context.Background()
	p8 := testP8(t)
	sa := testSA(t)

	// Configure Apple + Google fully (including the CLI-recorded notification
	// URL, which persistNotificationURL writes via an Apple-only request).
	if _, err := h.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
		ProjectId: project.ID,
		Apple: &adminv1.AppleBillingConfig{IapKeyId: "K1", IapIssuerId: "I1", IapKeyP8: p8,
			BundleId: "com.demo.app", AppAppleId: "1234567890", NotificationSecret: "asn-secret",
			NotificationUrl: "https://moth.example.com/billing/apple/notifications/demo"},
		Google: &adminv1.GoogleBillingConfig{ServiceAccountJson: sa, PackageName: "com.demo.app",
			PubsubTopic: "projects/p/topics/moth-rtdn", RtdnSecret: "rtdn-secret"},
	})); err != nil {
		t.Fatalf("UpdateBillingCredentials apple+google: %v", err)
	}
	before, err := st.GetBillingCredentials(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Stripe-only update, as storeCredentials AND wireStripeWebhook send it.
	resp, err := h.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
		ProjectId: project.ID,
		Stripe:    &adminv1.StripeBillingConfig{SecretKey: "sk_test_abc", WebhookSecret: "whsec_s", WebhookEndpointId: "we_1"},
	}))
	if err != nil {
		t.Fatalf("UpdateBillingCredentials stripe-only: %v", err)
	}
	if !resp.Msg.Stripe.GetHasSecretKey() || !resp.Msg.Apple.GetHasIapKey() || !resp.Msg.Google.GetHasServiceAccount() {
		t.Fatalf("response lost sections: %+v", resp.Msg)
	}

	after, err := st.GetBillingCredentials(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Every Apple field survived.
	if after.AppleIAPKeyID != "K1" || after.AppleIAPIssuerID != "I1" || after.AppleBundleID != "com.demo.app" ||
		after.AppleAppAppleID != "1234567890" ||
		after.AppleNotificationURL != "https://moth.example.com/billing/apple/notifications/demo" {
		t.Fatalf("apple fields wiped by a stripe-only update: %+v", after)
	}
	if !bytes.Equal(after.AppleIAPKeyEnc, before.AppleIAPKeyEnc) ||
		!bytes.Equal(after.AppleNotificationSecretEnc, before.AppleNotificationSecretEnc) {
		t.Fatal("apple secrets wiped by a stripe-only update")
	}
	// Every Google field survived.
	if after.GooglePackageName != "com.demo.app" || after.GooglePubsubTopic != "projects/p/topics/moth-rtdn" {
		t.Fatalf("google fields wiped by a stripe-only update: %+v", after)
	}
	if !bytes.Equal(after.GoogleServiceAccountEnc, before.GoogleServiceAccountEnc) ||
		!bytes.Equal(after.GoogleRTDNSecretEnc, before.GoogleRTDNSecretEnc) {
		t.Fatal("google secrets wiped by a stripe-only update")
	}
	// And the Stripe section landed.
	if len(after.StripeSecretKeyEnc) == 0 || len(after.StripeWebhookSecretEnc) == 0 || after.StripeWebhookEndpointID != "we_1" {
		t.Fatalf("stripe section not stored: %+v", after)
	}

	// The reverse direction too: an Apple-only update (persistNotificationURL's
	// shape — ids + URL, no secrets) keeps Google and Stripe intact.
	if _, err := h.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
		ProjectId: project.ID,
		Apple: &adminv1.AppleBillingConfig{IapKeyId: "K1", IapIssuerId: "I1",
			BundleId: "com.demo.app", AppAppleId: "1234567890",
			NotificationUrl: "https://moth.example.com/billing/apple/notifications/demo"},
	})); err != nil {
		t.Fatalf("UpdateBillingCredentials apple-only: %v", err)
	}
	final, err := st.GetBillingCredentials(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.GooglePackageName != "com.demo.app" || len(final.GoogleServiceAccountEnc) == 0 {
		t.Fatalf("google wiped by an apple-only update: %+v", final)
	}
	if len(final.StripeSecretKeyEnc) == 0 || final.StripeWebhookEndpointID != "we_1" {
		t.Fatalf("stripe wiped by an apple-only update: %+v", final)
	}
	if !bytes.Equal(final.AppleIAPKeyEnc, before.AppleIAPKeyEnc) {
		t.Fatal("apple key lost on an apple-only update without a p8")
	}
}

func TestUpdateBillingCredentialsStripeRejectsBadKeys(t *testing.T) {
	h, _, _, project := newBillingTestHandler(t)
	ctx := context.Background()

	// Publishable keys (and anything not sk_/rk_) cannot call server APIs.
	for _, key := range []string{"pk_test_x", "sk_x", "whsec_x", "sk_test_", "secret"} {
		_, err := h.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
			ProjectId: project.ID,
			Stripe:    &adminv1.StripeBillingConfig{SecretKey: key},
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("key %q: want InvalidArgument, got %v", key, err)
		}
	}
	// A webhook secret must be a whsec_ value.
	_, err := h.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
		ProjectId: project.ID,
		Stripe:    &adminv1.StripeBillingConfig{WebhookSecret: "not-a-secret"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument for bad webhook secret, got %v", err)
	}
	// Valid shapes are accepted.
	for _, key := range []string{"sk_test_ok", "sk_live_ok", "rk_test_ok", "rk_live_ok"} {
		if _, err := h.UpdateBillingCredentials(ctx, connect.NewRequest(&adminv1.UpdateBillingCredentialsRequest{
			ProjectId: project.ID,
			Stripe:    &adminv1.StripeBillingConfig{SecretKey: key},
		})); err != nil {
			t.Fatalf("key %q rejected: %v", key, err)
		}
	}
}

func TestProductProtoThreadsStripeIDs(t *testing.T) {
	h, _, _, project := newBillingTestHandler(t)
	ctx := context.Background()

	created, err := h.CreateProduct(ctx, connect.NewRequest(&adminv1.CreateProductRequest{
		ProjectId: project.ID,
		Product: &adminv1.Product{
			Identifier: "monthly", DisplayName: "Pro", BillingPeriod: "monthly",
			PriceAmountMicros: 9_990_000, Currency: "USD",
			StripePriceId: "price_1", StripeProductId: "prod_1",
		},
	}))
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	p := created.Msg.Product
	if p.GetStripePriceId() != "price_1" || p.GetStripeProductId() != "prod_1" {
		t.Fatalf("stripe ids not threaded: %+v", p)
	}
}
