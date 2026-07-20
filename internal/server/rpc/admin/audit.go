package adminrpc

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/audit"
	"github.com/aloisdeniel/moth/internal/store"
)

// Audit action names. Machine-readable, dotted <target>.<verb>; the human
// summary carries the specifics.
const (
	ActionProjectCreate    = "project.create"
	ActionProjectUpdate    = "project.update"
	ActionProjectDelete    = "project.delete"
	ActionSecretKeyRegen   = "project.secret_key.regenerate"
	ActionSigningKeyReset  = "signing_key.reset"
	ActionSigningKeyRotate = "signing_key.rotate"
	ActionProjectExport    = "project.export"
	ActionProjectImport    = "project.import"
	ActionProviderUpdate   = "provider.update"
	ActionUserCreate       = "user.create"
	ActionUserUpdate       = "user.update"
	ActionUserDisable      = "user.disable"
	ActionUserEnable       = "user.enable"
	ActionUserDelete       = "user.delete"
	ActionUserSessionsRvk  = "user.sessions.revoke"
	ActionUserPwReset      = "user.password_reset.send"
	ActionPATCreate        = "pat.create"
	ActionPATRevoke        = "pat.revoke"
	ActionAdminInvite      = "admin.invite"
	ActionAdminAccept      = "admin.invite.accept"
	ActionAdminPassword    = "admin.password.change"
	ActionSMTPUpdate       = "smtp.update"
	ActionThemeUpdate      = "theme.update"
	ActionThemeRestore     = "theme.restore"
	ActionThemeReset       = "theme.reset"
	ActionThemeLogoUpload  = "theme.logo.upload"
	ActionThemeLogoDelete  = "theme.logo.delete"
	// Milestone 11 — subscriptions & entitlements.
	ActionEntitlementCreate  = "entitlement.create"
	ActionEntitlementUpdate  = "entitlement.update"
	ActionEntitlementDelete  = "entitlement.delete"
	ActionProductCreate      = "product.create"
	ActionProductUpdate      = "product.update"
	ActionProductDelete      = "product.delete"
	ActionGrantCreate        = "subscription.grant"
	ActionGrantRevoke        = "subscription.grant.revoke"
	ActionBillingCredsUpdate = "billing.credentials.update"
	// Milestone 12 — store catalog provisioning.
	ActionStoreCatalogSync = "billing.store_catalog.sync"
	ActionOfferingReorder  = "billing.offering.reorder"
	// Milestone 13 — themed paywall.
	ActionPaywallUpdate  = "paywall.update"
	ActionPaywallRestore = "paywall.restore"

	ActionCopyUpdate   = "copy.update"
	ActionCopyReset    = "copy.reset"
	ActionCopyRestore  = "copy.restore"
	ActionPaywallReset = "paywall.reset"

	// Milestone 20 — push device registry.
	ActionPushSettingsUpdate = "push.settings.update"
	ActionPushDeviceRevoke   = "push.device.revoke"

	// Milestone 22 — setup profile (the creation wizard's answers).
	ActionProfileUpdate = "profile.update"
)

type clientIPCtxKey struct{}

// withClientIP stores the coarse client IP resolved by the auth interceptor.
func withClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPCtxKey{}, ip)
}

// clientIPFromContext returns the coarse client IP for the current request.
func clientIPFromContext(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPCtxKey{}).(string)
	return ip
}

// Auditor records admin actions, attributing each to the credential that
// authenticated the request (browser session vs personal access token) and
// tagging it with the coarse client IP. Writes never fail the request.
type Auditor struct {
	sink *audit.Sink
}

// NewAuditor builds an Auditor over the audit sink.
func NewAuditor(sink *audit.Sink) *Auditor { return &Auditor{sink: sink} }

// entry is the per-action specifics a handler supplies; the actor, IP, id and
// timestamp are filled in from context.
type entry struct {
	Action      string
	TargetType  string
	TargetID    string
	ProjectID   string
	Summary     string
	BeforeAfter string
}

// record attributes e to the ctx credential and appends it. Safe on a nil
// Auditor (handlers built without auditing in tests).
func (a *Auditor) record(ctx context.Context, e entry) {
	if a == nil || a.sink == nil {
		return
	}
	actorType := store.AuditActorCookie
	if cred, ok := CredentialFromContext(ctx); ok && cred.Type == CredentialPAT {
		actorType = store.AuditActorPAT
	}
	var actorID, actorLabel string
	if admin, ok := AdminFromContext(ctx); ok {
		actorID = admin.ID
		actorLabel = admin.Email
	}
	a.recordAs(ctx, actorType, actorID, actorLabel, e)
}

// recordAs appends e attributed to an explicit actor, for flows that create
// their own credential (AcceptAdminInvite runs before any admin is in ctx).
func (a *Auditor) recordAs(ctx context.Context, actorType, actorID, actorLabel string, e entry) {
	if a == nil || a.sink == nil {
		return
	}
	a.sink.Append(ctx, store.AuditEntry{
		ActorType:   actorType,
		ActorID:     actorID,
		ActorLabel:  actorLabel,
		Action:      e.Action,
		TargetType:  e.TargetType,
		TargetID:    e.TargetID,
		ProjectID:   e.ProjectID,
		Summary:     e.Summary,
		BeforeAfter: e.BeforeAfter,
		IP:          clientIPFromContext(ctx),
	})
}

const (
	defaultAuditPageSize = 50
	maxAuditPageSize     = 200
)

// AuditHandler implements moth.admin.v1.AuditService.
type AuditHandler struct {
	store store.AuditStore
}

// NewAuditHandler builds the audit viewer service.
func NewAuditHandler(st store.AuditStore) *AuditHandler {
	return &AuditHandler{store: st}
}

// ListAuditLog returns audit entries newest-first, narrowed by the filters
// and paged with page_size / page_token. The token is the id of the last row
// of the previous page (audit ids are UUIDv7, so id order is time order).
func (h *AuditHandler) ListAuditLog(ctx context.Context, req *connect.Request[adminv1.ListAuditLogRequest]) (*connect.Response[adminv1.ListAuditLogResponse], error) {
	size := int(req.Msg.PageSize)
	if size <= 0 {
		size = defaultAuditPageSize
	}
	if size > maxAuditPageSize {
		size = maxAuditPageSize
	}
	filter := store.AuditFilter{
		ProjectID: req.Msg.ProjectId,
		ActorID:   req.Msg.ActorId,
		Action:    req.Msg.Action,
		AfterID:   req.Msg.PageToken,
		// One extra row decides whether a further page exists.
		Limit: size + 1,
	}
	if req.Msg.StartTime != nil {
		filter.From = req.Msg.StartTime.AsTime()
	}
	if req.Msg.EndTime != nil {
		filter.To = req.Msg.EndTime.AsTime()
	}
	entries, err := h.store.ListAudit(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	next := ""
	if len(entries) > size {
		entries = entries[:size]
		next = entries[len(entries)-1].ID
	}
	resp := &adminv1.ListAuditLogResponse{NextPageToken: next}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, auditEntryProto(e))
	}
	return connect.NewResponse(resp), nil
}

func auditEntryProto(e store.AuditEntry) *adminv1.AuditEntry {
	return &adminv1.AuditEntry{
		Id:          e.ID,
		ActorType:   e.ActorType,
		ActorId:     e.ActorID,
		ActorLabel:  e.ActorLabel,
		Action:      e.Action,
		TargetType:  e.TargetType,
		TargetId:    e.TargetID,
		ProjectId:   e.ProjectID,
		Summary:     e.Summary,
		BeforeAfter: e.BeforeAfter,
		Ip:          e.IP,
		CreateTime:  timestamppb.New(e.CreatedAt),
	}
}
