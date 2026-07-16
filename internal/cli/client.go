package cli

import (
	"net/http"
	"strings"
	"time"

	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
)

// Client bundles one connect client per moth.admin.v1 service, all sharing
// an http.Client that authenticates with a personal access token. The CLI
// command groups are thin wrappers over these — the same generated clients
// the admin SPA uses, so the two surfaces cannot diverge in capability.
type Client struct {
	Sessions  adminv1connect.SessionServiceClient
	Projects  adminv1connect.ProjectServiceClient
	Users     adminv1connect.UserServiceClient
	Account   adminv1connect.AdminAccountServiceClient
	Settings  adminv1connect.InstanceSettingsServiceClient
	Analytics adminv1connect.AnalyticsServiceClient
	Themes    adminv1connect.ThemeServiceClient
}

// New builds the admin clients for the server at baseURL, sending
// `authorization: Bearer <pat>` on every request (the credential the admin
// auth interceptor accepts alongside cookie sessions).
func New(baseURL, pat string) *Client {
	base := strings.TrimSuffix(baseURL, "/")
	hc := &http.Client{
		Timeout:   60 * time.Second,
		Transport: &bearerTransport{token: pat, next: http.DefaultTransport},
	}
	return &Client{
		Sessions:  adminv1connect.NewSessionServiceClient(hc, base),
		Projects:  adminv1connect.NewProjectServiceClient(hc, base),
		Users:     adminv1connect.NewUserServiceClient(hc, base),
		Account:   adminv1connect.NewAdminAccountServiceClient(hc, base),
		Settings:  adminv1connect.NewInstanceSettingsServiceClient(hc, base),
		Analytics: adminv1connect.NewAnalyticsServiceClient(hc, base),
		Themes:    adminv1connect.NewThemeServiceClient(hc, base),
	}
}

// bearerTransport attaches the PAT as a Bearer Authorization header.
type bearerTransport struct {
	token string
	next  http.RoundTripper
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.next.RoundTrip(req)
}
