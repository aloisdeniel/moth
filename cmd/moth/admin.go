package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/analytics"
	"github.com/aloisdeniel/moth/internal/password"
	adminrpc "github.com/aloisdeniel/moth/internal/server/rpc/admin"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

func newAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Manage admin accounts of the local instance",
	}
	cmd.AddCommand(newAdminCreateCmd())
	cmd.AddCommand(newAdminTokenCmd())
	cmd.AddCommand(newSeedAnalyticsCmd())
	return cmd
}

// openLocalStore opens (and migrates) the instance database of the local
// data directory, for the admin commands that bypass the RPC surface.
func openLocalStore(cmd *cobra.Command, flags *rootFlags) (*store.Store, error) {
	cfg, err := resolveConfig(cmd, flags)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	st, err := store.Open(filepath.Join(cfg.DataDir, "moth.db"))
	if err != nil {
		return nil, err
	}
	if err := st.Migrate(cmd.Context()); err != nil {
		st.Close()
		return nil, err
	}
	return st, nil
}

// newAdminTokenCmd manages personal access tokens directly against the
// local database — the bootstrap path for 'moth login' when no browser is
// at hand (the remote counterpart is 'moth token').
func newAdminTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Manage personal access tokens of the local instance",
	}
	cmd.AddCommand(newAdminTokenCreateCmd(), newAdminTokenListCmd(), newAdminTokenRevokeCmd())
	return cmd
}

func newAdminTokenCreateCmd() *cobra.Command {
	var flags rootFlags
	var email, name string
	var expiresInDays int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Mint a personal access token for an admin (printed exactly once)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			st, err := openLocalStore(cmd, &flags)
			if err != nil {
				return err
			}
			defer st.Close()
			admin, err := st.GetAdminByEmail(cmd.Context(), strings.ToLower(strings.TrimSpace(email)))
			if err != nil {
				return fmt.Errorf("admin %q: %w", email, err)
			}
			if expiresInDays < 0 {
				return errors.New("--expires-in-days must not be negative")
			}
			plain := token.New(token.PATPrefix)
			now := time.Now()
			pat := store.PersonalAccessToken{
				ID:        adminrpc.NewID(),
				AdminID:   admin.ID,
				Name:      name,
				TokenHash: token.Hash(plain),
				CreatedAt: now,
			}
			if expiresInDays > 0 {
				expires := now.AddDate(0, 0, expiresInDays)
				pat.ExpiresAt = &expires
			}
			if err := st.CreatePAT(cmd.Context(), pat); err != nil {
				return err
			}
			if asJSON {
				// Same message shape as the remote 'moth token create'.
				return printJSON(cmd, &adminv1.CreatePersonalAccessTokenResponse{
					Token:    plain,
					Metadata: adminrpc.PATProto(pat),
				})
			}
			fmt.Printf("token %q created for %s (id %s)\n%s\n", name, admin.Email, pat.ID, plain)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "admin email address (required)")
	cmd.Flags().StringVar(&name, "name", "cli", "token label")
	cmd.Flags().IntVar(&expiresInDays, "expires-in-days", 0, "days until expiry (0: never expires)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("email") // flag is registered just above
	addConfigFlags(cmd, &flags)
	return cmd
}

func newAdminTokenListCmd() *cobra.Command {
	var flags rootFlags
	var email string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List an admin's personal access tokens, newest first",
		RunE: func(cmd *cobra.Command, _ []string) error {
			st, err := openLocalStore(cmd, &flags)
			if err != nil {
				return err
			}
			defer st.Close()
			admin, err := st.GetAdminByEmail(cmd.Context(), strings.ToLower(strings.TrimSpace(email)))
			if err != nil {
				return fmt.Errorf("admin %q: %w", email, err)
			}
			pats, err := st.ListPATs(cmd.Context(), admin.ID)
			if err != nil {
				return err
			}
			if asJSON {
				// Same message shape as the remote 'moth token list'.
				resp := &adminv1.ListPersonalAccessTokensResponse{}
				for _, p := range pats {
					resp.Tokens = append(resp.Tokens, adminrpc.PATProto(p))
				}
				return printJSON(cmd, resp)
			}
			fmtAt := func(t *time.Time) string {
				if t == nil {
					return "-"
				}
				return t.Local().Format("2006-01-02 15:04")
			}
			for _, p := range pats {
				fmt.Printf("%s  %-20s created %s  last used %s  expires %s  revoked %s\n",
					p.ID, p.Name, p.CreatedAt.Local().Format("2006-01-02 15:04"),
					fmtAt(p.LastUsedAt), fmtAt(p.ExpiresAt), fmtAt(p.RevokedAt))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "admin email address (required)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("email") // flag is registered just above
	addConfigFlags(cmd, &flags)
	return cmd
}

func newAdminTokenRevokeCmd() *cobra.Command {
	var flags rootFlags
	var email, id string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke one of an admin's personal access tokens",
		RunE: func(cmd *cobra.Command, _ []string) error {
			st, err := openLocalStore(cmd, &flags)
			if err != nil {
				return err
			}
			defer st.Close()
			admin, err := st.GetAdminByEmail(cmd.Context(), strings.ToLower(strings.TrimSpace(email)))
			if err != nil {
				return fmt.Errorf("admin %q: %w", email, err)
			}
			if err := st.RevokePAT(cmd.Context(), admin.ID, id, time.Now()); err != nil {
				return fmt.Errorf("token %q: %w", id, err)
			}
			if asJSON {
				return printJSON(cmd, &adminv1.RevokePersonalAccessTokenResponse{})
			}
			fmt.Printf("revoked token %s\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "admin email address (required)")
	cmd.Flags().StringVar(&id, "id", "", "token id (required)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("email") // flag is registered just above
	_ = cmd.MarkFlagRequired("id")    // flag is registered just above
	addConfigFlags(cmd, &flags)
	return cmd
}

func newAdminCreateCmd() *cobra.Command {
	var flags rootFlags
	var email, pw string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an admin account (or reset its password if the email exists)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := resolveConfig(cmd, &flags)
			if err != nil {
				return err
			}

			email = strings.ToLower(strings.TrimSpace(email))
			if _, err := mail.ParseAddress(email); err != nil {
				return fmt.Errorf("invalid email address %q", email)
			}
			generated := false
			if pw == "" {
				pw = token.Random(12)
				generated = true
			}
			if len(pw) < 8 {
				return errors.New("password must be at least 8 characters")
			}
			hash, err := password.Hash(pw)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
				return fmt.Errorf("create data dir: %w", err)
			}
			st, err := store.Open(filepath.Join(cfg.DataDir, "moth.db"))
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.Migrate(cmd.Context()); err != nil {
				return err
			}

			now := time.Now()
			if err := st.UpsertAdmin(cmd.Context(), store.Admin{
				ID:           adminrpc.NewID(),
				Email:        email,
				PasswordHash: hash,
				CreatedAt:    now,
				UpdatedAt:    now,
			}); err != nil {
				return err
			}

			fmt.Printf("admin account ready: %s\n", email)
			if generated {
				fmt.Printf("generated password: %s\n", pw)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "admin email address (required)")
	cmd.Flags().StringVar(&pw, "password", "", "password (generated and printed if omitted)")
	_ = cmd.MarkFlagRequired("email") // flag is registered just above
	addConfigFlags(cmd, &flags)
	return cmd
}

// newSeedAnalyticsCmd is a hidden development helper: it fills a project
// with deterministic synthetic analytics events and rolls them up, so the
// dashboards have something to show in demos and manual testing.
func newSeedAnalyticsCmd() *cobra.Command {
	var flags rootFlags
	var slug string
	var days int
	var seed uint64
	cmd := &cobra.Command{
		Use:    "seed-analytics",
		Short:  "Fill a project with synthetic analytics events (dev helper)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := resolveConfig(cmd, &flags)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
				return fmt.Errorf("create data dir: %w", err)
			}
			st, err := store.Open(filepath.Join(cfg.DataDir, "moth.db"))
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.Migrate(cmd.Context()); err != nil {
				return err
			}
			project, err := st.GetProjectBySlug(cmd.Context(), slug)
			if err != nil {
				return fmt.Errorf("project %q: %w", slug, err)
			}

			n, err := analytics.Seed(cmd.Context(), st, project, analytics.SeedOptions{
				Days: days, Seed: seed,
			})
			if err != nil {
				return err
			}
			log := slog.New(slog.NewTextHandler(os.Stderr, nil))
			run, err := analytics.NewRollup(st, log, nil).Run(cmd.Context(), project.ID)
			if err != nil {
				return err
			}
			fmt.Printf("seeded %d events over %d days for %s; rollup processed %d days (pruned %d)\n",
				n, days, slug, run.DaysProcessed, run.EventsPruned)
			return nil
		},
	}
	cmd.Flags().StringVar(&slug, "project", "", "project slug (required)")
	cmd.Flags().IntVar(&days, "days", 90, "days of history to generate")
	cmd.Flags().Uint64Var(&seed, "seed", 1, "random seed (deterministic output)")
	_ = cmd.MarkFlagRequired("project") // flag is registered just above
	addConfigFlags(cmd, &flags)
	return cmd
}
