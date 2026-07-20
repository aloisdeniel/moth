package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"

	"github.com/aloisdeniel/moth/gen/moth/auth/v1/authv1connect"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/metrics"
)

// Metrics exposes the instrumentation registry so the serve loop can feed it
// the counters it owns (e.g. rollup runs).
func (s *Server) Metrics() *metrics.Registry { return s.metrics }

// authAttemptProcedures are the credential-facing RPCs counted as
// authentication attempts in moth_auth_attempts_total. The metrics
// interceptor already records every RPC by procedure/code; this narrower
// counter tracks the security-relevant subset regardless of service naming.
var authAttemptProcedures = map[string]struct{}{
	authv1connect.AuthServiceSignInProcedure:            {},
	authv1connect.AuthServiceSignUpProcedure:            {},
	authv1connect.AuthServiceSignInWithOAuthProcedure:   {},
	authv1connect.AuthServiceExchangeOAuthCodeProcedure: {},
}

// newAuthMetricsInterceptor records an auth attempt ("success"/"failure")
// for each credential-facing RPC. It is appended only to the auth service
// chain so non-auth procedures never touch the counter.
func newAuthMetricsInterceptor(reg *metrics.Registry) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			resp, err := next(ctx, req)
			if _, ok := authAttemptProcedures[req.Spec().Procedure]; ok {
				result := "success"
				if err != nil {
					result = "failure"
				}
				reg.IncAuthAttempt(result)
			}
			return resp, err
		}
	}
}

// metricsHandler serves the Prometheus exposition. Before rendering it folds
// the analytics writer's newly dropped-event delta into the registry so
// moth_event_buffer_drops_total reflects buffer overflow without a dedicated
// goroutine — the writer counts drops atomically and this pull reconciles the
// counter at scrape time.
func (s *Server) metricsHandler() http.Handler {
	inner := s.metrics.Handler()
	var (
		mu       sync.Mutex
		lastDrop uint64
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The exposition discloses operational data (auth success/failure
		// counts, per-RPC volume, latency histograms). It is served on the same
		// public listener as everything else, so it requires the admin
		// credential: a session cookie or a personal access token (the usual
		// way a Prometheus scraper authenticates). Unauthenticated scrapes are
		// rejected before any series is rendered.
		if !s.adminHTTPAuthed(r) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeJSONError(w, http.StatusUnauthorized,
				"admin session or personal access token required")
			return
		}
		mu.Lock()
		dropped := s.events.Stats().Dropped
		if dropped > lastDrop {
			s.metrics.AddEventBufferDrops(int(dropped - lastDrop))
			lastDrop = dropped
		}
		mu.Unlock()
		inner.ServeHTTP(w, r)
	})
}

// RollupObserver returns a callback for analytics.Rollup.RunPeriodically that
// records each scheduled run's outcome in moth_rollup_runs_total.
func (s *Server) RollupObserver() func(err error) {
	return func(err error) {
		status := "success"
		if err != nil {
			status = "error"
		}
		s.metrics.IncRollupRun(status)
	}
}

// healthProbe reports whether the instance can serve: the database answers a
// trivial query and the data directory is writable. It is the shared body of
// GET /healthz and the gRPC health service. Both endpoints are unauthenticated
// and unthrottled, so the result is cached for a short window (see healthCache)
// to keep a burst of probes from amplifying into sustained disk writes and DB
// round-trips onto the very disk the probe reports on.
func (s *Server) healthProbe(ctx context.Context) error {
	return s.health.check(ctx)
}

// runHealthProbe performs the actual liveness checks. It is called through the
// cache, never directly.
func (s *Server) runHealthProbe(ctx context.Context) error {
	if _, err := s.store.CountAdmins(ctx); err != nil {
		return fmt.Errorf("database unreachable: %w", err)
	}
	if err := checkWritable(s.cfg.DataDir); err != nil {
		return fmt.Errorf("data dir not writable: %w", err)
	}
	return nil
}

// healthCacheTTL collapses bursts of health probes: within this window a
// repeated probe returns the last result instead of hitting disk and the
// database again.
const healthCacheTTL = 2 * time.Second

// healthCache memoises the health probe for healthCacheTTL. The mutex is held
// across the probe so concurrent callers coalesce onto one in-flight check
// rather than each running their own disk write + query.
type healthCache struct {
	mu    sync.Mutex
	at    time.Time
	err   error
	valid bool
	now   func() time.Time
	probe func(context.Context) error
}

func (c *healthCache) check(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.valid && now.Sub(c.at) < healthCacheTTL {
		return c.err
	}
	c.err = c.probe(ctx)
	c.at = now
	c.valid = true
	return c.err
}

// checkWritable confirms dir accepts a create+write+remove of a probe file.
func checkWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".healthz-*")
	if err != nil {
		return err
	}
	name := f.Name()
	_, werr := f.Write([]byte("ok"))
	cerr := f.Close()
	rerr := os.Remove(name)
	if werr != nil {
		return werr
	}
	if cerr != nil {
		return cerr
	}
	return rerr
}

// healthChecker adapts healthProbe to the gRPC health protocol: known
// services report SERVING/NOT_SERVING from the live probe, unknown services
// yield CodeNotFound (per the health spec).
type healthChecker struct {
	services map[string]struct{}
	probe    func(context.Context) error
}

func newHealthChecker(probe func(context.Context) error, services ...string) *healthChecker {
	set := make(map[string]struct{}, len(services))
	for _, name := range services {
		set[name] = struct{}{}
	}
	return &healthChecker{services: set, probe: probe}
}

func (h *healthChecker) Check(ctx context.Context, req *grpchealth.CheckRequest) (*grpchealth.CheckResponse, error) {
	if req.Service != "" {
		if _, ok := h.services[req.Service]; !ok {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("unknown service %s", req.Service))
		}
	}
	if err := h.probe(ctx); err != nil {
		return &grpchealth.CheckResponse{Status: grpchealth.StatusNotServing}, nil
	}
	return &grpchealth.CheckResponse{Status: grpchealth.StatusServing}, nil
}

// canaryPlaintext is the fixed value sealed under the master key on first run
// and re-verified on every subsequent start.
const canaryPlaintext = "moth-master-key-canary-v1"

// canaryFile is where the encrypted canary lives, next to the master key.
func canaryFile(dataDir string) string {
	return filepath.Join(dataDir, "keys", "master.canary")
}

// SelfCheck validates the preconditions for serving and returns an actionable
// error when one fails: the data directory must be writable, the system clock
// must be sane, and the master key must decrypt the persisted canary. The
// canary is written on first run; on later runs a decrypt failure means the
// supplied MOTH_MASTER_KEY (or master.key) does not match this data directory,
// which would otherwise surface only when a project's private key is touched.
// Run it before ListenAndServe.
func SelfCheck(dataDir string, master keys.MasterKey, now time.Time) error {
	if err := checkWritable(dataDir); err != nil {
		return fmt.Errorf("data dir %q is not writable (check permissions and disk space): %w", dataDir, err)
	}
	// Clock sanity: a grossly wrong clock breaks token expiry, TLS and JWT
	// verification. tzdata is embedded, so this is a lower bound only.
	if now.Year() < 2024 {
		return fmt.Errorf("system clock looks wrong (%s); tokens and TLS will misbehave — sync NTP", now.Format(time.RFC3339))
	}
	return checkMasterKeyCanary(dataDir, master)
}

func checkMasterKeyCanary(dataDir string, master keys.MasterKey) error {
	path := canaryFile(dataDir)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		// First run for this data directory: seal the canary so later starts
		// can detect a swapped key.
		enc, encErr := master.Encrypt([]byte(canaryPlaintext))
		if encErr != nil {
			return fmt.Errorf("master key cannot encrypt canary: %w", encErr)
		}
		if mkErr := os.MkdirAll(filepath.Dir(path), 0o700); mkErr != nil {
			return fmt.Errorf("create keys dir: %w", mkErr)
		}
		if wErr := os.WriteFile(path, enc, 0o600); wErr != nil {
			return fmt.Errorf("write master-key canary: %w", wErr)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read master-key canary: %w", err)
	}
	dec, err := master.Decrypt(raw)
	if err != nil {
		return fmt.Errorf("master key cannot decrypt the stored canary — MOTH_MASTER_KEY does not match this data directory (%s): %w", path, err)
	}
	if string(dec) != canaryPlaintext {
		return fmt.Errorf("master-key canary mismatch — wrong MOTH_MASTER_KEY for this data directory (%s)", path)
	}
	return nil
}
