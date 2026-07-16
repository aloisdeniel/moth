package billingrpc

import (
	"context"

	"github.com/aloisdeniel/moth/internal/billing"
	"github.com/aloisdeniel/moth/internal/store"
)

// reconcileBatch caps how many subscriptions one sweep re-reads, keeping the
// job cheap on a large instance; the next tick picks up the rest.
const reconcileBatch = 100

// Reconcile re-reads store subscriptions whose paid period has lapsed while
// they are still marked access-granting — the tell-tale of a store notification
// moth missed. It fetches authoritative state from the store and upserts it, so
// a silent renewal or a silent expiry is corrected without a webhook. Best
// effort: per-subscription failures are logged and skipped so one bad row never
// stalls the sweep.
func (h *Handler) Reconcile(ctx context.Context) error {
	now := h.now()
	subs, err := h.store.ListSubscriptionsForReconciliation(ctx, now, reconcileBatch)
	if err != nil {
		return err
	}
	// One credentials lookup per project across the batch.
	credByProject := make(map[string]store.BillingCredentials)
	for _, sub := range subs {
		cred, ok := credByProject[sub.ProjectID]
		if !ok {
			cred, err = h.store.GetBillingCredentials(ctx, sub.ProjectID)
			if err != nil {
				h.log.WarnContext(ctx, "reconcile: no billing credentials", "project", sub.ProjectID)
				credByProject[sub.ProjectID] = store.BillingCredentials{} // memoize the miss
				continue
			}
			credByProject[sub.ProjectID] = cred
		}
		if cred.ProjectID == "" {
			continue // memoized miss
		}
		if err := h.reconcileOne(ctx, sub, cred); err != nil {
			h.log.WarnContext(ctx, "reconcile: subscription re-read failed",
				"project", sub.ProjectID, "subscription", sub.ID, "error", err.Error())
		}
	}
	return nil
}

func (h *Handler) reconcileOne(ctx context.Context, sub store.Subscription, cred store.BillingCredentials) error {
	var norm billing.NormalizedSubscription
	var err error
	switch sub.Store {
	case store.SubscriptionStoreApple:
		var client *billing.AppleClient
		if client, err = h.appleClient(cred); err != nil {
			return err
		}
		norm, err = client.GetAllSubscriptionStatuses(ctx, sub.StoreTransactionID)
	case store.SubscriptionStoreGoogle:
		var client *billing.GoogleClient
		if client, err = h.googleClient(cred); err != nil {
			return err
		}
		norm, _, err = client.GetSubscriptionV2(ctx, sub.StoreTransactionID)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	_, err = h.applyNormalized(ctx, sub.ProjectID, sub.UserID, norm, false)
	return err
}
