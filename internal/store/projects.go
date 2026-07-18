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
	// Theme is the raw design-system protobuf document
	// (moth.projectconfig.v1.StoredTheme, see internal/theme); empty means the
	// built-in default theme. Written through SetProjectTheme, never
	// UpdateProject.
	Theme []byte
	// ThemeRevisionID identifies the revision Theme came from ("" when
	// Theme is empty).
	ThemeRevisionID string
	// Paywall is the raw paywall-config protobuf document
	// (moth.projectconfig.v1.StoredPaywall, see internal/paywall); empty means the
	// built-in default paywall. Written through SetProjectPaywall, never
	// UpdateProject.
	Paywall []byte
	// PaywallRevisionID identifies the revision Paywall came from ("" when
	// Paywall is empty).
	PaywallRevisionID string
	// Copy is the raw copy-override protobuf document
	// (moth.projectconfig.v1.StoredCopy, a locale → key → string map, see
	// CopyOverrides); empty means the project renders the bundled catalog
	// defaults. Written through SetProjectCopy / the copy mutators, never
	// UpdateProject.
	Copy []byte
	// CopyRevisionID identifies the revision Copy came from ("" when Copy is
	// empty).
	CopyRevisionID string
	// Push is the raw push-settings protobuf document
	// (moth.projectconfig.v1.StoredPush, see internal/push); empty means push
	// was never configured (disabled). Plain config with no revision history,
	// written through SetProjectPush, never UpdateProject.
	Push      []byte
	CreatedAt time.Time
	UpdatedAt time.Time
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
	// RotatedAt is when a graceful rotation stopped this key signing; nil
	// while the key is active or was hard-retired.
	RotatedAt *time.Time
	// NotAfter is when a grace-period key leaves the JWKS and becomes
	// eligible for pruning; nil for active and hard-retired keys.
	NotAfter  *time.Time
	CreatedAt time.Time
}

// ProjectKeyStatusActive marks the key currently signing new tokens and
// served in the project JWKS.
const ProjectKeyStatusActive = "active"

// ProjectKeyStatusGrace marks a key retired by a graceful rotation: it no
// longer signs but stays in the JWKS until NotAfter so tokens it already
// signed keep validating until they expire.
const ProjectKeyStatusGrace = "grace"

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

const projectColumns = `id, name, slug, publishable_key, secret_key_hash, settings, theme_pb, theme_revision, paywall_pb, paywall_revision, copy_pb, copy_revision, push_pb, created_at, updated_at`

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

// SetProjectPush installs push as the project's stored push-settings document
// (a moth.projectconfig.v1.StoredPush protobuf, see internal/push). Plain
// config: a full replacement with no revision history, unlike the
// paywall/theme documents.
func (s *Store) SetProjectPush(ctx context.Context, projectID string, push []byte, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE projects SET push_pb = ?, updated_at = ? WHERE id = ?`,
		push, formatTime(now), projectID)
	if err != nil {
		return fmt.Errorf("set project push: %w", err)
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

	// Retire every key that could still validate a token — the active key AND
	// any grace-period keys left by an earlier graceful rotation. A hard reset
	// means every access token dies, so no rotated-out key may survive in the
	// JWKS to keep validating tokens signed by it.
	if _, err := tx.ExecContext(ctx,
		`UPDATE project_keys SET status = ? WHERE project_id = ? AND status != ?`,
		ProjectKeyStatusRetired, projectID, ProjectKeyStatusRetired); err != nil {
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

const projectKeyColumns = `id, project_id, kid, algorithm, public_key_pem,
	private_key_enc, status, rotated_at, not_after, created_at`

func scanProjectKeyRow(row rowScanner) (ProjectKey, error) {
	var k ProjectKey
	var rotatedAt, notAfter sql.NullString
	var createdAt string
	if err := row.Scan(&k.ID, &k.ProjectID, &k.Kid, &k.Algorithm,
		&k.PublicKeyPEM, &k.PrivateKeyEnc, &k.Status, &rotatedAt, &notAfter, &createdAt); err != nil {
		return ProjectKey{}, err
	}
	var err error
	if k.RotatedAt, err = parseNullTime(rotatedAt); err != nil {
		return ProjectKey{}, fmt.Errorf("parse project key rotated_at: %w", err)
	}
	if k.NotAfter, err = parseNullTime(notAfter); err != nil {
		return ProjectKey{}, fmt.Errorf("parse project key not_after: %w", err)
	}
	if k.CreatedAt, err = parseTime(createdAt); err != nil {
		return ProjectKey{}, fmt.Errorf("parse project key created_at: %w", err)
	}
	return k, nil
}

func scanProjectKeys(rows *sql.Rows) ([]ProjectKey, error) {
	defer rows.Close()
	var keys []ProjectKey
	for rows.Next() {
		k, err := scanProjectKeyRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan project key: %w", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list project keys: %w", err)
	}
	return keys, nil
}

// ListActiveProjectKeys returns only the keys signing new tokens (status
// "active"), excluding grace-period keys.
func (s *Store) ListActiveProjectKeys(ctx context.Context, projectID string) ([]ProjectKey, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+projectKeyColumns+`
		 FROM project_keys WHERE project_id = ? AND status = ? ORDER BY created_at, id`,
		projectID, ProjectKeyStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list project keys: %w", err)
	}
	return scanProjectKeys(rows)
}

// ListActiveAndGraceKeys returns the keys a project's JWKS must publish: the
// active signing key plus any grace-period keys whose grace has not expired
// at now. Callers build the JWKS from these so tokens signed by a rotated-out
// key keep validating until they expire.
func (s *Store) ListActiveAndGraceKeys(ctx context.Context, projectID string, now time.Time) ([]ProjectKey, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+projectKeyColumns+`
		 FROM project_keys
		 WHERE project_id = ?
		   AND (status = ? OR (status = ? AND (not_after IS NULL OR not_after > ?)))
		 ORDER BY created_at, id`,
		projectID, ProjectKeyStatusActive, ProjectKeyStatusGrace, formatTime(now))
	if err != nil {
		return nil, fmt.Errorf("list active and grace keys: %w", err)
	}
	return scanProjectKeys(rows)
}

// RotateSigningKey installs k as the project's new active signing key while
// moving the current active key(s) to grace status (kept in the JWKS until
// graceUntil). Unlike ResetProjectSigningKey, it does NOT revoke refresh
// tokens: existing sessions and in-flight access tokens survive so no user
// is signed out.
func (s *Store) RotateSigningKey(ctx context.Context, projectID string, k ProjectKey, graceUntil, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin rotate signing key: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`UPDATE project_keys SET status = ?, rotated_at = ?, not_after = ?
		 WHERE project_id = ? AND status = ?`,
		ProjectKeyStatusGrace, formatTime(now), formatTime(graceUntil),
		projectID, ProjectKeyStatusActive); err != nil {
		return fmt.Errorf("move active keys to grace: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO project_keys (id, project_id, kid, algorithm, public_key_pem, private_key_enc, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		k.ID, k.ProjectID, k.Kid, k.Algorithm, k.PublicKeyPEM, k.PrivateKeyEnc,
		k.Status, formatTime(k.CreatedAt)); err != nil {
		return fmt.Errorf("insert rotated signing key: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rotate signing key: %w", err)
	}
	return nil
}

// PruneExpiredKeys deletes grace-period keys whose grace ended at or before
// now, across all projects, and reports how many were removed. Active and
// hard-retired keys are untouched. Run periodically by the maintenance loop.
func (s *Store) PruneExpiredKeys(ctx context.Context, now time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM project_keys
		 WHERE status = ? AND not_after IS NOT NULL AND not_after <= ?`,
		ProjectKeyStatusGrace, formatTime(now))
	if err != nil {
		return 0, fmt.Errorf("prune expired keys: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prune expired keys: %w", err)
	}
	return n, nil
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
		&settings, &p.Theme, &p.ThemeRevisionID, &p.Paywall, &p.PaywallRevisionID,
		&p.Copy, &p.CopyRevisionID, &p.Push, &createdAt, &updatedAt)
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
