package authrpc

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/gen/moth/auth/v1/authv1connect"
)

// Handler also implements moth.auth.v1.ConfigService: the public,
// non-secret project configuration the SDK login screen renders from. It
// runs behind the same publishable-key interceptor as AuthService.
var _ authv1connect.ConfigServiceHandler = (*Handler)(nil)

func (h *Handler) GetProjectConfig(ctx context.Context, _ *connect.Request[authv1.GetProjectConfigRequest]) (*connect.Response[authv1.GetProjectConfigResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	s := project.Settings
	// Public values only — client IDs are embeddable by design; secrets
	// (Google web client secret, Apple .p8) never appear here.
	return connect.NewResponse(&authv1.GetProjectConfigResponse{
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
	}), nil
}
