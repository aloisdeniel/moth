package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// PaywallRevisionKeep is how many revisions of a project's paywall config are
// kept for undo; SetProjectPaywall prunes older ones. Mirrors
// ThemeRevisionKeep.
const PaywallRevisionKeep = 10

// PaywallRevision is one saved version of a project's paywall config. Paywall
// is the raw versioned JSON document produced by internal/paywall; the store
// never interprets it.
type PaywallRevision struct {
	ID        string
	ProjectID string
	Paywall   string
	CreatedAt time.Time
}

// SetProjectPaywall installs rev as the project's current paywall config and
// appends it to the revision history in one transaction, pruning the history
// to the newest PaywallRevisionKeep entries.
//
// The install is a compare-and-swap: prevRevisionID is the revision the
// caller read before mutating (empty for a never-customized or reset
// project), and the save returns ErrConflict when a concurrent save moved the
// project past it — so two racing read-modify-write cycles can never silently
// drop each other's changes. Mirrors SetProjectTheme.
func (s *Store) SetProjectPaywall(ctx context.Context, rev PaywallRevision, prevRevisionID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin set project paywall: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE projects SET paywall = ?, paywall_revision = ?, updated_at = ?
		 WHERE id = ? AND paywall_revision = ?`,
		rev.Paywall, rev.ID, formatTime(rev.CreatedAt), rev.ProjectID, prevRevisionID)
	if err != nil {
		return fmt.Errorf("update project paywall: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("update project paywall: %w", err)
	} else if n == 0 {
		// Zero rows is either a missing project or a lost CAS; tell the caller
		// which, so a conflict can be retried.
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
		`INSERT INTO paywall_revisions (id, project_id, paywall, created_at) VALUES (?, ?, ?, ?)`,
		rev.ID, rev.ProjectID, rev.Paywall, formatTime(rev.CreatedAt)); err != nil {
		return fmt.Errorf("insert paywall revision: %w", err)
	}
	// Order by the UUIDv7 id, which is monotonic and lexically sortable in
	// creation order. created_at is RFC3339Nano TEXT: trailing zeros in the
	// fractional seconds are trimmed, so ".5Z" sorts after ".54Z" and a text
	// sort is not chronological. The id is the reliable newest-first key.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM paywall_revisions WHERE project_id = ? AND id NOT IN (
			SELECT id FROM paywall_revisions WHERE project_id = ?
			ORDER BY id DESC LIMIT ?
		)`, rev.ProjectID, rev.ProjectID, PaywallRevisionKeep); err != nil {
		return fmt.Errorf("prune paywall revisions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit set project paywall: %w", err)
	}
	return nil
}

// ClearProjectPaywall resets the project to the built-in default paywall
// config. The revision history is kept, so the previous config stays
// restorable.
func (s *Store) ClearProjectPaywall(ctx context.Context, projectID string, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE projects SET paywall = '', paywall_revision = '', updated_at = ? WHERE id = ?`,
		formatTime(now), projectID)
	if err != nil {
		return fmt.Errorf("clear project paywall: %w", err)
	}
	return requireRow(res)
}

func (s *Store) GetPaywallRevision(ctx context.Context, projectID, revisionID string) (PaywallRevision, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, paywall, created_at FROM paywall_revisions
		 WHERE project_id = ? AND id = ?`, projectID, revisionID)
	rev, err := scanPaywallRevision(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PaywallRevision{}, ErrNotFound
	}
	return rev, err
}

// ListPaywallRevisions returns the project's saved revisions, newest first.
// A limit <= 0 or above PaywallRevisionKeep returns everything kept.
func (s *Store) ListPaywallRevisions(ctx context.Context, projectID string, limit int) ([]PaywallRevision, error) {
	if limit <= 0 || limit > PaywallRevisionKeep {
		limit = PaywallRevisionKeep
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, paywall, created_at FROM paywall_revisions
		 WHERE project_id = ? ORDER BY id DESC LIMIT ?`,
		projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("list paywall revisions: %w", err)
	}
	defer rows.Close()

	var revs []PaywallRevision
	for rows.Next() {
		rev, err := scanPaywallRevision(rows)
		if err != nil {
			return nil, err
		}
		revs = append(revs, rev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list paywall revisions: %w", err)
	}
	return revs, nil
}

func scanPaywallRevision(row rowScanner) (PaywallRevision, error) {
	var rev PaywallRevision
	var createdAt string
	if err := row.Scan(&rev.ID, &rev.ProjectID, &rev.Paywall, &createdAt); err != nil {
		return PaywallRevision{}, err
	}
	var err error
	if rev.CreatedAt, err = parseTime(createdAt); err != nil {
		return PaywallRevision{}, fmt.Errorf("parse paywall revision created_at: %w", err)
	}
	return rev, nil
}
