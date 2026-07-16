// Package backup produces and restores a single-file archive of a moth data
// directory: an online-consistent snapshot of the SQLite database plus the
// uploads and key material.
//
// The database is never copied byte-for-byte — a live WAL-mode file is not
// safe to read while the server is writing. Instead VACUUM INTO writes a
// fresh, transactionally-consistent copy that captures every committed write
// up to the moment it runs, so a backup can be taken under load. The result
// is a gzip-compressed tar; Restore reverses it with traversal and
// overwrite guards.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// DBArchiveName is the path of the vacuumed database inside the archive; on
// Restore it lands at <dataDir>/moth.db.
const DBArchiveName = "moth.db"

// backupSubtrees are the data-dir subdirectories copied verbatim alongside
// the database: uploaded project logos and the master/signing key material.
var backupSubtrees = []string{"uploads", "keys"}

// Backup writes a gzip-compressed tar snapshot to out. dbPath is the live
// SQLite database; dataDir holds the uploads/ and keys/ subtrees. Missing
// subtrees are skipped (a fresh instance may have no uploads yet). The
// snapshot is taken with VACUUM INTO, so it is safe to call while the server
// is serving traffic.
func Backup(ctx context.Context, dbPath, dataDir string, out io.Writer) error {
	// The intermediate VACUUM INTO copy is written under the data directory,
	// not os.TempDir(): the shipped scratch Docker image has no /tmp, and the
	// data dir is the one path guaranteed to exist and be writable (it is a
	// mounted volume). It must live on the same filesystem as the database
	// anyway for the VACUUM to be cheap.
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("backup: create data dir: %w", err)
	}
	tmp, err := os.CreateTemp(dataDir, ".moth-backup-*.db")
	if err != nil {
		return fmt.Errorf("backup: temp file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	// VACUUM INTO requires the destination not to exist.
	if err := os.Remove(tmpPath); err != nil {
		return fmt.Errorf("backup: clear temp: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	if err := vacuumInto(ctx, dbPath, tmpPath); err != nil {
		return err
	}

	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)

	if err := addFile(tw, tmpPath, DBArchiveName); err != nil {
		return err
	}
	for _, sub := range backupSubtrees {
		if err := addTree(tw, filepath.Join(dataDir, sub), sub); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("backup: close tar: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("backup: close gzip: %w", err)
	}
	return nil
}

// vacuumInto opens dbPath read-only-ish and writes a consistent copy to dest.
func vacuumInto(ctx context.Context, dbPath, dest string) error {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(10000)")
	if err != nil {
		return fmt.Errorf("backup: open db: %w", err)
	}
	defer db.Close()
	// dest is a locally-generated temp path; still, quote it as a SQL string
	// literal defensively.
	quoted := "'" + strings.ReplaceAll(dest, "'", "''") + "'"
	if _, err := db.ExecContext(ctx, "VACUUM INTO "+quoted); err != nil {
		return fmt.Errorf("backup: vacuum into: %w", err)
	}
	return nil
}

// Restore extracts an archive produced by Backup into dataDir. Unless force
// is set it refuses to write into a non-empty dataDir, so an accidental
// restore cannot silently clobber a live instance. Every archive member is
// validated against path traversal before anything is written.
func Restore(archive io.Reader, dataDir string, force bool) error {
	if !force {
		if entries, err := os.ReadDir(dataDir); err == nil && len(entries) > 0 {
			return fmt.Errorf("backup: data dir %q is not empty; pass force to overwrite", dataDir)
		}
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("backup: create data dir: %w", err)
	}

	// Remove any leftover WAL/SHM sidecars of the database being replaced. The
	// archive's moth.db is a VACUUM INTO copy in rollback-journal mode with no
	// companion WAL; if a crashed instance left an uncheckpointed
	// moth.db-wal/-shm next to the old database, SQLite would open the restored
	// file, find the stale WAL and replay its frames onto a now-different
	// database — resurrecting pre-restore rows or corrupting the snapshot. The
	// restored moth.db is self-contained, so the sidecars must not survive it.
	for _, sidecar := range []string{DBArchiveName + "-wal", DBArchiveName + "-shm"} {
		if err := os.Remove(filepath.Join(dataDir, sidecar)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("backup: remove stale %s: %w", sidecar, err)
		}
	}

	gz, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("backup: open gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	sawDB := false
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("backup: read tar: %w", err)
		}
		if !hdr.FileInfo().Mode().IsRegular() {
			continue
		}
		dest, err := safeJoin(dataDir, hdr.Name)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return fmt.Errorf("backup: mkdir for %q: %w", hdr.Name, err)
		}
		if err := writeMember(dest, tr); err != nil {
			return err
		}
		if hdr.Name == DBArchiveName {
			sawDB = true
		}
	}
	if !sawDB {
		return fmt.Errorf("backup: archive is missing %s", DBArchiveName)
	}
	return nil
}

// safeJoin resolves an archive member name under base, rejecting absolute
// paths and any ".." traversal.
func safeJoin(base, name string) (string, error) {
	slash := filepath.ToSlash(name)
	if slash == "" {
		return "", errors.New("backup: empty archive member name")
	}
	if strings.HasPrefix(slash, "/") || filepath.IsAbs(name) {
		return "", fmt.Errorf("backup: absolute path in archive: %q", name)
	}
	clean := path.Clean(slash)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("backup: path traversal in archive: %q", name)
	}
	for _, part := range strings.Split(clean, "/") {
		if part == ".." {
			return "", fmt.Errorf("backup: path traversal in archive: %q", name)
		}
	}
	return filepath.Join(base, filepath.FromSlash(clean)), nil
}

func writeMember(dest string, r io.Reader) error {
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("backup: create %q: %w", dest, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("backup: write %q: %w", dest, err)
	}
	return f.Close()
}

func addFile(tw *tar.Writer, srcPath, name string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("backup: open %q: %w", srcPath, err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("backup: stat %q: %w", srcPath, err)
	}
	hdr := &tar.Header{
		Name:     name,
		Mode:     0o600,
		Size:     info.Size(),
		ModTime:  info.ModTime().UTC(),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("backup: write header %q: %w", name, err)
	}
	if _, err := io.Copy(tw, f); err != nil {
		return fmt.Errorf("backup: copy %q: %w", name, err)
	}
	return nil
}

// addTree walks root and adds every regular file under the archive prefix.
// A missing root is not an error (skipped).
func addTree(tw *tar.Writer, root, prefix string) error {
	info, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("backup: stat %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		return addFile(tw, p, prefix+"/"+filepath.ToSlash(rel))
	})
}
