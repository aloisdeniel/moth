package config

import (
	"testing"
	"time"
)

func TestLogFormatDefaultAndValidation(t *testing.T) {
	inDir(t)
	cfg, err := Load(Overrides{}, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogFormat != "text" {
		t.Fatalf("default log format: %q", cfg.LogFormat)
	}

	if _, err := Load(Overrides{LogFormat: ptr("json")}, noEnv); err != nil {
		t.Fatalf("json log format should be accepted: %v", err)
	}
	if _, err := Load(Overrides{LogFormat: ptr("xml")}, noEnv); err == nil {
		t.Fatal("invalid log format must be rejected")
	}
}

func TestBackupSchedulingResolution(t *testing.T) {
	inDir(t)

	// No backup dir: scheduler stays disabled.
	cfg, err := Load(Overrides{}, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BackupDir != "" {
		t.Fatalf("backup dir should default empty: %q", cfg.BackupDir)
	}

	// A backup dir without an interval falls back to the default.
	cfg, err = Load(Overrides{BackupDir: ptr("/var/backups")}, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BackupInterval != DefaultBackupInterval {
		t.Fatalf("default backup interval: %v", cfg.BackupInterval)
	}

	// Env-supplied interval is parsed.
	env := func(k string) string {
		if k == "MOTH_BACKUP_INTERVAL" {
			return "15m"
		}
		return ""
	}
	cfg, err = Load(Overrides{BackupDir: ptr("/var/backups")}, env)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BackupInterval != 15*time.Minute {
		t.Fatalf("env backup interval: %v", cfg.BackupInterval)
	}

	// A bad interval is a hard error.
	badEnv := func(k string) string {
		if k == "MOTH_BACKUP_INTERVAL" {
			return "soon"
		}
		return ""
	}
	if _, err := Load(Overrides{}, badEnv); err == nil {
		t.Fatal("invalid backup interval must be rejected")
	}
}

func TestReflectionResolution(t *testing.T) {
	inDir(t)
	cfg, err := Load(Overrides{}, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Reflection {
		t.Fatal("reflection must default off")
	}
	on := true
	cfg, err = Load(Overrides{Reflection: &on}, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Reflection {
		t.Fatal("reflection flag should enable it")
	}
}
