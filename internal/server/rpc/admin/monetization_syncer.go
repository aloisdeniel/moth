package adminrpc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/aloisdeniel/moth/internal/billing"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/setup"
	"github.com/aloisdeniel/moth/internal/store"
)

// NewLiveStoreSyncer builds the production storeSyncer for server.New. The
// returned value drives the real Android Publisher client for Google and emits a
// guided CLI step for Apple (the ASC catalog key is never persisted).
func NewLiveStoreSyncer(master keys.MasterKey) LiveStoreSyncer {
	return LiveStoreSyncer{master: master, httpc: &http.Client{}}
}

// LiveStoreSyncer is the production storeSyncer wired by server.New. It drives
// the real store-catalog clients from moth's encrypted billing credentials.
//
// Apple: the App Store Connect *catalog* API key is never persisted by moth, so
// this cannot create Apple subscriptions — it returns a guided ManualStep
// pointing at `moth setup billing` (which supplies the ASC key in-process). This
// is the honest boundary from the capability spike, not a silent skip.
//
// Google: the service account is stored, so the Android Publisher subscription
// push runs here. The RTDN Pub/Sub topic needs a separate pubsub-scoped
// credential moth does not hold, so topic/subscription wiring degrades to a
// guided notification result.
type LiveStoreSyncer struct {
	master keys.MasterKey
	httpc  billing.Doer
}

func (s LiveStoreSyncer) Sync(ctx context.Context, storeName, slug, baseURL string, creds store.BillingCredentials, cat setup.DesiredCatalog) (*setup.SyncResult, error) {
	switch storeName {
	case store.SubscriptionStoreApple:
		return s.syncApple(slug), nil
	case store.SubscriptionStoreGoogle:
		return s.syncGoogle(ctx, slug, baseURL, creds, cat)
	default:
		return nil, fmt.Errorf("unknown store %q", storeName)
	}
}

func (s LiveStoreSyncer) syncApple(slug string) *setup.SyncResult {
	res := &setup.SyncResult{Store: store.SubscriptionStoreApple}
	res.ManualSteps = append(res.ManualSteps, setup.ManualStep{
		Title:  "Push the Apple catalog with the CLI",
		Reason: "moth never persists the App Store Connect catalog API key; the admin cannot create Apple subscriptions itself.",
		URL:    "https://appstoreconnect.apple.com",
		Instructions: []string{
			"moth setup billing --project " + slug + " --asc-p8 <path> --asc-key-id <id> --asc-issuer-id <id> --apple-app-id <id>",
			"Then submit the new subscriptions for review in App Store Connect (no API).",
		},
	})
	return res
}

func (s LiveStoreSyncer) syncGoogle(ctx context.Context, slug, baseURL string, creds store.BillingCredentials, cat setup.DesiredCatalog) (*setup.SyncResult, error) {
	saJSON, err := s.master.Decrypt(creds.GoogleServiceAccountEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt google service account: %w", err)
	}
	sa, err := billing.ParseServiceAccount(saJSON)
	if err != nil {
		return nil, fmt.Errorf("parse google service account: %w", err)
	}
	tokens := billing.NewGoogleTokenSource(sa, "", s.httpc, nil)
	gc := &setup.GoogleCatalog{PackageName: creds.GooglePackageName, Tokens: tokens, HTTPC: s.httpc}
	res, err := gc.Sync(ctx, cat)
	if err != nil {
		return nil, err
	}
	// RTDN topic creation needs a pubsub-scoped credential moth does not hold;
	// degrade to a guided step (WireRTDN with a nil PubSub emits guided-only).
	if creds.GooglePubsubTopic != "" {
		topicID := creds.GooglePubsubTopic
		if i := strings.LastIndex(topicID, "/"); i >= 0 {
			topicID = topicID[i+1:]
		}
		endpoint := s.rtdnEndpoint(slug, baseURL, creds)
		_ = setup.WireRTDN(ctx, nil, topicID, "moth-"+slug+"-rtdn", endpoint, res)
	}
	return res, nil
}

// rtdnEndpoint builds the RTDN push endpoint including the shared-secret
// ?token= query the receiver authenticates every push against. The token is
// decrypted from the stored RTDN secret; without it Google's tokenless
// deliveries are rejected 401 (see internal/server/billing.go handleGoogleRTDN).
func (s LiveStoreSyncer) rtdnEndpoint(slug, baseURL string, creds store.BillingCredentials) string {
	endpoint := strings.TrimSuffix(baseURL, "/") + "/billing/google/rtdn/" + slug
	if len(creds.GoogleRTDNSecretEnc) > 0 {
		if secret, err := s.master.Decrypt(creds.GoogleRTDNSecretEnc); err == nil && len(secret) > 0 {
			endpoint += "?token=" + url.QueryEscape(string(secret))
		}
	}
	return endpoint
}
