package authrpc

import (
	"context"
	"net/url"
	"time"

	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// Email-token lifetimes.
const (
	verifyTokenTTL      = 24 * time.Hour
	resetTokenTTL       = time.Hour
	emailChangeTokenTTL = 24 * time.Hour
	// EmailRevertWindow is how long the old address can undo an email
	// change.
	EmailRevertWindow = 72 * time.Hour
)

// issueEmailToken creates a fresh single-use token of one purpose,
// dropping the user's previous tokens of that purpose so only the latest
// link works. It returns the plaintext for the email link.
func (h *Handler) issueEmailToken(ctx context.Context, projectID, userID, purpose, payload string, ttl time.Duration) (string, error) {
	if err := h.store.DeleteUserEmailTokens(ctx, projectID, userID, purpose); err != nil {
		return "", err
	}
	plain := token.Random(32)
	now := h.now()
	err := h.store.CreateEmailToken(ctx, store.EmailToken{
		ID:        NewID(),
		ProjectID: projectID,
		UserID:    userID,
		Purpose:   purpose,
		TokenHash: hashToken(plain),
		Payload:   payload,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	})
	if err != nil {
		return "", err
	}
	return plain, nil
}

// consumeEmailToken validates and consumes a single-use token of the given
// purposes, returning it. Unknown, expired, already-consumed and
// wrong-purpose tokens all fail identically.
func (h *Handler) consumeEmailToken(ctx context.Context, projectID, plain string, purposes ...string) (store.EmailToken, error) {
	et, err := h.store.GetEmailToken(ctx, projectID, hashToken(plain))
	if err != nil {
		return store.EmailToken{}, errInvalidEmailToken()
	}
	purposeOK := false
	for _, p := range purposes {
		purposeOK = purposeOK || et.Purpose == p
	}
	if !purposeOK || !et.Usable(h.now()) {
		return store.EmailToken{}, errInvalidEmailToken()
	}
	if err := h.store.ConsumeEmailToken(ctx, projectID, et.ID, h.now()); err != nil {
		return store.EmailToken{}, errInvalidEmailToken()
	}
	return et, nil
}

// Hosted confirmation-page URLs; email clients open a browser, so the
// links must be plain HTTP pages (they invoke the confirm RPCs
// in-process).
func (h *Handler) verifyLink(slug, tok string) string {
	return h.pageLink(slug, "verify", tok)
}

func (h *Handler) resetLink(slug, tok string) string {
	return h.pageLink(slug, "reset", tok)
}

func (h *Handler) confirmEmailLink(slug, tok string) string {
	return h.pageLink(slug, "confirm-email", tok)
}

func (h *Handler) pageLink(slug, page, tok string) string {
	return h.baseURL + "/p/" + url.PathEscape(slug) + "/" + page + "?token=" + url.QueryEscape(tok)
}
