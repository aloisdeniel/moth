package authrpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	mailpkg "github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/password"
	"github.com/aloisdeniel/moth/internal/store"
)

// sendVerification issues a verify token and emails its link.
func (h *Handler) sendVerification(ctx context.Context, project store.Project, user store.User) error {
	plain, err := h.issueEmailToken(ctx, project.ID, user.ID,
		store.EmailTokenPurposeVerify, "", verifyTokenTTL)
	if err != nil {
		return errInternal(err)
	}
	return h.send(ctx, mailpkg.Verification(h.Brand(project), user.Email,
		h.verifyLink(project.Slug, plain)), false)
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
	if err := h.sendVerification(ctx, project, user); err != nil {
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
	if err := h.send(ctx, mailpkg.PasswordReset(h.Brand(project), user.Email,
		h.resetLink(project.Slug, plain)), false); err != nil {
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
	// the account switches.
	if err := h.send(ctx, mailpkg.EmailChangeConfirm(h.Brand(project), newEmail,
		h.confirmEmailLink(project.Slug, plain)), true); err != nil {
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
		if err := h.send(ctx, mailpkg.EmailChangedNotice(h.Brand(project), oldEmail,
			user.Email, h.confirmEmailLink(project.Slug, revert)), false); err != nil {
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
