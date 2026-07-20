package adminrpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/store"
)

// rotationClockSkew is added to the access-token TTL to compute the default
// grace period of a signing-key rotation: a token minted just before the
// rotation stays verifiable through its whole lifetime plus this margin for
// client/server clock drift.
const rotationClockSkew = 5 * time.Minute

// RotateSigningKey mints a fresh active signing key and moves the current key
// to grace status: it stays in the JWKS until grace_expire_time so in-flight
// access tokens keep validating and no user is signed out. Unlike
// ResetSigningKey it never revokes refresh tokens.
func (h *ProjectHandler) RotateSigningKey(ctx context.Context, req *connect.Request[adminv1.RotateSigningKeyRequest]) (*connect.Response[adminv1.RotateSigningKeyResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	signing, err := keys.GenerateSigningKey(h.master)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	now := time.Now()
	grace := time.Duration(req.Msg.GraceSeconds) * time.Second
	if grace <= 0 {
		grace = time.Duration(p.Settings.AccessTokenTTLSeconds)*time.Second + rotationClockSkew
	}
	graceUntil := now.Add(grace)
	projectKey := store.ProjectKey{
		ID:            NewID(),
		ProjectID:     p.ID,
		Kid:           signing.Kid,
		Algorithm:     signing.Algorithm,
		PublicKeyPEM:  signing.PublicKeyPEM,
		PrivateKeyEnc: signing.PrivateKeyEnc,
		Status:        store.ProjectKeyStatusActive,
		CreatedAt:     now,
	}
	if err := h.store.RotateSigningKey(ctx, p.ID, projectKey, graceUntil, now); err != nil {
		return nil, projectErr(err)
	}
	h.audit.record(ctx, entry{
		Action: ActionSigningKeyRotate, TargetType: "signing_key", TargetID: projectKey.Kid,
		ProjectID: p.ID,
		Summary:   fmt.Sprintf("Rotated the signing key for %q (grace until %s)", p.Name, graceUntil.UTC().Format(time.RFC3339)),
	})
	return connect.NewResponse(&adminv1.RotateSigningKeyResponse{
		Key:             signingKeyProto(projectKey),
		GraceExpireTime: timestamppb.New(graceUntil),
	}), nil
}

// ExportProject returns every user of the project with password hashes and
// provider identities, for migration off moth.
func (h *ProjectHandler) ExportProject(ctx context.Context, req *connect.Request[adminv1.ExportProjectRequest]) (*connect.Response[adminv1.ExportProjectResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	exports, err := h.store.ExportUsers(ctx, p.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ExportProjectResponse{Users: make([]*adminv1.ExportedUser, 0, len(exports))}
	for _, ue := range exports {
		resp.Users = append(resp.Users, exportedUserProto(ue))
	}
	h.audit.record(ctx, entry{
		Action: ActionProjectExport, TargetType: "project", TargetID: p.ID,
		ProjectID: p.ID,
		Summary:   fmt.Sprintf("Exported %d user(s) from %q", len(resp.Users), p.Name),
	})
	return connect.NewResponse(resp), nil
}

// ImportProject bulk-creates users from a migration document, carrying their
// (possibly foreign) password hashes so the first sign-in verifies with the
// original algorithm and rehashes to argon2id. Users whose email already
// exists in the project are skipped.
func (h *ProjectHandler) ImportProject(ctx context.Context, req *connect.Request[adminv1.ImportProjectRequest]) (*connect.Response[adminv1.ImportProjectResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	now := time.Now()
	imports := make([]store.UserImport, 0, len(req.Msg.Users))
	for _, iu := range req.Msg.Users {
		imports = append(imports, userImportFromProto(p.ID, iu, now))
	}
	result, err := h.store.ImportUsers(ctx, p.ID, imports, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.audit.record(ctx, entry{
		Action: ActionProjectImport, TargetType: "project", TargetID: p.ID,
		ProjectID: p.ID,
		Summary:   fmt.Sprintf("Imported %d user(s) into %q (%d skipped)", result.Imported, p.Name, result.Skipped),
	})
	return connect.NewResponse(&adminv1.ImportProjectResponse{
		ImportedCount: int32(result.Imported),
		SkippedCount:  int32(result.Skipped),
	}), nil
}

// exportPasswordAlgorithm maps the stored PasswordAlgo marker to the export
// document's password_algorithm: the empty native marker becomes the explicit
// "argon2id", foreign markers pass through, and passwordless accounts stay
// empty.
func exportPasswordAlgorithm(u store.User) string {
	if u.PasswordHash == "" {
		return ""
	}
	if u.PasswordAlgo == store.PasswordAlgoNative {
		return "argon2id"
	}
	return u.PasswordAlgo
}

func exportedUserProto(ue store.UserExport) *adminv1.ExportedUser {
	u := ue.User
	out := &adminv1.ExportedUser{
		Id:                u.ID,
		Email:             u.Email,
		EmailVerified:     u.Verified(),
		DisplayName:       u.DisplayName,
		AvatarUrl:         u.AvatarURL,
		CustomClaims:      u.CustomClaims,
		Disabled:          u.Disabled(),
		CreateTime:        timestamppb.New(u.CreatedAt),
		PasswordHash:      u.PasswordHash,
		PasswordAlgorithm: exportPasswordAlgorithm(u),
	}
	if u.LastLoginAt != nil {
		out.LastLoginTime = timestamppb.New(*u.LastLoginAt)
	}
	for _, id := range ue.Identities {
		out.Identities = append(out.Identities, &adminv1.ExportedIdentity{
			Provider:        id.Provider,
			ProviderSubject: id.ProviderSubject,
			Email:           id.ProviderEmail,
		})
	}
	return out
}

// importPasswordAlgo maps a document's password_algorithm to the stored
// PasswordAlgo marker: "argon2id" and "" are the native format (empty
// marker), every other value is a foreign hash verified by internal/pwimport
// on first login.
func importPasswordAlgo(algo string) string {
	switch strings.ToLower(strings.TrimSpace(algo)) {
	case "", "argon2id":
		return store.PasswordAlgoNative
	default:
		return strings.ToLower(strings.TrimSpace(algo))
	}
}

func userImportFromProto(projectID string, iu *adminv1.ImportedUser, now time.Time) store.UserImport {
	u := store.User{
		ID:           NewID(),
		ProjectID:    projectID,
		Email:        strings.ToLower(strings.TrimSpace(iu.Email)),
		DisplayName:  iu.DisplayName,
		AvatarURL:    iu.AvatarUrl,
		CustomClaims: iu.CustomClaims,
		PasswordHash: iu.PasswordHash,
		PasswordAlgo: importPasswordAlgo(iu.PasswordAlgorithm),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if iu.EmailVerified {
		u.EmailVerifiedAt = &now
	}
	if iu.Disabled {
		u.DisabledAt = &now
	}
	ui := store.UserImport{User: u}
	for _, id := range iu.Identities {
		ui.Identities = append(ui.Identities, store.Identity{
			ID:              NewID(),
			ProjectID:       projectID,
			UserID:          u.ID,
			Provider:        id.Provider,
			ProviderSubject: id.ProviderSubject,
			ProviderEmail:   id.Email,
			CreatedAt:       now,
		})
	}
	return ui
}
