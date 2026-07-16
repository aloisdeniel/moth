package adminrpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	mailpkg "github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/password"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

const (
	defaultUserPageSize = 50
	maxUserPageSize     = 200
	// userInviteTTL is the lifetime of a set-password invite link (longer
	// than a self-service reset: the recipient did not just ask for it).
	userInviteTTL = 72 * time.Hour
	// maxCustomClaimsLen bounds the JSON embedded in every JWT.
	maxCustomClaimsLen = 4096
)

// UserHandler implements moth.admin.v1.UserService — the operator's user
// management, a cookie-authed façade over the same domain layer as
// moth.server.v1.
type UserHandler struct {
	store  Store
	auth   *authrpc.Handler // issues hosted-page reset/invite links
	mailer mailpkg.Mailer
	now    func() time.Time
}

// NewUserHandler builds the admin user service.
func NewUserHandler(st Store, auth *authrpc.Handler, mailer mailpkg.Mailer) *UserHandler {
	return &UserHandler{store: st, auth: auth, mailer: mailer, now: time.Now}
}

func (h *UserHandler) ListUsers(ctx context.Context, req *connect.Request[adminv1.ListUsersRequest]) (*connect.Response[adminv1.ListUsersResponse], error) {
	project, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	size := int(req.Msg.PageSize)
	if size <= 0 {
		size = defaultUserPageSize
	}
	if size > maxUserPageSize {
		size = maxUserPageSize
	}
	afterID, err := decodePageToken(req.Msg.PageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	query := strings.TrimSpace(req.Msg.Query)

	// One extra row decides whether a next page exists.
	users, err := h.store.ListUsersPage(ctx, project.ID, store.UserPage{
		Query:   query,
		AfterID: afterID,
		Limit:   size + 1,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	nextToken := ""
	if len(users) > size {
		users = users[:size]
		nextToken = encodePageToken(users[len(users)-1].ID)
	}
	total, err := h.store.CountUsers(ctx, project.ID, query)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	identities, err := h.identitiesFor(ctx, project.ID, users)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &adminv1.ListUsersResponse{
		NextPageToken: nextToken,
		TotalSize:     int64(total),
	}
	for _, u := range users {
		resp.Users = append(resp.Users, adminUserProto(u, identities[u.ID]))
	}
	return connect.NewResponse(resp), nil
}

func (h *UserHandler) GetUser(ctx context.Context, req *connect.Request[adminv1.GetUserRequest]) (*connect.Response[adminv1.GetUserResponse], error) {
	user, err := h.getUser(ctx, req.Msg.ProjectId, req.Msg.UserId)
	if err != nil {
		return nil, err
	}
	identities, err := h.identitiesFor(ctx, user.ProjectID, []store.User{user})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	sessions, err := h.store.ListActiveUserRefreshTokens(ctx, user.ProjectID, user.ID, h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.GetUserResponse{User: adminUserProto(user, identities[user.ID])}
	for _, rt := range sessions {
		resp.Sessions = append(resp.Sessions, &adminv1.UserSession{
			Id:         rt.ID,
			DeviceInfo: rt.DeviceInfo,
			CreateTime: timestamppb.New(rt.CreatedAt),
			ExpireTime: timestamppb.New(rt.ExpiresAt),
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *UserHandler) CreateUser(ctx context.Context, req *connect.Request[adminv1.CreateUserRequest]) (*connect.Response[adminv1.CreateUserResponse], error) {
	project, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	email := strings.ToLower(strings.TrimSpace(req.Msg.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("invalid email address"))
	}
	if req.Msg.Password == "" && !req.Msg.SendInvite {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("set a password or send an invite"))
	}

	now := h.now()
	user := store.User{
		ID:           NewID(),
		ProjectID:    project.ID,
		Email:        email,
		DisplayName:  strings.TrimSpace(req.Msg.DisplayName),
		CustomClaims: "{}",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if req.Msg.EmailVerified {
		user.EmailVerifiedAt = &now
	}
	var identities []store.Identity
	if req.Msg.Password != "" {
		if len(req.Msg.Password) < project.Settings.PasswordMinLength {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("password must be at least %d characters", project.Settings.PasswordMinLength))
		}
		if user.PasswordHash, err = password.Hash(req.Msg.Password); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		identities = append(identities, store.Identity{
			ID:              NewID(),
			ProjectID:       project.ID,
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

	if req.Msg.SendInvite {
		link, err := h.auth.IssuePasswordResetLink(ctx, project, user, userInviteTTL)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := h.mailer.Send(ctx, mailpkg.UserInvite(project.Name, user.Email, link)); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable,
				fmt.Errorf("the account was created but the invite email failed: %w", err))
		}
	}
	return connect.NewResponse(&adminv1.CreateUserResponse{
		User: adminUserProto(user, identities),
	}), nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *connect.Request[adminv1.UpdateUserRequest]) (*connect.Response[adminv1.UpdateUserResponse], error) {
	if req.Msg.UpdateMask == nil || len(req.Msg.UpdateMask.Paths) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("update_mask is required"))
	}
	if req.Msg.User == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("user is required"))
	}
	user, err := h.getUser(ctx, req.Msg.ProjectId, req.Msg.UserId)
	if err != nil {
		return nil, err
	}
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "display_name":
			user.DisplayName = strings.TrimSpace(req.Msg.User.DisplayName)
		case "custom_claims":
			claims, err := validClaims(req.Msg.User.CustomClaims)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			user.CustomClaims = claims
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("unsupported update_mask path %q", path))
		}
	}
	user.UpdatedAt = h.now()
	if err := h.store.UpdateUser(ctx, user); err != nil {
		return nil, userErr(err)
	}
	identities, err := h.identitiesFor(ctx, user.ProjectID, []store.User{user})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.UpdateUserResponse{
		User: adminUserProto(user, identities[user.ID]),
	}), nil
}

func (h *UserHandler) DisableUser(ctx context.Context, req *connect.Request[adminv1.DisableUserRequest]) (*connect.Response[adminv1.DisableUserResponse], error) {
	user, err := h.setDisabled(ctx, req.Msg.ProjectId, req.Msg.UserId, true)
	if err != nil {
		return nil, err
	}
	identities, err := h.identitiesFor(ctx, user.ProjectID, []store.User{user})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.DisableUserResponse{User: adminUserProto(user, identities[user.ID])}), nil
}

func (h *UserHandler) EnableUser(ctx context.Context, req *connect.Request[adminv1.EnableUserRequest]) (*connect.Response[adminv1.EnableUserResponse], error) {
	user, err := h.setDisabled(ctx, req.Msg.ProjectId, req.Msg.UserId, false)
	if err != nil {
		return nil, err
	}
	identities, err := h.identitiesFor(ctx, user.ProjectID, []store.User{user})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.EnableUserResponse{User: adminUserProto(user, identities[user.ID])}), nil
}

func (h *UserHandler) DeleteUser(ctx context.Context, req *connect.Request[adminv1.DeleteUserRequest]) (*connect.Response[adminv1.DeleteUserResponse], error) {
	if _, err := h.store.GetProject(ctx, req.Msg.ProjectId); err != nil {
		return nil, projectErr(err)
	}
	if err := h.store.DeleteUser(ctx, req.Msg.ProjectId, req.Msg.UserId); err != nil {
		return nil, userErr(err)
	}
	return connect.NewResponse(&adminv1.DeleteUserResponse{}), nil
}

func (h *UserHandler) RevokeUserSessions(ctx context.Context, req *connect.Request[adminv1.RevokeUserSessionsRequest]) (*connect.Response[adminv1.RevokeUserSessionsResponse], error) {
	user, err := h.getUser(ctx, req.Msg.ProjectId, req.Msg.UserId)
	if err != nil {
		return nil, err
	}
	now := h.now()
	active, err := h.store.ListActiveUserRefreshTokens(ctx, user.ProjectID, user.ID, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := h.store.RevokeUserRefreshTokens(ctx, user.ProjectID, user.ID, now); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.RevokeUserSessionsResponse{
		RevokedCount: int64(len(active)),
	}), nil
}

func (h *UserHandler) SendPasswordReset(ctx context.Context, req *connect.Request[adminv1.SendPasswordResetRequest]) (*connect.Response[adminv1.SendPasswordResetResponse], error) {
	project, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	user, err := h.getUser(ctx, project.ID, req.Msg.UserId)
	if err != nil {
		return nil, err
	}
	if user.Disabled() {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("the user is disabled"))
	}
	link, err := h.auth.IssuePasswordResetLink(ctx, project, user, authrpc.ResetTokenTTL)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := h.mailer.Send(ctx, mailpkg.PasswordReset(project.Name, user.Email, link)); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable,
			fmt.Errorf("send password reset email: %w", err))
	}
	return connect.NewResponse(&adminv1.SendPasswordResetResponse{}), nil
}

func userErr(err error) *connect.Error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	case errors.Is(err, store.ErrConflict):
		return connect.NewError(connect.CodeAlreadyExists,
			errors.New("an account with this email already exists"))
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func (h *UserHandler) getUser(ctx context.Context, projectID, userID string) (store.User, error) {
	user, err := h.store.GetUser(ctx, projectID, userID)
	if errors.Is(err, store.ErrNotFound) {
		return store.User{}, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}
	if err != nil {
		return store.User{}, connect.NewError(connect.CodeInternal, err)
	}
	return user, nil
}

func (h *UserHandler) setDisabled(ctx context.Context, projectID, userID string, disabled bool) (store.User, error) {
	user, err := h.getUser(ctx, projectID, userID)
	if err != nil {
		return store.User{}, err
	}
	if user.Disabled() == disabled {
		return user, nil
	}
	now := h.now()
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

func (h *UserHandler) identitiesFor(ctx context.Context, projectID string, users []store.User) (map[string][]store.Identity, error) {
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	return h.store.ListIdentitiesForUsers(ctx, projectID, ids)
}

// validClaims checks that raw is a JSON object of reasonable size and
// returns it compacted; "" normalizes to "{}".
func validClaims(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}", nil
	}
	if len(raw) > maxCustomClaimsLen {
		return "", fmt.Errorf("custom claims must be at most %d bytes", maxCustomClaimsLen)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil || m == nil {
		return "", errors.New("custom claims must be a JSON object")
	}
	compact, err := json.Marshal(m)
	if err != nil {
		return "", errors.New("custom claims must be a JSON object")
	}
	return string(compact), nil
}

func encodePageToken(id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(id))
}

func decodePageToken(tok string) (string, error) {
	if tok == "" {
		return "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		return "", errors.New("invalid page_token")
	}
	return string(raw), nil
}

func adminUserProto(u store.User, identities []store.Identity) *adminv1.User {
	providers := make([]string, 0, len(identities))
	for _, id := range identities {
		providers = append(providers, id.Provider)
	}
	msg := &adminv1.User{
		Id:            u.ID,
		Email:         u.Email,
		EmailVerified: u.Verified(),
		DisplayName:   u.DisplayName,
		Disabled:      u.Disabled(),
		CreateTime:    timestamppb.New(u.CreatedAt),
		UpdateTime:    timestamppb.New(u.UpdatedAt),
		Providers:     providers,
		CustomClaims:  u.CustomClaims,
	}
	if u.LastLoginAt != nil {
		msg.LastLoginTime = timestamppb.New(*u.LastLoginAt)
	}
	return msg
}
