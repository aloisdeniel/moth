package adminrpc

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	storagev1 "github.com/aloisdeniel/moth/gen/moth/storage/v1"
	"github.com/aloisdeniel/moth/internal/i18n"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// CopyHandler implements moth.admin.v1.CopyService: the per-project
// localization overrides on top of moth's bundled catalog (internal/i18n).
// It mirrors ThemeService — a curated, closed key set with bundled defaults,
// per-project overrides, and versioned revisions — keyed by screen × locale
// instead of by color. Every override save is validated against the catalog
// (known key, required placeholders, length) and recorded as a new revision;
// the store keeps the last store.CopyRevisionKeep for undo.
type CopyHandler struct {
	store Store
	audit *Auditor
	now   func() time.Time
}

var _ adminv1connect.CopyServiceHandler = (*CopyHandler)(nil)

// NewCopyHandler builds the copy service.
func NewCopyHandler(st Store, auditor *Auditor) *CopyHandler {
	return &CopyHandler{store: st, audit: auditor, now: time.Now}
}

// editorScreens are the SDK/paywall surfaces the admin editor exposes, in
// display order. Hosted-page and email catalog keys are localized server-side
// but are not editable in the admin (plan/15), so they are not offered here.
var editorScreens = []i18n.Screen{
	i18n.ScreenSignIn, i18n.ScreenSignUp, i18n.ScreenPasswordReset,
	i18n.ScreenVerifyEmail, i18n.ScreenPaywall,
}

// localeDisplayNames labels the bundled locales for the editor's selector; a
// custom (non-bundled) locale falls back to its raw tag.
var localeDisplayNames = map[i18n.Locale]string{
	"en": "English", "fr": "French", "de": "German", "es": "Spanish",
	"pt": "Portuguese", "it": "Italian", "ja": "Japanese",
}

func localeDisplayName(l i18n.Locale) string {
	if n, ok := localeDisplayNames[l]; ok {
		return n
	}
	return string(l)
}

func i18nScreenToProto(s i18n.Screen) adminv1.CopyScreen {
	switch s {
	case i18n.ScreenSignIn:
		return adminv1.CopyScreen_COPY_SCREEN_SIGN_IN
	case i18n.ScreenSignUp:
		return adminv1.CopyScreen_COPY_SCREEN_SIGN_UP
	case i18n.ScreenPasswordReset:
		return adminv1.CopyScreen_COPY_SCREEN_PASSWORD_RESET
	case i18n.ScreenVerifyEmail:
		return adminv1.CopyScreen_COPY_SCREEN_VERIFY_EMAIL
	case i18n.ScreenPaywall:
		return adminv1.CopyScreen_COPY_SCREEN_PAYWALL
	default:
		return adminv1.CopyScreen_COPY_SCREEN_UNSPECIFIED
	}
}

// screensFor maps a request's CopyScreen to the i18n screens to return:
// UNSPECIFIED returns the whole editable surface, a specific value returns one.
func screensFor(s adminv1.CopyScreen) ([]i18n.Screen, error) {
	if s == adminv1.CopyScreen_COPY_SCREEN_UNSPECIFIED {
		return editorScreens, nil
	}
	for _, sc := range editorScreens {
		if i18nScreenToProto(sc) == s {
			return []i18n.Screen{sc}, nil
		}
	}
	return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown copy screen %v", s))
}

func (h *CopyHandler) GetProjectCopy(ctx context.Context, req *connect.Request[adminv1.GetProjectCopyRequest]) (*connect.Response[adminv1.GetProjectCopyResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	screens, err := screensFor(req.Msg.Screen)
	if err != nil {
		return nil, err
	}
	locale := req.Msg.Locale
	if locale == "" {
		locale = string(authrpc.ProjectDefaultLocale)
	}
	nloc := i18n.NormalizeLocale(locale)

	ov, storeRev, err := h.store.GetProjectCopy(ctx, p.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	overrides := ov[string(nloc)] // raw overrides for this locale, or nil
	// The server resolves copy in three layers (i18n.Resolve): bundled default →
	// project-default-locale override → this-locale override. The default-locale
	// (en) override bleeds into every untranslated locale, so DefaultValue — the
	// effective value when this locale has no override of its own — must include
	// it, or the editor/preview would disagree with what GetProjectConfig ships.
	defLoc := i18n.NormalizeLocale(string(authrpc.ProjectDefaultLocale))
	defOverrides := ov[string(defLoc)]

	resp := &adminv1.GetProjectCopyResponse{
		Locale:     string(nloc),
		RevisionId: storeRev,
		IsDefault:  storeRev == "",
	}
	for _, screen := range screens {
		for _, key := range i18n.ScreenKeys(screen) {
			def, _ := i18n.BundledValue(key, nloc)
			// Layer 2: a non-default locale inherits the default-locale override
			// as its effective default. (For the default locale itself, layer 2
			// is the operator's own override, surfaced separately as OverrideValue.)
			if nloc != defLoc {
				if dv := defOverrides[string(key)]; dv != "" {
					def = dv
				}
			}
			resp.Keys = append(resp.Keys, &adminv1.CopyKey{
				Key:           string(key),
				Screen:        i18nScreenToProto(screen),
				DefaultValue:  def,
				OverrideValue: overrides[string(key)],
				Placeholders:  i18n.RequiredPlaceholders(key),
				MaxLength:     int32(i18n.MaxValueLength),
			})
		}
	}
	return connect.NewResponse(resp), nil
}

func (h *CopyHandler) UpdateProjectCopy(ctx context.Context, req *connect.Request[adminv1.UpdateProjectCopyRequest]) (*connect.Response[adminv1.UpdateProjectCopyResponse], error) {
	if req.Msg.Locale == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("locale is required"))
	}
	nloc := i18n.NormalizeLocale(req.Msg.Locale)
	rev, err := h.store.UpdateProjectCopy(ctx, req.Msg.ProjectId, string(nloc),
		req.Msg.Values, NewID(), h.now(), authrpc.NewCopyValidator())
	if err != nil {
		return nil, copyErr(err)
	}
	h.audit.record(ctx, entry{
		Action: ActionCopyUpdate, TargetType: "copy", TargetID: req.Msg.ProjectId,
		ProjectID: req.Msg.ProjectId,
		Summary:   fmt.Sprintf("Updated the %s copy", nloc),
	})
	return connect.NewResponse(&adminv1.UpdateProjectCopyResponse{RevisionId: rev}), nil
}

func (h *CopyHandler) ResetCopy(ctx context.Context, req *connect.Request[adminv1.ResetCopyRequest]) (*connect.Response[adminv1.ResetCopyResponse], error) {
	if req.Msg.Locale == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("locale is required"))
	}
	nloc := i18n.NormalizeLocale(req.Msg.Locale)
	rev, err := h.store.ResetCopy(ctx, req.Msg.ProjectId, string(nloc), req.Msg.Key, NewID(), h.now())
	if err != nil {
		return nil, copyErr(err)
	}
	summary := fmt.Sprintf("Reset the %s copy to defaults", nloc)
	if req.Msg.Key != "" {
		summary = fmt.Sprintf("Reset %s (%s) to default", req.Msg.Key, nloc)
	}
	h.audit.record(ctx, entry{
		Action: ActionCopyReset, TargetType: "copy", TargetID: req.Msg.ProjectId,
		ProjectID: req.Msg.ProjectId, Summary: summary,
	})
	return connect.NewResponse(&adminv1.ResetCopyResponse{RevisionId: rev}), nil
}

func (h *CopyHandler) RestoreCopyRevision(ctx context.Context, req *connect.Request[adminv1.RestoreCopyRevisionRequest]) (*connect.Response[adminv1.RestoreCopyRevisionResponse], error) {
	rev, err := h.store.RestoreCopyRevision(ctx, req.Msg.ProjectId, req.Msg.RevisionId, NewID(), h.now())
	if errors.Is(err, store.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("copy revision not found"))
	}
	if err != nil {
		return nil, copyErr(err)
	}
	h.audit.record(ctx, entry{
		Action: ActionCopyRestore, TargetType: "copy", TargetID: req.Msg.ProjectId,
		ProjectID: req.Msg.ProjectId,
		Summary:   fmt.Sprintf("Restored copy revision %s", req.Msg.RevisionId),
	})
	return connect.NewResponse(&adminv1.RestoreCopyRevisionResponse{RevisionId: rev}), nil
}

func (h *CopyHandler) ListCopyRevisions(ctx context.Context, req *connect.Request[adminv1.ListCopyRevisionsRequest]) (*connect.Response[adminv1.ListCopyRevisionsResponse], error) {
	revs, err := h.store.ListCopyRevisions(ctx, req.Msg.ProjectId, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListCopyRevisionsResponse{}
	for _, rev := range revs {
		resp.Revisions = append(resp.Revisions, &adminv1.CopyRevision{
			RevisionId: rev.ID,
			CreateTime: timestamppb.New(rev.CreatedAt),
			Locales:    revisionLocales(rev.Copy),
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *CopyHandler) ListLocales(ctx context.Context, req *connect.Request[adminv1.ListLocalesRequest]) (*connect.Response[adminv1.ListLocalesResponse], error) {
	ov, _, err := h.store.GetProjectCopy(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	def := authrpc.ProjectDefaultLocale
	customized := map[i18n.Locale]bool{}
	set := map[i18n.Locale]bool{def: true}
	for locale := range ov {
		n := i18n.NormalizeLocale(locale)
		set[n] = true
		customized[n] = true
	}
	for _, l := range i18n.BundledLocales {
		set[l] = true
	}
	tags := make([]i18n.Locale, 0, len(set))
	for l := range set {
		tags = append(tags, l)
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i] < tags[j] })

	resp := &adminv1.ListLocalesResponse{DefaultLocale: string(def)}
	for _, l := range tags {
		resp.Locales = append(resp.Locales, &adminv1.Locale{
			Tag:         string(l),
			DisplayName: localeDisplayName(l),
			Bundled:     i18n.IsBundled(l),
			Customized:  customized[l],
			IsDefault:   l == def,
		})
	}
	return connect.NewResponse(resp), nil
}

// revisionLocales extracts the locale tags a stored revision document
// (moth.storage.v1.StoredCopy) carries, sorted, for a compact history label.
func revisionLocales(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var doc storagev1.StoredCopy
	if err := proto.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	locales := make([]string, 0, len(doc.GetLocales()))
	for l := range doc.GetLocales() {
		locales = append(locales, l)
	}
	sort.Strings(locales)
	return locales
}

// copyErr maps the copy store's sentinel errors onto connect codes.
func copyErr(err error) error {
	switch {
	case errors.Is(err, store.ErrInvalidCopy):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, store.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New("project not found"))
	case errors.Is(err, store.ErrConflict):
		return connect.NewError(connect.CodeAborted, errors.New("copy changed concurrently; retry the edit"))
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
