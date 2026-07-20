package billingrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"connectrpc.com/connect"

	billingv1 "github.com/aloisdeniel/moth/gen/moth/billing/v1"
	"github.com/aloisdeniel/moth/internal/billing"
	"github.com/aloisdeniel/moth/internal/entitlements"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// stripeSignatureTolerance bounds how old (or future-dated) a Stripe-Signature
// timestamp may be — Stripe's own SDKs default to five minutes.
const stripeSignatureTolerance = 5 * time.Minute

// Stripe webhook event types moth acts on. Anything else is recorded as an
// audit row and acknowledged.
const (
	stripeEventCheckoutCompleted   = "checkout.session.completed"
	stripeEventSubscriptionCreated = "customer.subscription.created"
	stripeEventSubscriptionUpdated = "customer.subscription.updated"
	stripeEventSubscriptionDeleted = "customer.subscription.deleted"
)

// CreateCheckoutSession starts a Stripe-hosted Checkout for the signed-in user:
// it resolves the tier's stripe_price_id, lazily creates the user's Stripe
// customer, and returns the hosted Checkout URL. moth never renders a card
// field — the money surface stays Stripe-hosted.
func (h *Handler) CreateCheckoutSession(ctx context.Context, req *connect.Request[billingv1.CreateCheckoutSessionRequest]) (*connect.Response[billingv1.CreateCheckoutSessionResponse], error) {
	project, user, err := h.auth.AuthenticateUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	if !validRedirectURL(req.Msg.SuccessUrl) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("success_url must be an absolute http(s) URL"))
	}
	if !validRedirectURL(req.Msg.CancelUrl) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cancel_url must be an absolute http(s) URL"))
	}
	products, err := h.store.ListProducts(ctx, project.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var product store.Product
	found := false
	for _, p := range products {
		if p.Identifier == req.Msg.ProductIdentifier {
			product, found = p, true
			break
		}
	}
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("unknown product %q", req.Msg.ProductIdentifier))
	}
	if product.StripePriceID == "" {
		return nil, authrpc.NewError(connect.CodeFailedPrecondition, authrpc.ReasonProductNotOnStore,
			"this product is not available for web purchase")
	}

	// Stripe happily lets the same customer subscribe to the same price twice;
	// moth refuses the double-bill up front: an access-granting Stripe
	// subscription on this tier blocks a second checkout.
	subs, err := h.store.ListUserSubscriptions(ctx, project.ID, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, s := range subs {
		if s.Store == store.SubscriptionStoreStripe && s.ProductID == product.ID &&
			entitlements.StatusGrantsAccess(s.Status) {
			return nil, authrpc.NewError(connect.CodeFailedPrecondition, authrpc.ReasonAlreadySubscribed,
				"an active subscription to this product already exists")
		}
	}

	cred, err := h.store.GetBillingCredentials(ctx, project.ID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, errBillingNotConfigured()
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	client, err := h.stripeClient(cred)
	if err != nil {
		if errors.Is(err, errNotConfigured) {
			return nil, errBillingNotConfigured()
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	customerID, err := h.stripeCustomerID(ctx, client, project, user)
	if err != nil {
		return nil, err
	}

	// A trial vocabulary Stripe cannot express fails the checkout: silently
	// charging a user against an advertised free trial is never an option.
	trialDays, err := billing.StripeTrialDays(product.TrialPeriod)
	if err != nil {
		h.log.WarnContext(ctx, "stripe: unsupported trial period for web checkout",
			"product", product.Identifier, "trial_period", product.TrialPeriod)
		return nil, authrpc.NewError(connect.CodeFailedPrecondition, authrpc.ReasonTrialNotSupported,
			"this tier's trial period is not supported for web checkout")
	}

	params := billing.StripeCheckoutParams{
		PriceID:           product.StripePriceID,
		CustomerID:        customerID,
		SuccessURL:        req.Msg.SuccessUrl,
		CancelURL:         req.Msg.CancelUrl,
		ClientReferenceID: user.ID,
		// Mirrored onto the resulting subscription via subscription_data, so
		// both the session and the subscription carry the moth identity.
		Metadata: map[string]string{
			"project_id": project.ID,
			"user_id":    user.ID,
			"product":    product.Identifier,
		},
		TrialPeriodDays: trialDays,
	}
	sess, err := client.CreateCheckoutSession(ctx, params)
	if isStripeResourceMissing(err) {
		// The stored customer mapping is stale — swapped test/live keys, or the
		// customer was deleted in the Stripe dashboard. Stripe will reject that
		// id forever, so retrying cannot heal it: mint a fresh customer,
		// overwrite the mapping, and retry the session create exactly once.
		h.log.WarnContext(ctx, "stripe: stored customer rejected, refreshing mapping",
			"user", user.ID, "customer", customerID, "error", err.Error())
		created, cerr := client.CreateCustomer(ctx, user.Email, map[string]string{
			"moth_project_id": project.ID,
			"moth_user_id":    user.ID,
		})
		if cerr != nil {
			return nil, h.stripeCallError(ctx, cerr)
		}
		if uerr := h.store.UpdateStripeCustomer(ctx, store.StripeCustomer{
			ProjectID: project.ID, UserID: user.ID,
			StripeCustomerID: created.ID, CreatedAt: h.now(),
		}); uerr != nil {
			return nil, connect.NewError(connect.CodeInternal, uerr)
		}
		params.CustomerID = created.ID
		sess, err = client.CreateCheckoutSession(ctx, params)
	}
	if err != nil {
		return nil, h.stripeCallError(ctx, err)
	}
	return connect.NewResponse(&billingv1.CreateCheckoutSessionResponse{Url: sess.URL}), nil
}

// CreateBillingPortalSession returns a Stripe Billing Portal URL for the
// signed-in user — cancel, payment-method and invoice management stay
// Stripe-hosted.
func (h *Handler) CreateBillingPortalSession(ctx context.Context, req *connect.Request[billingv1.CreateBillingPortalSessionRequest]) (*connect.Response[billingv1.CreateBillingPortalSessionResponse], error) {
	project, user, err := h.auth.AuthenticateUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	if !validRedirectURL(req.Msg.ReturnUrl) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("return_url must be an absolute http(s) URL"))
	}
	cred, err := h.store.GetBillingCredentials(ctx, project.ID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, errBillingNotConfigured()
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	client, err := h.stripeClient(cred)
	if err != nil {
		if errors.Is(err, errNotConfigured) {
			return nil, errBillingNotConfigured()
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	sc, err := h.store.GetStripeCustomer(ctx, project.ID, user.ID)
	if errors.Is(err, store.ErrNotFound) {
		// The user never went through checkout: there is no Stripe customer,
		// hence nothing for the portal to manage.
		return nil, authrpc.NewError(connect.CodeFailedPrecondition, authrpc.ReasonNoBillingHistory,
			"no web billing history for this user")
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	sess, err := client.CreateBillingPortalSession(ctx, sc.StripeCustomerID, req.Msg.ReturnUrl)
	if isStripeResourceMissing(err) {
		// The stored mapping names a customer Stripe no longer recognizes
		// (swapped test/live keys, dashboard deletion): retrying cannot heal it
		// and there is no usable billing history behind it to manage.
		h.log.WarnContext(ctx, "stripe: stored customer rejected for portal",
			"user", user.ID, "customer", sc.StripeCustomerID, "error", err.Error())
		return nil, authrpc.NewError(connect.CodeFailedPrecondition, authrpc.ReasonNoBillingHistory,
			"no web billing history for this user")
	}
	if err != nil {
		return nil, h.stripeCallError(ctx, err)
	}
	return connect.NewResponse(&billingv1.CreateBillingPortalSessionResponse{Url: sess.URL}), nil
}

// stripeCustomerID returns the user's Stripe customer id, creating the Stripe
// customer (and the moth mapping row) on first checkout. A concurrent first
// checkout losing the insert race re-reads the winner's mapping so the user
// still ends up with exactly one Stripe customer per project.
func (h *Handler) stripeCustomerID(ctx context.Context, client *billing.StripeClient, project store.Project, user store.User) (string, error) {
	sc, err := h.store.GetStripeCustomer(ctx, project.ID, user.ID)
	if err == nil {
		return sc.StripeCustomerID, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return "", connect.NewError(connect.CodeInternal, err)
	}
	created, err := client.CreateCustomer(ctx, user.Email, map[string]string{
		"moth_project_id": project.ID,
		"moth_user_id":    user.ID,
	})
	if err != nil {
		return "", h.stripeCallError(ctx, err)
	}
	err = h.store.CreateStripeCustomer(ctx, store.StripeCustomer{
		ProjectID:        project.ID,
		UserID:           user.ID,
		StripeCustomerID: created.ID,
		CreatedAt:        h.now(),
	})
	if errors.Is(err, store.ErrConflict) {
		// A concurrent checkout created the mapping first; use the winner's
		// customer (the just-created duplicate stays unused on Stripe's side).
		sc, rerr := h.store.GetStripeCustomer(ctx, project.ID, user.ID)
		if rerr != nil {
			return "", connect.NewError(connect.CodeInternal, rerr)
		}
		return sc.StripeCustomerID, nil
	}
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, err)
	}
	return created.ID, nil
}

// validRedirectURL reports whether a checkout/portal redirect target is an
// absolute http(s) URL — the only shapes Stripe accepts and the only ones moth
// forwards.
func validRedirectURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// stripeCallError maps an outbound Stripe API failure to a client-facing
// connect error (purchaseError's shape for the checkout RPCs: everything
// outbound is transient from the client's perspective). The detailed Stripe
// error — which can carry customer ids, key mode and other account internals —
// is logged server-side only; the client gets a generic message with the
// stable reason.
func (h *Handler) stripeCallError(ctx context.Context, err error) *connect.Error {
	if errors.Is(err, errNotConfigured) {
		return errBillingNotConfigured()
	}
	h.log.WarnContext(ctx, "stripe: outbound call failed", "error", err.Error())
	return authrpc.NewError(connect.CodeUnavailable, authrpc.ReasonStoreUnavailable,
		"the store could not be reached")
}

// isStripeResourceMissing reports whether an outbound Stripe call failed
// because a referenced id no longer exists in the account/mode — Stripe's
// stable "resource_missing" error code, the tell-tale of a stale stored id
// (never healed by retrying).
func isStripeResourceMissing(err error) bool {
	var apiErr *billing.StripeAPIError
	return errors.As(err, &apiErr) && apiErr.Code == "resource_missing"
}

// --- webhook ---------------------------------------------------------------

// ProcessStripeWebhook handles a Stripe webhook event: it verifies the
// Stripe-Signature header against the project's webhook secret, dedupes on the
// event id, then re-reads authoritative subscription state from the Stripe API
// before touching anything — the event body is a nudge, never trusted for
// state. A replayed event is a no-op; a transient failure propagates so the
// HTTP layer 503s and Stripe redelivers.
func (h *Handler) ProcessStripeWebhook(ctx context.Context, project store.Project, body []byte, sigHeader string) error {
	cred, err := h.store.GetBillingCredentials(ctx, project.ID)
	if err != nil {
		return err
	}
	if len(cred.StripeWebhookSecretEnc) == 0 {
		return errNotConfigured
	}
	secret, err := h.master.Decrypt(cred.StripeWebhookSecretEnc)
	if err != nil {
		return fmt.Errorf("decrypt stripe webhook secret: %w", err)
	}
	if err := billing.VerifyStripeSignature(body, sigHeader, string(secret), h.now(), stripeSignatureTolerance); err != nil {
		return err
	}
	evt, err := billing.ParseStripeEvent(body)
	if err != nil {
		return err
	}
	id, done, err := h.claimNotification(ctx, project.ID, store.SubscriptionStoreStripe, evt.ID, evt.Type, "", body)
	if err != nil {
		return err
	}
	if done {
		return nil // already applied: idempotent replay
	}
	switch evt.Type {
	case stripeEventCheckoutCompleted:
		return h.processStripeCheckoutCompleted(ctx, project, cred, evt, id)
	case stripeEventSubscriptionCreated, stripeEventSubscriptionUpdated, stripeEventSubscriptionDeleted:
		return h.processStripeSubscriptionEvent(ctx, project, cred, evt, id)
	default:
		// Any other event type is recorded as an audit row and acknowledged.
		return h.markProcessed(ctx, project.ID, id)
	}
}

// processStripeCheckoutCompleted is the Stripe acquisition point: unlike
// Apple/Google (where SubmitPurchase creates the subscription row), the
// completed hosted checkout carries the moth identity, so this handler creates
// the subscription — after re-reading it from the Stripe API.
func (h *Handler) processStripeCheckoutCompleted(ctx context.Context, project store.Project, cred store.BillingCredentials, evt billing.StripeEvent, notifID string) error {
	session, err := evt.CheckoutSession()
	if err != nil {
		return err
	}
	if session.Subscription == "" {
		// A non-subscription checkout (one-time payment): recorded, nothing to
		// apply.
		return h.markProcessed(ctx, project.ID, notifID)
	}
	// A session provisioned for another project must not mutate this one — but
	// on a shared Stripe account every project's webhook endpoint receives
	// every event, so a validly signed event that merely belongs to another
	// tenant is acknowledged (recorded as the audit row, nothing applied). A
	// 400 here would put Stripe into its 3-day retry storm and eventually
	// auto-disable the endpoint for everyone.
	if pid, ok := session.Metadata["project_id"]; ok && pid != project.ID {
		h.log.InfoContext(ctx, "stripe: checkout completed for another project; recorded, not applied",
			"session", session.ID, "event_project", pid)
		return h.markProcessed(ctx, project.ID, notifID)
	}
	userID := session.ClientReferenceID
	if userID == "" {
		userID = session.Metadata["user_id"]
	}
	if userID == "" {
		h.log.WarnContext(ctx, "stripe: checkout completed without a moth user reference; recorded, not applied",
			"session", session.ID)
		return h.markProcessed(ctx, project.ID, notifID)
	}
	if _, err := h.store.GetUser(ctx, project.ID, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.log.WarnContext(ctx, "stripe: checkout completed for unknown user; recorded, not applied",
				"session", session.ID, "user", userID)
			return h.markProcessed(ctx, project.ID, notifID)
		}
		return err
	}
	// Reconcile the customer mapping: normally CreateCheckoutSession created it,
	// but a checkout provisioned outside moth (or a lost row) still names the
	// customer here.
	if session.Customer != "" {
		if _, err := h.store.GetStripeCustomer(ctx, project.ID, userID); errors.Is(err, store.ErrNotFound) {
			if cerr := h.store.CreateStripeCustomer(ctx, store.StripeCustomer{
				ProjectID: project.ID, UserID: userID,
				StripeCustomerID: session.Customer, CreatedAt: h.now(),
			}); cerr != nil && !errors.Is(cerr, store.ErrConflict) {
				return cerr
			}
		} else if err != nil {
			return err
		}
	}
	client, err := h.stripeClient(cred)
	if err != nil {
		return err
	}
	norm, _, err := client.GetSubscription(ctx, session.Subscription)
	if err != nil {
		return err
	}
	// emitEvent=true makes this the acquisition point: purchased/trial_started
	// is emitted exactly once because applyNormalized only emits when its
	// upsert actually inserted the row — a redelivered-but-unprocessed claim
	// (or a racing subscription.created) re-applies against the existing row
	// and cannot double-emit.
	if _, err := h.applyNormalized(ctx, project.ID, userID, norm, true); err != nil {
		return err
	}
	return h.markProcessed(ctx, project.ID, notifID)
}

// processStripeSubscriptionEvent handles the customer.subscription.* family:
// resolve the subscription to a moth user, re-read authoritative state from the
// Stripe API, derive the analytics event from the stored→new transition, then
// apply.
func (h *Handler) processStripeSubscriptionEvent(ctx context.Context, project store.Project, cred store.BillingCredentials, evt billing.StripeEvent, notifID string) error {
	obj, err := evt.SubscriptionObject()
	if err != nil {
		return err
	}
	if obj.ID == "" {
		return fmt.Errorf("%w: stripe subscription event without subscription id", billing.ErrMalformed)
	}
	existing, err := h.store.GetSubscriptionByStoreID(ctx, project.ID, store.SubscriptionStoreStripe, obj.ID)
	known := err == nil
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	userID := existing.UserID
	if !known {
		// The subscription is not in moth yet — usually subscription.created
		// racing checkout.session.completed. Attribution normally happens at
		// checkout completion, but if that webhook was missed the customer
		// mapping (created by CreateCheckoutSession) still names the user.
		if obj.Customer != "" {
			if sc, cerr := h.store.GetStripeCustomerByStripeID(ctx, project.ID, obj.Customer); cerr == nil {
				userID = sc.UserID
			} else if !errors.Is(cerr, store.ErrNotFound) {
				return cerr
			}
		}
		if userID == "" {
			h.log.InfoContext(ctx, "stripe: event for unknown subscription; recorded, not applied",
				"type", evt.Type, "subscription", obj.ID)
			return h.markProcessed(ctx, project.ID, notifID)
		}
	}
	client, err := h.stripeClient(cred)
	if err != nil {
		return err
	}
	norm, _, err := client.GetSubscription(ctx, obj.ID)
	if err != nil {
		// On customer.subscription.deleted Stripe's API normally still returns
		// the subscription with status=canceled — but once Stripe's retention
		// window drops it the re-read 404s. That 404 is expected for a deletion
		// and cannot heal by retrying, so synthesize the terminal state from the
		// stored row instead of erroring forever. Any other failure (or a 404 on
		// created/updated) is transient: propagate so Stripe redelivers.
		tolerable := errors.Is(err, billing.ErrNotFound) &&
			evt.Type == stripeEventSubscriptionDeleted && known
		if !tolerable {
			return err
		}
		norm = expiredStripeNorm(ctx, h, project.ID, existing, evt)
	}
	eventTypes := stripeSubscriptionEventTypes(evt.Type, known, existing, norm)
	// emitEvent=true is only consequential when the upsert inserts a new row
	// (the recovered missed-checkout attribution): applyNormalized then emits
	// the acquisition exactly once; for a known row it emits nothing extra.
	stored, err := h.applyNormalized(ctx, project.ID, userID, norm, true)
	if err != nil {
		return err
	}
	if len(eventTypes) > 0 {
		products, _ := h.store.ListProducts(ctx, project.ID)
		_, product := matchProduct(products, stored.Store, norm.ProductID)
		price, currency := eventPrice(norm, product)
		for _, eventType := range eventTypes {
			if isAcquisitionEvent(eventType) {
				continue
			}
			h.emitEvent(ctx, project.ID, stored, price, currency, eventType)
		}
	}
	return h.markProcessed(ctx, project.ID, notifID)
}

// stripeSubscriptionEventTypes derives the analytics events for a
// customer.subscription.* webhook by comparing the stored row against the
// authoritative re-read — Stripe's event vocabulary alone cannot distinguish a
// renewal from a cancel flip (both arrive as "updated"), and one event can
// legitimately carry several transitions at once (a cancel flip observed
// together with a period advance), so each delta is derived independently and
// every qualifying one is emitted:
//
//	deleted                                  -> expired
//	auto_renew true -> false                 -> canceled (cancel_at_period_end)
//	auto_renew false -> true                 -> nothing (resubscribed; record only)
//	period end advanced AND new status is
//	  active/trialing                        -> renewed
//	in_billing_retry -> active               -> renewed (recovery: the retried
//	                                            charge finally succeeded)
//	anything else                            -> nothing (record only)
//
// Stripe advances current_period_end even when the renewal charge FAILS
// (status past_due): booking 'renewed' then would overstate revenue by one
// full period per involuntary churn. The period advance while past_due books
// nothing; the deferred renewal books on the past_due→active recovery flip —
// the moment the charge actually succeeded (Google's RECOVERED precedent) —
// which by then shows no fresh period delta.
func stripeSubscriptionEventTypes(evtType string, known bool, existing store.Subscription, norm billing.NormalizedSubscription) []string {
	if evtType == stripeEventSubscriptionDeleted {
		return []string{store.SubscriptionEventExpired}
	}
	if !known {
		return nil
	}
	var events []string
	if existing.AutoRenew && !norm.AutoRenew {
		events = append(events, store.SubscriptionEventCanceled)
	}
	periodAdvanced := existing.CurrentPeriodEnd != nil && !norm.CurrentPeriodEnd.IsZero() &&
		norm.CurrentPeriodEnd.After(*existing.CurrentPeriodEnd)
	renewedNow := periodAdvanced &&
		(norm.Status == store.SubscriptionStatusActive || norm.Status == store.SubscriptionStatusTrialing)
	recovered := existing.Status == store.SubscriptionStatusInBillingRetry &&
		norm.Status == store.SubscriptionStatusActive
	if renewedNow || recovered {
		events = append(events, store.SubscriptionEventRenewed)
	}
	return events
}

// stripeRawProductID extracts the Stripe product id ("prod_...") from a raw
// Stripe subscription state blob. NormalizedSubscription only carries the
// price id as its SKU, but the product id — which survives catalog price
// re-points — rides along in the raw state moth already keeps, so product
// matching can fall back to it without widening the normalized model.
func stripeRawProductID(raw []byte) string {
	var sub struct {
		Items struct {
			Data []struct {
				Price struct {
					Product string `json:"product"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
	}
	if json.Unmarshal(raw, &sub) != nil || len(sub.Items.Data) == 0 {
		return ""
	}
	return sub.Items.Data[0].Price.Product
}

// expiredStripeNorm synthesizes the terminal normalized state for a deleted
// subscription whose API re-read 404s (dropped from Stripe's retention): the
// stored row's identity with status expired and auto-renew off. The stored moth
// product id is mapped back to its stripe_price_id so applyNormalized keeps the
// product link.
func expiredStripeNorm(ctx context.Context, h *Handler, projectID string, existing store.Subscription, evt billing.StripeEvent) billing.NormalizedSubscription {
	norm := billing.NormalizedSubscription{
		Store:              store.SubscriptionStoreStripe,
		StoreTransactionID: existing.StoreTransactionID,
		SubscriptionID:     existing.SubscriptionID,
		Status:             store.SubscriptionStatusExpired,
		AutoRenew:          false,
		Environment:        existing.Environment,
		RawState:           evt.Data.Object,
	}
	if existing.CurrentPeriodEnd != nil {
		norm.CurrentPeriodEnd = *existing.CurrentPeriodEnd
	}
	if existing.ProductID != "" {
		if p, err := h.store.GetProduct(ctx, projectID, existing.ProductID); err == nil {
			norm.ProductID = p.StripePriceID
		}
	}
	return norm
}
