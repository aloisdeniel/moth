package main

import (
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

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
