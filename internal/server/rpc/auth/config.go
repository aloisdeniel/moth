package authrpc

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/gen/moth/auth/v1/authv1connect"
	"github.com/aloisdeniel/moth/internal/push"
	"github.com/aloisdeniel/moth/internal/store"
)

// Handler also implements moth.auth.v1.ConfigService: the public,
// non-secret project configuration the SDK login screen renders from. It
// runs behind the same publishable-key interceptor as AuthService.
var _ authv1connect.ConfigServiceHandler = (*Handler)(nil)

// BuildProjectConfigResponse assembles a project's full public configuration
// (providers, password policy, sign-up state, push, theme and localized copy)
// — the moth.auth.v1 GetProjectConfig payload with every body populated,
// before the caching contract omits anything. GetProjectConfig applies the
// known-revision omission on top; the per-project pub repository reuses it to
// bake the config into a generated SDK package. cs reads the project's copy
// overrides; h carries the requested locale (nil negotiates the project
// default). Returns the response and the negotiated locale.
func BuildProjectConfigResponse(ctx context.Context, cs store.CopyStore, project store.Project, baseURL string, h http.Header) (*authv1.GetProjectConfigResponse, string, error) {
	s := project.Settings
	// Public values only — client IDs are embeddable by design; secrets
	// (Google web client secret, Apple .p8) never appear here.
	resp := &authv1.GetProjectConfigResponse{
		Google: &authv1.GoogleConfig{
			Enabled:         s.Google.Enabled,
			WebClientId:     s.Google.WebClientID,
			IosClientId:     s.Google.IOSClientID,
			AndroidClientId: s.Google.AndroidClientID,
		},
		Apple: &authv1.AppleConfig{
			Enabled: s.Apple.Enabled,
		},
		PasswordMinLength: int32(s.PasswordMinLength),
		SignUpOpen:        s.AllowPublicSignup,
	}
	// Push config (milestone 20): always present, no revision caching — it is
	// two fields, unlike the theme/copy documents. Public values only: the
	// VAPID public key is designed to be embedded in clients.
	pc := push.FromStored(project.Push)
	resp.Push = &authv1.PushConfig{
		Enabled:               pc.Enabled,
		WebpushVapidPublicKey: pc.WebPushVAPIDPublicKey,
	}
	// Full theme body; the caching contract (omit when known_theme_revision
	// matches) is applied by the caller.
	resp.Theme = publicTheme(project, baseURL)
	// Localized copy for the negotiated locale, keyed by (locale,
	// override-revision). The Copy message always carries the locale + token;
	// the messages map is populated here and the caller omits it when the
	// client's token still matches.
	loc, err := NewLocalizer(ctx, cs, project.ID, h)
	if err != nil {
		return nil, "", err
	}
	resp.Copy = &authv1.Copy{
		CopyRevision: loc.Token(),
		Locale:       string(loc.Locale),
		Messages:     loc.Messages(ConfigCopyScreens, map[string]string{"app": project.Name}),
	}
	return resp, string(loc.Locale), nil
}

func (h *Handler) GetProjectConfig(ctx context.Context, req *connect.Request[authv1.GetProjectConfigRequest]) (*connect.Response[authv1.GetProjectConfigResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	resp, locale, err := BuildProjectConfigResponse(ctx, h.store, project, h.baseURL, req.Header())
	if err != nil {
		return nil, errInternal(err)
	}
	// Theme caching contract (see the proto): the body is omitted only when
	// the client already holds the current revision.
	if req.Msg.KnownThemeRevision == resp.Theme.RevisionId {
		resp.Theme = nil
	}
	// Copy caching contract: the Copy message stays (locale + token), but its
	// messages map is dropped when the client's token still matches.
	if req.Msg.KnownCopyRevision == resp.Copy.CopyRevision {
		resp.Copy.Messages = nil
	}
	out := connect.NewResponse(resp)
	out.Header().Set(LocaleHeader, locale)
	return out, nil
}
