package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/aloisdeniel/moth/internal/billing"
	"github.com/aloisdeniel/moth/internal/store"
)

// Store notification webhooks. Renewals and refunds are store round-trips, not
// RPCs, so these are plain-HTTP, project-scoped endpoints. Both do the
// validating store read; a verification failure is a 400 so the store stops
// retrying a payload that can never verify, while a transient internal failure
// (store API blip, DB write) is logged and 503'd so the store REDELIVERS —
// the notification row is left unprocessed and the redelivery re-drives the
// authoritative re-read. A mid-period refund/revoke would otherwise be dropped
// with no self-healing (the reconciliation sweep only re-reads subscriptions
// whose period has already lapsed). Every state change goes through a
// validating store read — a notification body is never trusted on its own.

// webhookMaxBody caps a notification body; store payloads are a few KiB.
const webhookMaxBody = 1 << 20

// handleAppleNotification receives an App Store Server Notification V2:
// POST /billing/apple/notifications/{slug} with a {"signedPayload": "<JWS>"}
// body.
func (s *Server) handleAppleNotification(w http.ResponseWriter, r *http.Request) {
	project, ok := s.billingProject(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, webhookMaxBody))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	var envelope struct {
		SignedPayload string `json:"signedPayload"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.SignedPayload == "" {
		http.Error(w, "invalid notification body", http.StatusBadRequest)
		return
	}
	if err := s.billing.ProcessAppleNotification(r.Context(), project, envelope.SignedPayload); err != nil {
		s.writeWebhookError(w, r, "apple", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleGoogleRTDN receives a Play Real-time Developer Notification delivered as
// a Cloud Pub/Sub push: POST /billing/google/rtdn/{slug}?token=<secret>. The
// push is authenticated against the project's stored RTDN secret before any
// work is done.
func (s *Server) handleGoogleRTDN(w http.ResponseWriter, r *http.Request) {
	project, ok := s.billingProject(w, r)
	if !ok {
		return
	}
	cred, err := s.billing.Credentials(r.Context(), project.ID)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if !s.billing.AuthenticateGooglePush(cred, r.URL.Query().Get("token")) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, webhookMaxBody))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	if err := s.billing.ProcessGoogleNotification(r.Context(), project, body); err != nil {
		s.writeWebhookError(w, r, "google", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleStripeWebhook receives a Stripe webhook event:
// POST /billing/stripe/webhook/{slug} with a Stripe-Signature header. The raw
// body bytes are what the HMAC signs, so they are read verbatim and verified
// inside ProcessStripeWebhook before anything is parsed or persisted.
func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	project, ok := s.billingProject(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, webhookMaxBody))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	if err := s.billing.ProcessStripeWebhook(r.Context(), project, body, r.Header.Get("Stripe-Signature")); err != nil {
		s.writeWebhookError(w, r, "stripe", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// billingProject resolves the {slug} path value to a project, writing a 404 when
// it is unknown.
func (s *Server) billingProject(w http.ResponseWriter, r *http.Request) (store.Project, bool) {
	project, err := s.store.GetProjectBySlug(r.Context(), r.PathValue("slug"))
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return store.Project{}, false
	}
	if err != nil {
		s.internalError(w, r, err)
		return store.Project{}, false
	}
	return project, true
}

// writeWebhookError maps a processing error to an HTTP status: a permanent
// verification failure is a 400 (the payload can never verify — stop retrying),
// while a transient fault (store API unavailable, DB write failure) is a 503 so
// the store redelivers the notification. Because the notification row is only
// marked processed after a successful re-read+apply, that redelivery re-drives
// the state change instead of dropping it — critical for a refund/revoke that
// arrives mid-period, which the reconciliation sweep does not cover.
func (s *Server) writeWebhookError(w http.ResponseWriter, r *http.Request, storeName string, err error) {
	switch {
	case errors.Is(err, billing.ErrMalformed),
		errors.Is(err, billing.ErrInvalidSignature),
		errors.Is(err, billing.ErrUntrustedChain),
		errors.Is(err, billing.ErrBundleMismatch):
		s.log.WarnContext(r.Context(), "billing webhook rejected", "store", storeName, "error", err.Error())
		http.Error(w, "invalid notification", http.StatusBadRequest)
	default:
		s.log.ErrorContext(r.Context(), "billing webhook processing failed", "store", storeName, "error", err.Error())
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}
}
