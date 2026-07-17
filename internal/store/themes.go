package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ThemeRevisionKeep is how many revisions of a project's theme are kept
// for undo; SetProjectTheme prunes older ones.
const ThemeRevisionKeep = 10

// ThemeRevision is one saved version of a project's design-system theme.
// Theme is the raw versioned protobuf document (moth.storage.v1.StoredTheme)
// produced by internal/theme; the store never interprets it.
type ThemeRevision struct {
	ID        string
	ProjectID string
	Theme     []byte
	CreatedAt time.Time
}

// SetProjectTheme installs rev as the project's current theme and appends
// it to the revision history in one transaction, pruning the history to
// the newest ThemeRevisionKeep entries.
//
// The install is a compare-and-swap: prevRevisionID is the revision the
// caller read before mutating (empty for a never-customized or reset
// project), and the save returns ErrConflict when a concurrent save moved
// the project past it — so two racing read-modify-write cycles can never
// silently drop each other's changes.
func (s *Store) SetProjectTheme(ctx context.Context, rev ThemeRevision, prevRevisionID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin set project theme: %w", err)
	}
	defer tx.Rollback()

	// The legacy TEXT column is frozen at '' since migration 0019; the
	// document lives in theme_pb.
	res, err := tx.ExecContext(ctx,
		`UPDATE projects SET theme = '', theme_pb = ?, theme_revision = ?, updated_at = ?
		 WHERE id = ? AND theme_revision = ?`,
		rev.Theme, rev.ID, formatTime(rev.CreatedAt), rev.ProjectID, prevRevisionID)
	if err != nil {
		return fmt.Errorf("update project theme: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("update project theme: %w", err)
	} else if n == 0 {
		// Zero rows is either a missing project or a lost CAS; tell the
		// caller which, so a conflict can be retried.
		var exists bool
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) > 0 FROM projects WHERE id = ?`, rev.ProjectID,
		).Scan(&exists); err != nil {
			return fmt.Errorf("check project exists: %w", err)
		}
		if !exists {
			return ErrNotFound
		}
		return ErrConflict
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO theme_revisions (id, project_id, theme, theme_pb, created_at) VALUES (?, ?, '', ?, ?)`,
		rev.ID, rev.ProjectID, rev.Theme, formatTime(rev.CreatedAt)); err != nil {
		return fmt.Errorf("insert theme revision: %w", err)
	}
	// Order by the UUIDv7 id, which is monotonic and lexically sortable in
	// creation order. created_at is RFC3339Nano TEXT: trailing zeros in the
	// fractional seconds are trimmed, so ".5Z" sorts after ".54Z" and a text
	// sort is not chronological. The id is the reliable newest-first key.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM theme_revisions WHERE project_id = ? AND id NOT IN (
			SELECT id FROM theme_revisions WHERE project_id = ?
			ORDER BY id DESC LIMIT ?
		)`, rev.ProjectID, rev.ProjectID, ThemeRevisionKeep); err != nil {
		return fmt.Errorf("prune theme revisions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit set project theme: %w", err)
	}
	return nil
}

// ClearProjectTheme resets the project to the built-in default theme. The
// revision history is kept, so the previous theme stays restorable.
func (s *Store) ClearProjectTheme(ctx context.Context, projectID string, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE projects SET theme = '', theme_pb = X'', theme_revision = '', updated_at = ? WHERE id = ?`,
		formatTime(now), projectID)
	if err != nil {
		return fmt.Errorf("clear project theme: %w", err)
	}
	return requireRow(res)
}

func (s *Store) GetThemeRevision(ctx context.Context, projectID, revisionID string) (ThemeRevision, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, theme_pb, created_at FROM theme_revisions
		 WHERE project_id = ? AND id = ?`, projectID, revisionID)
	rev, err := scanThemeRevision(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ThemeRevision{}, ErrNotFound
	}
	return rev, err
}

// ListThemeRevisions returns the project's saved revisions, newest first.
// A limit <= 0 or above ThemeRevisionKeep returns everything kept.
func (s *Store) ListThemeRevisions(ctx context.Context, projectID string, limit int) ([]ThemeRevision, error) {
	if limit <= 0 || limit > ThemeRevisionKeep {
		limit = ThemeRevisionKeep
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, theme_pb, created_at FROM theme_revisions
		 WHERE project_id = ? ORDER BY id DESC LIMIT ?`,
		projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("list theme revisions: %w", err)
	}
	defer rows.Close()

	var revs []ThemeRevision
	for rows.Next() {
		rev, err := scanThemeRevision(rows)
		if err != nil {
			return nil, err
		}
		revs = append(revs, rev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list theme revisions: %w", err)
	}
	return revs, nil
}

func scanThemeRevision(row rowScanner) (ThemeRevision, error) {
	var rev ThemeRevision
	var createdAt string
	if err := row.Scan(&rev.ID, &rev.ProjectID, &rev.Theme, &createdAt); err != nil {
		return ThemeRevision{}, err
	}
	var err error
	if rev.CreatedAt, err = parseTime(createdAt); err != nil {
		return ThemeRevision{}, fmt.Errorf("parse theme revision created_at: %w", err)
	}
	return rev, nil
}
