package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Push targets (the `target` column): which push service the credential
// belongs to, i.e. which API the developer's backend must call to reach the
// device. An iOS app using FCM registers as fcm — platform is display
// metadata only.
const (
	PushTargetAPNs    = "apns"
	PushTargetFCM     = "fcm"
	PushTargetWebPush = "webpush"
)

// OS-level notification-permission states the client last reported (the
// `permission` column). A denied registration is kept (data pushes may still
// work) but flagged so senders can skip alert pushes.
const (
	PushPermissionGranted     = "granted"
	PushPermissionProvisional = "provisional"
	PushPermissionDenied      = "denied"
	PushPermissionUnknown     = "unknown"
)

// Push revoke reasons (the `revoke_reason` column). Rows are revoked, never
// deleted, so invalidation is auditable and idempotent.
const (
	PushRevokeReasonSignedOut       = "signed_out"       // client UnregisterDevice on sign-out
	PushRevokeReasonReportedInvalid = "reported_invalid" // sender feedback (410 Gone / UNREGISTERED)
	PushRevokeReasonStale           = "stale"            // staleness sweep
	PushRevokeReasonReplaced        = "replaced"         // superseded by a re-register upsert
	PushRevokeReasonAdmin           = "admin"            // operator revoke from the admin panel
)

// PushDevice is one push-notification device registration of a (project,
// user). Token is the raw push credential, stored plaintext — senders need it
// back over the secret-key surface; it never appears in admin RPCs.
type PushDevice struct {
	ID           string
	ProjectID    string
	UserID       string
	Target       string
	Token        string
	DeviceID     string
	Permission   string
	Platform     string
	Model        string
	OSVersion    string
	AppVersion   string
	Locale       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastSeenAt   time.Time
	RevokedAt    *time.Time
	RevokeReason string // "" while active
}

// PushDevicePage selects one page of a project's active registrations, newest
// first (IDs are UUIDv7, so ID order is creation order). AfterID is the last
// ID of the previous page; empty means the first page. Target "" matches all
// targets.
type PushDevicePage struct {
	Target  string
	AfterID string
	Limit   int
}

// UpsertPushDevice registers d for its (project, user). An active row with
// the same (user, target, token, device_id) is refreshed in place —
// permission, metadata, updated_at and last_seen_at are overwritten from d
// while id/created_at are preserved. Any other active row sharing d's
// device_id or its (target, token) credential is superseded (revoked
// 'replaced', which also handles token rotation and a device changing users)
// and a fresh row is inserted with d's id and timestamps. Returns the stored
// row.
func (s *Store) UpsertPushDevice(ctx context.Context, d PushDevice) (PushDevice, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PushDevice{}, fmt.Errorf("upsert push device: %w", err)
	}
	defer tx.Rollback()

	// Every active row this registration collides with: the installation's own
	// row (device_id) and/or the credential's current owner (target, token).
	rows, err := tx.QueryContext(ctx,
		selectPushDevice+` WHERE project_id = ? AND revoked_at IS NULL
		        AND (device_id = ? OR (target = ? AND token = ?))`,
		d.ProjectID, d.DeviceID, d.Target, d.Token)
	if err != nil {
		return PushDevice{}, fmt.Errorf("upsert push device: %w", err)
	}
	existing, err := collectPushDevices(rows)
	if err != nil {
		return PushDevice{}, err
	}

	// Exact re-register (same installation, same credential, same user):
	// refresh in place instead of superseding, so created_at keeps meaning
	// "first registered" and permission/metadata changes round-trip.
	if len(existing) == 1 {
		e := existing[0]
		if e.UserID == d.UserID && e.DeviceID == d.DeviceID && e.Target == d.Target && e.Token == d.Token {
			if _, err := tx.ExecContext(ctx,
				`UPDATE push_devices SET permission = ?, platform = ?, model = ?, os_version = ?,
				        app_version = ?, locale = ?, updated_at = ?, last_seen_at = ?
				  WHERE id = ?`,
				d.Permission, d.Platform, d.Model, d.OSVersion, d.AppVersion, d.Locale,
				formatTime(d.UpdatedAt), formatTime(d.LastSeenAt), e.ID); err != nil {
				return PushDevice{}, fmt.Errorf("refresh push device: %w", err)
			}
			stored, err := getPushDeviceTx(ctx, tx, d.ProjectID, e.ID)
			if err != nil {
				return PushDevice{}, err
			}
			if err := tx.Commit(); err != nil {
				return PushDevice{}, fmt.Errorf("upsert push device: %w", err)
			}
			return stored, nil
		}
	}

	// Supersede whatever the new registration displaces (rotated token, new
	// installation for the token, or a new user on the same device).
	for _, e := range existing {
		if _, err := tx.ExecContext(ctx,
			`UPDATE push_devices SET revoked_at = ?, revoke_reason = ?, updated_at = ?
			  WHERE id = ? AND revoked_at IS NULL`,
			formatTime(d.UpdatedAt), PushRevokeReasonReplaced, formatTime(d.UpdatedAt), e.ID); err != nil {
			return PushDevice{}, fmt.Errorf("supersede push device: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO push_devices (id, project_id, user_id, target, token, device_id, permission,
		        platform, model, os_version, app_version, locale, created_at, updated_at,
		        last_seen_at, revoked_at, revoke_reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL)`,
		d.ID, d.ProjectID, d.UserID, d.Target, d.Token, d.DeviceID, d.Permission,
		d.Platform, d.Model, d.OSVersion, d.AppVersion, d.Locale,
		formatTime(d.CreatedAt), formatTime(d.UpdatedAt), formatTime(d.LastSeenAt)); err != nil {
		return PushDevice{}, fmt.Errorf("insert push device: %w", err)
	}
	stored, err := getPushDeviceTx(ctx, tx, d.ProjectID, d.ID)
	if err != nil {
		return PushDevice{}, err
	}
	if err := tx.Commit(); err != nil {
		return PushDevice{}, fmt.Errorf("upsert push device: %w", err)
	}
	return stored, nil
}

// GetPushDevice returns one registration (active or revoked) by id, or
// ErrNotFound.
func (s *Store) GetPushDevice(ctx context.Context, projectID, id string) (PushDevice, error) {
	row := s.db.QueryRowContext(ctx,
		selectPushDevice+` WHERE project_id = ? AND id = ?`, projectID, id)
	d, err := scanPushDevice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PushDevice{}, ErrNotFound
	}
	return d, err
}

// ListActivePushDevicesByUser returns a user's active registrations, newest
// first — everything a sender needs to fan out to one user's devices.
func (s *Store) ListActivePushDevicesByUser(ctx context.Context, projectID, userID string) ([]PushDevice, error) {
	rows, err := s.db.QueryContext(ctx,
		selectPushDevice+` WHERE project_id = ? AND user_id = ? AND revoked_at IS NULL
		  ORDER BY id DESC`, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("list user push devices: %w", err)
	}
	return collectPushDevices(rows)
}

// ListActivePushDevices returns one page of a project's active registrations
// for broadcast/segment sends, newest first, optionally filtered by target.
func (s *Store) ListActivePushDevices(ctx context.Context, projectID string, page PushDevicePage) ([]PushDevice, error) {
	q := selectPushDevice + ` WHERE project_id = ? AND revoked_at IS NULL`
	args := []any{projectID}
	if page.Target != "" {
		q += ` AND target = ?`
		args = append(args, page.Target)
	}
	if page.AfterID != "" {
		q += ` AND id < ?`
		args = append(args, page.AfterID)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, page.Limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list push devices: %w", err)
	}
	return collectPushDevices(rows)
}

// ListPushDevicesForAdmin returns a user's registrations for the admin
// Devices panel: every active row plus rows revoked after revokedSince (so
// the panel can show why a device recently disappeared), most recently seen
// first.
func (s *Store) ListPushDevicesForAdmin(ctx context.Context, projectID, userID string, revokedSince time.Time) ([]PushDevice, error) {
	rows, err := s.db.QueryContext(ctx,
		selectPushDevice+` WHERE project_id = ? AND user_id = ?
		        AND (revoked_at IS NULL OR revoked_at > ?)
		  ORDER BY last_seen_at DESC, id DESC`,
		projectID, userID, timeBound(revokedSince))
	if err != nil {
		return nil, fmt.Errorf("list push devices for admin: %w", err)
	}
	return collectPushDevices(rows)
}

// RevokePushDevice revokes one registration by row id (the admin revoke).
// Unknown or already-revoked ids are no-ops — revocation is idempotent.
func (s *Store) RevokePushDevice(ctx context.Context, projectID, id, reason string, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE push_devices SET revoked_at = ?, revoke_reason = ?, updated_at = ?
		  WHERE project_id = ? AND id = ? AND revoked_at IS NULL`,
		formatTime(now), reason, formatTime(now), projectID, id)
	if err != nil {
		return fmt.Errorf("revoke push device: %w", err)
	}
	return nil
}

// RevokePushDeviceByDeviceID revokes the calling user's registration for one
// installation (the client UnregisterDevice). Scoped to (project, user) so a
// user can only revoke their own row; unknown or already-revoked device ids
// are no-ops.
func (s *Store) RevokePushDeviceByDeviceID(ctx context.Context, projectID, userID, deviceID, reason string, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE push_devices SET revoked_at = ?, revoke_reason = ?, updated_at = ?
		  WHERE project_id = ? AND user_id = ? AND device_id = ? AND revoked_at IS NULL`,
		formatTime(now), reason, formatTime(now), projectID, userID, deviceID)
	if err != nil {
		return fmt.Errorf("revoke push device by device id: %w", err)
	}
	return nil
}

// RevokePushDeviceByTokenOrDevice revokes active registrations matching the
// token and/or device id — the secret-key feedback loop reporting a dead
// credential. Project-scoped only (the sender knows the credential, not the
// user); empty selectors match nothing, and unknown or already-revoked
// credentials are no-ops.
func (s *Store) RevokePushDeviceByTokenOrDevice(ctx context.Context, projectID, token, deviceID, reason string, now time.Time) error {
	conds := ""
	args := []any{formatTime(now), reason, formatTime(now), projectID}
	if token != "" {
		conds = `token = ?`
		args = append(args, token)
	}
	if deviceID != "" {
		if conds != "" {
			conds += ` OR `
		}
		conds += `device_id = ?`
		args = append(args, deviceID)
	}
	if conds == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE push_devices SET revoked_at = ?, revoke_reason = ?, updated_at = ?
		  WHERE project_id = ? AND (`+conds+`) AND revoked_at IS NULL`, args...)
	if err != nil {
		return fmt.Errorf("revoke push device by token or device: %w", err)
	}
	return nil
}

// RevokeStalePushDevices revokes active registrations not seen since cutoff,
// across every project (reason 'stale') — the milestone-10 background sweep
// decaying dead installs. Returns how many rows were revoked.
func (s *Store) RevokeStalePushDevices(ctx context.Context, cutoff, now time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE push_devices SET revoked_at = ?, revoke_reason = ?, updated_at = ?
		  WHERE revoked_at IS NULL AND last_seen_at < ?`,
		formatTime(now), PushRevokeReasonStale, formatTime(now), timeBound(cutoff))
	if err != nil {
		return 0, fmt.Errorf("revoke stale push devices: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("revoke stale push devices: %w", err)
	}
	return n, nil
}

// CountActivePushDevicesByTarget returns a project's active-registration
// counts keyed by target — the overview gauge read off push_devices.
func (s *Store) CountActivePushDevicesByTarget(ctx context.Context, projectID string) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT target, COUNT(*) FROM push_devices
		  WHERE project_id = ? AND revoked_at IS NULL GROUP BY target`, projectID)
	if err != nil {
		return nil, fmt.Errorf("count push devices: %w", err)
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var target string
		var n int
		if err := rows.Scan(&target, &n); err != nil {
			return nil, fmt.Errorf("scan push device count: %w", err)
		}
		counts[target] = n
	}
	return counts, rows.Err()
}

func getPushDeviceTx(ctx context.Context, tx *sql.Tx, projectID, id string) (PushDevice, error) {
	row := tx.QueryRowContext(ctx,
		selectPushDevice+` WHERE project_id = ? AND id = ?`, projectID, id)
	d, err := scanPushDevice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PushDevice{}, ErrNotFound
	}
	return d, err
}

func collectPushDevices(rows *sql.Rows) ([]PushDevice, error) {
	defer rows.Close()
	var out []PushDevice
	for rows.Next() {
		d, err := scanPushDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

const selectPushDevice = `SELECT id, project_id, user_id, target, token, device_id, permission,
	platform, model, os_version, app_version, locale, created_at, updated_at, last_seen_at,
	revoked_at, COALESCE(revoke_reason, '') FROM push_devices`

func scanPushDevice(row rowScanner) (PushDevice, error) {
	var d PushDevice
	var createdAt, updatedAt, lastSeenAt string
	var revokedAt sql.NullString
	if err := row.Scan(&d.ID, &d.ProjectID, &d.UserID, &d.Target, &d.Token, &d.DeviceID,
		&d.Permission, &d.Platform, &d.Model, &d.OSVersion, &d.AppVersion, &d.Locale,
		&createdAt, &updatedAt, &lastSeenAt, &revokedAt, &d.RevokeReason); err != nil {
		return PushDevice{}, err
	}
	var err error
	if d.CreatedAt, err = parseTime(createdAt); err != nil {
		return PushDevice{}, fmt.Errorf("parse push device created_at: %w", err)
	}
	if d.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return PushDevice{}, fmt.Errorf("parse push device updated_at: %w", err)
	}
	if d.LastSeenAt, err = parseTime(lastSeenAt); err != nil {
		return PushDevice{}, fmt.Errorf("parse push device last_seen_at: %w", err)
	}
	if d.RevokedAt, err = parseNullTime(revokedAt); err != nil {
		return PushDevice{}, fmt.Errorf("parse push device revoked_at: %w", err)
	}
	return d, nil
}
