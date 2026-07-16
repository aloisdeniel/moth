package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Project is one mobile app hosted by the instance: a sealed tenant with
// its own users, keys and configuration.
type Project struct {
	ID             string
	Name           string
	Slug           string
	PublishableKey string
	SecretKeyHash  string
	Settings       ProjectSettings
	// Theme is the raw design-system JSON document (internal/theme
	// schema); empty means the built-in default theme. Written through
	// SetProjectTheme, never UpdateProject.
	Theme string
	// ThemeRevisionID identifies the revision Theme came from ("" when
	// Theme is empty).
	ThemeRevisionID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ProjectKey is one ES256 signing keypair belonging to a project. The
// private key is encrypted under the instance master key.
type ProjectKey struct {
	ID            string
	ProjectID     string
	Kid           string
	Algorithm     string
	PublicKeyPEM  string
	PrivateKeyEnc []byte
	Status        string
	CreatedAt     time.Time
}

// ProjectKeyStatusActive marks keys currently served in the project JWKS.
const ProjectKeyStatusActive = "active"

// ProjectKeyStatusRetired marks keys dropped from the JWKS by a signing
// key reset; tokens they signed no longer validate.
const ProjectKeyStatusRetired = "retired"

func (s *Store) CreateProject(ctx context.Context, p Project, k ProjectKey) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create project: %w", err)
	}
	defer tx.Rollback()

	settings, err := encodeProjectSettings(p.Settings)
	if err != nil {
		return fmt.Errorf("encode project settings: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO projects (id, name, slug, publishable_key, secret_key_hash, settings, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Slug, p.PublishableKey, p.SecretKeyHash, settings,
		formatTime(p.CreatedAt), formatTime(p.UpdatedAt)); err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO project_keys (id, project_id, kid, algorithm, public_key_pem, private_key_enc, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		k.ID, k.ProjectID, k.Kid, k.Algorithm, k.PublicKeyPEM, k.PrivateKeyEnc,
		k.Status, formatTime(k.CreatedAt)); err != nil {
		return fmt.Errorf("insert project key: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create project: %w", err)
	}
	return nil
}

const projectColumns = `id, name, slug, publishable_key, secret_key_hash, settings, theme, theme_revision, created_at, updated_at`

func (s *Store) GetProject(ctx context.Context, id string) (Project, error) {
	return scanProject(s.db.QueryRowContext(ctx,
		`SELECT `+projectColumns+` FROM projects WHERE id = ?`, id))
}

func (s *Store) GetProjectBySlug(ctx context.Context, slug string) (Project, error) {
	return scanProject(s.db.QueryRowContext(ctx,
		`SELECT `+projectColumns+` FROM projects WHERE slug = ?`, slug))
}

func (s *Store) GetProjectByPublishableKey(ctx context.Context, key string) (Project, error) {
	return scanProject(s.db.QueryRowContext(ctx,
		`SELECT `+projectColumns+` FROM projects WHERE publishable_key = ?`, key))
}

func (s *Store) GetProjectBySecretKeyHash(ctx context.Context, keyHash string) (Project, error) {
	return scanProject(s.db.QueryRowContext(ctx,
		`SELECT `+projectColumns+` FROM projects WHERE secret_key_hash = ?`, keyHash))
}

func (s *Store) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+projectColumns+` FROM projects ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		p, err := scanProjectRow(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

func (s *Store) UpdateProject(ctx context.Context, p Project) error {
	settings, err := encodeProjectSettings(p.Settings)
	if err != nil {
		return fmt.Errorf("encode project settings: %w", err)
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE projects SET name = ?, settings = ?, updated_at = ? WHERE id = ?`,
		p.Name, settings, formatTime(p.UpdatedAt), p.ID)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	return requireRow(res)
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return requireRow(res)
}

// UpdateProjectSecretKey replaces the project's secret key hash; the old
// key stops authenticating immediately.
func (s *Store) UpdateProjectSecretKey(ctx context.Context, id, secretKeyHash string, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE projects SET secret_key_hash = ?, updated_at = ? WHERE id = ?`,
		secretKeyHash, formatTime(now), id)
	if err != nil {
		return fmt.Errorf("update project secret key: %w", err)
	}
	return requireRow(res)
}

// ResetProjectSigningKey atomically retires every signing key of the
// project, installs the replacement and revokes all refresh tokens: every
// issued token is dead and all users must sign in again.
func (s *Store) ResetProjectSigningKey(ctx context.Context, projectID string, k ProjectKey, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reset signing key: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`UPDATE project_keys SET status = ? WHERE project_id = ? AND status = ?`,
		ProjectKeyStatusRetired, projectID, ProjectKeyStatusActive); err != nil {
		return fmt.Errorf("retire project keys: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO project_keys (id, project_id, kid, algorithm, public_key_pem, private_key_enc, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		k.ID, k.ProjectID, k.Kid, k.Algorithm, k.PublicKeyPEM, k.PrivateKeyEnc,
		k.Status, formatTime(k.CreatedAt)); err != nil {
		return fmt.Errorf("insert replacement project key: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ? WHERE project_id = ? AND revoked_at IS NULL`,
		formatTime(now), projectID); err != nil {
		return fmt.Errorf("revoke project refresh tokens: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reset signing key: %w", err)
	}
	return nil
}

func (s *Store) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) > 0 FROM projects WHERE slug = ?`, slug).Scan(&exists); err != nil {
		return false, fmt.Errorf("check slug: %w", err)
	}
	return exists, nil
}

func (s *Store) ListActiveProjectKeys(ctx context.Context, projectID string) ([]ProjectKey, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, kid, algorithm, public_key_pem, private_key_enc, status, created_at
		 FROM project_keys WHERE project_id = ? AND status = ? ORDER BY created_at, id`,
		projectID, ProjectKeyStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list project keys: %w", err)
	}
	defer rows.Close()

	var keys []ProjectKey
	for rows.Next() {
		var k ProjectKey
		var createdAt string
		if err := rows.Scan(&k.ID, &k.ProjectID, &k.Kid, &k.Algorithm,
			&k.PublicKeyPEM, &k.PrivateKeyEnc, &k.Status, &createdAt); err != nil {
			return nil, fmt.Errorf("scan project key: %w", err)
		}
		if k.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, fmt.Errorf("parse project key created_at: %w", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list project keys: %w", err)
	}
	return keys, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProject(row *sql.Row) (Project, error) {
	p, err := scanProjectRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	return p, err
}

func scanProjectRow(row rowScanner) (Project, error) {
	var p Project
	var settings, createdAt, updatedAt string
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.PublishableKey, &p.SecretKeyHash,
		&settings, &p.Theme, &p.ThemeRevisionID, &createdAt, &updatedAt)
	if err != nil {
		return Project{}, err
	}
	if p.Settings, err = parseProjectSettings(settings); err != nil {
		return Project{}, fmt.Errorf("parse project settings: %w", err)
	}
	if p.CreatedAt, err = parseTime(createdAt); err != nil {
		return Project{}, fmt.Errorf("parse project created_at: %w", err)
	}
	if p.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return Project{}, fmt.Errorf("parse project updated_at: %w", err)
	}
	return p, nil
}

func requireRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
