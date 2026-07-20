package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Audit actor types.
const (
	// AuditActorCookie is an admin acting through a browser session cookie.
	AuditActorCookie = "cookie"
	// AuditActorPAT is an admin acting through a personal access token.
	AuditActorPAT = "pat"
	// AuditActorSystem is the server itself (background jobs, automatic
	// security actions such as refresh-token family revocation).
	AuditActorSystem = "system"
)

// AuditEntry is one append-only audit record. IDs are UUIDv7, so id order is
// creation order and doubles as the pagination cursor.
type AuditEntry struct {
	ID         string
	ActorType  string
	ActorID    string
	ActorLabel string
	Action     string
	TargetType string
	TargetID   string
	// ProjectID is empty for instance-level actions.
	ProjectID string
	Summary   string
	// BeforeAfter is an optional JSON change description; empty when absent.
	BeforeAfter string
	// IP is a coarse or hashed client address.
	IP        string
	CreatedAt time.Time
}

// AuditFilter narrows and pages ListAudit. Empty fields are ignored. Results
// are newest-first; AfterID is the id of the last row of the previous page.
type AuditFilter struct {
	ProjectID string
	ActorID   string
	Action    string
	// From and To bound created_at (inclusive From, exclusive To); zero
	// values disable the respective bound.
	From    time.Time
	To      time.Time
	AfterID string
	Limit   int
}

// AppendAudit inserts one audit record. The log is append-only; there is no
// update or single-row delete.
func (s *Store) AppendAudit(ctx context.Context, e AuditEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_log (id, actor_type, actor_id, actor_label, action,
		                        target_type, target_id, project_id, summary,
		                        before_after, ip, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.ActorType, e.ActorID, e.ActorLabel, e.Action,
		e.TargetType, e.TargetID, nullString(e.ProjectID), e.Summary,
		nullString(e.BeforeAfter), e.IP, formatTime(e.CreatedAt))
	if err != nil {
		return fmt.Errorf("append audit: %w", err)
	}
	return nil
}

const auditColumns = `id, actor_type, actor_id, actor_label, action,
	target_type, target_id, project_id, summary, before_after, ip, created_at`

// ListAudit returns audit entries matching filter, newest first.
func (s *Store) ListAudit(ctx context.Context, filter AuditFilter) ([]AuditEntry, error) {
	q := `SELECT ` + auditColumns + ` FROM audit_log WHERE 1 = 1`
	args := []any{}
	if filter.ProjectID != "" {
		q += ` AND project_id = ?`
		args = append(args, filter.ProjectID)
	}
	if filter.ActorID != "" {
		q += ` AND actor_id = ?`
		args = append(args, filter.ActorID)
	}
	if filter.Action != "" {
		q += ` AND action = ?`
		args = append(args, filter.Action)
	}
	if !filter.From.IsZero() {
		q += ` AND created_at >= ?`
		args = append(args, formatTime(filter.From))
	}
	if !filter.To.IsZero() {
		q += ` AND created_at < ?`
		args = append(args, formatTime(filter.To))
	}
	if filter.AfterID != "" {
		q += ` AND id < ?`
		args = append(args, filter.AfterID)
	}
	q += ` ORDER BY id DESC`
	if filter.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		e, err := scanAuditRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	return entries, nil
}

func scanAuditRow(row rowScanner) (AuditEntry, error) {
	var e AuditEntry
	var projectID, beforeAfter sql.NullString
	var createdAt string
	if err := row.Scan(&e.ID, &e.ActorType, &e.ActorID, &e.ActorLabel, &e.Action,
		&e.TargetType, &e.TargetID, &projectID, &e.Summary, &beforeAfter,
		&e.IP, &createdAt); err != nil {
		return AuditEntry{}, fmt.Errorf("scan audit: %w", err)
	}
	e.ProjectID = projectID.String
	e.BeforeAfter = beforeAfter.String
	var err error
	if e.CreatedAt, err = parseTime(createdAt); err != nil {
		return AuditEntry{}, fmt.Errorf("parse audit created_at: %w", err)
	}
	return e, nil
}
