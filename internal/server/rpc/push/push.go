// Package pushrpc implements moth.push.v1.PushService — the publishable-key +
// Bearer client API for the push-device registry (milestone 20). moth
// registers; the developer's backend sends: this surface only records which
// push credential reaches the signed-in user's device, its permission state
// and its liveness. The registration always hangs off (project,
// authenticated user) — the user id comes from the Bearer token, never from
// the request — and the push credential is write-only here: RegisterDevice
// never echoes the token back (tokens are returned only over the secret-key
// surface, moth.server.v1).
package pushrpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	pushv1 "github.com/aloisdeniel/moth/gen/moth/push/v1"
	"github.com/aloisdeniel/moth/gen/moth/push/v1/pushv1connect"
	"github.com/aloisdeniel/moth/internal/push"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// Request bounds. Tokens are bounded generously because a Web Push
// registration is a whole serialized subscription (endpoint URL + keys), not
// a compact device token; the metadata fields are display strings.
const (
	maxTokenLen    = 4096
	maxDeviceIDLen = 128
	maxMetaLen     = 128
)

// Store is everything the push service needs from persistence. User
// authentication is delegated to the auth handler.
type Store interface {
	store.ProjectStore
	store.UserStore
	store.PushDeviceStore
}

var _ pushv1connect.PushServiceHandler = (*Handler)(nil)

// Handler implements moth.push.v1.PushService.
type Handler struct {
	store Store
	auth  *authrpc.Handler // Bearer user authentication (shared with auth.v1)
	now   func() time.Time
}

// Options configures the push handler.
type Options struct {
	Store Store
	Auth  *authrpc.Handler
	Now   func() time.Time
}

// New builds the push handler.
func New(o Options) *Handler {
	if o.Now == nil {
		o.Now = time.Now
	}
	return &Handler{store: o.Store, auth: o.Auth, now: o.Now}
}

// RegisterDevice upserts the calling user's registration and returns the
// stored row. Idempotent by design: the SDK calls it on every app launch,
// token rotation and permission change; the store resolves collisions (same
// device_id, or the same (target, token) under a new user) by superseding the
// displaced row (`replaced`). Every call refreshes last_seen_time.
func (h *Handler) RegisterDevice(ctx context.Context, req *connect.Request[pushv1.RegisterDeviceRequest]) (*connect.Response[pushv1.RegisterDeviceResponse], error) {
	project, user, err := h.auth.AuthenticateUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	// The enabled switch is enforced on registration writes: a project that
	// never turned push on must not accumulate credentials. FailedPrecondition
	// (not a silent no-op) so a misconfigured SDK integration surfaces
	// immediately instead of registering into the void.
	if !push.FromStored(project.Push).Enabled {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("push is not enabled for this project"))
	}
	target, err := targetFromProto(req.Msg.Target)
	if err != nil {
		return nil, err
	}
	if err := validateRegistration(req.Msg); err != nil {
		return nil, err
	}
	now := h.now()
	d := store.PushDevice{
		ID:         authrpc.NewID(),
		ProjectID:  project.ID,
		UserID:     user.ID,
		Target:     target,
		Token:      req.Msg.Token,
		DeviceID:   req.Msg.DeviceId,
		Permission: permissionFromProto(req.Msg.Permission),
		CreatedAt:  now,
		UpdatedAt:  now,
		LastSeenAt: now,
	}
	if m := req.Msg.Metadata; m != nil {
		d.Platform, d.Model, d.OSVersion, d.AppVersion, d.Locale =
			m.Platform, m.Model, m.OsVersion, m.AppVersion, m.Locale
	}
	stored, err := h.store.UpsertPushDevice(ctx, d)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pushv1.RegisterDeviceResponse{Device: deviceProto(stored)}), nil
}

// UnregisterDevice revokes the calling user's registration for one
// installation (`signed_out`). Idempotent: unknown or already-revoked device
// ids succeed — the SDK calls this on sign-out without bookkeeping.
func (h *Handler) UnregisterDevice(ctx context.Context, req *connect.Request[pushv1.UnregisterDeviceRequest]) (*connect.Response[pushv1.UnregisterDeviceResponse], error) {
	project, user, err := h.auth.AuthenticateUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	if req.Msg.DeviceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("deviceId is required"))
	}
	// Deliberately not gated on the enabled switch: revoking a credential is
	// always allowed, even after an operator turned push off.
	if err := h.store.RevokePushDeviceByDeviceID(ctx, project.ID, user.ID,
		req.Msg.DeviceId, store.PushRevokeReasonSignedOut, h.now()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pushv1.UnregisterDeviceResponse{}), nil
}

// validateRegistration bounds the client-supplied strings (the target enum is
// validated separately by targetFromProto).
func validateRegistration(msg *pushv1.RegisterDeviceRequest) error {
	if msg.Token == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("token is required"))
	}
	if len(msg.Token) > maxTokenLen {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("token: %d bytes exceeds the %d maximum", len(msg.Token), maxTokenLen))
	}
	if msg.DeviceId == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("deviceId is required"))
	}
	if len(msg.DeviceId) > maxDeviceIDLen {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("deviceId: %d bytes exceeds the %d maximum", len(msg.DeviceId), maxDeviceIDLen))
	}
	if m := msg.Metadata; m != nil {
		for name, v := range map[string]string{
			"platform": m.Platform, "model": m.Model, "osVersion": m.OsVersion,
			"appVersion": m.AppVersion, "locale": m.Locale,
		} {
			if len(v) > maxMetaLen {
				return connect.NewError(connect.CodeInvalidArgument,
					fmt.Errorf("metadata.%s: %d bytes exceeds the %d maximum", name, len(v), maxMetaLen))
			}
		}
	}
	return nil
}

// --- proto mappers ---------------------------------------------------------

func targetFromProto(t pushv1.PushTarget) (string, error) {
	switch t {
	case pushv1.PushTarget_PUSH_TARGET_APNS:
		return store.PushTargetAPNs, nil
	case pushv1.PushTarget_PUSH_TARGET_FCM:
		return store.PushTargetFCM, nil
	case pushv1.PushTarget_PUSH_TARGET_WEBPUSH:
		return store.PushTargetWebPush, nil
	default:
		return "", connect.NewError(connect.CodeInvalidArgument,
			errors.New("target is required (apns, fcm or webpush)"))
	}
}

func targetProto(t string) pushv1.PushTarget {
	switch t {
	case store.PushTargetAPNs:
		return pushv1.PushTarget_PUSH_TARGET_APNS
	case store.PushTargetFCM:
		return pushv1.PushTarget_PUSH_TARGET_FCM
	case store.PushTargetWebPush:
		return pushv1.PushTarget_PUSH_TARGET_WEBPUSH
	default:
		return pushv1.PushTarget_PUSH_TARGET_UNSPECIFIED
	}
}

// permissionFromProto maps the reported permission onto the stored state; an
// unspecified permission reads as unknown, so a minimal SDK that never asks
// still registers a valid row.
func permissionFromProto(p pushv1.PushPermission) string {
	switch p {
	case pushv1.PushPermission_PUSH_PERMISSION_GRANTED:
		return store.PushPermissionGranted
	case pushv1.PushPermission_PUSH_PERMISSION_PROVISIONAL:
		return store.PushPermissionProvisional
	case pushv1.PushPermission_PUSH_PERMISSION_DENIED:
		return store.PushPermissionDenied
	default:
		return store.PushPermissionUnknown
	}
}

func permissionProto(p string) pushv1.PushPermission {
	switch p {
	case store.PushPermissionGranted:
		return pushv1.PushPermission_PUSH_PERMISSION_GRANTED
	case store.PushPermissionProvisional:
		return pushv1.PushPermission_PUSH_PERMISSION_PROVISIONAL
	case store.PushPermissionDenied:
		return pushv1.PushPermission_PUSH_PERMISSION_DENIED
	case store.PushPermissionUnknown:
		return pushv1.PushPermission_PUSH_PERMISSION_UNKNOWN
	default:
		return pushv1.PushPermission_PUSH_PERMISSION_UNSPECIFIED
	}
}

// deviceProto builds the client view of a registration. No token by design:
// the client already holds its own credential, and tokens are only ever
// returned over the secret-key surface.
func deviceProto(d store.PushDevice) *pushv1.PushDevice {
	return &pushv1.PushDevice{
		Id:         d.ID,
		Target:     targetProto(d.Target),
		DeviceId:   d.DeviceID,
		Permission: permissionProto(d.Permission),
		Metadata: &pushv1.PushDeviceMetadata{
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
