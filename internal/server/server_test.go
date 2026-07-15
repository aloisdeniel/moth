package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/config"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/store"
)

type testEnv struct {
	url      string
	client   *http.Client // has a cookie jar: acts like one browser
	sessions adminv1connect.SessionServiceClient
	projects adminv1connect.ProjectServiceClient
	store    *store.Store
}

func newTestEnv(t *testing.T, setupToken string) *testEnv {
	t.Helper()
	dir := t.TempDir()

	st, err := store.Open(filepath.Join(dir, "moth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	master, err := keys.LoadOrCreateMasterKey(dir, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}

	srv := New(Options{
		Config:     config.Config{Addr: ":0", DataDir: dir, BaseURL: "http://localhost:8080"},
		Store:      st,
		Master:     master,
		Logger:     slog.New(slog.DiscardHandler),
		SetupToken: setupToken,
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}
	return &testEnv{
		url:      ts.URL,
		client:   client,
		sessions: adminv1connect.NewSessionServiceClient(client, ts.URL),
		projects: adminv1connect.NewProjectServiceClient(client, ts.URL),
		store:    st,
	}
}

func (e *testEnv) postJSON(t *testing.T, path string, body any) (*http.Response, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	resp, err := e.client.Post(e.url+path, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func (e *testEnv) setup(t *testing.T, token string) {
	t.Helper()
	resp, body := e.postJSON(t, "/admin/setup", map[string]string{
		"token": token, "email": "ops@example.com", "password": "hunter22",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setup failed: %d %v", resp.StatusCode, body)
	}
}

func TestHealthz(t *testing.T) {
	e := newTestEnv(t, "")
	resp, err := e.client.Get(e.url + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz: %d", resp.StatusCode)
	}
}

func TestFirstRunSetupFlow(t *testing.T) {
	e := newTestEnv(t, "setup-token")

	// Status reports setup needed.
	resp, err := e.client.Get(e.url + "/admin/status")
	if err != nil {
		t.Fatal(err)
	}
	var status struct{ NeedsSetup bool }
	json.NewDecoder(resp.Body).Decode(&status)
	resp.Body.Close()
	if !status.NeedsSetup {
		t.Fatal("fresh instance should need setup")
	}

	// Wrong token refused.
	r, _ := e.postJSON(t, "/admin/setup", map[string]string{
		"token": "wrong", "email": "ops@example.com", "password": "hunter22"})
	if r.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong token: want 403, got %d", r.StatusCode)
	}
	// Weak password refused.
	r, _ = e.postJSON(t, "/admin/setup", map[string]string{
		"token": "setup-token", "email": "ops@example.com", "password": "short"})
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("weak password: want 400, got %d", r.StatusCode)
	}

	// Valid setup creates the admin and logs the browser in via cookie.
	e.setup(t, "setup-token")
	who, err := e.sessions.GetCurrentAdmin(context.Background(),
		connect.NewRequest(&adminv1.GetCurrentAdminRequest{}))
	if err != nil {
		t.Fatalf("setup should have set a session cookie: %v", err)
	}
	if who.Msg.Admin.Email != "ops@example.com" {
		t.Fatalf("whoami: %+v", who.Msg.Admin)
	}

	// Setup is one-time: token no longer works.
	r, _ = e.postJSON(t, "/admin/setup", map[string]string{
		"token": "setup-token", "email": "two@example.com", "password": "hunter22"})
	if r.StatusCode == http.StatusOK {
		t.Fatal("setup must not work twice")
	}
}

func TestSessionInterceptorAndLoginLogout(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()

	// A separate client without cookies is rejected.
	anon := adminv1connect.NewProjectServiceClient(http.DefaultClient, e.url)
	_, err := anon.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous RPC: want unauthenticated, got %v", err)
	}

	// Logout invalidates the session server-side.
	if _, err := e.sessions.Logout(ctx, connect.NewRequest(&adminv1.LogoutRequest{})); err != nil {
		t.Fatal(err)
	}
	_, err = e.projects.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("after logout: want unauthenticated, got %v", err)
	}

	// Wrong password refused, right password sets a fresh session cookie.
	_, err = e.sessions.Login(ctx, connect.NewRequest(&adminv1.LoginRequest{
		Email: "ops@example.com", Password: "wrong-password"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("bad login: want unauthenticated, got %v", err)
	}
	_, err = e.sessions.Login(ctx, connect.NewRequest(&adminv1.LoginRequest{
		Email: "OPS@example.com", Password: "hunter22"}))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := e.projects.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{})); err != nil {
		t.Fatalf("authenticated RPC after login: %v", err)
	}
}

func TestProjectLifecycleAndJWKS(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	ctx := context.Background()

	created, err := e.projects.CreateProject(ctx,
		connect.NewRequest(&adminv1.CreateProjectRequest{Name: "My Cool App!"}))
	if err != nil {
		t.Fatal(err)
	}
	p := created.Msg.Project
	if p.Slug != "my-cool-app" {
		t.Errorf("slug: %q", p.Slug)
	}
	if len(created.Msg.SecretKey) < 10 || created.Msg.SecretKey[:3] != "sk_" {
		t.Errorf("secret key: %q", created.Msg.SecretKey)
	}
	if p.PublishableKey[:3] != "pk_" {
		t.Errorf("publishable key: %q", p.PublishableKey)
	}

	// The secret is never returned again.
	got, err := e.projects.GetProject(ctx, connect.NewRequest(&adminv1.GetProjectRequest{Id: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.Project.PublishableKey != p.PublishableKey {
		t.Error("publishable key should be readable")
	}

	// Same name → distinct slug.
	created2, err := e.projects.CreateProject(ctx,
		connect.NewRequest(&adminv1.CreateProjectRequest{Name: "My Cool App"}))
	if err != nil {
		t.Fatal(err)
	}
	if created2.Msg.Project.Slug == p.Slug {
		t.Errorf("slug collision: %q", created2.Msg.Project.Slug)
	}

	// Distinct signing keys per project, and JWKS serves the right one.
	keys1, err := e.store.ListActiveProjectKeys(ctx, p.Id)
	if err != nil || len(keys1) != 1 {
		t.Fatalf("project 1 keys: %v %d", err, len(keys1))
	}
	keys2, _ := e.store.ListActiveProjectKeys(ctx, created2.Msg.Project.Id)
	if keys1[0].Kid == keys2[0].Kid {
		t.Fatal("two projects share a kid")
	}

	resp, err := e.client.Get(fmt.Sprintf("%s/p/%s/.well-known/jwks.json", e.url, p.Slug))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("jwks: %d", resp.StatusCode)
	}
	var jwks struct {
		Keys []struct{ Kid string }
	}
	json.NewDecoder(resp.Body).Decode(&jwks)
	if len(jwks.Keys) != 1 || jwks.Keys[0].Kid != keys1[0].Kid {
		t.Fatalf("jwks content: %+v (want kid %s)", jwks, keys1[0].Kid)
	}

	// Unknown slug → 404.
	r404, err := e.client.Get(e.url + "/p/nope/.well-known/jwks.json")
	if err != nil {
		t.Fatal(err)
	}
	r404.Body.Close()
	if r404.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown project jwks: want 404, got %d", r404.StatusCode)
	}

	// Update + delete.
	upd, err := e.projects.UpdateProject(ctx, connect.NewRequest(&adminv1.UpdateProjectRequest{
		Id: p.Id, Name: "Renamed"}))
	if err != nil || upd.Msg.Project.Name != "Renamed" {
		t.Fatalf("update: %v %+v", err, upd)
	}
	if _, err := e.projects.DeleteProject(ctx, connect.NewRequest(&adminv1.DeleteProjectRequest{Id: p.Id})); err != nil {
		t.Fatal(err)
	}
	_, err = e.projects.GetProject(ctx, connect.NewRequest(&adminv1.GetProjectRequest{Id: p.Id}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("deleted project: want not_found, got %v", err)
	}

	// Empty name rejected.
	_, err = e.projects.CreateProject(ctx, connect.NewRequest(&adminv1.CreateProjectRequest{Name: "  "}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty name: want invalid_argument, got %v", err)
	}
}

func TestAdminPageServed(t *testing.T) {
	e := newTestEnv(t, "")
	resp, err := e.client.Get(e.url + "/admin")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin page: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("content type: %s", ct)
	}
}
