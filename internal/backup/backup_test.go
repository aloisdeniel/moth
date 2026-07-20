package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// makeDB creates a small SQLite database at path with one seeded row.
func makeDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE widgets (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO widgets (name) VALUES ('alpha'), ('beta')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
}

func TestBackupRestoreRoundTrip(t *testing.T) {
	src := t.TempDir()
	dbPath := filepath.Join(src, "moth.db")
	makeDB(t, dbPath)

	// Seed uploads/ and keys/ with representative files.
	writeFile(t, filepath.Join(src, "uploads", "proj1", "logo.png"), []byte("PNGDATA"))
	writeFile(t, filepath.Join(src, "keys", "master.key"), []byte("deadbeef"))

	var buf bytes.Buffer
	if err := Backup(context.Background(), dbPath, src, &buf); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("empty archive")
	}

	dst := t.TempDir()
	// dst from TempDir is empty, so a non-force restore must succeed.
	if err := Restore(bytes.NewReader(buf.Bytes()), dst, false); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	// Files restored verbatim.
	if got := readFile(t, filepath.Join(dst, "uploads", "proj1", "logo.png")); string(got) != "PNGDATA" {
		t.Fatalf("logo mismatch: %q", got)
	}
	if got := readFile(t, filepath.Join(dst, "keys", "master.key")); string(got) != "deadbeef" {
		t.Fatalf("key mismatch: %q", got)
	}

	// Restored database opens and the seeded rows survive.
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dst, "moth.db")+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open restored: %v", err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM widgets`).Scan(&n); err != nil {
		t.Fatalf("query restored: %v", err)
	}
	if n != 2 {
		t.Fatalf("row count = %d, want 2", n)
	}
}

// TestBackupUnderWriteLoad is the acceptance criterion "backup taken under
// write load restores to a working instance": a background writer commits rows
// continuously while the backup runs, and the restored database must open,
// pass an integrity check, and preserve committed rows. It exercises the
// VACUUM INTO snapshot's under-load consistency guarantee that a raw WAL
// byte-copy could not provide.
func TestBackupUnderWriteLoad(t *testing.T) {
	src := t.TempDir()
	dbPath := filepath.Join(src, "moth.db")

	db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE events (id INTEGER PRIMARY KEY AUTOINCREMENT, payload TEXT)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	// A baseline committed row guarantees the snapshot is non-empty regardless
	// of interleaving.
	if _, err := db.Exec(`INSERT INTO events (payload) VALUES ('seed')`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				_, _ = db.Exec(`INSERT INTO events (payload) VALUES ('x')`)
			}
		}
	}()
	// Let some writes land so the VACUUM overlaps live traffic.
	time.Sleep(20 * time.Millisecond)

	var buf bytes.Buffer
	backupErr := Backup(context.Background(), dbPath, src, &buf)
	close(stop)
	<-done
	if backupErr != nil {
		t.Fatalf("Backup under load: %v", backupErr)
	}

	dst := t.TempDir()
	if err := Restore(bytes.NewReader(buf.Bytes()), dst, false); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	rdb, err := sql.Open("sqlite", "file:"+filepath.Join(dst, "moth.db")+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open restored: %v", err)
	}
	defer rdb.Close()
	var integrity string
	if err := rdb.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
		t.Fatalf("integrity_check: %v", err)
	}
	if integrity != "ok" {
		t.Fatalf("restored db integrity = %q, want ok", integrity)
	}
	var n int
	if err := rdb.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&n); err != nil {
		t.Fatalf("query restored: %v", err)
	}
	if n < 1 {
		t.Fatalf("restored under-load snapshot has %d rows, want >= 1", n)
	}
}

// TestRestoreClearsStaleWALSidecars: a force restore over a crashed instance's
// data dir (a database left with uncheckpointed moth.db-wal/-shm) must remove
// the stale sidecars so SQLite cannot replay pre-restore frames onto the
// freshly restored, self-contained database.
func TestRestoreClearsStaleWALSidecars(t *testing.T) {
	src := t.TempDir()
	dbPath := filepath.Join(src, "moth.db")
	makeDB(t, dbPath)
	var buf bytes.Buffer
	if err := Backup(context.Background(), dbPath, src, &buf); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	dst := t.TempDir()
	writeFile(t, filepath.Join(dst, "moth.db"), []byte("stale-db"))
	writeFile(t, filepath.Join(dst, "moth.db-wal"), []byte("stale-wal"))
	writeFile(t, filepath.Join(dst, "moth.db-shm"), []byte("stale-shm"))

	if err := Restore(bytes.NewReader(buf.Bytes()), dst, true); err != nil {
		t.Fatalf("force Restore: %v", err)
	}
	for _, sidecar := range []string{"moth.db-wal", "moth.db-shm"} {
		if _, err := os.Stat(filepath.Join(dst, sidecar)); !os.IsNotExist(err) {
			t.Fatalf("stale %s survived restore (stat err=%v)", sidecar, err)
		}
	}
	rdb, err := sql.Open("sqlite", "file:"+filepath.Join(dst, "moth.db")+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open restored: %v", err)
	}
	defer rdb.Close()
	var n int
	if err := rdb.QueryRow(`SELECT COUNT(*) FROM widgets`).Scan(&n); err != nil {
		t.Fatalf("query restored: %v", err)
	}
	if n != 2 {
		t.Fatalf("restored row count = %d, want 2", n)
	}
}

func TestRestoreRefusesNonEmptyDir(t *testing.T) {
	src := t.TempDir()
	dbPath := filepath.Join(src, "moth.db")
	makeDB(t, dbPath)
	var buf bytes.Buffer
	if err := Backup(context.Background(), dbPath, src, &buf); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	dst := t.TempDir()
	writeFile(t, filepath.Join(dst, "existing"), []byte("x"))

	if err := Restore(bytes.NewReader(buf.Bytes()), dst, false); err == nil {
		t.Fatal("expected refusal on non-empty dir")
	}
	// force overwrites.
	if err := Restore(bytes.NewReader(buf.Bytes()), dst, true); err != nil {
		t.Fatalf("force Restore: %v", err)
	}
}

func TestRestoreRejectsTraversal(t *testing.T) {
	// Hand-craft a malicious archive with a "../escape" member.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	// Include a valid db member so the missing-db check is not what fails.
	writeTar(t, tw, DBArchiveName, []byte("db"))
	writeTar(t, tw, "../escape.txt", []byte("evil"))
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	if err := Restore(bytes.NewReader(buf.Bytes()), dst, true); err == nil {
		t.Fatal("expected traversal rejection")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dst), "escape.txt")); err == nil {
		t.Fatal("traversal wrote a file outside the data dir")
	}
}

func TestRestoreRejectsMissingDB(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	writeTar(t, tw, "uploads/x", []byte("y"))
	tw.Close()
	gz.Close()
	if err := Restore(bytes.NewReader(buf.Bytes()), t.TempDir(), true); err == nil {
		t.Fatal("expected missing-db error")
	}
}

func writeTar(t *testing.T, tw *tar.Writer, name string, data []byte) {
	t.Helper()
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
