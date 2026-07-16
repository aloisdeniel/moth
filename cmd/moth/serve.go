package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

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
	return cmd
}

func serve(ctx context.Context, cfg config.Config) error {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

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

	master, err := keys.LoadOrCreateMasterKey(cfg.DataDir, os.Getenv)
	if err != nil {
		return err
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

	log.Info("moth serving", "version", version.Version, "addr", cfg.Addr, "base_url", cfg.BaseURL, "data_dir", cfg.DataDir)
	if setupToken != "" {
		fmt.Fprintf(os.Stderr, "\n  No admin account exists yet. Create one at:\n\n    %s/admin?setup=%s\n\n", cfg.BaseURL, setupToken)
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.ListenAndServe() }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return nil
	}
}
