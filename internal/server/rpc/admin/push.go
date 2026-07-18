package adminrpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/push"
	"github.com/aloisdeniel/moth/internal/store"
)

// PushHandler implements moth.admin.v1.PushService: the project push settings
// (enable switch + Web Push VAPID public key — plain config, full
// replacement, no revisions, unlike the paywall/theme documents) and the user
// detail Devices panel. Registrations are metadata-only here: push tokens are
// credentials and appear exclusively on the secret-key surface
// (moth.server.v1), never in admin responses.
type PushHandler struct {
	store Store
	audit *Auditor
	now   func() time.Time
}

var _ adminv1connect.PushServiceHandler = (*PushHandler)(nil)

// NewPushHandler builds the push admin service. now is injectable for tests;
// nil means time.Now.
func NewPushHandler(st Store, auditor *Auditor, now func() time.Time) *PushHandler {
	if now == nil {
		now = time.Now
	}
	return &PushHandler{store: st, audit: auditor, now: now}
}

// GetPushSettings returns the project's push settings; a project that never
// configured push gets the defaults (disabled, no VAPID key).
func (h *PushHandler) GetPushSettings(ctx context.Context, req *connect.Request[adminv1.GetPushSettingsRequest]) (*connect.Response[adminv1.GetPushSettingsResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	return connect.NewResponse(&adminv1.GetPushSettingsResponse{
		Settings: pushSettingsProto(push.FromStored(p.Push)),
	}), nil
}

// UpdatePushSettings validates and installs a full replacement of the push
// settings.
func (h *PushHandler) UpdatePushSettings(ctx context.Context, req *connect.Request[adminv1.UpdatePushSettingsRequest]) (*connect.Response[adminv1.UpdatePushSettingsResponse], error) {
	if req.Msg.Settings == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("settings is required"))
	}
	c := push.Config{
		Version:               push.SchemaVersion,
		Enabled:               req.Msg.Settings.Enabled,
		WebPushVAPIDPublicKey: req.Msg.Settings.WebpushVapidPublicKey,
	}
	if err := c.Validate(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	raw, err := push.Encode(c)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := h.store.SetProjectPush(ctx, req.Msg.ProjectId, raw, h.now()); err != nil {
		return nil, projectErr(err)
	}
	h.audit.record(ctx, entry{
		Action: ActionPushSettingsUpdate, TargetType: "push_settings", TargetID: req.Msg.ProjectId,
		ProjectID: req.Msg.ProjectId, Summary: "Updated the push settings",
	})
	return connect.NewResponse(&adminv1.UpdatePushSettingsResponse{
		Settings: pushSettingsProto(c),
	}), nil
}

// ListUserPushDevices returns one user's registrations for the user detail
// Devices panel, most recently seen first — active and revoked (revocation is
// auditable, not a delete), never the tokens.
func (h *PushHandler) ListUserPushDevices(ctx context.Context, req *connect.Request[adminv1.ListUserPushDevicesRequest]) (*connect.Response[adminv1.ListUserPushDevicesResponse], error) {
	if _, err := h.store.GetUser(ctx, req.Msg.ProjectId, req.Msg.UserId); err != nil {
		return nil, userErr(err)
	}
	// The zero revokedSince keeps every revoked row visible: rows are never
	// deleted, so the panel is the full auditable history.
	devices, err := h.store.ListPushDevicesForAdmin(ctx, req.Msg.ProjectId, req.Msg.UserId, time.Time{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListUserPushDevicesResponse{}
	for _, d := range devices {
		resp.Devices = append(resp.Devices, adminPushDeviceProto(d))
	}
	return connect.NewResponse(resp), nil
}

// Page bounds for the Push tab's project-wide listing, matching the
// secret-key surface's ListPushDevices.
const (
	defaultAdminPushPageSize = 50
	maxAdminPushPageSize     = 200
)

// ListPushDevices returns the project's active registrations for the Push
// tab, newest first with keyset pagination and an optional target filter.
// Each row carries the owning user's id and email so the operator can tell
// whose device it is; tokens never appear (adminv1.PushDevice has no token
// field). Project-wide per-target totals ride along on every page.
func (h *PushHandler) ListPushDevices(ctx context.Context, req *connect.Request[adminv1.ListPushDevicesRequest]) (*connect.Response[adminv1.ListPushDevicesResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	size := int(req.Msg.PageSize)
	if size < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pageSize must not be negative"))
	}
	if size == 0 {
		size = defaultAdminPushPageSize
	}
	if size > maxAdminPushPageSize {
		size = maxAdminPushPageSize
	}
	target := ""
	if req.Msg.Target != adminv1.PushTarget_PUSH_TARGET_UNSPECIFIED {
		if target, err = adminPushTargetFromProto(req.Msg.Target); err != nil {
			return nil, err
		}
	}
	// One extra row decides whether a further page exists.
	devices, err := h.store.ListActivePushDevices(ctx, p.ID, store.PushDevicePage{
		Target:  target,
		AfterID: req.Msg.PageToken,
		Limit:   size + 1,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	next := ""
	if len(devices) > size {
		devices = devices[:size]
		next = devices[len(devices)-1].ID
	}
	userIDs := make([]string, 0, len(devices))
	seen := make(map[string]bool, len(devices))
	for _, d := range devices {
		if !seen[d.UserID] {
			seen[d.UserID] = true
			userIDs = append(userIDs, d.UserID)
		}
	}
	emails, err := h.store.UserEmails(ctx, p.ID, userIDs)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	counts, err := h.store.CountActivePushDevicesByTarget(ctx, p.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListPushDevicesResponse{
		NextPageToken: next,
		ApnsCount:     int64(counts[store.PushTargetAPNs]),
		FcmCount:      int64(counts[store.PushTargetFCM]),
		WebpushCount:  int64(counts[store.PushTargetWebPush]),
	}
	for _, d := range devices {
		resp.Devices = append(resp.Devices, &adminv1.ProjectPushDevice{
			Device:    adminPushDeviceProto(d),
			UserId:    d.UserID,
			UserEmail: emails[d.UserID],
		})
	}
	return connect.NewResponse(resp), nil
}

// RevokePushDevice revokes one registration by its row id (`admin` reason,
// audit-logged). Idempotent: revoking an already-revoked registration
// succeeds, keeps the original reason and is not re-audited.
func (h *PushHandler) RevokePushDevice(ctx context.Context, req *connect.Request[adminv1.RevokePushDeviceRequest]) (*connect.Response[adminv1.RevokePushDeviceResponse], error) {
	// The read-first gives an unknown id a clean NotFound (the store revoke is
	// a deliberate no-op there) and tells replay apart from first revoke.
	d, err := h.store.GetPushDevice(ctx, req.Msg.ProjectId, req.Msg.PushDeviceId)
	if errors.Is(err, store.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("push device not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if d.RevokedAt == nil {
		if err := h.store.RevokePushDevice(ctx, req.Msg.ProjectId, d.ID,
			store.PushRevokeReasonAdmin, h.now()); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if d, err = h.store.GetPushDevice(ctx, req.Msg.ProjectId, d.ID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		h.audit.record(ctx, entry{
			Action: ActionPushDeviceRevoke, TargetType: "push_device", TargetID: d.ID,
			ProjectID: req.Msg.ProjectId,
			Summary:   fmt.Sprintf("Revoked push device %s (%s) of user %s", d.DeviceID, d.Target, d.UserID),
		})
	}
	return connect.NewResponse(&adminv1.RevokePushDeviceResponse{
		Device: adminPushDeviceProto(d),
	}), nil
}

// --- proto mappers ---------------------------------------------------------

func pushSettingsProto(c push.Config) *adminv1.PushSettings {
	return &adminv1.PushSettings{
		Enabled:               c.Enabled,
		WebpushVapidPublicKey: c.WebPushVAPIDPublicKey,
	}
}

func adminPushTargetProto(t string) adminv1.PushTarget {
	switch t {
	case store.PushTargetAPNs:
		return adminv1.PushTarget_PUSH_TARGET_APNS
	case store.PushTargetFCM:
		return adminv1.PushTarget_PUSH_TARGET_FCM
	case store.PushTargetWebPush:
		return adminv1.PushTarget_PUSH_TARGET_WEBPUSH
	default:
		return adminv1.PushTarget_PUSH_TARGET_UNSPECIFIED
	}
}

func adminPushTargetFromProto(t adminv1.PushTarget) (string, error) {
	switch t {
	case adminv1.PushTarget_PUSH_TARGET_APNS:
		return store.PushTargetAPNs, nil
	case adminv1.PushTarget_PUSH_TARGET_FCM:
		return store.PushTargetFCM, nil
	case adminv1.PushTarget_PUSH_TARGET_WEBPUSH:
		return store.PushTargetWebPush, nil
	default:
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown target %d", t))
	}
}

func adminPushPermissionProto(p string) adminv1.PushPermission {
	switch p {
	case store.PushPermissionGranted:
		return adminv1.PushPermission_PUSH_PERMISSION_GRANTED
	case store.PushPermissionProvisional:
		return adminv1.PushPermission_PUSH_PERMISSION_PROVISIONAL
	case store.PushPermissionDenied:
		return adminv1.PushPermission_PUSH_PERMISSION_DENIED
	case store.PushPermissionUnknown:
		return adminv1.PushPermission_PUSH_PERMISSION_UNKNOWN
	default:
		return adminv1.PushPermission_PUSH_PERMISSION_UNSPECIFIED
	}
}

func adminPushRevokeReasonProto(r string) adminv1.PushRevokeReason {
	switch r {
	case store.PushRevokeReasonSignedOut:
		return adminv1.PushRevokeReason_PUSH_REVOKE_REASON_SIGNED_OUT
	case store.PushRevokeReasonReportedInvalid:
		return adminv1.PushRevokeReason_PUSH_REVOKE_REASON_REPORTED_INVALID
	case store.PushRevokeReasonStale:
		return adminv1.PushRevokeReason_PUSH_REVOKE_REASON_STALE
	case store.PushRevokeReasonReplaced:
		return adminv1.PushRevokeReason_PUSH_REVOKE_REASON_REPLACED
	case store.PushRevokeReasonAdmin:
		return adminv1.PushRevokeReason_PUSH_REVOKE_REASON_ADMIN
	default:
		return adminv1.PushRevokeReason_PUSH_REVOKE_REASON_UNSPECIFIED
	}
}

// adminPushDeviceProto builds the operator view of a registration.
// Deliberately token-free: adminv1.PushDevice has no token field, and the
// store row's Token is never copied anywhere here.
func adminPushDeviceProto(d store.PushDevice) *adminv1.PushDevice {
	msg := &adminv1.PushDevice{
		Id:         d.ID,
		Target:     adminPushTargetProto(d.Target),
		DeviceId:   d.DeviceID,
		Permission: adminPushPermissionProto(d.Permission),
		Metadata: &adminv1.PushDeviceMetadata{
			Platform:   d.Platform,
			Model:      d.Model,
			OsVersion:  d.OSVersion,
			AppVersion: d.AppVersion,
			Locale:     d.Locale,
		},
		CreateTime:   timestamppb.New(d.CreatedAt),
		UpdateTime:   timestamppb.New(d.UpdatedAt),
		LastSeenTime: timestamppb.New(d.LastSeenAt),
		RevokeReason: adminPushRevokeReasonProto(d.RevokeReason),
	}
	if d.RevokedAt != nil {
		msg.RevokeTime = timestamppb.New(*d.RevokedAt)
	}
	return msg
}
