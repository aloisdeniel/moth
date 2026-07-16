package authrpc

import (
	"context"
	"errors"
	"net/mail"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	mailpkg "github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/password"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// dummyHash keeps SignIn timing comparable whether or not the email
// exists.
var dummyHash, _ = password.Hash("moth-no-such-user")

func (h *Handler) SignUp(ctx context.Context, req *connect.Request[authv1.SignUpRequest]) (*connect.Response[authv1.SignUpResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	settings := project.Settings
	if !settings.AllowPublicSignup {
		return nil, newError(connect.CodePermissionDenied, ReasonSignupClosed,
			"public signup is closed for this project")
	}
	email, err := validEmail(req.Msg.Email)
	if err != nil {
		return nil, err
	}
	if err := validPassword(req.Msg.Password, settings); err != nil {
		return nil, err
	}

	now := h.now()
	user := store.User{
		ID:           NewID(),
		ProjectID:    project.ID,
		Email:        email,
		DisplayName:  req.Msg.DisplayName,
		CustomClaims: "{}",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if user.PasswordHash, err = password.Hash(req.Msg.Password); err != nil {
		return nil, errInternal(err)
	}
	identity := store.Identity{
		ID:              NewID(),
		ProjectID:       project.ID,
		UserID:          user.ID,
		Provider:        store.IdentityProviderPassword,
		ProviderSubject: user.ID,
		CreatedAt:       now,
	}
	if err := h.store.CreateUser(ctx, user, identity); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return h.signUpExisting(ctx, project, email)
		}
		return nil, errInternal(err)
	}
	h.insertEvent(ctx, project.ID, user.ID, store.EventUserSignedUp)

	// The verification email goes out on every signup so the address can
	// become verified even when the project does not require it.
	if err := h.sendVerification(ctx, project, user); err != nil {
		return nil, err
	}

	if settings.EnumerationSafeSignup {
		// Response must be indistinguishable from the already-registered
		// case: empty.
		return connect.NewResponse(&authv1.SignUpResponse{}), nil
	}
	resp := &authv1.SignUpResponse{User: userProto(user)}
	if !settings.RequireEmailVerification {
		if resp.Tokens, err = h.issueSession(ctx, project, user, req.Msg.DeviceInfo); err != nil {
			return nil, errInternal(err)
		}
	}
	return connect.NewResponse(resp), nil
}

// signUpExisting handles a signup against an already-registered email.
func (h *Handler) signUpExisting(ctx context.Context, project store.Project, email string) (*connect.Response[authv1.SignUpResponse], error) {
	if !project.Settings.EnumerationSafeSignup {
		return nil, newError(connect.CodeAlreadyExists, ReasonEmailAlreadyExists,
			"an account with this email already exists")
	}
	// Same OK-and-empty response as a fresh signup; the owner gets a
	// "you already have an account" note instead.
	if err := h.send(ctx, mailpkg.AccountExists(project.Name, email), false); err != nil {
		return nil, err
	}
	return connect.NewResponse(&authv1.SignUpResponse{}), nil
}

func (h *Handler) SignIn(ctx context.Context, req *connect.Request[authv1.SignInRequest]) (*connect.Response[authv1.SignInResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	user, err := h.store.GetUserByEmail(ctx, project.ID, normalizeEmail(req.Msg.Email))
	if err != nil || user.PasswordHash == "" {
		// Unknown email and social-only accounts fail exactly like a wrong
		// password, including the hashing cost.
		password.Verify(req.Msg.Password, dummyHash)
		return nil, errInvalidCredentials()
	}
	if !password.Verify(req.Msg.Password, user.PasswordHash) {
		return nil, errInvalidCredentials()
	}
	if user.Disabled() {
		return nil, errUserDisabled()
	}
	if project.Settings.RequireEmailVerification && !user.Verified() {
		return nil, newError(connect.CodeFailedPrecondition, ReasonEmailNotVerified,
			"email address is not verified")
	}

	tokens, err := h.issueSession(ctx, project, user, req.Msg.DeviceInfo)
	if err != nil {
		return nil, errInternal(err)
	}
	if err := h.store.SetUserLastLogin(ctx, project.ID, user.ID, h.now()); err != nil {
		h.log.ErrorContext(ctx, "set last login", "error", err.Error())
	}
	h.insertEvent(ctx, project.ID, user.ID, store.EventUserSignedIn)
	return connect.NewResponse(&authv1.SignInResponse{
		User:   userProto(user),
		Tokens: tokens,
	}), nil
}

func (h *Handler) RefreshToken(ctx context.Context, req *connect.Request[authv1.RefreshTokenRequest]) (*connect.Response[authv1.RefreshTokenResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	invalid := newError(connect.CodeUnauthenticated, ReasonInvalidRefreshToken,
		"invalid refresh token")

	rt, err := h.store.GetRefreshToken(ctx, project.ID, hashToken(req.Msg.RefreshToken))
	if errors.Is(err, store.ErrNotFound) {
		return nil, invalid
	}
	if err != nil {
		return nil, errInternal(err)
	}
	now := h.now()
	if rt.RotatedAt != nil {
		// Reuse of a rotated token is theft evidence: kill the family.
		if err := h.store.RevokeRefreshTokenFamily(ctx, project.ID, rt.FamilyID, now); err != nil {
			return nil, errInternal(err)
		}
		h.log.WarnContext(ctx, "refresh token reuse detected; family revoked",
			"project_id", project.ID, "user_id", rt.UserID, "family_id", rt.FamilyID)
		return nil, newError(connect.CodeUnauthenticated, ReasonRefreshTokenReused,
			"refresh token was already used; all sessions of this device were revoked")
	}
	if !rt.Usable(now) {
		return nil, invalid
	}
	user, err := h.store.GetUser(ctx, project.ID, rt.UserID)
	if err != nil {
		return nil, invalid
	}
	if user.Disabled() {
		return nil, errUserDisabled()
	}

	// Rotate within the family; the sliding window extends from now.
	refresh := token.Random(32)
	successor := store.RefreshToken{
		ID:         NewID(),
		ProjectID:  project.ID,
		UserID:     rt.UserID,
		TokenHash:  hashToken(refresh),
		FamilyID:   rt.FamilyID,
		DeviceInfo: rt.DeviceInfo,
		ExpiresAt:  now.Add(h.refreshTTL(project)),
		CreatedAt:  now,
	}
	if err := h.store.RotateRefreshToken(ctx, rt.ID, now, successor); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// Lost a rotation race — treat exactly like reuse.
			if err := h.store.RevokeRefreshTokenFamily(ctx, project.ID, rt.FamilyID, now); err != nil {
				return nil, errInternal(err)
			}
			return nil, newError(connect.CodeUnauthenticated, ReasonRefreshTokenReused,
				"refresh token was already used; all sessions of this device were revoked")
		}
		return nil, errInternal(err)
	}
	access, expiresIn, err := h.mintAccessToken(ctx, project, user)
	if err != nil {
		return nil, errInternal(err)
	}
	return connect.NewResponse(&authv1.RefreshTokenResponse{
		User: userProto(user),
		Tokens: &authv1.TokenPair{
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresIn:    expiresIn,
		},
	}), nil
}

func (h *Handler) SignOut(ctx context.Context, req *connect.Request[authv1.SignOutRequest]) (*connect.Response[authv1.SignOutResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	// Unknown tokens sign out silently: the caller's goal is already met.
	rt, err := h.store.GetRefreshToken(ctx, project.ID, hashToken(req.Msg.RefreshToken))
	if errors.Is(err, store.ErrNotFound) {
		return connect.NewResponse(&authv1.SignOutResponse{}), nil
	}
	if err != nil {
		return nil, errInternal(err)
	}
	now := h.now()
	if req.Msg.AllDevices {
		err = h.store.RevokeUserRefreshTokens(ctx, project.ID, rt.UserID, now)
	} else {
		err = h.store.RevokeRefreshTokenFamily(ctx, project.ID, rt.FamilyID, now)
	}
	if err != nil {
		return nil, errInternal(err)
	}
	return connect.NewResponse(&authv1.SignOutResponse{}), nil
}

func validEmail(email string) (string, error) {
	email = normalizeEmail(email)
	if _, err := mail.ParseAddress(email); err != nil {
		return "", newError(connect.CodeInvalidArgument, ReasonInvalidEmail,
			"invalid email address")
	}
	return email, nil
}

func validPassword(pw string, settings store.ProjectSettings) error {
	if len(pw) < settings.PasswordMinLength {
		return newError(connect.CodeInvalidArgument, ReasonWeakPassword,
			"password does not meet the project's minimum length")
	}
	return nil
}
