package adminrpc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/theme"
)

// ThemeHandler implements moth.admin.v1.ThemeService: the per-project
// design system (plan/06). Token sets are validated by internal/theme —
// WCAG AA contrast included — before every save, and every save is a new
// revision (the store keeps the last store.ThemeRevisionKeep for undo).
type ThemeHandler struct {
	store Store
	// uploadsDir is where logo files live:
	// {uploadsDir}/{projectID}/logo-{variant}.{png|svg}, served back at
	// /assets/{projectID}/....
	uploadsDir string
	now        func() time.Time
}

var _ adminv1connect.ThemeServiceHandler = (*ThemeHandler)(nil)

// NewThemeHandler builds the theme service. uploadsDir is created lazily on
// the first logo upload.
func NewThemeHandler(st Store, uploadsDir string) *ThemeHandler {
	return &ThemeHandler{store: st, uploadsDir: uploadsDir, now: time.Now}
}

func (h *ThemeHandler) GetTheme(ctx context.Context, req *connect.Request[adminv1.GetThemeRequest]) (*connect.Response[adminv1.GetThemeResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	t, isDefault := currentTheme(p)
	resp := &adminv1.GetThemeResponse{Theme: themeProto(t), IsDefault: isDefault}
	if !isDefault {
		resp.RevisionId = p.ThemeRevisionID
	}
	return connect.NewResponse(resp), nil
}

func (h *ThemeHandler) UpdateTheme(ctx context.Context, req *connect.Request[adminv1.UpdateThemeRequest]) (*connect.Response[adminv1.UpdateThemeResponse], error) {
	if req.Msg.Theme == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("theme is required"))
	}
	t, rev, err := h.mutateTheme(ctx, req.Msg.ProjectId, func(p store.Project) (theme.Theme, error) {
		t := themeFromProto(req.Msg.Theme)
		// Logo paths are output-only, managed through UploadLogo/DeleteLogo;
		// whatever the client sent is replaced by the current state.
		cur, err := writableTheme(p)
		if err != nil {
			return theme.Theme{}, err
		}
		t.Logo = cur.Logo
		if err := t.Validate(); err != nil {
			return theme.Theme{}, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return t, nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&adminv1.UpdateThemeResponse{
		Theme:      themeProto(t),
		RevisionId: rev,
	}), nil
}

func (h *ThemeHandler) ListThemeRevisions(ctx context.Context, req *connect.Request[adminv1.ListThemeRevisionsRequest]) (*connect.Response[adminv1.ListThemeRevisionsResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	revs, err := h.store.ListThemeRevisions(ctx, p.ID, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListThemeRevisionsResponse{}
	for _, rev := range revs {
		t, err := theme.Parse([]byte(rev.Theme))
		if err != nil {
			// A revision this binary cannot parse (written by a newer
			// schema, or corrupted) is skipped rather than failing the whole
			// list: the remaining revisions must stay restorable.
			continue
		}
		resp.Revisions = append(resp.Revisions, &adminv1.ThemeRevision{
			RevisionId: rev.ID,
			Theme:      themeProto(t),
			CreateTime: timestamppb.New(rev.CreatedAt),
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *ThemeHandler) RestoreThemeRevision(ctx context.Context, req *connect.Request[adminv1.RestoreThemeRevisionRequest]) (*connect.Response[adminv1.RestoreThemeRevisionResponse], error) {
	rev, err := h.store.GetThemeRevision(ctx, req.Msg.ProjectId, req.Msg.RevisionId)
	if errors.Is(err, store.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("theme revision not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	restored, err := theme.Parse([]byte(rev.Theme))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("stored theme revision %s: %w", rev.ID, err))
	}
	// Restore deliberately replaces the current theme whatever its state,
	// so it stays usable as the recovery path for a corrupt document; the
	// mutation never reads the current theme.
	t, newRev, err := h.mutateTheme(ctx, req.Msg.ProjectId, func(p store.Project) (theme.Theme, error) {
		t := restored
		// Logo assets are stored per project, not per revision — a restored
		// theme points at the logo files as they exist on disk today, not at
		// whatever the revision recorded (a re-upload may have changed the
		// file's extension since).
		t.Logo.Light = h.diskLogoPath(p.ID, "light")
		t.Logo.Dark = h.diskLogoPath(p.ID, "dark")
		return t, nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&adminv1.RestoreThemeRevisionResponse{
		Theme:      themeProto(t),
		RevisionId: newRev,
	}), nil
}

func (h *ThemeHandler) ResetTheme(ctx context.Context, req *connect.Request[adminv1.ResetThemeRequest]) (*connect.Response[adminv1.ResetThemeResponse], error) {
	// Logo files stay on disk: the revision history is kept too, and a
	// later restore re-attaches any file that still exists.
	if err := h.store.ClearProjectTheme(ctx, req.Msg.ProjectId, h.now()); err != nil {
		return nil, projectErr(err)
	}
	return connect.NewResponse(&adminv1.ResetThemeResponse{
		Theme: themeProto(theme.Default()),
	}), nil
}

func (h *ThemeHandler) UploadLogo(ctx context.Context, req *connect.Request[adminv1.UploadLogoRequest]) (*connect.Response[adminv1.UploadLogoResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	variant, err := logoVariantName(req.Msg.Variant)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if len(req.Msg.Data) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("logo data is required"))
	}
	if len(req.Msg.Data) > maxLogoBytes {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("logo is %d bytes; at most %d KiB are accepted", len(req.Msg.Data), maxLogoBytes/1024))
	}
	// Decode + re-encode (PNG) or parse + sanitize (SVG): whatever comes
	// out contains only what the pipeline understands, never embedded
	// payloads from the upload.
	clean, ext, err := processLogo(req.Msg.Data, req.Msg.ContentType)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	dir, err := h.projectUploadDir(p.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	name := "logo-" + variant + "." + ext
	if err := os.WriteFile(filepath.Join(dir, name), clean, 0o644); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// A PNG upload replaces an earlier SVG for the same variant (and vice
	// versa); drop the stale file so /assets serves exactly one truth. A
	// missing file is the normal case.
	for _, other := range []string{"png", "svg"} {
		if other != ext {
			_ = os.Remove(filepath.Join(dir, "logo-"+variant+"."+other))
		}
	}

	path := "/assets/" + p.ID + "/" + name
	t, rev, err := h.mutateTheme(ctx, p.ID, func(p store.Project) (theme.Theme, error) {
		t, err := writableTheme(p)
		if err != nil {
			return theme.Theme{}, err
		}
		if variant == "light" {
			t.Logo.Light = path
		} else {
			t.Logo.Dark = path
		}
		return t, nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&adminv1.UploadLogoResponse{
		Theme:      themeProto(t),
		RevisionId: rev,
		Path:       path,
	}), nil
}

func (h *ThemeHandler) DeleteLogo(ctx context.Context, req *connect.Request[adminv1.DeleteLogoRequest]) (*connect.Response[adminv1.DeleteLogoResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	variant, err := logoVariantName(req.Msg.Variant)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	dir, err := h.projectUploadDir(p.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, ext := range []string{"png", "svg"} {
		// Only one extension exists at a time; the other is long gone.
		_ = os.Remove(filepath.Join(dir, "logo-"+variant+"."+ext))
	}
	t, rev, err := h.mutateTheme(ctx, p.ID, func(p store.Project) (theme.Theme, error) {
		t, err := writableTheme(p)
		if err != nil {
			return theme.Theme{}, err
		}
		if variant == "light" {
			t.Logo.Light = ""
		} else {
			t.Logo.Dark = ""
		}
		return t, nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&adminv1.DeleteLogoResponse{
		Theme:      themeProto(t),
		RevisionId: rev,
	}), nil
}

// themeSaveAttempts bounds mutateTheme's compare-and-swap retries; racing
// admin edits resolve in one or two rounds, anything more is a bug.
const themeSaveAttempts = 5

// mutateTheme runs a read-modify-write cycle on the project's theme under
// optimistic concurrency: mut builds the new theme from a fresh project
// read, and the save is retried from scratch when a concurrent save lands
// in between (SetProjectTheme's revision CAS), so racing edits — a logo
// upload against a token save, two uploads — never silently drop each
// other's changes. Returned errors are already connect-coded, as are the
// ones mut returns.
func (h *ThemeHandler) mutateTheme(ctx context.Context, projectID string, mut func(p store.Project) (theme.Theme, error)) (theme.Theme, string, error) {
	for attempt := 1; ; attempt++ {
		p, err := h.store.GetProject(ctx, projectID)
		if err != nil {
			return theme.Theme{}, "", projectErr(err)
		}
		t, err := mut(p)
		if err != nil {
			return theme.Theme{}, "", err
		}
		raw, err := theme.Encode(t)
		if err != nil {
			return theme.Theme{}, "", connect.NewError(connect.CodeInternal, err)
		}
		rev := store.ThemeRevision{
			ID:        NewID(),
			ProjectID: projectID,
			Theme:     string(raw),
			CreatedAt: h.now(),
		}
		err = h.store.SetProjectTheme(ctx, rev, p.ThemeRevisionID)
		if errors.Is(err, store.ErrConflict) {
			if attempt < themeSaveAttempts {
				continue
			}
			return theme.Theme{}, "", connect.NewError(connect.CodeAborted,
				errors.New("theme changed concurrently; retry the edit"))
		}
		if err != nil {
			return theme.Theme{}, "", projectErr(err)
		}
		return t, rev.ID, nil
	}
}

// projectUploadDir returns the directory logo files for projectID live in.
// IDs are server-generated UUIDs, but the value crossed the store, so it is
// still verified to stay inside the uploads root.
func (h *ThemeHandler) projectUploadDir(projectID string) (string, error) {
	if projectID == "" || projectID != filepath.Base(projectID) || !filepath.IsLocal(projectID) {
		return "", fmt.Errorf("unsafe project id %q", projectID)
	}
	return filepath.Join(h.uploadsDir, projectID), nil
}

// diskLogoPath returns the served asset path of the logo file that exists
// on disk today for variant ("light" or "dark"), or "" when none does.
// Logo assets live per project, not per revision, and a re-upload may have
// changed the file's extension since a revision was recorded, so restores
// rebuild the pointer from disk instead of trusting the stored one.
func (h *ThemeHandler) diskLogoPath(projectID, variant string) string {
	dir, err := h.projectUploadDir(projectID)
	if err != nil {
		return ""
	}
	for _, ext := range []string{"png", "svg"} {
		name := "logo-" + variant + "." + ext
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return "/assets/" + projectID + "/" + name
		}
	}
	return ""
}

// currentTheme returns the theme the project renders with and whether it is
// the built-in default (never customized, reset, or — defensively — a
// corrupt stored document). Read paths only; write paths go through
// writableTheme.
func currentTheme(p store.Project) (theme.Theme, bool) {
	if p.Theme == "" {
		return theme.Default(), true
	}
	t, err := theme.Parse([]byte(p.Theme))
	if err != nil {
		return theme.Default(), true
	}
	return t, false
}

// writableTheme is currentTheme for read-modify-write paths: a stored
// document this binary cannot parse (written by a newer schema version, or
// corrupted) must block the write instead of being silently replaced by
// the default — restore a revision or reset the theme to recover.
func writableTheme(p store.Project) (theme.Theme, error) {
	if p.Theme == "" {
		return theme.Default(), nil
	}
	t, err := theme.Parse([]byte(p.Theme))
	if err != nil {
		return theme.Theme{}, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("the stored theme cannot be edited by this server version (%v); restore a revision or reset the theme", err))
	}
	return t, nil
}

func logoVariantName(v adminv1.LogoVariant) (string, error) {
	switch v {
	case adminv1.LogoVariant_LOGO_VARIANT_LIGHT:
		return "light", nil
	case adminv1.LogoVariant_LOGO_VARIANT_DARK:
		return "dark", nil
	default:
		return "", errors.New("variant must be LOGO_VARIANT_LIGHT or LOGO_VARIANT_DARK")
	}
}

func themeProto(t theme.Theme) *adminv1.Theme {
	msg := &adminv1.Theme{
		Colors: themeColorsProto(t.Colors),
		Typography: &adminv1.ThemeTypography{
			FontFamily: t.Typography.FontFamily,
			Scale:      t.Typography.Scale,
		},
		Spacing: &adminv1.ThemeSpacing{Unit: int32(t.Spacing.Unit)},
		Shape:   &adminv1.ThemeShape{CornerRadius: int32(t.Shape.CornerRadius)},
		Logo: &adminv1.ThemeLogo{
			LightPath: t.Logo.Light,
			DarkPath:  t.Logo.Dark,
		},
		Legal: &adminv1.ThemeLegal{
			TermsUrl:   t.Legal.TermsURL,
			PrivacyUrl: t.Legal.PrivacyURL,
		},
	}
	if t.DarkColors != nil {
		msg.DarkColors = &adminv1.ThemeColorOverrides{
			Primary:      t.DarkColors.Primary,
			OnPrimary:    t.DarkColors.OnPrimary,
			Background:   t.DarkColors.Background,
			OnBackground: t.DarkColors.OnBackground,
			Surface:      t.DarkColors.Surface,
			OnSurface:    t.DarkColors.OnSurface,
			Error:        t.DarkColors.Error,
			OnError:      t.DarkColors.OnError,
		}
	}
	return msg
}

func themeColorsProto(c theme.Colors) *adminv1.ThemeColors {
	return &adminv1.ThemeColors{
		Primary:      c.Primary,
		OnPrimary:    c.OnPrimary,
		Background:   c.Background,
		OnBackground: c.OnBackground,
		Surface:      c.Surface,
		OnSurface:    c.OnSurface,
		Error:        c.Error,
		OnError:      c.OnError,
	}
}

// themeFromProto converts the client message into the domain model (nil
// sub-messages become zero values, which Validate rejects with a precise
// message). Logo paths are intentionally not copied — output only.
func themeFromProto(msg *adminv1.Theme) theme.Theme {
	t := theme.Theme{Version: theme.SchemaVersion}
	if c := msg.Colors; c != nil {
		t.Colors = theme.Colors{
			Primary:      c.Primary,
			OnPrimary:    c.OnPrimary,
			Background:   c.Background,
			OnBackground: c.OnBackground,
			Surface:      c.Surface,
			OnSurface:    c.OnSurface,
			Error:        c.Error,
			OnError:      c.OnError,
		}
	}
	if d := msg.DarkColors; d != nil {
		ov := theme.ColorOverrides{
			Primary:      d.Primary,
			OnPrimary:    d.OnPrimary,
			Background:   d.Background,
			OnBackground: d.OnBackground,
			Surface:      d.Surface,
			OnSurface:    d.OnSurface,
			Error:        d.Error,
			OnError:      d.OnError,
		}
		if ov != (theme.ColorOverrides{}) {
			t.DarkColors = &ov
		}
	}
	if ty := msg.Typography; ty != nil {
		t.Typography = theme.Typography{FontFamily: ty.FontFamily, Scale: ty.Scale}
	}
	if sp := msg.Spacing; sp != nil {
		t.Spacing = theme.Spacing{Unit: int(sp.Unit)}
	}
	if sh := msg.Shape; sh != nil {
		t.Shape = theme.Shape{CornerRadius: int(sh.CornerRadius)}
	}
	if l := msg.Legal; l != nil {
		t.Legal = theme.Legal{TermsURL: l.TermsUrl, PrivacyURL: l.PrivacyUrl}
	}
	return t
}
