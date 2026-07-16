// Package adminrpc implements the moth.admin.v1 connect services.
package adminrpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/password"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// Store is everything the admin services need from persistence.
type Store interface {
	store.AdminStore
	store.AdminInviteStore
	store.SessionStore
	store.InstanceSettingStore
	store.ProjectStore
	store.UserStore
	store.RefreshTokenStore
	store.ProviderSecretStore
	store.ThemeStore
}

// CookieName is the admin session cookie.
const CookieName = "moth_admin_session"

// SessionTTL is how long an admin stays logged in.
const SessionTTL = 7 * 24 * time.Hour

// dummyHash keeps Login timing comparable whether or not the email exists.
var dummyHash, _ = password.Hash("moth-no-such-admin")

// SessionHandler implements moth.admin.v1.SessionService.
type SessionHandler struct {
	store  Store
	secure bool
}

// NewSessionHandler builds the session service. secure controls the Secure
// attribute on cookies (true when the instance is served over https).
func NewSessionHandler(st Store, secure bool) *SessionHandler {
	return &SessionHandler{store: st, secure: secure}
}

func (h *SessionHandler) Login(ctx context.Context, req *connect.Request[adminv1.LoginRequest]) (*connect.Response[adminv1.LoginResponse], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.Email))
	invalid := connect.NewError(connect.CodeUnauthenticated, errors.New("invalid email or password"))

	admin, err := h.store.GetAdminByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		password.Verify(req.Msg.Password, dummyHash)
		return nil, invalid
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !password.Verify(req.Msg.Password, admin.PasswordHash) {
		return nil, invalid
	}

	cookie, err := IssueSession(ctx, h.store, admin.ID, h.secure)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := connect.NewResponse(&adminv1.LoginResponse{Admin: adminProto(admin)})
	resp.Header().Add("Set-Cookie", cookie.String())
	return resp, nil
}

func (h *SessionHandler) Logout(ctx context.Context, req *connect.Request[adminv1.LogoutRequest]) (*connect.Response[adminv1.LogoutResponse], error) {
	if tok, ok := sessionToken(req.Header()); ok {
		if err := h.store.DeleteSession(ctx, token.Hash(tok)); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	resp := connect.NewResponse(&adminv1.LogoutResponse{})
	resp.Header().Add("Set-Cookie", ClearCookie(h.secure).String())
	return resp, nil
}

func (h *SessionHandler) GetCurrentAdmin(ctx context.Context, _ *connect.Request[adminv1.GetCurrentAdminRequest]) (*connect.Response[adminv1.GetCurrentAdminResponse], error) {
	admin, ok := AdminFromContext(ctx)
	if !ok {
		return nil, errUnauthenticated()
	}
	return connect.NewResponse(&adminv1.GetCurrentAdminResponse{Admin: adminProto(admin)}), nil
}

// IssueSession creates a server-side session for adminID and returns the
// cookie to set. Shared with the first-run setup HTTP handler.
func IssueSession(ctx context.Context, st store.SessionStore, adminID string, secure bool) (*http.Cookie, error) {
	tok := token.Random(32)
	now := time.Now()
	if err := st.CreateSession(ctx, store.AdminSession{
		TokenHash: token.Hash(tok),
		AdminID:   adminID,
		CreatedAt: now,
		ExpiresAt: now.Add(SessionTTL),
	}); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return sessionCookie(tok, int(SessionTTL.Seconds()), secure), nil
}

// ClearCookie returns an expired session cookie.
func ClearCookie(secure bool) *http.Cookie {
	return sessionCookie("", -1, secure)
}

func sessionCookie(value string, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// sessionToken extracts the session cookie value from request headers.
func sessionToken(h http.Header) (string, bool) {
	for _, line := range h.Values("Cookie") {
		cookies, err := http.ParseCookie(line)
		if err != nil {
			continue
		}
		for _, c := range cookies {
			if c.Name == CookieName && c.Value != "" {
				return c.Value, true
			}
		}
	}
	return "", false
}

func adminProto(a store.Admin) *adminv1.Admin {
	return &adminv1.Admin{
		Id:         a.ID,
		Email:      a.Email,
		CreateTime: timestamppb.New(a.CreatedAt),
	}
}

// NewID returns a UUIDv7 string (time-sortable primary keys).
func NewID() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Only fails if crypto/rand fails; nothing sensible to do.
		panic(fmt.Sprintf("uuidv7: %v", err))
	}
	return id.String()
}
