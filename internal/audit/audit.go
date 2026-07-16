// Package audit records the append-only admin/security audit log. It wraps
// store.AuditStore with fire-and-forget semantics: a write failure is logged
// but never propagated, so auditing can never fail the request it describes.
package audit

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/google/uuid"

	"github.com/aloisdeniel/moth/internal/store"
)

// Sink appends audit records. Construct one per instance and share it across
// the handlers and the security-event emitters.
type Sink struct {
	store store.AuditStore
	log   *slog.Logger
	now   func() time.Time
}

// New builds a Sink. now defaults to time.Now.
func New(st store.AuditStore, log *slog.Logger, now func() time.Time) *Sink {
	if log == nil {
		log = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &Sink{store: st, log: log, now: now}
}

// Append writes one audit entry, filling in its id and timestamp when the
// caller left them zero. It never blocks the caller on failure: a store error
// is logged and swallowed, because losing an audit line must not fail (or
// roll back) the action it records.
func (s *Sink) Append(ctx context.Context, e store.AuditEntry) {
	if s == nil {
		return
	}
	if e.ID == "" {
		if id, err := uuid.NewV7(); err == nil {
			e.ID = id.String()
		}
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = s.now().UTC()
	}
	// Detach from the request context so a cancelled/closed request cannot
	// abort the audit write.
	writeCtx := context.WithoutCancel(ctx)
	if err := s.store.AppendAudit(writeCtx, e); err != nil {
		s.log.ErrorContext(ctx, "append audit log",
			"action", e.Action, "target_type", e.TargetType,
			"target_id", e.TargetID, "error", err.Error())
	}
}

// CoarseIP reduces a "host:port" or bare-host address to a coarse network
// prefix so the audit log locates an action without storing a precise
// client address: IPv4 is masked to /24, IPv6 to /48. Unparseable input is
// returned as-is (already coarse or empty).
func CoarseIP(addr string) string {
	host := addr
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.Mask(net.CIDRMask(24, 32)).String() + "/24"
	}
	return ip.Mask(net.CIDRMask(48, 128)).String() + "/48"
}
