// Package store is the SQLite persistence layer: connection setup,
// embedded migrations, and hand-written queries behind small per-domain
// interfaces (no ORM).
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// AdminStore persists instance operators.
type AdminStore interface {
	CreateAdmin(ctx context.Context, a Admin) error
	// UpsertAdmin creates the admin or, when the email already exists,
	// resets its password hash. Used by `moth admin create`.
	UpsertAdmin(ctx context.Context, a Admin) error
	GetAdmin(ctx context.Context, id string) (Admin, error)
	GetAdminByEmail(ctx context.Context, email string) (Admin, error)
	CountAdmins(ctx context.Context) (int, error)
}

// SessionStore persists admin browser sessions (cookie tokens are stored
// hashed).
type SessionStore interface {
	CreateSession(ctx context.Context, s AdminSession) error
	GetSession(ctx context.Context, tokenHash string) (AdminSession, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteExpiredSessions(ctx context.Context, now time.Time) error
}

// ProjectStore persists projects and their signing keys.
type ProjectStore interface {
	// CreateProject inserts the project and its first signing key in one
	// transaction: a project must never exist without a keypair.
	CreateProject(ctx context.Context, p Project, k ProjectKey) error
	GetProject(ctx context.Context, id string) (Project, error)
	GetProjectBySlug(ctx context.Context, slug string) (Project, error)
	ListProjects(ctx context.Context) ([]Project, error)
	UpdateProject(ctx context.Context, p Project) error
	DeleteProject(ctx context.Context, id string) error
	SlugExists(ctx context.Context, slug string) (bool, error)
	ListActiveProjectKeys(ctx context.Context, projectID string) ([]ProjectKey, error)
}

// Store implements AdminStore, SessionStore and ProjectStore on SQLite.
type Store struct {
	db *sql.DB
}

var (
	_ AdminStore   = (*Store)(nil)
	_ SessionStore = (*Store)(nil)
	_ ProjectStore = (*Store)(nil)
)

// Open opens (creating if needed) the SQLite database at path with WAL
// mode, foreign keys and a busy timeout.
func Open(path string) (*Store, error) {
	dsn := "file:" + path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// Migrate applies all embedded migrations that are not yet recorded in
// schema_migrations. It is idempotent and runs on every startup.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		version, err := migrationVersion(name)
		if err != nil {
			return err
		}
		var applied bool
		if err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) > 0 FROM schema_migrations WHERE version = ?`, version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied {
			continue
		}
		sqlText, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, string(sqlText)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			version, formatTime(time.Now()),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}

// migrationVersion extracts the numeric prefix of "0001_init.sql".
func migrationVersion(name string) (int, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("migration %s: name must be NNNN_description.sql", name)
	}
	v, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("migration %s: invalid version prefix: %w", name, err)
	}
	return v, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
