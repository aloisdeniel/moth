package billingrpc

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	billingv1 "github.com/aloisdeniel/moth/gen/moth/billing/v1"
	"github.com/aloisdeniel/moth/internal/billing"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// SubmitPurchase validates a receipt the app just obtained from a native
// purchase, links the resulting subscription to the current user, and returns
// fresh CustomerInfo. moth reads authoritative state from the store — it never
// trusts the client's claim of what was bought.
func (h *Handler) SubmitPurchase(ctx context.Context, req *connect.Request[billingv1.SubmitPurchaseRequest]) (*connect.Response[billingv1.SubmitPurchaseResponse], error) {
	project, user, err := h.auth.AuthenticateUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	cred, err := h.store.GetBillingCredentials(ctx, project.ID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, errBillingNotConfigured()
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var norm billing.NormalizedSubscription
	switch req.Msg.Store {
	case billingv1.Store_STORE_APPLE:
		norm, err = h.validateApple(ctx, cred, req.Msg.GetAppleJwsTransaction())
	case billingv1.Store_STORE_GOOGLE:
		norm, err = h.validateGoogle(ctx, cred, req.Msg.GetGooglePurchaseToken(), req.Msg.GetGoogleSubscriptionId())
	default:
		return nil, authrpc.NewError(connect.CodeInvalidArgument, authrpc.ReasonInvalidReceipt, "unknown or unspecified store")
	}
	if err != nil {
		return nil, purchaseError(err)
	}

	if _, err := h.applyNormalized(ctx, project.ID, user.ID, norm, true); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	info, err := h.customerInfo(ctx, project.ID, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&billingv1.SubmitPurchaseResponse{CustomerInfo: info}), nil
}

// RestorePurchases re-links a user's existing store purchases to the current
// account (new device, reinstall, account change). Each receipt is re-read from
// the store and upserted with the current user id; because subscriptions are
// keyed on store identity, a transaction previously owned by another user in
// the project is transferred to the caller — moth follows the store's own
// transfer outcome (the store having granted the caller's device the purchase),
// last write wins.
func (h *Handler) RestorePurchases(ctx context.Context, req *connect.Request[billingv1.RestorePurchasesRequest]) (*connect.Response[billingv1.RestorePurchasesResponse], error) {
	project, user, err := h.auth.AuthenticateUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	cred, err := h.store.GetBillingCredentials(ctx, project.ID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, errBillingNotConfigured()
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var firstErr error
	for _, receipt := range req.Msg.Receipts {
		if receipt == "" {
			continue
		}
		var norm billing.NormalizedSubscription
		switch req.Msg.Store {
		case billingv1.Store_STORE_APPLE:
			norm, err = h.validateApple(ctx, cred, receipt)
		case billingv1.Store_STORE_GOOGLE:
			norm, err = h.validateGoogle(ctx, cred, receipt, "")
		default:
			return nil, authrpc.NewError(connect.CodeInvalidArgument, authrpc.ReasonInvalidReceipt, "unknown or unspecified store")
		}
		if err != nil {
			// A single bad receipt does not fail the whole restore; record it
			// and continue re-linking the rest.
			h.log.WarnContext(ctx, "restore: receipt rejected", "store", req.Msg.Store.String(), "error", err.Error())
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if _, err := h.applyNormalized(ctx, project.ID, user.ID, norm, false); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	info, err := h.customerInfo(ctx, project.ID, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// If nothing restored and every receipt was rejected, surface the reason.
	if len(info.Subscriptions) == 0 && firstErr != nil {
		return nil, purchaseError(firstErr)
	}
	return connect.NewResponse(&billingv1.RestorePurchasesResponse{CustomerInfo: info}), nil
}

// validateApple verifies a StoreKit 2 signed transaction locally (rejecting a
// tampered signature or a mismatched bundle) and then reads authoritative
// renewal state from the App Store Server API.
func (h *Handler) validateApple(ctx context.Context, cred store.BillingCredentials, jws string) (billing.NormalizedSubscription, error) {
	if jws == "" {
		return billing.NormalizedSubscription{}, billing.ErrMalformed
	}
	txn, err := h.appleVerifier(cred).VerifyTransaction(jws)
	if err != nil {
		return billing.NormalizedSubscription{}, err
	}
	client, err := h.appleClient(cred)
	if err != nil {
		return billing.NormalizedSubscription{}, err
	}
	return client.GetAllSubscriptionStatuses(ctx, txn.OriginalTransactionID)
}

// validateGoogle reads authoritative state for a purchase token and, when the
// purchase is awaiting acknowledgement, acknowledges it (Google auto-refunds an
// un-acknowledged purchase after three days).
func (h *Handler) validateGoogle(ctx context.Context, cred store.BillingCredentials, purchaseToken, subscriptionID string) (billing.NormalizedSubscription, error) {
	if purchaseToken == "" {
		return billing.NormalizedSubscription{}, billing.ErrMalformed
	}
	client, err := h.googleClient(cred)
	if err != nil {
		return billing.NormalizedSubscription{}, err
	}
	norm, raw, err := client.GetSubscriptionV2(ctx, purchaseToken)
	if err != nil {
		return billing.NormalizedSubscription{}, err
	}
	if raw.AcknowledgementState == "acknowledgementStatePending" && subscriptionID != "" {
		if ackErr := client.AcknowledgeSubscription(ctx, subscriptionID, purchaseToken); ackErr != nil {
			h.log.WarnContext(ctx, "google: acknowledge failed", "error", ackErr.Error())
		}
	}
	return norm, nil
}

// applyNormalized maps a normalized store subscription onto a stored row keyed
// on store identity, upserting it against the given user, and emits a revenue
// event on first insert. It returns the stored subscription. emitEvent controls
// whether a purchase/trial event is written (webhooks and restores emit their
// own or none).
func (h *Handler) applyNormalized(ctx context.Context, projectID, userID string, norm billing.NormalizedSubscription, emitEvent bool) (store.Subscription, error) {
	products, err := h.store.ListProducts(ctx, projectID)
	if err != nil {
		return store.Subscription{}, err
	}
	productID, product := matchProduct(products, norm.Store, norm.ProductID)

	existing, err := h.store.GetSubscriptionByStoreID(ctx, projectID, norm.Store, norm.StoreTransactionID)
	wasNew := errors.Is(err, store.ErrNotFound)
	if err != nil && !wasNew {
		return store.Subscription{}, err
	}

	now := h.now()
	sub := store.Subscription{
		ID:                 authrpc.NewID(),
		ProjectID:          projectID,
		UserID:             userID,
		Store:              norm.Store,
		ProductID:          productID,
		StoreTransactionID: norm.StoreTransactionID,
		SubscriptionID:     norm.SubscriptionID,
		Status:             norm.Status,
		AutoRenew:          norm.AutoRenew,
		Environment:        norm.Environment,
		RawState:           string(norm.RawState),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if !norm.CurrentPeriodEnd.IsZero() {
		end := norm.CurrentPeriodEnd
		sub.CurrentPeriodEnd = &end
	}
	if !wasNew {
		sub.ID = existing.ID
		sub.CreatedAt = existing.CreatedAt
	}
	stored, err := h.store.UpsertSubscription(ctx, sub)
	if err != nil {
		return store.Subscription{}, err
	}
	if emitEvent && wasNew {
		price, currency := eventPrice(norm, product)
		h.emitEvent(ctx, projectID, stored, price, currency, subscriptionEventType(norm.Status))
	}
	// Trial-to-paid conversion: an existing trialing subscription that the store
	// now reports as active. This is what feeds the trial-to-paid dashboard —
	// no store webhook maps to "converted", so moth derives it from the status
	// transition. The status flip is persisted by UpsertSubscription above, so
	// whichever path (SubmitPurchase, restore or webhook) observes the flip
	// first emits exactly once; later observers see active→active and do not.
	// The paid charge's revenue rides the accompanying renewal event; this
	// event is a count only.
	if !wasNew && existing.Status == store.SubscriptionStatusTrialing &&
		norm.Status == store.SubscriptionStatusActive {
		h.emitEvent(ctx, projectID, stored, 0, "", store.SubscriptionEventConverted)
	}
	return stored, nil
}

// eventPrice resolves the revenue amount + currency for an emitted event: the
// store-reported transaction price when the store supplied one (the
// storefront-localized amount the buyer actually paid), otherwise the moth
// catalog price for the mapped product. Falling back to the catalog keeps a
// receipt from an older SDK — or a store that omits price — from booking zero
// revenue, while store-reported prices make per-currency revenue honest across
// storefronts.
func eventPrice(norm billing.NormalizedSubscription, product store.Product) (int64, string) {
	if norm.PriceAmountMicros > 0 && norm.Currency != "" {
		return norm.PriceAmountMicros, norm.Currency
	}
	return product.PriceAmountMicros, product.Currency
}

// matchProduct resolves a store SKU to a moth product for the given store. It
// returns the empty id and product when the SKU is not mapped.
func matchProduct(products []store.Product, storeName, sku string) (string, store.Product) {
	if sku == "" {
		return "", store.Product{}
	}
	for _, p := range products {
		if storeName == store.SubscriptionStoreApple && p.AppleProductID == sku {
			return p.ID, p
		}
		if storeName == store.SubscriptionStoreGoogle && p.GoogleProductID == sku {
			return p.ID, p
		}
	}
	return "", store.Product{}
}

// subscriptionEventType picks the revenue event for a freshly created
// subscription.
func subscriptionEventType(status string) string {
	if status == store.SubscriptionStatusTrialing {
		return store.SubscriptionEventTrialStarted
	}
	return store.SubscriptionEventPurchased
}

// emitEvent writes one revenue event (best effort; a failure is logged, never
// surfaced). priceMicros/currency are store-reported when available, otherwise
// the mapped catalog product's (see eventPrice); count-only events pass 0/"".
func (h *Handler) emitEvent(ctx context.Context, projectID string, sub store.Subscription, priceMicros int64, currency, eventType string) {
	e := store.SubscriptionEvent{
		ID:                authrpc.NewID(),
		ProjectID:         projectID,
		Type:              eventType,
		UserID:            sub.UserID,
		ProductID:         sub.ProductID,
		Store:             sub.Store,
		PriceAmountMicros: priceMicros,
		Currency:          currency,
		Environment:       sub.Environment,
		CreatedAt:         h.now(),
	}
	if err := h.store.InsertSubscriptionEvent(ctx, e); err != nil {
		h.log.ErrorContext(ctx, "insert subscription event", "type", eventType, "error", err.Error())
	}
}

// --- error mapping --------------------------------------------------------

func errBillingNotConfigured() *connect.Error {
	return authrpc.NewError(connect.CodeFailedPrecondition, authrpc.ReasonBillingNotConfigured,
		"billing is not configured for this project")
}

// purchaseError maps a store-validation error to a client-facing connect error
// with a stable ErrorInfo reason.
func purchaseError(err error) *connect.Error {
	switch {
	case errors.Is(err, errNotConfigured):
		return errBillingNotConfigured()
	case errors.Is(err, billing.ErrMalformed),
		errors.Is(err, billing.ErrInvalidSignature),
		errors.Is(err, billing.ErrUntrustedChain),
		errors.Is(err, billing.ErrBundleMismatch),
		errors.Is(err, billing.ErrNotFound):
		return authrpc.NewError(connect.CodeInvalidArgument, authrpc.ReasonInvalidReceipt,
			"the purchase receipt could not be validated")
	default:
		// An outbound store failure (network, 5xx, auth) — transient from the
		// client's perspective.
		return authrpc.NewError(connect.CodeUnavailable, authrpc.ReasonStoreUnavailable,
			fmt.Sprintf("the store could not be reached: %v", err))
	}
}
