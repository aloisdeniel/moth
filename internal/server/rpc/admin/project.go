package adminrpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/keys"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

const maxProjectNameLen = 100

// ProjectHandler implements moth.admin.v1.ProjectService.
type ProjectHandler struct {
	store   Store
	master  keys.MasterKey
	baseURL string // no trailing slash; JWKS/issuer values hang off it
}

// NewProjectHandler builds the project service. The master key encrypts
// each new project's signing key at rest.
func NewProjectHandler(st Store, master keys.MasterKey, baseURL string) *ProjectHandler {
	return &ProjectHandler{store: st, master: master, baseURL: strings.TrimSuffix(baseURL, "/")}
}

func (h *ProjectHandler) CreateProject(ctx context.Context, req *connect.Request[adminv1.CreateProjectRequest]) (*connect.Response[adminv1.CreateProjectResponse], error) {
	name, err := validName(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	slug, err := h.uniqueSlug(ctx, name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	secretKey := token.New(token.SecretKeyPrefix)
	now := time.Now()
	project := store.Project{
		ID:             NewID(),
		Name:           name,
		Slug:           slug,
		PublishableKey: token.New(token.PublishableKeyPrefix),
		SecretKeyHash:  token.Hash(secretKey),
		Settings:       store.DefaultProjectSettings(),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	signing, err := keys.GenerateSigningKey(h.master)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	projectKey := store.ProjectKey{
		ID:            NewID(),
		ProjectID:     project.ID,
		Kid:           signing.Kid,
		Algorithm:     signing.Algorithm,
		PublicKeyPEM:  signing.PublicKeyPEM,
		PrivateKeyEnc: signing.PrivateKeyEnc,
		Status:        store.ProjectKeyStatusActive,
		CreatedAt:     now,
	}

	if err := h.store.CreateProject(ctx, project, projectKey); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.CreateProjectResponse{
		Project:   projectProto(project, 0),
		SecretKey: secretKey,
	}), nil
}

func (h *ProjectHandler) GetProject(ctx context.Context, req *connect.Request[adminv1.GetProjectRequest]) (*connect.Response[adminv1.GetProjectResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.Id)
	if err != nil {
		return nil, projectErr(err)
	}
	count, err := h.store.CountUsers(ctx, p.ID, "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.GetProjectResponse{Project: projectProto(p, count)}), nil
}

func (h *ProjectHandler) ListProjects(ctx context.Context, _ *connect.Request[adminv1.ListProjectsRequest]) (*connect.Response[adminv1.ListProjectsResponse], error) {
	projects, err := h.store.ListProjects(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	counts, err := h.store.CountUsersByProject(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListProjectsResponse{}
	for _, p := range projects {
		resp.Projects = append(resp.Projects, projectProto(p, counts[p.ID]))
	}
	return connect.NewResponse(resp), nil
}

func (h *ProjectHandler) UpdateProject(ctx context.Context, req *connect.Request[adminv1.UpdateProjectRequest]) (*connect.Response[adminv1.UpdateProjectResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.Id)
	if err != nil {
		return nil, projectErr(err)
	}
	paths := []string{"name"}
	if req.Msg.Settings != nil {
		paths = append(paths, "settings")
	}
	if mask := req.Msg.UpdateMask; mask != nil {
		paths = mask.Paths
	}
	for _, path := range paths {
		switch path {
		case "name":
			name, err := validName(req.Msg.Name)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			p.Name = name
		case "settings":
			if req.Msg.Settings == nil {
				return nil, connect.NewError(connect.CodeInvalidArgument,
					errors.New("update_mask names settings but none were provided"))
			}
			p.Settings = settingsFromProto(req.Msg.Settings)
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("unsupported update_mask path %q", path))
		}
	}
	p.UpdatedAt = time.Now()
	if err := h.store.UpdateProject(ctx, p); err != nil {
		return nil, projectErr(err)
	}
	count, err := h.store.CountUsers(ctx, p.ID, "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.UpdateProjectResponse{Project: projectProto(p, count)}), nil
}

func (h *ProjectHandler) RegenerateSecretKey(ctx context.Context, req *connect.Request[adminv1.RegenerateSecretKeyRequest]) (*connect.Response[adminv1.RegenerateSecretKeyResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	secretKey := token.New(token.SecretKeyPrefix)
	now := time.Now()
	if err := h.store.UpdateProjectSecretKey(ctx, p.ID, token.Hash(secretKey), now); err != nil {
		return nil, projectErr(err)
	}
	p.UpdatedAt = now
	count, err := h.store.CountUsers(ctx, p.ID, "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.RegenerateSecretKeyResponse{
		Project:   projectProto(p, count),
		SecretKey: secretKey,
	}), nil
}

func (h *ProjectHandler) GetSigningKey(ctx context.Context, req *connect.Request[adminv1.GetSigningKeyRequest]) (*connect.Response[adminv1.GetSigningKeyResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	active, err := h.store.ListActiveProjectKeys(ctx, p.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if len(active) == 0 {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("project %s has no active signing key", p.ID))
	}
	newest := active[len(active)-1]
	return connect.NewResponse(&adminv1.GetSigningKeyResponse{
		Key:      signingKeyProto(newest),
		JwksUrl:  h.baseURL + "/p/" + p.Slug + "/.well-known/jwks.json",
		Issuer:   authrpc.Issuer(h.baseURL, p.Slug),
		Audience: p.Slug,
	}), nil
}

func (h *ProjectHandler) ResetSigningKey(ctx context.Context, req *connect.Request[adminv1.ResetSigningKeyRequest]) (*connect.Response[adminv1.ResetSigningKeyResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	signing, err := keys.GenerateSigningKey(h.master)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	now := time.Now()
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
	if err := h.store.ResetProjectSigningKey(ctx, p.ID, projectKey, now); err != nil {
		return nil, projectErr(err)
	}
	return connect.NewResponse(&adminv1.ResetSigningKeyResponse{
		Key: signingKeyProto(projectKey),
	}), nil
}

func (h *ProjectHandler) DeleteProject(ctx context.Context, req *connect.Request[adminv1.DeleteProjectRequest]) (*connect.Response[adminv1.DeleteProjectResponse], error) {
	if err := h.store.DeleteProject(ctx, req.Msg.Id); err != nil {
		return nil, projectErr(err)
	}
	return connect.NewResponse(&adminv1.DeleteProjectResponse{}), nil
}

// uniqueSlug derives a URL-safe slug from name, appending -2, -3, ... on
// collision and falling back to a random suffix.
func (h *ProjectHandler) uniqueSlug(ctx context.Context, name string) (string, error) {
	base := Slugify(name)
	slug := base
	for i := 2; ; i++ {
		exists, err := h.store.SlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		if i > 20 {
			return base + "-" + token.Random(4), nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

// Slugify lowercases name and reduces it to [a-z0-9-].
func Slugify(name string) string {
	var b strings.Builder
	lastDash := true // suppress leading dash
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) && r < 128, unicode.IsDigit(r) && r < 128:
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "project"
	}
	return slug
}

func validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("name is required")
	}
	if len(name) > maxProjectNameLen {
		return "", fmt.Errorf("name must be at most %d characters", maxProjectNameLen)
	}
	return name, nil
}

func projectErr(err error) *connect.Error {
	if errors.Is(err, store.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, errors.New("project not found"))
	}
	return connect.NewError(connect.CodeInternal, err)
}

func projectProto(p store.Project, userCount int) *adminv1.Project {
	return &adminv1.Project{
		Id:             p.ID,
		Name:           p.Name,
		Slug:           p.Slug,
		PublishableKey: p.PublishableKey,
		CreateTime:     timestamppb.New(p.CreatedAt),
		UpdateTime:     timestamppb.New(p.UpdatedAt),
		Settings:       settingsProto(p.Settings),
		UserCount:      int64(userCount),
	}
}

func signingKeyProto(k store.ProjectKey) *adminv1.SigningKey {
	return &adminv1.SigningKey{
		Kid:          k.Kid,
		Algorithm:    k.Algorithm,
		PublicKeyPem: k.PublicKeyPEM,
		CreateTime:   timestamppb.New(k.CreatedAt),
	}
}

func settingsProto(s store.ProjectSettings) *adminv1.ProjectSettings {
	return &adminv1.ProjectSettings{
		PasswordMinLength:        int32(s.PasswordMinLength),
		RequireEmailVerification: s.RequireEmailVerification,
		AllowPublicSignup:        s.AllowPublicSignup,
		EnumerationSafeSignup:    s.EnumerationSafeSignup,
		AccessTokenTtlSeconds:    int32(s.AccessTokenTTLSeconds),
		RefreshTokenTtlDays:      int32(s.RefreshTokenTTLDays),
	}
}

// settingsFromProto converts the admin message; zero numeric fields fall
// back to defaults when the row is next loaded.
func settingsFromProto(s *adminv1.ProjectSettings) store.ProjectSettings {
	return store.ProjectSettings{
		PasswordMinLength:        int(s.PasswordMinLength),
		RequireEmailVerification: s.RequireEmailVerification,
		AllowPublicSignup:        s.AllowPublicSignup,
		EnumerationSafeSignup:    s.EnumerationSafeSignup,
		AccessTokenTTLSeconds:    int(s.AccessTokenTtlSeconds),
		RefreshTokenTTLDays:      int(s.RefreshTokenTtlDays),
	}
}
