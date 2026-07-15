package serverapi

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"connectrpc.com/connect"

	serverv1 "github.com/aloisdeniel/moth/gen/moth/server/v1"
	"github.com/aloisdeniel/moth/internal/password"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// UserHandler implements moth.server.v1.UserService.
type UserHandler struct {
	store Store
	now   func() time.Time
}

// NewUserHandler builds the user management service. now is injectable
// for tests; nil means time.Now.
func NewUserHandler(st Store, now func() time.Time) *UserHandler {
	if now == nil {
		now = time.Now
	}
	return &UserHandler{store: st, now: now}
}

func (h *UserHandler) GetUser(ctx context.Context, req *connect.Request[serverv1.GetUserRequest]) (*connect.Response[serverv1.GetUserResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	user, err := h.store.GetUser(ctx, proj.ID, req.Msg.UserId)
	if err != nil {
		return nil, userErr(err)
	}
	return connect.NewResponse(&serverv1.GetUserResponse{User: userProto(user)}), nil
}

func (h *UserHandler) ListUsers(ctx context.Context, _ *connect.Request[serverv1.ListUsersRequest]) (*connect.Response[serverv1.ListUsersResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	users, err := h.store.ListUsers(ctx, proj.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &serverv1.ListUsersResponse{}
	for _, u := range users {
		resp.Users = append(resp.Users, userProto(u))
	}
	return connect.NewResponse(resp), nil
}

func (h *UserHandler) CreateUser(ctx context.Context, req *connect.Request[serverv1.CreateUserRequest]) (*connect.Response[serverv1.CreateUserResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	email := strings.ToLower(strings.TrimSpace(req.Msg.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("invalid email address"))
	}
	claims, err := claimsJSON(req.Msg.CustomClaims)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	now := h.now()
	user := store.User{
		ID:           authrpc.NewID(),
		ProjectID:    proj.ID,
		Email:        email,
		DisplayName:  req.Msg.DisplayName,
		CustomClaims: claims,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if req.Msg.EmailVerified {
		user.EmailVerifiedAt = &now
	}
	var identities []store.Identity
	if req.Msg.Password != "" {
		if len(req.Msg.Password) < proj.Settings.PasswordMinLength {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("password does not meet the project's minimum length"))
		}
		if user.PasswordHash, err = password.Hash(req.Msg.Password); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		identities = append(identities, store.Identity{
			ID:              authrpc.NewID(),
			ProjectID:       proj.ID,
			UserID:          user.ID,
			Provider:        store.IdentityProviderPassword,
			ProviderSubject: user.ID,
			CreatedAt:       now,
		})
	}
	if err := h.store.CreateUser(ctx, user, identities...); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists,
				errors.New("an account with this email already exists"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&serverv1.CreateUserResponse{User: userProto(user)}), nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *connect.Request[serverv1.UpdateUserRequest]) (*connect.Response[serverv1.UpdateUserResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	user, err := h.store.GetUser(ctx, proj.ID, req.Msg.UserId)
	if err != nil {
		return nil, userErr(err)
	}
	if req.Msg.DisplayName != nil {
		user.DisplayName = *req.Msg.DisplayName
	}
	if req.Msg.AvatarUrl != nil {
		user.AvatarURL = *req.Msg.AvatarUrl
	}
	if req.Msg.CustomClaims != nil {
		if user.CustomClaims, err = claimsJSON(req.Msg.CustomClaims); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	user.UpdatedAt = h.now()
	if err := h.store.UpdateUser(ctx, user); err != nil {
		return nil, userErr(err)
	}
	return connect.NewResponse(&serverv1.UpdateUserResponse{User: userProto(user)}), nil
}

func (h *UserHandler) DisableUser(ctx context.Context, req *connect.Request[serverv1.DisableUserRequest]) (*connect.Response[serverv1.DisableUserResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	user, err := h.store.GetUser(ctx, proj.ID, req.Msg.UserId)
	if err != nil {
		return nil, userErr(err)
	}
	now := h.now()
	if !user.Disabled() {
		user.DisabledAt = &now
		user.UpdatedAt = now
		if err := h.store.UpdateUser(ctx, user); err != nil {
			return nil, userErr(err)
		}
		// End every session so the block takes effect at the next refresh
		// at the latest.
		if err := h.store.RevokeUserRefreshTokens(ctx, proj.ID, user.ID, now); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&serverv1.DisableUserResponse{User: userProto(user)}), nil
}

func (h *UserHandler) EnableUser(ctx context.Context, req *connect.Request[serverv1.EnableUserRequest]) (*connect.Response[serverv1.EnableUserResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	user, err := h.store.GetUser(ctx, proj.ID, req.Msg.UserId)
	if err != nil {
		return nil, userErr(err)
	}
	if user.Disabled() {
		user.DisabledAt = nil
		user.UpdatedAt = h.now()
		if err := h.store.UpdateUser(ctx, user); err != nil {
			return nil, userErr(err)
		}
	}
	return connect.NewResponse(&serverv1.EnableUserResponse{User: userProto(user)}), nil
}

func (h *UserHandler) DeleteUser(ctx context.Context, req *connect.Request[serverv1.DeleteUserRequest]) (*connect.Response[serverv1.DeleteUserResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.store.DeleteUser(ctx, proj.ID, req.Msg.UserId); err != nil {
		return nil, userErr(err)
	}
	return connect.NewResponse(&serverv1.DeleteUserResponse{}), nil
}

func (h *UserHandler) RevokeUserSessions(ctx context.Context, req *connect.Request[serverv1.RevokeUserSessionsRequest]) (*connect.Response[serverv1.RevokeUserSessionsResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	// Confirm the user exists so callers get NotFound instead of silence.
	if _, err := h.store.GetUser(ctx, proj.ID, req.Msg.UserId); err != nil {
		return nil, userErr(err)
	}
	if err := h.store.RevokeUserRefreshTokens(ctx, proj.ID, req.Msg.UserId, h.now()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&serverv1.RevokeUserSessionsResponse{}), nil
}
