package authrpc

import (
	"context"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/internal/password"
	"github.com/aloisdeniel/moth/internal/store"
)

func (h *Handler) GetMe(ctx context.Context, req *connect.Request[authv1.GetMeRequest]) (*connect.Response[authv1.GetMeResponse], error) {
	_, user, err := h.requireUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&authv1.GetMeResponse{User: userProto(user)}), nil
}

func (h *Handler) UpdateMe(ctx context.Context, req *connect.Request[authv1.UpdateMeRequest]) (*connect.Response[authv1.UpdateMeResponse], error) {
	_, user, err := h.requireUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	if req.Msg.DisplayName != nil {
		user.DisplayName = *req.Msg.DisplayName
	}
	if req.Msg.AvatarUrl != nil {
		user.AvatarURL = *req.Msg.AvatarUrl
	}
	user.UpdatedAt = h.now()
	if err := h.store.UpdateUser(ctx, user); err != nil {
		return nil, errInternal(err)
	}
	return connect.NewResponse(&authv1.UpdateMeResponse{User: userProto(user)}), nil
}

func (h *Handler) ChangePassword(ctx context.Context, req *connect.Request[authv1.ChangePasswordRequest]) (*connect.Response[authv1.ChangePasswordResponse], error) {
	project, user, err := h.requireUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	if user.PasswordHash == "" || !password.Verify(req.Msg.CurrentPassword, user.PasswordHash) {
		return nil, errInvalidCredentials()
	}
	if err := validPassword(req.Msg.NewPassword, project.Settings); err != nil {
		return nil, err
	}
	if user.PasswordHash, err = password.Hash(req.Msg.NewPassword); err != nil {
		return nil, errInternal(err)
	}
	user.UpdatedAt = h.now()
	if err := h.store.UpdateUser(ctx, user); err != nil {
		return nil, errInternal(err)
	}
	// Revoke every session, then hand this device a fresh one so only the
	// caller stays signed in.
	if err := h.store.RevokeUserRefreshTokens(ctx, project.ID, user.ID, h.now()); err != nil {
		return nil, errInternal(err)
	}
	tokens, err := h.issueSession(ctx, project, user, "")
	if err != nil {
		return nil, errInternal(err)
	}
	return connect.NewResponse(&authv1.ChangePasswordResponse{Tokens: tokens}), nil
}

// deleteReauthWindow is how recent a social-only account's sign-in must be
// for DeleteAccount: the user's last credential sign-in may be at most
// this old.
const deleteReauthWindow = 5 * time.Minute

func (h *Handler) DeleteAccount(ctx context.Context, req *connect.Request[authv1.DeleteAccountRequest]) (*connect.Response[authv1.DeleteAccountResponse], error) {
	project, user, err := h.requireUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	// Fresh re-authentication (App Store guideline 5.1.1). Accounts with a
	// password always re-enter it; social-only accounts instead prove a
	// recent provider sign-in: last_login_at must be within
	// deleteReauthWindow. Token freshness would not do — RefreshToken mints
	// a new access token from a stored refresh token without any provider
	// interaction, while last_login_at only ever advances on real
	// credential/provider sign-ins.
	if user.PasswordHash != "" {
		if !password.Verify(req.Msg.Password, user.PasswordHash) {
			return nil, errInvalidCredentials()
		}
	} else if !h.recentSignIn(user) {
		return nil, newError(connect.CodeFailedPrecondition, ReasonInvalidCredentials,
			"re-authentication required: sign in with the provider again, then retry")
	}
	// App Store review also requires revoking Apple tokens on deletion;
	// best effort before the identities cascade away.
	identities, err := h.store.ListUserIdentities(ctx, project.ID, user.ID)
	if err != nil {
		h.log.ErrorContext(ctx, "list identities before delete", "error", err.Error())
	}
	for _, id := range identities {
		h.revokeAppleRefreshToken(ctx, project, id)
	}
	// Identities, refresh tokens and email tokens cascade with the user
	// row.
	if err := h.store.DeleteUser(ctx, project.ID, user.ID); err != nil {
		return nil, errInternal(err)
	}
	h.insertEvent(ctx, project.ID, user.ID, store.EventUserDeleted)
	return connect.NewResponse(&authv1.DeleteAccountResponse{}), nil
}

// recentSignIn reports whether the user's last credential/provider sign-in
// happened within deleteReauthWindow.
func (h *Handler) recentSignIn(user store.User) bool {
	return user.LastLoginAt != nil && h.now().Sub(*user.LastLoginAt) <= deleteReauthWindow
}
