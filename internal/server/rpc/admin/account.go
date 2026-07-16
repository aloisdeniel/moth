package adminrpc

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	mailpkg "github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/password"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

const (
	// adminInviteTTL is the lifetime of an operator invite link.
	adminInviteTTL = 72 * time.Hour
	// adminMinPasswordLen matches the first-run setup screen's policy.
	adminMinPasswordLen = 8
)

// AccountHandler implements moth.admin.v1.AdminAccountService.
type AccountHandler struct {
	store   Store
	mailer  mailpkg.Mailer
	baseURL string // no trailing slash; invite links hang off it
	secure  bool   // Secure attribute on the session cookie
	// smtpOn reports whether a real transport (not the console logger) is
	// behind the mailer, so InviteAdmin can say whether the invite was
	// actually emailed.
	smtpOn func() bool
	now    func() time.Time
}

// NewAccountHandler builds the admin account service.
func NewAccountHandler(st Store, mailer mailpkg.Mailer, baseURL string, secure bool, smtpOn func() bool) *AccountHandler {
	return &AccountHandler{
		store:   st,
		mailer:  mailer,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		secure:  secure,
		smtpOn:  smtpOn,
		now:     time.Now,
	}
}

func (h *AccountHandler) ListAdmins(ctx context.Context, _ *connect.Request[adminv1.ListAdminsRequest]) (*connect.Response[adminv1.ListAdminsResponse], error) {
	admins, err := h.store.ListAdmins(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListAdminsResponse{}
	for _, a := range admins {
		resp.Admins = append(resp.Admins, adminProto(a))
	}
	return connect.NewResponse(resp), nil
}

func (h *AccountHandler) InviteAdmin(ctx context.Context, req *connect.Request[adminv1.InviteAdminRequest]) (*connect.Response[adminv1.InviteAdminResponse], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("invalid email address"))
	}
	if _, err := h.store.GetAdminByEmail(ctx, email); err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists,
			errors.New("an admin with this email already exists"))
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	plain := token.Random(32)
	now := h.now()
	inv := store.AdminInvite{
		ID:        NewID(),
		Email:     email,
		TokenHash: token.Hash(plain),
		ExpiresAt: now.Add(adminInviteTTL),
		CreatedAt: now,
	}
	if err := h.store.CreateAdminInvite(ctx, inv); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	inviteURL := h.baseURL + "/admin/invite?token=" + url.QueryEscape(plain)

	emailed := false
	if h.smtpOn() {
		if err := h.mailer.Send(ctx, mailpkg.AdminInvite(email, inviteURL)); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable,
				fmt.Errorf("the invite was created but the email failed: %w", err))
		}
		emailed = true
	}
	return connect.NewResponse(&adminv1.InviteAdminResponse{
		Invite:    adminInviteProto(inv),
		InviteUrl: inviteURL,
		Emailed:   emailed,
	}), nil
}

func (h *AccountHandler) ListAdminInvites(ctx context.Context, _ *connect.Request[adminv1.ListAdminInvitesRequest]) (*connect.Response[adminv1.ListAdminInvitesResponse], error) {
	invites, err := h.store.ListAdminInvites(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListAdminInvitesResponse{}
	for _, inv := range invites {
		resp.Invites = append(resp.Invites, adminInviteProto(inv))
	}
	return connect.NewResponse(resp), nil
}

func (h *AccountHandler) RevokeAdminInvite(ctx context.Context, req *connect.Request[adminv1.RevokeAdminInviteRequest]) (*connect.Response[adminv1.RevokeAdminInviteResponse], error) {
	if err := h.store.DeleteAdminInvite(ctx, req.Msg.Id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("invite not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.RevokeAdminInviteResponse{}), nil
}

func (h *AccountHandler) AcceptAdminInvite(ctx context.Context, req *connect.Request[adminv1.AcceptAdminInviteRequest]) (*connect.Response[adminv1.AcceptAdminInviteResponse], error) {
	invalid := connect.NewError(connect.CodeInvalidArgument,
		errors.New("invalid or expired invite"))
	inv, err := h.store.GetAdminInviteByTokenHash(ctx, token.Hash(req.Msg.Token))
	if errors.Is(err, store.ErrNotFound) {
		return nil, invalid
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	now := h.now()
	if now.After(inv.ExpiresAt) {
		return nil, invalid
	}
	if len(req.Msg.Password) < adminMinPasswordLen {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("password must be at least %d characters", adminMinPasswordLen))
	}
	hash, err := password.Hash(req.Msg.Password)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	admin := store.Admin{
		ID:           NewID(),
		Email:        inv.Email,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.store.CreateAdmin(ctx, admin); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := h.store.DeleteAdminInvite(ctx, inv.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cookie, err := IssueSession(ctx, h.store, admin.ID, h.secure)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := connect.NewResponse(&adminv1.AcceptAdminInviteResponse{Admin: adminProto(admin)})
	resp.Header().Add("Set-Cookie", cookie.String())
	return resp, nil
}

func (h *AccountHandler) ChangePassword(ctx context.Context, req *connect.Request[adminv1.ChangePasswordRequest]) (*connect.Response[adminv1.ChangePasswordResponse], error) {
	admin, ok := AdminFromContext(ctx)
	if !ok {
		return nil, errUnauthenticated()
	}
	if !password.Verify(req.Msg.CurrentPassword, admin.PasswordHash) {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			errors.New("current password is incorrect"))
	}
	if len(req.Msg.NewPassword) < adminMinPasswordLen {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("password must be at least %d characters", adminMinPasswordLen))
	}
	hash, err := password.Hash(req.Msg.NewPassword)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := h.store.UpdateAdminPassword(ctx, admin.ID, hash, h.now()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// End the admin's other browser sessions; the current one stays. Over a
	// PAT there is no session hash in the context, so the empty "current"
	// ends every browser session — rotating a password from the CLI is
	// exactly the "my browser session was stolen" recovery path.
	current, _ := SessionHashFromContext(ctx)
	if err := h.store.DeleteAdminSessionsExcept(ctx, admin.ID, current); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.ChangePasswordResponse{}), nil
}

func adminInviteProto(inv store.AdminInvite) *adminv1.AdminInvite {
	return &adminv1.AdminInvite{
		Id:         inv.ID,
		Email:      inv.Email,
		CreateTime: timestamppb.New(inv.CreatedAt),
		ExpireTime: timestamppb.New(inv.ExpiresAt),
	}
}
