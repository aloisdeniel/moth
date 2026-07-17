package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/aloisdeniel/moth/internal/acme"
	"github.com/aloisdeniel/moth/internal/backup"
	"github.com/aloisdeniel/moth/internal/config"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/server"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
	"github.com/aloisdeniel/moth/internal/version"
)

func newServeCmd() *cobra.Command {
	var flags rootFlags
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the moth server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := resolveConfig(cmd, &flags)
			if err != nil {
				return err
			}
			return serve(cmd.Context(), cfg)
		},
	}
	addConfigFlags(cmd, &flags)
	addServeOpsFlags(cmd, &flags)
	return cmd
}

func serve(ctx context.Context, cfg config.Config) error {
	log := slog.New(newLogHandler(os.Stderr, cfg.LogFormat))

	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	st, err := store.Open(filepath.Join(cfg.DataDir, "moth.db"))
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return err
	}
	if err := st.DeleteExpiredSessions(ctx, time.Now()); err != nil {
		return err
	}
	if err := st.DeleteExpiredOAuthTokens(ctx, time.Now()); err != nil {
		return err
	}
	if err := st.DeleteExpiredPATs(ctx, time.Now()); err != nil {
		return err
	}

	master, err := keys.LoadOrCreateMasterKey(cfg.DataDir, os.Getenv)
	if err != nil {
		return err
	}

	// Fail loudly before accepting traffic if a precondition is wrong: a
	// read-only data dir, a badly skewed clock, or a master key that cannot
	// round-trip (a wrong MOTH_MASTER_KEY would otherwise surface only when
	// the first project key is decrypted).
	if err := server.SelfCheck(cfg.DataDir, master, time.Now()); err != nil {
		return fmt.Errorf("startup self-check failed: %w", err)
	}

	adminCount, err := st.CountAdmins(ctx)
	if err != nil {
		return err
	}
	setupToken := ""
	if adminCount == 0 {
		setupToken = token.Random(16)
	}

	// SMTP resolution (database settings > config > console) happens
	// inside server.New, which also lets the admin console reconfigure it
	// at runtime.
	if cfg.SMTP.Enabled() {
		log.Info("smtp transport configured", "host", cfg.SMTP.Host, "port", cfg.SMTP.Port)
	} else {
		log.Info("no smtp in config; console transport unless configured in the admin")
	}

	srv, err := server.New(server.Options{
		Config:     cfg,
		Store:      st,
		Master:     master,
		Logger:     log,
		SetupToken: setupToken,
		Reflection: cfg.Reflection,
		// Testing hook (env-only, deliberately not part of the resolved
		// config or user docs, like MOTH_MASTER_KEY): points the billing
		// engine's Stripe API calls at a local test double so the SDK e2e
		// suites can drive the full checkout/webhook loop against a spawned
		// binary. Empty means the real https://api.stripe.com.
		StripeBaseURL: os.Getenv("MOTH_STRIPE_API_URL"),
	})
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		Protocols:         server.Protocols(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Built-in ACME/Let's Encrypt: obtain and renew a certificate straight from
	// the binary, serve HTTPS on :443 and answer http-01 challenges (plus
	// redirect plain HTTP) on :80. challengeServer is nil when ACME is off.
	var challengeServer *http.Server
	if len(cfg.AcmeDomains) > 0 {
		mgr, err := acme.Manager(cfg.DataDir, cfg.AcmeDomains...)
		if err != nil {
			return err
		}
		httpServer.Addr = ":443"
		httpServer.TLSConfig = acme.TLSConfig(mgr)
		// A TLS listener negotiates encrypted HTTP/2 via ALPN, so drop the h2c
		// (unencrypted HTTP/2) protocol set used on the plain-HTTP path and let
		// the server auto-configure HTTP/1.1 + HTTP/2 over TLS.
		httpServer.Protocols = nil
		challengeServer = &http.Server{
			Addr:              ":80",
			Handler:           mgr.HTTPHandler(nil),
			ReadHeaderTimeout: 10 * time.Second,
		}
		log.Info("acme enabled", "domains", cfg.AcmeDomains, "https_addr", httpServer.Addr, "challenge_addr", challengeServer.Addr)
	}

	log.Info("moth serving", "version", version.Version, "addr", httpServer.Addr, "base_url", cfg.BaseURL, "data_dir", cfg.DataDir)
	if setupToken != "" {
		fmt.Fprintf(os.Stderr, "\n  No admin account exists yet. Create one at:\n\n    %s/admin?setup=%s\n\n", cfg.BaseURL, setupToken)
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Periodic sweep of expired single-use rows: the unauthenticated
	// /oauth/{provider}/start endpoint inserts a state row per hit and
	// consumed codes/states are kept until expiry, so a long-running
	// instance must not rely on the startup sweep alone.
	go sweepExpired(ctx, st, log)

	// Analytics rollup: hourly with jitter, processing only completed local
	// days (see analytics.RunPeriodically) and pruning expired raw events.
	// The observer feeds moth_rollup_runs_total.
	go srv.Rollup().RunPeriodically(ctx, srv.RollupObserver())

	// Subscription reconciliation: re-read store state for subscriptions whose
	// paid period lapsed while still marked active, catching missed store
	// notifications.
	go reconcileBilling(ctx, srv, log)

	// Scheduled local backups when a backup directory is configured.
	if cfg.BackupDir != "" {
		go scheduledBackup(ctx, cfg, log)
	}

	errCh := make(chan error, 2)
	if challengeServer != nil {
		// ListenAndServeTLS with empty cert/key paths uses the manager's
		// GetCertificate from TLSConfig.
		go func() { errCh <- httpServer.ListenAndServeTLS("", "") }()
		go func() { errCh <- challengeServer.ListenAndServe() }()
	} else {
		go func() { errCh <- httpServer.ListenAndServe() }()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if challengeServer != nil {
			if err := challengeServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
				log.Warn("acme challenge server shutdown", "error", err.Error())
			}
		}
		if err := httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		// Drain the buffered analytics events once no request can emit more.
		if err := srv.Close(shutdownCtx); err != nil {
			log.Warn("analytics event writer drain", "error", err.Error())
		}
		return nil
	}
}

// reconcileBillingInterval is how often the subscription reconciliation sweep
// runs. Renewals fire at most daily, so an hourly sweep with the webhook path
// as the fast lane is ample.
const reconcileBillingInterval = time.Hour

// reconcileBilling periodically re-validates subscriptions near expiry to catch
// missed store notifications, until ctx is done. Failures are logged, never
// fatal.
func reconcileBilling(ctx context.Context, srv *server.Server, log *slog.Logger) {
	ticker := time.NewTicker(reconcileBillingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := srv.Billing().Reconcile(ctx); err != nil {
				log.Error("subscription reconciliation sweep", "error", err.Error())
			}
		}
	}
}

// newLogHandler builds the slog handler for the given format: "json" for
// structured logs (aggregation-friendly), anything else for the
// human-readable text default.
func newLogHandler(w io.Writer, format string) slog.Handler {
	if format == "json" {
		return slog.NewJSONHandler(w, nil)
	}
	return slog.NewTextHandler(w, nil)
}

// scheduledBackup writes a compressed archive of the database, uploads and
// keys to cfg.BackupDir on cfg.BackupInterval until ctx is done. Failures are
// logged, never fatal; the VACUUM INTO snapshot is safe under write load.
func scheduledBackup(ctx context.Context, cfg config.Config, log *slog.Logger) {
	if err := os.MkdirAll(cfg.BackupDir, 0o700); err != nil {
		log.Error("scheduled backup: create backup dir", "error", err.Error())
		return
	}
	log.Info("scheduled backups enabled", "dir", cfg.BackupDir, "interval", cfg.BackupInterval.String())
	ticker := time.NewTicker(cfg.BackupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := runBackup(ctx, cfg.DataDir, cfg.BackupDir); err != nil {
				log.Error("scheduled backup failed", "error", err.Error())
				continue
			}
			log.Info("scheduled backup written", "dir", cfg.BackupDir)
		}
	}
}

// runBackup writes one timestamped archive of dataDir into destDir.
func runBackup(ctx context.Context, dataDir, destDir string) error {
	name := fmt.Sprintf("moth-backup-%s.tar.gz", time.Now().UTC().Format("20060102T150405Z"))
	path := filepath.Join(destDir, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	dbPath := filepath.Join(dataDir, "moth.db")
	if err := backup.Backup(ctx, dbPath, dataDir, f); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}

// sweepExpiredInterval is how often expired sessions and OAuth artifacts
// are deleted while serving.
const sweepExpiredInterval = time.Hour

// sweepExpired deletes expired admin sessions, single-use OAuth rows and
// personal access tokens on a ticker until ctx is done; failures are
// logged, never fatal.
func sweepExpired(ctx context.Context, st *store.Store, log *slog.Logger) {
	ticker := time.NewTicker(sweepExpiredInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if err := st.DeleteExpiredSessions(ctx, now); err != nil {
				log.Error("sweep expired sessions", "error", err.Error())
			}
			if err := st.DeleteExpiredOAuthTokens(ctx, now); err != nil {
				log.Error("sweep expired oauth tokens", "error", err.Error())
			}
			if err := st.DeleteExpiredPATs(ctx, now); err != nil {
				log.Error("sweep expired personal access tokens", "error", err.Error())
			}
			// Drop grace-period signing keys whose grace has ended: tokens
			// they signed have all expired, so they leave the JWKS.
			if _, err := st.PruneExpiredKeys(ctx, now); err != nil {
				log.Error("prune expired signing keys", "error", err.Error())
			}
			// Rate-limit buckets are fixed short windows; anything older than
			// an hour is dead weight.
			if _, err := st.DeleteStaleRateLimits(ctx, now.Add(-time.Hour)); err != nil {
				log.Error("sweep stale rate limits", "error", err.Error())
			}
		}
	}
}
