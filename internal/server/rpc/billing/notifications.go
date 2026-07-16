package billingrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aloisdeniel/moth/internal/billing"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// Credentials returns a project's billing credentials (or store.ErrNotFound),
// exposed so the plain-HTTP webhook can authenticate a push before handing the
// body to Process*.
func (h *Handler) Credentials(ctx context.Context, projectID string) (store.BillingCredentials, error) {
	return h.store.GetBillingCredentials(ctx, projectID)
}

// AuthenticateGooglePush verifies the shared-secret token guarding the RTDN
// endpoint against the project's stored (decrypted) RTDN secret, in constant
// time.
func (h *Handler) AuthenticateGooglePush(cred store.BillingCredentials, token string) bool {
	if len(cred.GoogleRTDNSecretEnc) == 0 {
		return false
	}
	secret, err := h.master.Decrypt(cred.GoogleRTDNSecretEnc)
	if err != nil {
		h.log.Error("decrypt google rtdn secret", "error", err.Error())
		return false
	}
	return billing.AuthenticatePushToken(token, string(secret))
}

// ProcessAppleNotification handles an App Store Server Notification V2: it
// verifies the signedPayload JWS chain, dedupes on the notification id, then
// re-reads authoritative state from the App Store Server API before touching
// the subscription — the notification body is a nudge, never trusted for state.
// A replayed notification is a no-op.
func (h *Handler) ProcessAppleNotification(ctx context.Context, project store.Project, signedPayload string) error {
	cred, err := h.store.GetBillingCredentials(ctx, project.ID)
	if err != nil {
		return err
	}
	notif, err := h.appleVerifier(cred).VerifyNotification(signedPayload)
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(map[string]string{"signedPayload": signedPayload})
	id, done, err := h.claimNotification(ctx, project.ID, store.SubscriptionStoreApple, notif.NotificationID, notif.Type, notif.Subtype, raw)
	if err != nil {
		return err
	}
	if done {
		return nil // already applied: idempotent replay
	}

	storeTxnID := notif.Subscription.StoreTransactionID
	if storeTxnID == "" {
		// A test notification or a body without a transaction: nothing to
		// re-read, but it has been recorded.
		return h.markProcessed(ctx, project.ID, id)
	}
	client, err := h.appleClient(cred)
	if err != nil {
		return err
	}
	norm, err := client.GetAllSubscriptionStatuses(ctx, storeTxnID)
	if err != nil {
		return err
	}
	if err := h.applyFromNotification(ctx, project.ID, norm, appleEventType(notif.Type)); err != nil {
		return err
	}
	return h.markProcessed(ctx, project.ID, id)
}

// ProcessGoogleNotification handles a Play RTDN Pub/Sub push: it parses the
// envelope, dedupes on the Pub/Sub message id, then re-reads authoritative
// state via the Play Developer API before touching the subscription. The push
// must already be authenticated (see AuthenticateGooglePush).
func (h *Handler) ProcessGoogleNotification(ctx context.Context, project store.Project, body []byte) error {
	cred, err := h.store.GetBillingCredentials(ctx, project.ID)
	if err != nil {
		return err
	}
	dn, messageID, err := billing.ParsePubSubPush(body)
	if err != nil {
		return err
	}
	notifType, subtype := "", ""
	var purchaseToken string
	if dn.SubscriptionNotification != nil {
		notifType = fmt.Sprintf("%d", dn.SubscriptionNotification.NotificationType)
		purchaseToken = dn.SubscriptionNotification.PurchaseToken
	} else if dn.TestNotification != nil {
		notifType = "test"
	}
	id, done, err := h.claimNotification(ctx, project.ID, store.SubscriptionStoreGoogle, messageID, notifType, subtype, body)
	if err != nil {
		return err
	}
	if done {
		return nil // already applied: idempotent replay
	}
	if purchaseToken == "" {
		return h.markProcessed(ctx, project.ID, id)
	}
	client, err := h.googleClient(cred)
	if err != nil {
		return err
	}
	norm, _, err := client.GetSubscriptionV2(ctx, purchaseToken)
	if err != nil {
		return err
	}
	if err := h.applyFromNotification(ctx, project.ID, norm, googleEventType(dn.SubscriptionNotification)); err != nil {
		return err
	}
	return h.markProcessed(ctx, project.ID, id)
}

// claimNotification records the raw notification (for idempotency + audit) and
// reports the row id to stamp once applied, plus whether it has ALREADY been
// applied (processed_at set). Crucially, an existing but still-UNPROCESSED row —
// left behind by an earlier transient failure between recording and applying —
// is reported as not-yet-done, so a store redelivery re-drives the authoritative
// re-read instead of being silently swallowed as a replay. Dedupe is therefore
// keyed on "successfully applied", not on mere existence, which is what makes a
// dropped refund/revoke self-heal on retry.
func (h *Handler) claimNotification(ctx context.Context, projectID, storeName, notificationID, notifType, subtype string, raw []byte) (id string, done bool, err error) {
	if notificationID == "" {
		return "", false, fmt.Errorf("%w: notification without id", billing.ErrMalformed)
	}
	newID := authrpc.NewID()
	fresh, err := h.store.InsertStoreNotificationIfNew(ctx, store.StoreNotification{
		ID:             newID,
		ProjectID:      projectID,
		Store:          storeName,
		NotificationID: notificationID,
		Type:           notifType,
		Subtype:        subtype,
		RawPayload:     string(raw),
		CreatedAt:      h.now(),
	})
	if err != nil {
		return "", false, err
	}
	if fresh {
		return newID, false, nil
	}
	// The notification id already exists: reprocess only if it was never applied.
	existing, err := h.store.GetStoreNotification(ctx, projectID, storeName, notificationID)
	if err != nil {
		return "", false, err
	}
	return existing.ID, existing.ProcessedAt != nil, nil
}

// markProcessed stamps processed_at on the recorded notification row, sealing
// the idempotency guard. A failure is propagated: the webhook then returns a
// retryable status so the store redelivers and the (idempotent) re-read+apply
// runs again, rather than leaving the row wrongly marked done.
func (h *Handler) markProcessed(ctx context.Context, projectID, id string) error {
	if err := h.store.MarkStoreNotificationProcessed(ctx, projectID, id, h.now()); err != nil {
		return fmt.Errorf("mark store notification processed: %w", err)
	}
	return nil
}

// applyFromNotification re-links an authoritative store read to the user who
// already owns the subscription. A notification for a subscription moth has
// never seen (no SubmitPurchase yet) cannot be attributed to a user, so it is
// recorded and skipped; the next client SubmitPurchase/RestorePurchases will
// create it.
func (h *Handler) applyFromNotification(ctx context.Context, projectID string, norm billing.NormalizedSubscription, eventType string) error {
	existing, err := h.store.GetSubscriptionByStoreID(ctx, projectID, norm.Store, norm.StoreTransactionID)
	if errors.Is(err, store.ErrNotFound) {
		h.log.InfoContext(ctx, "notification for unknown subscription; recorded, not applied",
			"store", norm.Store, "transaction", norm.StoreTransactionID)
		return nil
	}
	if err != nil {
		return err
	}
	stored, err := h.applyNormalized(ctx, projectID, existing.UserID, norm, false)
	if err != nil {
		return err
	}
	// The subscription already existed (SubmitPurchase created it and already
	// emitted its purchase/trial event). A SUBSCRIBED / OFFER_REDEEMED /
	// PURCHASED notification for that same subscription must not re-emit an
	// acquisition event, or the milestone-14 revenue stream double-counts the
	// purchase. Only genuine post-acquisition transitions (renewal, expiry,
	// refund, revoke, cancel) are emitted from the notification path.
	if eventType != "" && !isAcquisitionEvent(eventType) {
		products, _ := h.store.ListProducts(ctx, projectID)
		_, product := matchProduct(products, stored.Store, norm.ProductID)
		h.emitEvent(ctx, projectID, stored, product, eventType)
	}
	return nil
}

// isAcquisitionEvent reports whether an event type marks an initial purchase or
// trial start — events owned by SubmitPurchase, never re-emitted from a webhook.
func isAcquisitionEvent(eventType string) bool {
	return eventType == store.SubscriptionEventPurchased ||
		eventType == store.SubscriptionEventTrialStarted
}

// appleEventType maps an ASSN V2 notificationType to a revenue event type.
func appleEventType(nType string) string {
	switch nType {
	case "SUBSCRIBED", "OFFER_REDEEMED":
		return store.SubscriptionEventPurchased
	case "DID_RENEW", "RENEWAL_EXTENDED":
		return store.SubscriptionEventRenewed
	case "EXPIRED", "GRACE_PERIOD_EXPIRED":
		return store.SubscriptionEventExpired
	case "REFUND":
		return store.SubscriptionEventRefunded
	case "REVOKE":
		return store.SubscriptionEventRevoked
	case "DID_CHANGE_RENEWAL_STATUS":
		return store.SubscriptionEventCanceled
	default:
		return ""
	}
}

// googleEventType maps an RTDN notificationType to a revenue event type.
func googleEventType(n *struct {
	Version          string `json:"version"`
	NotificationType int    `json:"notificationType"`
	PurchaseToken    string `json:"purchaseToken"`
	SubscriptionID   string `json:"subscriptionId"`
}) string {
	if n == nil {
		return ""
	}
	switch n.NotificationType {
	case billing.GoogleNotifPurchased:
		return store.SubscriptionEventPurchased
	case billing.GoogleNotifRenewed, billing.GoogleNotifRecovered, billing.GoogleNotifRestarted:
		return store.SubscriptionEventRenewed
	case billing.GoogleNotifCanceled:
		return store.SubscriptionEventCanceled
	case billing.GoogleNotifExpired:
		return store.SubscriptionEventExpired
	case billing.GoogleNotifRevoked:
		return store.SubscriptionEventRefunded
	default:
		return ""
	}
}
