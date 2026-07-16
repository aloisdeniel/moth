package adminrpc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// maxPATNameLen bounds the operator-chosen token label.
const maxPATNameLen = 100

func (h *AccountHandler) CreatePersonalAccessToken(ctx context.Context, req *connect.Request[adminv1.CreatePersonalAccessTokenRequest]) (*connect.Response[adminv1.CreatePersonalAccessTokenResponse], error) {
	admin, ok := AdminFromContext(ctx)
	if !ok {
		return nil, errUnauthenticated()
	}
	name := strings.TrimSpace(req.Msg.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if len(name) > maxPATNameLen {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("name must be at most %d characters", maxPATNameLen))
	}
	if req.Msg.ExpiresInDays < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("expires_in_days must not be negative"))
	}

	plain := token.New(token.PATPrefix)
	now := h.now()
	pat := store.PersonalAccessToken{
		ID:        NewID(),
		AdminID:   admin.ID,
		Name:      name,
		TokenHash: token.Hash(plain),
		CreatedAt: now,
	}
	if d := req.Msg.ExpiresInDays; d > 0 {
		expires := now.AddDate(0, 0, int(d))
		pat.ExpiresAt = &expires
	}
	// A leaked short-lived PAT must not launder itself into a longer-lived
	// credential: tokens minted over a PAT never outlive their creator.
	if cred, ok := CredentialFromContext(ctx); ok && cred.Type == CredentialPAT && cred.PATExpiresAt != nil {
		if pat.ExpiresAt == nil || pat.ExpiresAt.After(*cred.PATExpiresAt) {
			capped := *cred.PATExpiresAt
			pat.ExpiresAt = &capped
		}
	}
	if err := h.store.CreatePAT(ctx, pat); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.audit.record(ctx, entry{
		Action: ActionPATCreate, TargetType: "personal_access_token", TargetID: pat.ID,
		Summary: fmt.Sprintf("Created personal access token %q", pat.Name),
	})
	return connect.NewResponse(&adminv1.CreatePersonalAccessTokenResponse{
		Token:    plain,
		Metadata: PATProto(pat),
	}), nil
}

func (h *AccountHandler) ListPersonalAccessTokens(ctx context.Context, _ *connect.Request[adminv1.ListPersonalAccessTokensRequest]) (*connect.Response[adminv1.ListPersonalAccessTokensResponse], error) {
	admin, ok := AdminFromContext(ctx)
	if !ok {
		return nil, errUnauthenticated()
	}
	pats, err := h.store.ListPATs(ctx, admin.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListPersonalAccessTokensResponse{}
	for _, pat := range pats {
		resp.Tokens = append(resp.Tokens, PATProto(pat))
	}
	return connect.NewResponse(resp), nil
}

func (h *AccountHandler) RevokePersonalAccessToken(ctx context.Context, req *connect.Request[adminv1.RevokePersonalAccessTokenRequest]) (*connect.Response[adminv1.RevokePersonalAccessTokenResponse], error) {
	admin, ok := AdminFromContext(ctx)
	if !ok {
		return nil, errUnauthenticated()
	}
	if err := h.store.RevokePAT(ctx, admin.ID, req.Msg.Id, h.now()); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("token not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.audit.record(ctx, entry{
		Action: ActionPATRevoke, TargetType: "personal_access_token", TargetID: req.Msg.Id,
		Summary: "Revoked a personal access token",
	})
	return connect.NewResponse(&adminv1.RevokePersonalAccessTokenResponse{}), nil
}

// PATProto converts a stored personal access token to its proto metadata
// (exported: the local `moth admin token` commands reuse it for --json).
func PATProto(t store.PersonalAccessToken) *adminv1.PersonalAccessToken {
	msg := &adminv1.PersonalAccessToken{
		Id:         t.ID,
		Name:       t.Name,
		CreateTime: timestamppb.New(t.CreatedAt),
	}
	if t.LastUsedAt != nil {
		msg.LastUsedTime = timestamppb.New(*t.LastUsedAt)
	}
	if t.ExpiresAt != nil {
		msg.ExpireTime = timestamppb.New(*t.ExpiresAt)
	}
	if t.RevokedAt != nil {
		msg.RevokeTime = timestamppb.New(*t.RevokedAt)
	}
	return msg
}
