package server

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/internal/keys"
)

// TestMetricsExposition drives a few RPCs and asserts the /metrics endpoint
// renders the expected Prometheus families, including the auth-attempt counter
// bumped by a failed SignIn.
func TestMetricsExposition(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "Metrics App")
	ctx := context.Background()

	// A failed sign-in (no such user) must register as an auth attempt.
	_, err := e.authClient(p.PublishableKey).SignIn(ctx,
		connect.NewRequest(&authv1.SignInRequest{Email: "nobody@example.com", Password: "nope-nope-1"}))
	if err == nil {
		t.Fatal("sign-in against an empty project should fail")
	}

	// The endpoint requires the admin credential: an unauthenticated scrape is
	// rejected before any series is rendered.
	noAuth := &http.Client{}
	unauth, err := noAuth.Get(e.url + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /metrics: want 401, got %d", unauth.StatusCode)
	}

	body := e.getMetrics(t)
	for _, want := range []string{
		"moth_rpc_requests_total",
		"moth_rpc_duration_seconds",
		"moth_auth_attempts_total",
		"moth_event_buffer_drops_total",
		"moth_rollup_runs_total",
		`moth_auth_attempts_total{result="failure"}`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics exposition missing %q\n%s", want, body)
		}
	}
	// The failed SignIn RPC itself should be counted by procedure.
	if !strings.Contains(body, "/moth.auth.v1.AuthService/SignIn") {
		t.Errorf("rpc counter missing SignIn procedure:\n%s", body)
	}
}

func (e *testEnv) getMetrics(t *testing.T) string {
	t.Helper()
	resp, err := e.client.Get(e.url + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/metrics: %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	return string(raw)
}

// TestHealthzBrokenDB confirms GET /healthz flips to 503 once the database is
// unreachable (store closed under it).
func TestHealthzBrokenDB(t *testing.T) {
	// The probe is cached for healthCacheTTL, so advance an injectable clock
	// past the window between the healthy and broken checks to bust the cache.
	clock := newFakeClock()
	e := newTestEnv(t, "", func(o *Options) { o.Now = clock.Now })

	resp, err := e.client.Get(e.url + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthy /healthz: want 200, got %d", resp.StatusCode)
	}

	// Break the database.
	if err := e.store.Close(); err != nil {
		t.Fatal(err)
	}
	clock.Advance(healthCacheTTL + time.Second)

	resp, err = e.client.Get(e.url + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("broken-db /healthz: want 503, got %d", resp.StatusCode)
	}
}

// TestHealthChecker unit-tests the gRPC health checker that backs the health
// service: known services report the live probe result, a broken probe yields
// NOT_SERVING, and an unknown service is CodeNotFound.
func TestHealthChecker(t *testing.T) {
	healthy := errNil
	broken := func(context.Context) error { return io.ErrUnexpectedEOF }
	const svc = "moth.auth.v1.AuthService"

	ok := newHealthChecker(healthy, svc)
	resp, err := ok.Check(context.Background(), &grpchealth.CheckRequest{Service: svc})
	if err != nil || resp.Status != grpchealth.StatusServing {
		t.Fatalf("healthy: status=%v err=%v", resp, err)
	}

	down := newHealthChecker(broken, svc)
	resp, err = down.Check(context.Background(), &grpchealth.CheckRequest{Service: svc})
	if err != nil {
		t.Fatalf("broken probe should not error, got %v", err)
	}
	if resp.Status != grpchealth.StatusNotServing {
		t.Fatalf("broken probe: want NOT_SERVING, got %v", resp.Status)
	}

	_, err = ok.Check(context.Background(), &grpchealth.CheckRequest{Service: "does.not.Exist"})
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown service: want not_found, got %v", err)
	}
}

func errNil(context.Context) error { return nil }

// TestSelfCheck covers the startup self-check: the happy path seals and
// re-verifies a canary, a mismatched master key is rejected, a wildly wrong
// clock fails, and a read-only data dir fails.
func TestSelfCheck(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	dirA := t.TempDir()
	keyA, err := keys.LoadOrCreateMasterKey(dirA, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	// First run seals the canary; second run re-verifies it.
	if err := SelfCheck(dirA, keyA, now); err != nil {
		t.Fatalf("first self-check: %v", err)
	}
	if _, err := os.Stat(canaryFile(dirA)); err != nil {
		t.Fatalf("canary not written: %v", err)
	}
	if err := SelfCheck(dirA, keyA, now); err != nil {
		t.Fatalf("second self-check: %v", err)
	}

	// A different master key against dirA's canary must be rejected.
	dirB := t.TempDir()
	keyB, err := keys.LoadOrCreateMasterKey(dirB, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if err := SelfCheck(dirA, keyB, now); err == nil {
		t.Fatal("wrong master key must fail the self-check")
	}

	// A wildly wrong clock is rejected.
	if err := SelfCheck(dirB, keyB, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("bad clock must fail the self-check")
	}

	// A read-only data dir is rejected (skipped when running as root, which
	// bypasses permission bits).
	if os.Geteuid() != 0 && runtime.GOOS != "windows" {
		ro := filepath.Join(t.TempDir(), "readonly")
		if err := os.Mkdir(ro, 0o500); err != nil {
			t.Fatal(err)
		}
		if err := SelfCheck(ro, keyA, now); err == nil {
			t.Fatal("read-only data dir must fail the self-check")
		}
	}
}
