package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"google.golang.org/protobuf/proto"

	projectconfigv1 "github.com/aloisdeniel/moth/gen/moth/projectconfig/v1"
)

// CopyRevisionKeep is how many revisions of a project's copy overrides are
// kept for undo; SetProjectCopy prunes older ones. Mirrors ThemeRevisionKeep
// and PaywallRevisionKeep.
const CopyRevisionKeep = 10

// ErrInvalidCopy is returned by the copy mutators when the overrides fail
// catalog validation or a required argument is missing; RPC handlers map it to
// connect.CodeInvalidArgument.
var ErrInvalidCopy = errors.New("invalid copy")

// copySaveAttempts bounds the compare-and-swap retries the read-modify-write
// helpers do; racing admin edits resolve in one or two rounds, anything more
// is a bug. Mirrors adminrpc.themeSaveAttempts.
const copySaveAttempts = 5

// CopyOverrides is a project's copy customization: BCP-47 locale tag →
// message key → override string. It is the full override document stored as
// a moth.projectconfig.v1.StoredCopy protobuf message in projects.copy_pb. An
// empty map (or an absent document) means the
// project renders the bundled catalog defaults everywhere; overrides are
// additive on top of the catalog. The store treats the message keys and
// values opaquely — validation against the bundled catalog is a CopyValidator
// concern (internal/i18n), injected so the store never depends on it.
type CopyOverrides map[string]map[string]string

// CopyValidator validates a project's copy overrides against the bundled
// message catalog: every key must exist in the catalog, every locale must be
// a well-formed tag, and each value must satisfy its key's required-placeholder
// and length contract. The catalog (internal/i18n) is injected through this
// interface so the store has no dependency on it. Implementations return an
// error whose message is safe to map to connect.CodeInvalidArgument.
type CopyValidator interface {
	ValidateCopyOverrides(overrides CopyOverrides) error
}

// CopyRevision is one saved version of a project's copy overrides. Copy is the
// raw override protobuf document (moth.projectconfig.v1.StoredCopy, a CopyOverrides
// map); the low-level primitives treat it opaquely, exactly like
// ThemeRevision/PaywallRevision.
type CopyRevision struct {
	ID        string
	ProjectID string
	Copy      []byte
	CreatedAt time.Time
}

// parseCopyOverrides decodes a stored override document
// (moth.projectconfig.v1.StoredCopy); empty bytes (a never-customized or reset
// project) decode to an empty, non-nil map.
func parseCopyOverrides(raw []byte) (CopyOverrides, error) {
	o := CopyOverrides{}
	if len(raw) == 0 {
		return o, nil
	}
	var msg projectconfigv1.StoredCopy
	if err := proto.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("parse copy overrides: %w", err)
	}
	for locale, msgs := range msg.GetLocales() {
		kept := map[string]string{}
		for k, v := range msgs.GetMessages() {
			kept[k] = v
		}
		o[locale] = kept
	}
	return o, nil
}

// encodeCopyOverrides serializes overrides for storage as a
// moth.projectconfig.v1.StoredCopy message, dropping empty locales and empty values
// so the document only ever holds meaningful overrides. It returns nil when
// nothing is left, so an emptied document reads back as the bundled default —
// the same empty == default convention themes/paywalls use.
func encodeCopyOverrides(o CopyOverrides) ([]byte, error) {
	msg := &projectconfigv1.StoredCopy{Locales: map[string]*projectconfigv1.CopyLocaleMessages{}}
	for locale, msgs := range o {
		kept := map[string]string{}
		for k, v := range msgs {
			if v != "" {
				kept[k] = v
			}
		}
		if len(kept) > 0 {
			msg.Locales[locale] = &projectconfigv1.CopyLocaleMessages{Messages: kept}
		}
	}
	if len(msg.Locales) == 0 {
		return nil, nil
	}
	raw, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("encode copy overrides: %w", err)
	}
	return raw, nil
}

// SetProjectCopy installs rev as the project's current copy overrides and
// appends it to the revision history in one transaction, pruning the history
// to the newest CopyRevisionKeep entries.
//
// The install is a compare-and-swap: prevRevisionID is the revision the caller
// read before mutating (empty for a never-customized or reset project), and
// the save returns ErrConflict when a concurrent save moved the project past
// it — so two racing read-modify-write cycles can never silently drop each
// other's changes. Mirrors SetProjectTheme / SetProjectPaywall.
func (s *Store) SetProjectCopy(ctx context.Context, rev CopyRevision, prevRevisionID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin set project copy: %w", err)
	}
	defer tx.Rollback()

	// The legacy TEXT column is frozen at '' since migration 0019; the
	// document lives in copy_pb.
	res, err := tx.ExecContext(ctx,
		`UPDATE projects SET copy = '', copy_pb = ?, copy_revision = ?, updated_at = ?
		 WHERE id = ? AND copy_revision = ?`,
		rev.Copy, rev.ID, formatTime(rev.CreatedAt), rev.ProjectID, prevRevisionID)
	if err != nil {
		return fmt.Errorf("update project copy: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("update project copy: %w", err)
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
		`INSERT INTO copy_revisions (id, project_id, copy, copy_pb, created_at) VALUES (?, ?, '', ?, ?)`,
		rev.ID, rev.ProjectID, rev.Copy, formatTime(rev.CreatedAt)); err != nil {
		return fmt.Errorf("insert copy revision: %w", err)
	}
	// Order by the UUIDv7 id, which is monotonic and lexically sortable in
	// creation order. created_at is RFC3339Nano TEXT: trailing zeros in the
	// fractional seconds are trimmed, so ".5Z" sorts after ".54Z" and a text
	// sort is not chronological. The id is the reliable newest-first key.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM copy_revisions WHERE project_id = ? AND id NOT IN (
			SELECT id FROM copy_revisions WHERE project_id = ?
			ORDER BY id DESC LIMIT ?
		)`, rev.ProjectID, rev.ProjectID, CopyRevisionKeep); err != nil {
		return fmt.Errorf("prune copy revisions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit set project copy: %w", err)
	}
	return nil
}

// ClearProjectCopy resets the project to the built-in default copy (the
// bundled catalog with no overrides). The revision history is kept, so the
// previous overrides stay restorable. Mirrors ClearProjectTheme.
func (s *Store) ClearProjectCopy(ctx context.Context, projectID string, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE projects SET copy = '', copy_pb = X'', copy_revision = '', updated_at = ? WHERE id = ?`,
		formatTime(now), projectID)
	if err != nil {
		return fmt.Errorf("clear project copy: %w", err)
	}
	return requireRow(res)
}

func (s *Store) GetCopyRevision(ctx context.Context, projectID, revisionID string) (CopyRevision, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, copy_pb, created_at FROM copy_revisions
		 WHERE project_id = ? AND id = ?`, projectID, revisionID)
	rev, err := scanCopyRevision(row)
	if errors.Is(err, sql.ErrNoRows) {
		return CopyRevision{}, ErrNotFound
	}
	return rev, err
}

// ListCopyRevisions returns the project's saved revisions, newest first. A
// limit <= 0 or above CopyRevisionKeep returns everything kept.
func (s *Store) ListCopyRevisions(ctx context.Context, projectID string, limit int) ([]CopyRevision, error) {
	if limit <= 0 || limit > CopyRevisionKeep {
		limit = CopyRevisionKeep
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, copy_pb, created_at FROM copy_revisions
		 WHERE project_id = ? ORDER BY id DESC LIMIT ?`,
		projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("list copy revisions: %w", err)
	}
	defer rows.Close()

	var revs []CopyRevision
	for rows.Next() {
		rev, err := scanCopyRevision(rows)
		if err != nil {
			return nil, err
		}
		revs = append(revs, rev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list copy revisions: %w", err)
	}
	return revs, nil
}

// GetProjectCopy returns the project's current copy overrides (parsed) and the
// id of the revision they came from ("" when the project renders the bundled
// defaults). Read path only; write paths go through the mutators below.
func (s *Store) GetProjectCopy(ctx context.Context, projectID string) (CopyOverrides, string, error) {
	p, err := s.GetProject(ctx, projectID)
	if err != nil {
		return nil, "", err
	}
	o, err := parseCopyOverrides(p.Copy)
	if err != nil {
		return nil, "", err
	}
	return o, p.CopyRevisionID, nil
}

// UpdateProjectCopy merges values (a message key → override string map) into one
// locale's overrides and saves a new revision, returning its id. Each entry is
// upserted (non-empty value) or cleared (empty value) key-by-key; keys the
// caller does not send are left untouched, so an editor that saves a single
// screen never clobbers another screen's overrides for the same locale. A
// locale left with no overrides is removed. When validate is non-nil the
// fully-merged override document is checked against the bundled catalog before
// the save (ErrInvalidCopy on failure). id is the new revision id
// (caller-generated UUIDv7). The save is optimistic: racing edits retry the
// read-modify-write and never drop each other. When the result leaves the
// project with no overrides at all it clears to the bundled default and returns
// an empty revision id. Mirrors adminrpc.mutateTheme, living in the store so the
// copy contract is self-contained.
func (s *Store) UpdateProjectCopy(ctx context.Context, projectID, locale string, values map[string]string, id string, now time.Time, validate CopyValidator) (string, error) {
	if locale == "" {
		return "", fmt.Errorf("%w: locale is required", ErrInvalidCopy)
	}
	return s.mutateCopy(ctx, projectID, id, now, validate, func(o CopyOverrides) {
		cur := o[locale]
		if cur == nil {
			cur = map[string]string{}
		}
		for k, v := range values {
			if v != "" {
				cur[k] = v
			} else {
				// An empty value clears that key's override (reset to default),
				// so an editor can clear a field and save without a separate RPC.
				delete(cur, k)
			}
		}
		if len(cur) == 0 {
			delete(o, locale)
			return
		}
		o[locale] = cur
	})
}

// ResetCopy reverts one key (key != "") or a whole locale (key == "") to the
// bundled default by removing it from the overrides, saving a new revision and
// returning its id. Removing the last override clears the project back to the
// bundled default and returns an empty revision id. No catalog validation is
// needed — removing an override can only make the document more default.
func (s *Store) ResetCopy(ctx context.Context, projectID, locale, key, id string, now time.Time) (string, error) {
	if locale == "" {
		return "", fmt.Errorf("%w: locale is required", ErrInvalidCopy)
	}
	return s.mutateCopy(ctx, projectID, id, now, nil, func(o CopyOverrides) {
		if key == "" {
			delete(o, locale)
			return
		}
		delete(o[locale], key)
		if len(o[locale]) == 0 {
			delete(o, locale)
		}
	})
}

// RestoreCopyRevision re-installs an old revision's override document as a new
// revision (history only ever moves forward), returning the new revision id.
// The stored document is trusted (it was valid when saved); no re-validation.
// An empty stored document restores the project to the bundled default and
// returns an empty revision id.
func (s *Store) RestoreCopyRevision(ctx context.Context, projectID, revisionID, id string, now time.Time) (string, error) {
	rev, err := s.GetCopyRevision(ctx, projectID, revisionID)
	if err != nil {
		return "", err
	}
	restored, err := parseCopyOverrides(rev.Copy)
	if err != nil {
		return "", err
	}
	// Restore replaces the current overrides wholesale, whatever their state,
	// so it stays usable as the recovery path; the mutation ignores what it
	// reads.
	return s.mutateCopy(ctx, projectID, id, now, nil, func(o CopyOverrides) {
		for k := range o {
			delete(o, k)
		}
		for locale, msgs := range restored {
			cp := map[string]string{}
			for k, v := range msgs {
				cp[k] = v
			}
			o[locale] = cp
		}
	})
}

// ListAvailableLocales returns the project's available locales: its default
// locale (always present) plus every locale it has overrides for, sorted and
// deduplicated. defaultLocale is injected — it is the negotiation/settings
// concern, not stored with the copy document.
func (s *Store) ListAvailableLocales(ctx context.Context, projectID, defaultLocale string) ([]string, error) {
	o, _, err := s.GetProjectCopy(ctx, projectID)
	if err != nil {
		return nil, err
	}
	set := map[string]struct{}{}
	if defaultLocale != "" {
		set[defaultLocale] = struct{}{}
	}
	for locale := range o {
		set[locale] = struct{}{}
	}
	locales := make([]string, 0, len(set))
	for locale := range set {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	return locales, nil
}

// mutateCopy runs a read-modify-write cycle on the project's copy overrides
// under optimistic concurrency: mut edits the parsed overrides in place, the
// result is (optionally) validated, encoded and saved with a revision CAS,
// retried from a fresh read when a concurrent save lands in between. Returns
// the new revision id, or "" when the mutation emptied the document (the
// project is cleared back to the bundled default).
func (s *Store) mutateCopy(ctx context.Context, projectID, id string, now time.Time, validate CopyValidator, mut func(CopyOverrides)) (string, error) {
	for attempt := 1; ; attempt++ {
		p, err := s.GetProject(ctx, projectID)
		if err != nil {
			return "", err
		}
		o, err := parseCopyOverrides(p.Copy)
		if err != nil {
			return "", err
		}
		mut(o)
		if validate != nil {
			if err := validate.ValidateCopyOverrides(o); err != nil {
				return "", fmt.Errorf("%w: %s", ErrInvalidCopy, err)
			}
		}
		raw, err := encodeCopyOverrides(o)
		if err != nil {
			return "", err
		}
		if len(raw) == 0 {
			// The document is now fully default; clear rather than storing an
			// empty revision, mirroring theme/paywall reset semantics.
			if err := s.ClearProjectCopy(ctx, projectID, now); err != nil {
				return "", err
			}
			return "", nil
		}
		rev := CopyRevision{ID: id, ProjectID: projectID, Copy: raw, CreatedAt: now}
		err = s.SetProjectCopy(ctx, rev, p.CopyRevisionID)
		if errors.Is(err, ErrConflict) {
			if attempt < copySaveAttempts {
				continue
			}
			return "", ErrConflict
		}
		if err != nil {
			return "", err
		}
		return rev.ID, nil
	}
}

func scanCopyRevision(row rowScanner) (CopyRevision, error) {
	var rev CopyRevision
	var createdAt string
	if err := row.Scan(&rev.ID, &rev.ProjectID, &rev.Copy, &createdAt); err != nil {
		return CopyRevision{}, err
	}
	var err error
	if rev.CreatedAt, err = parseTime(createdAt); err != nil {
		return CopyRevision{}, fmt.Errorf("parse copy revision created_at: %w", err)
	}
	return rev, nil
}
