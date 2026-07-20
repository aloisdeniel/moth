package authrpc

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/internal/events"
	"github.com/aloisdeniel/moth/internal/i18n"
	mailpkg "github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/password"
	"github.com/aloisdeniel/moth/internal/store"
)

// VerificationEmail builds the "confirm your email" message localized for the
// locale negotiated from header (falling back to the project default when the
// copy store cannot be read). Shared with the admin user handlers.
func (h *Handler) VerificationEmail(ctx context.Context, project store.Project, to, link string, header http.Header) mailpkg.Message {
	loc := h.emailLocalizer(ctx, project, header)
	vars := map[string]string{"app": project.Name}
	return mailpkg.RenderContent(h.Brand(project), to, mailpkg.Content{
		Subject:     loc.Value(i18n.EmailVerifySubject, vars),
		Paragraphs:  []string{loc.Value(i18n.EmailVerifyBody, vars), loc.Value(i18n.EmailIgnoreNotice, nil)},
		ButtonLabel: loc.Value(i18n.EmailVerifyButton, nil),
		ButtonURL:   link,
	})
}

// PasswordResetEmail builds the "reset your password" message localized for the
// locale negotiated from header. Shared with the admin user handlers.
func (h *Handler) PasswordResetEmail(ctx context.Context, project store.Project, to, link string, header http.Header) mailpkg.Message {
	loc := h.emailLocalizer(ctx, project, header)
	vars := map[string]string{"app": project.Name}
	return mailpkg.RenderContent(h.Brand(project), to, mailpkg.Content{
		Subject:     loc.Value(i18n.EmailResetSubject, vars),
		Paragraphs:  []string{loc.Value(i18n.EmailResetBody, vars), loc.Value(i18n.EmailIgnoreNotice, nil)},
		ButtonLabel: loc.Value(i18n.EmailResetButton, nil),
		ButtonURL:   link,
	})
}

// EmailChangeEmail builds the "confirm your new email" message (sent to the new
// address to prove ownership) localized for the locale negotiated from header.
func (h *Handler) EmailChangeEmail(ctx context.Context, project store.Project, to, link string, header http.Header) mailpkg.Message {
	loc := h.emailLocalizer(ctx, project, header)
	vars := map[string]string{"app": project.Name}
	return mailpkg.RenderContent(h.Brand(project), to, mailpkg.Content{
		Subject:     loc.Value(i18n.EmailChangeSubject, vars),
		Paragraphs:  []string{loc.Value(i18n.EmailChangeBody, vars), loc.Value(i18n.EmailIgnoreNotice, nil)},
		ButtonLabel: loc.Value(i18n.EmailChangeButton, nil),
		ButtonURL:   link,
	})
}

// EmailChangedEmail builds the "your email was changed" notice (sent to the old
// address with a revert link) localized for the locale negotiated from header.
func (h *Handler) EmailChangedEmail(ctx context.Context, project store.Project, to, newEmail, revertLink string, header http.Header) mailpkg.Message {
	loc := h.emailLocalizer(ctx, project, header)
	return mailpkg.RenderContent(h.Brand(project), to, mailpkg.Content{
		Subject:     loc.Value(i18n.EmailChangedSubject, map[string]string{"app": project.Name}),
		Paragraphs:  []string{loc.Value(i18n.EmailChangedBody, map[string]string{"app": project.Name, "email": newEmail}), loc.Value(i18n.EmailChangedRevert, nil)},
		ButtonLabel: loc.Value(i18n.EmailChangedButton, nil),
		ButtonURL:   revertLink,
	})
}

// emailLocalizer negotiates the recipient's locale for an outgoing email; a
// copy-store read failure degrades to bundled defaults rather than failing the
// send.
func (h *Handler) emailLocalizer(ctx context.Context, project store.Project, header http.Header) Localizer {
	loc, err := NewLocalizer(ctx, h.store, project.ID, header)
	if err != nil {
		h.log.ErrorContext(ctx, "email localize", "error", err.Error())
		return NewFallbackLocalizer()
	}
	return loc
}

// sendVerification issues a verify token and emails its link, localized for the
// requester's negotiated locale (header carries the SDK's x-moth-language).
func (h *Handler) sendVerification(ctx context.Context, project store.Project, user store.User, header http.Header) error {
	plain, err := h.issueEmailToken(ctx, project.ID, user.ID,
		store.EmailTokenPurposeVerify, "", verifyTokenTTL)
	if err != nil {
		return errInternal(err)
	}
	return h.send(ctx, h.VerificationEmail(ctx, project, user.Email,
		h.verifyLink(project.Slug, plain), header), false)
}

func (h *Handler) RequestEmailVerification(ctx context.Context, req *connect.Request[authv1.RequestEmailVerificationRequest]) (*connect.Response[authv1.RequestEmailVerificationResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	// Always OK: the response must not reveal whether an account exists.
	resp := connect.NewResponse(&authv1.RequestEmailVerificationResponse{})
	user, err := h.store.GetUserByEmail(ctx, project.ID, normalizeEmail(req.Msg.Email))
	if err != nil || user.Verified() || user.Disabled() {
		return resp, nil
	}
	if err := h.sendVerification(ctx, project, user, req.Header()); err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *Handler) ConfirmEmailVerification(ctx context.Context, req *connect.Request[authv1.ConfirmEmailVerificationRequest]) (*connect.Response[authv1.ConfirmEmailVerificationResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	et, err := h.consumeEmailToken(ctx, project.ID, req.Msg.Token, store.EmailTokenPurposeVerify)
	if err != nil {
		return nil, err
	}
	user, err := h.store.GetUser(ctx, project.ID, et.UserID)
	if err != nil {
		return nil, errInvalidEmailToken()
	}
	if !user.Verified() {
		now := h.now()
		user.EmailVerifiedAt = &now
		user.UpdatedAt = now
		if err := h.store.UpdateUser(ctx, user); err != nil {
			return nil, errInternal(err)
		}
		h.emit(events.EmailVerified(ctx, project.ID, user.ID))
	}
	return connect.NewResponse(&authv1.ConfirmEmailVerificationResponse{}), nil
}

func (h *Handler) RequestPasswordReset(ctx context.Context, req *connect.Request[authv1.RequestPasswordResetRequest]) (*connect.Response[authv1.RequestPasswordResetResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	// Always OK: no account enumeration through this RPC.
	resp := connect.NewResponse(&authv1.RequestPasswordResetResponse{})
	user, err := h.store.GetUserByEmail(ctx, project.ID, normalizeEmail(req.Msg.Email))
	if err != nil || user.Disabled() {
		return resp, nil
	}
	plain, err := h.issueEmailToken(ctx, project.ID, user.ID,
		store.EmailTokenPurposeReset, "", resetTokenTTL)
	if err != nil {
		return nil, errInternal(err)
	}
	if err := h.send(ctx, h.PasswordResetEmail(ctx, project, user.Email,
		h.resetLink(project.Slug, plain), req.Header()), false); err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *Handler) ConfirmPasswordReset(ctx context.Context, req *connect.Request[authv1.ConfirmPasswordResetRequest]) (*connect.Response[authv1.ConfirmPasswordResetResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	if err := validPassword(req.Msg.NewPassword, project.Settings); err != nil {
		return nil, err
	}
	et, err := h.consumeEmailToken(ctx, project.ID, req.Msg.Token, store.EmailTokenPurposeReset)
	if err != nil {
		return nil, err
	}
	user, err := h.store.GetUser(ctx, project.ID, et.UserID)
	if err != nil {
		return nil, errInvalidEmailToken()
	}
	now := h.now()
	if user.PasswordHash, err = password.Hash(req.Msg.NewPassword); err != nil {
		return nil, errInternal(err)
	}
	if !user.Verified() {
		// Completing a reset proves control of the mailbox.
		user.EmailVerifiedAt = &now
	}
	user.UpdatedAt = now
	if err := h.store.UpdateUser(ctx, user); err != nil {
		return nil, errInternal(err)
	}
	// A completed reset revokes every refresh token.
	if err := h.store.RevokeUserRefreshTokens(ctx, project.ID, user.ID, now); err != nil {
		return nil, errInternal(err)
	}
	h.emit(events.PasswordResetCompleted(ctx, project.ID, user.ID))
	return connect.NewResponse(&authv1.ConfirmPasswordResetResponse{}), nil
}

func (h *Handler) RequestEmailChange(ctx context.Context, req *connect.Request[authv1.RequestEmailChangeRequest]) (*connect.Response[authv1.RequestEmailChangeResponse], error) {
	project, user, err := h.requireUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	newEmail, err := validEmail(req.Msg.NewEmail)
	if err != nil {
		return nil, err
	}
	if newEmail == user.Email {
		return nil, newError(connect.CodeInvalidArgument, ReasonInvalidEmail,
			"new email is the same as the current one")
	}
	if _, err := h.store.GetUserByEmail(ctx, project.ID, newEmail); err == nil {
		return nil, newError(connect.CodeAlreadyExists, ReasonEmailAlreadyExists,
			"an account with this email already exists")
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, errInternal(err)
	}

	plain, err := h.issueEmailToken(ctx, project.ID, user.ID,
		store.EmailTokenPurposeEmailChange, newEmail, emailChangeTokenTTL)
	if err != nil {
		return nil, errInternal(err)
	}
	// The confirmation goes to the NEW address: it must be verified before
	// the account switches. Localized for the requester's negotiated locale
	// (header carries the SDK's x-moth-language).
	if err := h.send(ctx, h.EmailChangeEmail(ctx, project, newEmail,
		h.confirmEmailLink(project.Slug, plain), req.Header()), true); err != nil {
		return nil, err
	}
	return connect.NewResponse(&authv1.RequestEmailChangeResponse{}), nil
}

func (h *Handler) ConfirmEmailChange(ctx context.Context, req *connect.Request[authv1.ConfirmEmailChangeRequest]) (*connect.Response[authv1.ConfirmEmailChangeResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	et, err := h.consumeEmailToken(ctx, project.ID, req.Msg.Token,
		store.EmailTokenPurposeEmailChange, store.EmailTokenPurposeEmailRevert)
	if err != nil {
		return nil, err
	}
	user, err := h.store.GetUser(ctx, project.ID, et.UserID)
	if err != nil {
		return nil, errInvalidEmailToken()
	}
	targetEmail := et.Payload
	if targetEmail == user.Email {
		// Nothing to do (e.g. revert after a manual change back).
		return connect.NewResponse(&authv1.ConfirmEmailChangeResponse{}), nil
	}
	if _, err := h.store.GetUserByEmail(ctx, project.ID, targetEmail); err == nil {
		return nil, newError(connect.CodeAlreadyExists, ReasonEmailAlreadyExists,
			"an account with this email already exists")
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, errInternal(err)
	}

	oldEmail := user.Email
	now := h.now()
	user.Email = targetEmail
	// Both directions prove mailbox control: the change token was mailed
	// to the new address, the revert token to the old one.
	user.EmailVerifiedAt = &now
	user.UpdatedAt = now
	if err := h.store.UpdateUser(ctx, user); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, newError(connect.CodeAlreadyExists, ReasonEmailAlreadyExists,
				"an account with this email already exists")
		}
		return nil, errInternal(err)
	}
	// Outstanding links sent to the previous address must die with it.
	for _, purpose := range []string{store.EmailTokenPurposeVerify, store.EmailTokenPurposeReset} {
		if err := h.store.DeleteUserEmailTokens(ctx, project.ID, user.ID, purpose); err != nil {
			return nil, errInternal(err)
		}
	}

	switch et.Purpose {
	case store.EmailTokenPurposeEmailChange:
		// Tell the old address, with a revert link valid 72 h.
		revert, err := h.issueEmailToken(ctx, project.ID, user.ID,
			store.EmailTokenPurposeEmailRevert, oldEmail, EmailRevertWindow)
		if err != nil {
			return nil, errInternal(err)
		}
		if err := h.send(ctx, h.EmailChangedEmail(ctx, project, oldEmail,
			user.Email, h.confirmEmailLink(project.Slug, revert), req.Header()), false); err != nil {
			return nil, err
		}
	case store.EmailTokenPurposeEmailRevert:
		// The legitimate owner took the address back; assume the account
		// was compromised and end every session.
		if err := h.store.RevokeUserRefreshTokens(ctx, project.ID, user.ID, now); err != nil {
			return nil, errInternal(err)
		}
	}
	return connect.NewResponse(&authv1.ConfirmEmailChangeResponse{}), nil
}
