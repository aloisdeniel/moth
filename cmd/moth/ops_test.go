package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/aloisdeniel/moth/internal/store"
)

// TestBackupRestoreRoundTrip exercises the backup and restore CLI commands
// against a temporary instance: a populated data directory is archived and
// restored into a fresh directory with its database, uploads and keys intact.
func TestBackupRestoreRoundTrip(t *testing.T) {
	src := t.TempDir()

	// A migrated database plus representative uploads and key material.
	st, err := store.Open(filepath.Join(src, "moth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	st.Close()
	mustWrite(t, filepath.Join(src, "uploads", "logo.png"), "PNGDATA")
	mustWrite(t, filepath.Join(src, "keys", "master.key"), "MASTERKEY")

	archive := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	runCmd(t, "backup", "--data-dir", src, "--to", archive)
	if fi, err := os.Stat(archive); err != nil || fi.Size() == 0 {
		t.Fatalf("archive not written: %v", err)
	}

	dst := filepath.Join(t.TempDir(), "restored")
	runCmd(t, "restore", archive, "--data-dir", dst)

	for _, rel := range []string{"moth.db", "uploads/logo.png", "keys/master.key"} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Errorf("restored file missing %s: %v", rel, err)
		}
	}
	if got := mustRead(t, filepath.Join(dst, "keys", "master.key")); got != "MASTERKEY" {
		t.Errorf("key material corrupted on restore: %q", got)
	}

	// Restore into a non-empty directory is refused without --force.
	if err := runCmdErr("restore", archive, "--data-dir", dst); err == nil {
		t.Fatal("restore into a populated data dir must fail without --force")
	}
	runCmd(t, "restore", archive, "--data-dir", dst, "--force")
}

// TestLogHandlerFormat verifies the --log-format toggle: json emits structured
// records, the default emits text.
func TestLogHandlerFormat(t *testing.T) {
	var jsonBuf bytes.Buffer
	slog.New(newLogHandler(&jsonBuf, "json")).Info("hello", "k", "v")
	var rec map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(jsonBuf.Bytes()), &rec); err != nil {
		t.Fatalf("json handler did not emit valid JSON: %v\n%s", err, jsonBuf.String())
	}
	if rec["msg"] != "hello" || rec["k"] != "v" {
		t.Fatalf("json record missing fields: %v", rec)
	}

	var textBuf bytes.Buffer
	slog.New(newLogHandler(&textBuf, "text")).Info("hello", "k", "v")
	line := textBuf.String()
	if json.Valid(bytes.TrimSpace(textBuf.Bytes())) {
		t.Fatalf("text handler should not emit JSON: %s", line)
	}
	if !bytes.Contains(textBuf.Bytes(), []byte("msg=hello")) {
		t.Fatalf("text record missing msg: %s", line)
	}
}

func runCmd(t *testing.T, args ...string) {
	t.Helper()
	if err := runCmdErr(args...); err != nil {
		t.Fatalf("moth %v: %v", args, err)
	}
}

func runCmdErr(args ...string) error {
	root := newRootCmd()
	root.SetArgs(args)
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	return root.Execute()
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
