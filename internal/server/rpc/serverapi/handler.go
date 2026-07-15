// Package serverapi implements moth.server.v1 — the services the
// developer's own backend calls with the project secret key (`x-moth-key:
// sk_...`): online token introspection and programmatic user management.
package serverapi

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	serverv1 "github.com/aloisdeniel/moth/gen/moth/server/v1"
	"github.com/aloisdeniel/moth/gen/moth/server/v1/serverv1connect"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// Store is everything the server API needs from persistence.
type Store interface {
	store.ProjectStore
	store.UserStore
	store.RefreshTokenStore
}

var (
	_ serverv1connect.TokenServiceHandler = (*TokenHandler)(nil)
	_ serverv1connect.UserServiceHandler  = (*UserHandler)(nil)
)

// NewSecretKeyInterceptor authenticates moth.server.v1 calls: it resolves
// the project from the secret key in x-moth-key metadata and injects it
// into the context (same context slot as the auth service, so helpers are
// shared).
func NewSecretKeyInterceptor(st store.ProjectStore) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			key := strings.TrimSpace(req.Header().Get(authrpc.KeyHeader))
			if !strings.HasPrefix(key, token.SecretKeyPrefix) {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					errors.New("secret key required in "+authrpc.KeyHeader+" metadata"))
			}
			project, err := st.GetProjectBySecretKeyHash(ctx, token.Hash(key))
			if errors.Is(err, store.ErrNotFound) {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					errors.New("unknown secret key"))
			}
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			return next(authrpc.WithProject(ctx, project), req)
		}
	}
}

// project returns the request's project; the interceptor guarantees it.
func project(ctx context.Context) (store.Project, error) {
	p, ok := authrpc.ProjectFromContext(ctx)
	if !ok {
		return store.Project{}, connect.NewError(connect.CodeInternal,
			errors.New("no project in context"))
	}
	return p, nil
}

func userErr(err error) *connect.Error {
	if errors.Is(err, store.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}
	return connect.NewError(connect.CodeInternal, err)
}

func userProto(u store.User) *serverv1.User {
	return &serverv1.User{
		Id:            u.ID,
		Email:         u.Email,
		EmailVerified: u.Verified(),
		DisplayName:   u.DisplayName,
		AvatarUrl:     u.AvatarURL,
		CustomClaims:  claimsStruct(u.CustomClaims),
		Disabled:      u.Disabled(),
		CreateTime:    timestamppb.New(u.CreatedAt),
		UpdateTime:    timestamppb.New(u.UpdatedAt),
	}
}

// claimsStruct decodes a stored custom_claims JSON object into a proto
// Struct; malformed or empty data becomes nil.
func claimsStruct(raw string) *structpb.Struct {
	if raw == "" || raw == "{}" {
		return nil
	}
	var s structpb.Struct
	if err := s.UnmarshalJSON([]byte(raw)); err != nil {
		return nil
	}
	return &s
}

// claimsJSON encodes a proto Struct into the stored custom_claims JSON.
func claimsJSON(s *structpb.Struct) (string, error) {
	if s == nil || len(s.Fields) == 0 {
		return "{}", nil
	}
	raw, err := s.MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
