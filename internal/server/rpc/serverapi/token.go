package serverapi

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	serverv1 "github.com/aloisdeniel/moth/gen/moth/server/v1"
	"github.com/aloisdeniel/moth/internal/jwt"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/store"
)

// Machine-readable causes for inactive introspection results.
const (
	InactiveMalformed        = "MALFORMED"
	InactiveInvalidSignature = "INVALID_SIGNATURE"
	InactiveExpired          = "EXPIRED"
	InactiveUserNotFound     = "USER_NOT_FOUND"
	InactiveUserDisabled     = "USER_DISABLED"
)

// TokenHandler implements moth.server.v1.TokenService.
type TokenHandler struct {
	store Store
	now   func() time.Time
}

// NewTokenHandler builds the token service. now is injectable for tests;
// nil means time.Now.
func NewTokenHandler(st Store, now func() time.Time) *TokenHandler {
	if now == nil {
		now = time.Now
	}
	return &TokenHandler{store: st, now: now}
}

func (h *TokenHandler) IntrospectToken(ctx context.Context, req *connect.Request[serverv1.IntrospectTokenRequest]) (*connect.Response[serverv1.IntrospectTokenResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	inactive := func(reason string) *connect.Response[serverv1.IntrospectTokenResponse] {
		return connect.NewResponse(&serverv1.IntrospectTokenResponse{InactiveReason: reason})
	}

	raw := req.Msg.AccessToken
	unverified, err := jwt.ParseUnverified(raw)
	if err != nil {
		return inactive(InactiveMalformed), nil
	}
	// A token minted for another project must not be introspectable with
	// this project's secret key — that would leak across tenants.
	if unverified.Audience != proj.Slug {
		return nil, connect.NewError(connect.CodePermissionDenied,
			errors.New("token does not belong to this project"))
	}

	claims, err := jwt.Verify(raw, h.publicKeyLookup(ctx, proj.ID), h.now())
	switch {
	case errors.Is(err, jwt.ErrExpired):
		return inactive(InactiveExpired), nil
	case err != nil:
		return inactive(InactiveInvalidSignature), nil
	}

	resp := &serverv1.IntrospectTokenResponse{
		UserId:        claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		IssueTime:     timestamppb.New(time.Unix(claims.IssuedAt, 0)),
		ExpireTime:    timestamppb.New(time.Unix(claims.ExpiresAt, 0)),
	}
	if len(claims.Custom) > 0 {
		if s, err := structpb.NewStruct(claims.Custom); err == nil {
			resp.CustomClaims = s
		}
	}

	// Live state that offline JWKS verification cannot see.
	user, err := h.store.GetUser(ctx, proj.ID, claims.Subject)
	if errors.Is(err, store.ErrNotFound) {
		resp.InactiveReason = InactiveUserNotFound
		return connect.NewResponse(resp), nil
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if user.Disabled() {
		resp.InactiveReason = InactiveUserDisabled
		return connect.NewResponse(resp), nil
	}
	resp.Active = true
	return connect.NewResponse(resp), nil
}

func (h *TokenHandler) publicKeyLookup(ctx context.Context, projectID string) func(kid string) (*ecdsa.PublicKey, error) {
	return func(kid string) (*ecdsa.PublicKey, error) {
		active, err := h.store.ListActiveAndGraceKeys(ctx, projectID, h.now())
		if err != nil {
			return nil, err
		}
		for _, k := range active {
			if k.Kid == kid {
				return keys.ParsePublicKeyPEM(k.PublicKeyPEM)
			}
		}
		return nil, errors.New("unknown kid")
	}
}
