package adminrpc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/oidc"
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
	msg, err := h.projectProto(ctx, project, 0)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.CreateProjectResponse{
		Project:   msg,
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
	msg, err := h.projectProto(ctx, p, count)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.GetProjectResponse{Project: msg}), nil
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
		msg, err := h.projectProto(ctx, p, counts[p.ID])
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		resp.Projects = append(resp.Projects, msg)
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
	// Write-only provider secrets ride on the settings message; they are
	// validated with it but persisted (encrypted) only after the project
	// row update succeeds.
	var pendingSecrets map[string]string
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
			settings, err := settingsFromProto(req.Msg.Settings)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			if pendingSecrets, err = providerSecretsFromProto(req.Msg.Settings); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			p.Settings = settings
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("unsupported update_mask path %q", path))
		}
	}
	now := time.Now()
	p.UpdatedAt = now
	if err := h.store.UpdateProject(ctx, p); err != nil {
		return nil, projectErr(err)
	}
	for name, plaintext := range pendingSecrets {
		enc, err := h.master.Encrypt([]byte(plaintext))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := h.store.SetProviderSecret(ctx, p.ID, name, enc, now); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	count, err := h.store.CountUsers(ctx, p.ID, "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	msg, err := h.projectProto(ctx, p, count)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.UpdateProjectResponse{Project: msg}), nil
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
	msg, err := h.projectProto(ctx, p, count)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.RegenerateSecretKeyResponse{
		Project:   msg,
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

func (h *ProjectHandler) projectProto(ctx context.Context, p store.Project, userCount int) (*adminv1.Project, error) {
	settings, err := h.settingsProto(ctx, p.ID, p.Settings)
	if err != nil {
		return nil, err
	}
	return &adminv1.Project{
		Id:             p.ID,
		Name:           p.Name,
		Slug:           p.Slug,
		PublishableKey: p.PublishableKey,
		CreateTime:     timestamppb.New(p.CreatedAt),
		UpdateTime:     timestamppb.New(p.UpdatedAt),
		Settings:       settings,
		UserCount:      int64(userCount),
	}, nil
}

func signingKeyProto(k store.ProjectKey) *adminv1.SigningKey {
	return &adminv1.SigningKey{
		Kid:          k.Kid,
		Algorithm:    k.Algorithm,
		PublicKeyPem: k.PublicKeyPEM,
		CreateTime:   timestamppb.New(k.CreatedAt),
	}
}

// settingsProto builds the admin view of the settings. Stored provider
// secrets are never returned; only their presence is reported (has_*).
func (h *ProjectHandler) settingsProto(ctx context.Context, projectID string, s store.ProjectSettings) (*adminv1.ProjectSettings, error) {
	hasGoogleSecret, err := h.hasProviderSecret(ctx, projectID, store.ProviderSecretGoogleWebClientSecret)
	if err != nil {
		return nil, err
	}
	hasAppleKey, err := h.hasProviderSecret(ctx, projectID, store.ProviderSecretApplePrivateKey)
	if err != nil {
		return nil, err
	}
	autoLink := s.AutoLinkEnabled()
	return &adminv1.ProjectSettings{
		PasswordMinLength:        int32(s.PasswordMinLength),
		RequireEmailVerification: s.RequireEmailVerification,
		AllowPublicSignup:        s.AllowPublicSignup,
		EnumerationSafeSignup:    s.EnumerationSafeSignup,
		AccessTokenTtlSeconds:    int32(s.AccessTokenTTLSeconds),
		RefreshTokenTtlDays:      int32(s.RefreshTokenTTLDays),
		Google: &adminv1.GoogleProviderConfig{
			Enabled:            s.Google.Enabled,
			WebClientId:        s.Google.WebClientID,
			IosClientId:        s.Google.IOSClientID,
			AndroidClientId:    s.Google.AndroidClientID,
			HasWebClientSecret: hasGoogleSecret,
		},
		Apple: &adminv1.AppleProviderConfig{
			Enabled:       s.Apple.Enabled,
			ServicesId:    s.Apple.ServicesID,
			TeamId:        s.Apple.TeamID,
			KeyId:         s.Apple.KeyID,
			HasPrivateKey: hasAppleKey,
			BundleIds:     s.Apple.BundleIDs,
		},
		AutoLinkVerifiedEmail:  &autoLink,
		RedirectSchemes:        s.RedirectSchemes,
		AnalyticsRetentionDays: int32(s.AnalyticsRetentionDays),
		RollupTimezone:         s.RollupTimezone,
	}, nil
}

func (h *ProjectHandler) hasProviderSecret(ctx context.Context, projectID, name string) (bool, error) {
	_, err := h.store.GetProviderSecret(ctx, projectID, name)
	if errors.Is(err, store.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// redirectSchemeRE matches a valid URL scheme (RFC 3986), lowercased.
var redirectSchemeRE = regexp.MustCompile(`^[a-z][a-z0-9+.-]*$`)

// Accepted analytics_retention_days range; the upper bound matches the
// rollup's maximum backfill window.
const (
	minAnalyticsRetentionDays = 1
	maxAnalyticsRetentionDays = 366
)

// settingsFromProto converts and validates the admin message; zero numeric
// fields fall back to defaults when the row is next loaded. Write-only
// secret fields are handled separately (providerSecretsFromProto).
func settingsFromProto(s *adminv1.ProjectSettings) (store.ProjectSettings, error) {
	out := store.ProjectSettings{
		PasswordMinLength:        int(s.PasswordMinLength),
		RequireEmailVerification: s.RequireEmailVerification,
		AllowPublicSignup:        s.AllowPublicSignup,
		EnumerationSafeSignup:    s.EnumerationSafeSignup,
		AccessTokenTTLSeconds:    int(s.AccessTokenTtlSeconds),
		RefreshTokenTTLDays:      int(s.RefreshTokenTtlDays),
		AutoLinkVerifiedEmail:    s.AutoLinkVerifiedEmail,
		AnalyticsRetentionDays:   int(s.AnalyticsRetentionDays),
	}
	// Zero means "default" (90 on the next load); anything else must stay
	// inside the range the rollup can honor — an unbounded value would keep
	// raw per-user events forever, breaking the plan's capped-retention
	// privacy guarantee.
	if d := s.AnalyticsRetentionDays; d != 0 && (d < minAnalyticsRetentionDays || d > maxAnalyticsRetentionDays) {
		return store.ProjectSettings{}, fmt.Errorf("analytics retention must be between %d and %d days",
			minAnalyticsRetentionDays, maxAnalyticsRetentionDays)
	}
	if tz := strings.TrimSpace(s.RollupTimezone); tz != "" {
		if _, err := time.LoadLocation(tz); err != nil {
			return store.ProjectSettings{}, fmt.Errorf("unknown rollup timezone %q", tz)
		}
		out.RollupTimezone = tz
	}
	if g := s.Google; g != nil {
		out.Google = store.GoogleProviderSettings{
			Enabled:         g.Enabled,
			WebClientID:     strings.TrimSpace(g.WebClientId),
			IOSClientID:     strings.TrimSpace(g.IosClientId),
			AndroidClientID: strings.TrimSpace(g.AndroidClientId),
		}
		if g.Enabled && out.Google.WebClientID == "" && out.Google.IOSClientID == "" &&
			out.Google.AndroidClientID == "" {
			return store.ProjectSettings{}, errors.New("enabling Google sign-in requires at least one client ID")
		}
	}
	if a := s.Apple; a != nil {
		out.Apple = store.AppleProviderSettings{
			Enabled:    a.Enabled,
			ServicesID: strings.TrimSpace(a.ServicesId),
			TeamID:     strings.TrimSpace(a.TeamId),
			KeyID:      strings.TrimSpace(a.KeyId),
		}
		for _, id := range a.BundleIds {
			if id = strings.TrimSpace(id); id != "" {
				out.Apple.BundleIDs = append(out.Apple.BundleIDs, id)
			}
		}
		if a.Enabled && out.Apple.ServicesID == "" && len(out.Apple.BundleIDs) == 0 {
			return store.ProjectSettings{}, errors.New("enabling Apple sign-in requires a Services ID or a bundle ID")
		}
	}
	for _, scheme := range s.RedirectSchemes {
		scheme = strings.ToLower(strings.TrimSpace(scheme))
		if scheme == "" {
			continue
		}
		if !redirectSchemeRE.MatchString(scheme) {
			return store.ProjectSettings{}, fmt.Errorf("invalid redirect scheme %q", scheme)
		}
		// The redirect check matches the scheme only, so registering
		// http(s) would let the OAuth callback redirect to any host (open
		// redirect); only custom app schemes are accepted.
		if scheme == "http" || scheme == "https" {
			return store.ProjectSettings{}, fmt.Errorf(
				"redirect scheme %q is not allowed; register the app's custom scheme instead", scheme)
		}
		out.RedirectSchemes = append(out.RedirectSchemes, scheme)
	}
	return out, nil
}

// providerSecretsFromProto extracts and validates the write-only secret
// fields of a settings update: name → plaintext. Empty fields keep the
// stored secret (same convention as the SMTP password).
func providerSecretsFromProto(s *adminv1.ProjectSettings) (map[string]string, error) {
	secrets := map[string]string{}
	if g := s.Google; g != nil && g.WebClientSecret != "" {
		secrets[store.ProviderSecretGoogleWebClientSecret] = g.WebClientSecret
	}
	if a := s.Apple; a != nil && a.PrivateKeyP8 != "" {
		// Reject malformed keys at write time, not at the first Apple code
		// exchange.
		if _, err := oidc.ParseP8([]byte(a.PrivateKeyP8)); err != nil {
			return nil, fmt.Errorf("invalid Apple private key: %w", err)
		}
		secrets[store.ProviderSecretApplePrivateKey] = a.PrivateKeyP8
	}
	return secrets, nil
}
