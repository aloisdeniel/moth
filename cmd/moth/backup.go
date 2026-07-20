package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/aloisdeniel/moth/internal/backup"
)

func newBackupCmd() *cobra.Command {
	var flags rootFlags
	var to string
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Write a compressed snapshot of the instance (database, uploads, keys)",
		Long: `Write a single gzip-compressed tar archive containing an
online-consistent snapshot of the SQLite database (taken with VACUUM INTO, so
it is safe to run against a live server) plus the uploads and key material.

The archive is self-contained; restore it with "moth restore".

Cron example — a daily backup at 03:30, keeping the last 14 days:

  30 3 * * *  moth backup --data-dir /var/lib/moth --to /backups/moth-$(date +\%F).tar.gz && find /backups -name 'moth-*.tar.gz' -mtime +14 -delete

For unattended backups without cron, set backup_dir (and optionally
backup_interval) in the config so "moth serve" writes them itself.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := resolveConfig(cmd, &flags)
			if err != nil {
				return err
			}
			out := to
			if out == "" {
				out = fmt.Sprintf("moth-backup-%s.tar.gz", time.Now().UTC().Format("20060102T150405Z"))
			}
			f, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
			if err != nil {
				return fmt.Errorf("create archive: %w", err)
			}
			dbPath := filepath.Join(cfg.DataDir, "moth.db")
			if err := backup.Backup(cmd.Context(), dbPath, cfg.DataDir, f); err != nil {
				_ = f.Close()
				_ = os.Remove(out)
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", out)
			return nil
		},
	}
	addConfigFlags(cmd, &flags)
	cmd.Flags().StringVar(&to, "to", "", "archive path (default moth-backup-<timestamp>.tar.gz in the working directory)")
	return cmd
}

func newRestoreCmd() *cobra.Command {
	var flags rootFlags
	var force bool
	cmd := &cobra.Command{
		Use:   "restore <archive>",
		Short: "Restore an instance from a backup archive into the data directory",
		Long: `Extract a "moth backup" archive into the data directory, recreating
the database, uploads and keys.

For safety the restore refuses to write into a non-empty data directory unless
--force is given, so an accidental restore cannot clobber a running instance.
Stop "moth serve" before restoring over an existing data directory.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := resolveConfig(cmd, &flags)
			if err != nil {
				return err
			}
			f, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("open archive: %w", err)
			}
			defer func() { _ = f.Close() }()
			if err := backup.Restore(f, cfg.DataDir, force); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "restored into %s\n", cfg.DataDir)
			return nil
		},
	}
	addConfigFlags(cmd, &flags)
	cmd.Flags().BoolVar(&force, "force", false, "overwrite a non-empty data directory")
	return cmd
}
