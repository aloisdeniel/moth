package serverapi

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	serverv1 "github.com/aloisdeniel/moth/gen/moth/server/v1"
	"github.com/aloisdeniel/moth/gen/moth/server/v1/serverv1connect"
	"github.com/aloisdeniel/moth/internal/store"
)

const (
	defaultPushPageSize = 50
	maxPushPageSize     = 200
)

// PushStore is what the developer-backend push service reads and revokes.
type PushStore interface {
	store.UserStore
	store.PushDeviceStore
}

var _ serverv1connect.PushServiceHandler = (*PushHandler)(nil)

// PushHandler implements moth.server.v1.PushService: it hands the developer's
// backend — the sender — the project's active push registrations, credentials
// included, and takes dead-credential reports back (the feedback loop). This
// is the only surface that ever returns tokens; project scoping comes from
// the secret key resolved by the interceptor, so one project's sender can
// never see another's registrations.
type PushHandler struct {
	store PushStore
	now   func() time.Time
}

// NewPushHandler builds the service. now is injectable for tests; nil means
// time.Now.
func NewPushHandler(st PushStore, now func() time.Time) *PushHandler {
	if now == nil {
		now = time.Now
	}
	return &PushHandler{store: st, now: now}
}

// ListUserPushDevices returns one user's active registrations, most recently
// seen first. A bad user id is NotFound; a user with no registrations is an
// empty list, never an error.
func (h *PushHandler) ListUserPushDevices(ctx context.Context, req *connect.Request[serverv1.ListUserPushDevicesRequest]) (*connect.Response[serverv1.ListUserPushDevicesResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	// Confirm the user exists so a bad id is NotFound, not a silent empty set.
	if _, err := h.store.GetUser(ctx, proj.ID, req.Msg.UserId); err != nil {
		return nil, userErr(err)
	}
	devices, err := h.store.ListActivePushDevicesByUser(ctx, proj.ID, req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// The store lists newest-created first (id DESC); the sender wants the
	// liveness order.
	sortByLastSeen(devices)
	resp := &serverv1.ListUserPushDevicesResponse{}
	for _, d := range devices {
		resp.Devices = append(resp.Devices, serverPushDeviceProto(d))
	}
	return connect.NewResponse(resp), nil
}

// ListPushDevices pages through the project's active registrations, newest
// first, optionally filtered by target. The page token is the id of the last
// row of the previous page (ids are UUIDv7, so id order is creation order).
func (h *PushHandler) ListPushDevices(ctx context.Context, req *connect.Request[serverv1.ListPushDevicesRequest]) (*connect.Response[serverv1.ListPushDevicesResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	size := int(req.Msg.PageSize)
	if size < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pageSize must not be negative"))
	}
	if size == 0 {
		size = defaultPushPageSize
	}
	if size > maxPushPageSize {
		size = maxPushPageSize
	}
	target := ""
	if req.Msg.Target != serverv1.PushTarget_PUSH_TARGET_UNSPECIFIED {
		if target, err = serverPushTargetFromProto(req.Msg.Target); err != nil {
			return nil, err
		}
	}
	// One extra row decides whether a further page exists.
	devices, err := h.store.ListActivePushDevices(ctx, proj.ID, store.PushDevicePage{
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
	resp := &serverv1.ListPushDevicesResponse{NextPageToken: next}
	for _, d := range devices {
		resp.Devices = append(resp.Devices, serverPushDeviceProto(d))
	}
	return connect.NewResponse(resp), nil
}

// RevokePushDevice is the feedback loop: the sender reports a credential the
// push service rejected (by token or by installation device_id) and moth
// revokes it (`reported_invalid`) so it never serves it again. Idempotent:
// unknown or already-revoked credentials succeed.
func (h *PushHandler) RevokePushDevice(ctx context.Context, req *connect.Request[serverv1.RevokePushDeviceRequest]) (*connect.Response[serverv1.RevokePushDeviceResponse], error) {
	proj, err := project(ctx)
	if err != nil {
		return nil, err
	}
	var token, deviceID string
	switch sel := req.Msg.Selector.(type) {
	case *serverv1.RevokePushDeviceRequest_Token:
		token = sel.Token
	case *serverv1.RevokePushDeviceRequest_DeviceId:
		deviceID = sel.DeviceId
	}
	if token == "" && deviceID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("either token or deviceId is required"))
	}
	if err := h.store.RevokePushDeviceByTokenOrDevice(ctx, proj.ID, token, deviceID,
		store.PushRevokeReasonReportedInvalid, h.now()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&serverv1.RevokePushDeviceResponse{}), nil
}

// sortByLastSeen orders registrations most-recently-seen first (id DESC as
// the stable tiebreak).
func sortByLastSeen(devices []store.PushDevice) {
	slices.SortStableFunc(devices, func(a, b store.PushDevice) int {
		if c := b.LastSeenAt.Compare(a.LastSeenAt); c != 0 {
			return c
		}
		return strings.Compare(b.ID, a.ID)
	})
}

// --- proto mappers ---------------------------------------------------------

func serverPushTargetFromProto(t serverv1.PushTarget) (string, error) {
	switch t {
	case serverv1.PushTarget_PUSH_TARGET_APNS:
		return store.PushTargetAPNs, nil
	case serverv1.PushTarget_PUSH_TARGET_FCM:
		return store.PushTargetFCM, nil
	case serverv1.PushTarget_PUSH_TARGET_WEBPUSH:
		return store.PushTargetWebPush, nil
	default:
		return "", connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unknown target %d", t))
	}
}

func serverPushTargetProto(t string) serverv1.PushTarget {
	switch t {
	case store.PushTargetAPNs:
		return serverv1.PushTarget_PUSH_TARGET_APNS
	case store.PushTargetFCM:
		return serverv1.PushTarget_PUSH_TARGET_FCM
	case store.PushTargetWebPush:
		return serverv1.PushTarget_PUSH_TARGET_WEBPUSH
	default:
		return serverv1.PushTarget_PUSH_TARGET_UNSPECIFIED
	}
}

func serverPushPermissionProto(p string) serverv1.PushPermission {
	switch p {
	case store.PushPermissionGranted:
		return serverv1.PushPermission_PUSH_PERMISSION_GRANTED
	case store.PushPermissionProvisional:
		return serverv1.PushPermission_PUSH_PERMISSION_PROVISIONAL
	case store.PushPermissionDenied:
		return serverv1.PushPermission_PUSH_PERMISSION_DENIED
	case store.PushPermissionUnknown:
		return serverv1.PushPermission_PUSH_PERMISSION_UNKNOWN
	default:
		return serverv1.PushPermission_PUSH_PERMISSION_UNSPECIFIED
	}
}

// serverPushDeviceProto builds the sender view of a registration — the only
// representation that carries the push credential.
func serverPushDeviceProto(d store.PushDevice) *serverv1.PushDevice {
	return &serverv1.PushDevice{
		Id:         d.ID,
		UserId:     d.UserID,
		Target:     serverPushTargetProto(d.Target),
		Token:      d.Token,
		DeviceId:   d.DeviceID,
		Permission: serverPushPermissionProto(d.Permission),
		Metadata: &serverv1.PushDeviceMetadata{
			Platform:   d.Platform,
			Model:      d.Model,
			OsVersion:  d.OSVersion,
			AppVersion: d.AppVersion,
			Locale:     d.Locale,
		},
		CreateTime:   timestamppb.New(d.CreatedAt),
		UpdateTime:   timestamppb.New(d.UpdatedAt),
		LastSeenTime: timestamppb.New(d.LastSeenAt),
	}
}
