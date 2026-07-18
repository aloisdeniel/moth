package authrpc

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/gen/moth/auth/v1/authv1connect"
	"github.com/aloisdeniel/moth/internal/push"
)

// Handler also implements moth.auth.v1.ConfigService: the public,
// non-secret project configuration the SDK login screen renders from. It
// runs behind the same publishable-key interceptor as AuthService.
var _ authv1connect.ConfigServiceHandler = (*Handler)(nil)

func (h *Handler) GetProjectConfig(ctx context.Context, req *connect.Request[authv1.GetProjectConfigRequest]) (*connect.Response[authv1.GetProjectConfigResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
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
	// Theme caching contract (see the proto): the body is omitted only when
	// the client already holds the current revision.
	t := publicTheme(project, h.baseURL)
	if req.Msg.KnownThemeRevision != t.RevisionId {
		resp.Theme = t
	}
	// Localized copy for the negotiated locale, same stale-while-revalidate
	// contract as the theme but keyed by (locale, override-revision). The
	// Copy message always ships (it carries the negotiated locale + token);
	// its messages map is omitted when the client's token still matches.
	loc, err := NewLocalizer(ctx, h.store, project.ID, req.Header())
	if err != nil {
		return nil, errInternal(err)
	}
	cp := &authv1.Copy{CopyRevision: loc.Token(), Locale: string(loc.Locale)}
	if req.Msg.KnownCopyRevision != loc.Token() {
		cp.Messages = loc.Messages(ConfigCopyScreens, map[string]string{"app": project.Name})
	}
	resp.Copy = cp
	out := connect.NewResponse(resp)
	out.Header().Set(LocaleHeader, string(loc.Locale))
	return out, nil
}
