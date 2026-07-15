// Package authrpc implements moth.auth.v1.AuthService — the public
// end-user authentication API. Every RPC runs behind an interceptor that
// resolves the project from the `x-moth-key: pk_...` request metadata, so
// all state it touches is project-scoped.
package authrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/gen/moth/auth/v1/authv1connect"
	"github.com/aloisdeniel/moth/internal/jwt"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// Store is everything the auth service needs from persistence.
type Store interface {
	store.ProjectStore
	store.UserStore
	store.RefreshTokenStore
	store.EmailTokenStore
	store.EventStore
}

// Handler implements moth.auth.v1.AuthService.
var _ authv1connect.AuthServiceHandler = (*Handler)(nil)

type Handler struct {
	store   Store
	master  keys.MasterKey
	mailer  mail.Mailer
	baseURL string // no trailing slash; hosted-page links hang off it
	log     *slog.Logger
	now     func() time.Time
}

// Options configures the auth service.
type Options struct {
	Store   Store
	Master  keys.MasterKey
	Mailer  mail.Mailer
	BaseURL string
	Logger  *slog.Logger
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

// New builds the auth service handler.
func New(o Options) *Handler {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return &Handler{
		store:   o.Store,
		master:  o.Master,
		mailer:  o.Mailer,
		baseURL: strings.TrimSuffix(o.BaseURL, "/"),
		log:     o.Logger,
		now:     o.Now,
	}
}

type projectCtxKey struct{}

// WithProject injects the resolved project, done by the publishable-key
// interceptor and by the hosted confirmation pages.
func WithProject(ctx context.Context, p store.Project) context.Context {
	return context.WithValue(ctx, projectCtxKey{}, p)
}

// ProjectFromContext returns the project this request is scoped to.
func ProjectFromContext(ctx context.Context) (store.Project, bool) {
	p, ok := ctx.Value(projectCtxKey{}).(store.Project)
	return p, ok
}

// project returns the request's project; the interceptor guarantees it.
func (h *Handler) project(ctx context.Context) (store.Project, error) {
	p, ok := ProjectFromContext(ctx)
	if !ok {
		return store.Project{}, errInternal(fmt.Errorf("no project in context"))
	}
	return p, nil
}

// requireUser authenticates the current user from the `authorization:
// Bearer ...` access token, scoped to the request's project.
func (h *Handler) requireUser(ctx context.Context, header http.Header) (store.Project, store.User, error) {
	project, err := h.project(ctx)
	if err != nil {
		return store.Project{}, store.User{}, err
	}
	raw, ok := bearerToken(header)
	if !ok {
		return store.Project{}, store.User{}, errInvalidAccessToken("missing bearer access token")
	}
	claims, err := jwt.Verify(raw, h.publicKeyLookup(ctx, project.ID), h.now())
	if err != nil {
		return store.Project{}, store.User{}, errInvalidAccessToken("invalid access token")
	}
	if claims.Audience != project.Slug {
		return store.Project{}, store.User{}, errInvalidAccessToken("access token audience mismatch")
	}
	user, err := h.store.GetUser(ctx, project.ID, claims.Subject)
	if err != nil {
		// A deleted user's token is just an invalid token.
		return store.Project{}, store.User{}, errInvalidAccessToken("invalid access token")
	}
	if user.Disabled() {
		return store.Project{}, store.User{}, errUserDisabled()
	}
	return project, user, nil
}

func bearerToken(header http.Header) (string, bool) {
	auth := header.Get("Authorization")
	const prefix = "Bearer "
	if len(auth) <= len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(auth[len(prefix):]), true
}

// insertEvent writes a stub analytics event; failures are logged, never
// surfaced (real analytics land in milestone 07).
func (h *Handler) insertEvent(ctx context.Context, projectID, userID, eventType string) {
	err := h.store.InsertEvent(ctx, store.Event{
		ID:        NewID(),
		ProjectID: projectID,
		UserID:    userID,
		Type:      eventType,
		Provider:  store.IdentityProviderPassword,
		CreatedAt: h.now(),
	})
	if err != nil {
		h.log.ErrorContext(ctx, "insert event", "type", eventType, "error", err.Error())
	}
}

// send delivers an email; failures are logged, and only reported to the
// caller when report is true (flows that must stay enumeration-safe pass
// false).
func (h *Handler) send(ctx context.Context, m mail.Message, report bool) error {
	if err := h.mailer.Send(ctx, m); err != nil {
		h.log.ErrorContext(ctx, "send email", "subject", m.Subject, "error", err.Error())
		if report {
			return errInternal(fmt.Errorf("send email: %w", err))
		}
	}
	return nil
}

func userProto(u store.User) *authv1.User {
	return &authv1.User{
		Id:            u.ID,
		Email:         u.Email,
		EmailVerified: u.Verified(),
		DisplayName:   u.DisplayName,
		AvatarUrl:     u.AvatarURL,
		CreateTime:    timestamppb.New(u.CreatedAt),
	}
}

// customClaims decodes the user's custom_claims JSON for embedding in the
// JWT; malformed data is treated as no claims.
func customClaims(u store.User) map[string]any {
	if u.CustomClaims == "" || u.CustomClaims == "{}" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(u.CustomClaims), &m); err != nil || len(m) == 0 {
		return nil
	}
	return m
}

// NewID returns a UUIDv7 string (time-sortable primary keys).
func NewID() string {
	id, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("uuidv7: %v", err))
	}
	return id.String()
}

// normalizeEmail lowercases and trims an email address for storage and
// lookups.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// hashToken aliases token.Hash for readability at call sites.
func hashToken(t string) string { return token.Hash(t) }
