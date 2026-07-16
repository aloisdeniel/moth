package setup

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/store"
)

// fakeProjects implements the generated ProjectServiceClient over an
// in-memory project list; unimplemented methods panic via the embedded nil
// interface.
type fakeProjects struct {
	adminv1connect.ProjectServiceClient
	projects []*adminv1.Project
	updates  []*adminv1.UpdateProjectRequest
	listErr  error
}

func (f *fakeProjects) ListProjects(context.Context, *connect.Request[adminv1.ListProjectsRequest]) (*connect.Response[adminv1.ListProjectsResponse], error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return connect.NewResponse(&adminv1.ListProjectsResponse{Projects: f.projects}), nil
}

func (f *fakeProjects) UpdateProject(_ context.Context, req *connect.Request[adminv1.UpdateProjectRequest]) (*connect.Response[adminv1.UpdateProjectResponse], error) {
	f.updates = append(f.updates, req.Msg)
	for _, p := range f.projects {
		if p.Id == req.Msg.Id {
			if req.Msg.Settings != nil {
				p.Settings = req.Msg.Settings
			}
			return connect.NewResponse(&adminv1.UpdateProjectResponse{Project: p}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, errors.New("project not found"))
}

// fakeSession implements SessionServiceClient for the doctor tests.
type fakeSession struct {
	adminv1connect.SessionServiceClient
	email   string
	version string
	err     error
}

func (f *fakeSession) GetCurrentAdmin(context.Context, *connect.Request[adminv1.GetCurrentAdminRequest]) (*connect.Response[adminv1.GetCurrentAdminResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&adminv1.GetCurrentAdminResponse{
		Admin:         &adminv1.Admin{Id: "a1", Email: f.email},
		ServerVersion: f.version,
	}), nil
}

// fakeSettings implements InstanceSettingsServiceClient for the doctor
// tests.
type fakeSettings struct {
	adminv1connect.InstanceSettingsServiceClient
	baseURL     string
	source      adminv1.SmtpSource
	host        string
	testEmailTo []string
	sendErr     error
}

func (f *fakeSettings) GetInstanceSettings(context.Context, *connect.Request[adminv1.GetInstanceSettingsRequest]) (*connect.Response[adminv1.GetInstanceSettingsResponse], error) {
	return connect.NewResponse(&adminv1.GetInstanceSettingsResponse{
		BaseUrl:    f.baseURL,
		Version:    "dev",
		Smtp:       &adminv1.SmtpSettings{Host: f.host},
		SmtpSource: f.source,
	}), nil
}

func (f *fakeSettings) SendTestEmail(_ context.Context, req *connect.Request[adminv1.SendTestEmailRequest]) (*connect.Response[adminv1.SendTestEmailResponse], error) {
	f.testEmailTo = append(f.testEmailTo, req.Msg.To)
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	return connect.NewResponse(&adminv1.SendTestEmailResponse{}), nil
}

// testProject returns a project shaped like a real read: every settings
// submessage populated.
func testProject(slug string) *adminv1.Project {
	return &adminv1.Project{
		Id:   "p-" + slug,
		Name: "Project " + slug,
		Slug: slug,
		Settings: &adminv1.ProjectSettings{
			Google: &adminv1.GoogleProviderConfig{},
			Apple:  &adminv1.AppleProviderConfig{},
		},
	}
}

// testP8 generates an EC P-256 key and returns it PEM-encoded as a .p8
// (PKCS#8) plus the parsed key.
func testP8(t *testing.T) ([]byte, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), key
}

// findCheck returns the first check with the given name.
func findCheck(t *testing.T, rep *Report, name string) Check {
	t.Helper()
	for _, c := range rep.Checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no check named %q in %+v", name, rep.Checks)
	return Check{}
}

// The advertised exit-code contract maps connect.CodeNotFound to exit 4;
// a typoed --project slug must carry that code, exactly like
// `moth project get`.
func TestFindProjectBySlugNotFound(t *testing.T) {
	fake := &fakeProjects{projects: []*adminv1.Project{testProject("demo")}}
	_, err := findProjectBySlug(context.Background(), fake, "nope")
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want not_found (err %v)", connect.CodeOf(err), err)
	}
	// The doctor branches on store.ErrNotFound to render a FAIL check.
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err %v should wrap store.ErrNotFound", err)
	}

	p, err := findProjectBySlug(context.Background(), fake, "demo")
	if err != nil || p.Slug != "demo" {
		t.Fatalf("found %v, %v", p, err)
	}
}

// The --json report shape is a scripting contract; keep it byte-stable.
func TestReportJSONGolden(t *testing.T) {
	rep := &Report{}
	rep.Pass("instance: health endpoint", "GET /healthz → 200")
	rep.Fail("project: Google web client ID resolves", "Google does not know 123",
		"re-run `moth setup google`")
	const want = `{
  "status": "FAIL",
  "checks": [
    {
      "name": "instance: health endpoint",
      "status": "PASS",
      "detail": "GET /healthz → 200"
    },
    {
      "name": "project: Google web client ID resolves",
      "status": "FAIL",
      "detail": "Google does not know 123",
      "remediation": "re-run ` + "`moth setup google`" + `"
    }
  ]
}`
	got, err := rep.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("golden mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestReportStatus(t *testing.T) {
	tests := []struct {
		name string
		fill func(*Report)
		want Status
	}{
		{"empty", func(*Report) {}, StatusPass},
		{"all pass", func(r *Report) { r.Pass("a", ""); r.Skip("b", "") }, StatusPass},
		{"warn wins over pass", func(r *Report) { r.Pass("a", ""); r.Warn("b", "", "") }, StatusWarn},
		{"fail wins over warn", func(r *Report) { r.Warn("a", "", ""); r.Fail("b", "", "") }, StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep := &Report{}
			tt.fill(rep)
			if got := rep.Status(); got != tt.want {
				t.Fatalf("Status() = %s, want %s", got, tt.want)
			}
		})
	}
}
