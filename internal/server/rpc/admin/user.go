package adminrpc

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/store"
)

// UserHandler implements moth.admin.v1.UserService — minimal operator
// visibility over a project's end users (full UI in milestone 03).
type UserHandler struct {
	store Store
}

// NewUserHandler builds the admin user service.
func NewUserHandler(st Store) *UserHandler {
	return &UserHandler{store: st}
}

func (h *UserHandler) ListUsers(ctx context.Context, req *connect.Request[adminv1.ListUsersRequest]) (*connect.Response[adminv1.ListUsersResponse], error) {
	if _, err := h.store.GetProject(ctx, req.Msg.ProjectId); err != nil {
		return nil, projectErr(err)
	}
	users, err := h.store.ListUsers(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListUsersResponse{}
	for _, u := range users {
		resp.Users = append(resp.Users, adminUserProto(u))
	}
	return connect.NewResponse(resp), nil
}

func (h *UserHandler) DisableUser(ctx context.Context, req *connect.Request[adminv1.DisableUserRequest]) (*connect.Response[adminv1.DisableUserResponse], error) {
	user, err := h.setDisabled(ctx, req.Msg.ProjectId, req.Msg.UserId, true)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&adminv1.DisableUserResponse{User: adminUserProto(user)}), nil
}

func (h *UserHandler) EnableUser(ctx context.Context, req *connect.Request[adminv1.EnableUserRequest]) (*connect.Response[adminv1.EnableUserResponse], error) {
	user, err := h.setDisabled(ctx, req.Msg.ProjectId, req.Msg.UserId, false)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&adminv1.EnableUserResponse{User: adminUserProto(user)}), nil
}

func (h *UserHandler) setDisabled(ctx context.Context, projectID, userID string, disabled bool) (store.User, error) {
	user, err := h.store.GetUser(ctx, projectID, userID)
	if errors.Is(err, store.ErrNotFound) {
		return store.User{}, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}
	if err != nil {
		return store.User{}, connect.NewError(connect.CodeInternal, err)
	}
	if user.Disabled() == disabled {
		return user, nil
	}
	now := time.Now()
	if disabled {
		user.DisabledAt = &now
	} else {
		user.DisabledAt = nil
	}
	user.UpdatedAt = now
	if err := h.store.UpdateUser(ctx, user); err != nil {
		return store.User{}, connect.NewError(connect.CodeInternal, err)
	}
	if disabled {
		// End every session so the block takes effect immediately.
		if err := h.store.RevokeUserRefreshTokens(ctx, projectID, userID, now); err != nil {
			return store.User{}, connect.NewError(connect.CodeInternal, err)
		}
	}
	return user, nil
}

func adminUserProto(u store.User) *adminv1.User {
	return &adminv1.User{
		Id:            u.ID,
		Email:         u.Email,
		EmailVerified: u.Verified(),
		DisplayName:   u.DisplayName,
		Disabled:      u.Disabled(),
		CreateTime:    timestamppb.New(u.CreatedAt),
	}
}
