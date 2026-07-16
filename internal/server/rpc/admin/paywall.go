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
	"github.com/aloisdeniel/moth/internal/paywall"
	"github.com/aloisdeniel/moth/internal/store"
)

// PaywallHandler implements moth.admin.v1.PaywallService: the per-project
// paywall configuration (plan/13). Configs are validated by internal/paywall
// (bounded copy, known layout, http(s) legal links) before every save, and
// every save is a new revision (the store keeps the last
// store.PaywallRevisionKeep for undo). It mirrors ThemeHandler exactly, minus
// the logo asset plumbing — the paywall inherits colors/typography/logo from
// the theme and owns no assets of its own.
type PaywallHandler struct {
	store Store
	audit *Auditor
	now   func() time.Time
}

var _ adminv1connect.PaywallServiceHandler = (*PaywallHandler)(nil)

// NewPaywallHandler builds the paywall service.
func NewPaywallHandler(st Store, auditor *Auditor) *PaywallHandler {
	return &PaywallHandler{store: st, audit: auditor, now: time.Now}
}

func (h *PaywallHandler) GetPaywallConfig(ctx context.Context, req *connect.Request[adminv1.GetPaywallConfigRequest]) (*connect.Response[adminv1.GetPaywallConfigResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	c, isDefault := currentPaywall(p)
	resp := &adminv1.GetPaywallConfigResponse{Config: paywallProto(c), IsDefault: isDefault}
	if !isDefault {
		resp.RevisionId = p.PaywallRevisionID
	}
	return connect.NewResponse(resp), nil
}

func (h *PaywallHandler) UpdatePaywallConfig(ctx context.Context, req *connect.Request[adminv1.UpdatePaywallConfigRequest]) (*connect.Response[adminv1.UpdatePaywallConfigResponse], error) {
	if req.Msg.Config == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("config is required"))
	}
	c, rev, err := h.mutatePaywall(ctx, req.Msg.ProjectId, func(store.Project) (paywall.Config, error) {
		c := paywallFromProto(req.Msg.Config)
		if err := c.Validate(); err != nil {
			return paywall.Config{}, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return c, nil
	})
	if err != nil {
		return nil, err
	}
	h.audit.record(ctx, entry{
		Action: ActionPaywallUpdate, TargetType: "paywall", TargetID: req.Msg.ProjectId,
		ProjectID: req.Msg.ProjectId, Summary: "Updated the paywall configuration",
	})
	return connect.NewResponse(&adminv1.UpdatePaywallConfigResponse{
		Config:     paywallProto(c),
		RevisionId: rev,
	}), nil
}

func (h *PaywallHandler) ListPaywallRevisions(ctx context.Context, req *connect.Request[adminv1.ListPaywallRevisionsRequest]) (*connect.Response[adminv1.ListPaywallRevisionsResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	revs, err := h.store.ListPaywallRevisions(ctx, p.ID, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &adminv1.ListPaywallRevisionsResponse{}
	for _, rev := range revs {
		c, err := paywall.Parse([]byte(rev.Paywall))
		if err != nil {
			// A revision this binary cannot parse (newer schema or corrupt) is
			// skipped rather than failing the whole list: the rest stay
			// restorable.
			continue
		}
		resp.Revisions = append(resp.Revisions, &adminv1.PaywallRevision{
			RevisionId: rev.ID,
			Config:     paywallProto(c),
			CreateTime: timestamppb.New(rev.CreatedAt),
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *PaywallHandler) RestorePaywallRevision(ctx context.Context, req *connect.Request[adminv1.RestorePaywallRevisionRequest]) (*connect.Response[adminv1.RestorePaywallRevisionResponse], error) {
	rev, err := h.store.GetPaywallRevision(ctx, req.Msg.ProjectId, req.Msg.RevisionId)
	if errors.Is(err, store.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("paywall revision not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	restored, err := paywall.Parse([]byte(rev.Paywall))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("stored paywall revision %s: %w", rev.ID, err))
	}
	// Restore deliberately replaces the current config whatever its state, so it
	// stays usable as the recovery path for a corrupt document; the mutation
	// never reads the current config.
	c, newRev, err := h.mutatePaywall(ctx, req.Msg.ProjectId, func(store.Project) (paywall.Config, error) {
		return restored, nil
	})
	if err != nil {
		return nil, err
	}
	h.audit.record(ctx, entry{
		Action: ActionPaywallRestore, TargetType: "paywall", TargetID: req.Msg.ProjectId,
		ProjectID: req.Msg.ProjectId,
		Summary:   fmt.Sprintf("Restored paywall revision %s", req.Msg.RevisionId),
	})
	return connect.NewResponse(&adminv1.RestorePaywallRevisionResponse{
		Config:     paywallProto(c),
		RevisionId: newRev,
	}), nil
}

func (h *PaywallHandler) ResetPaywall(ctx context.Context, req *connect.Request[adminv1.ResetPaywallRequest]) (*connect.Response[adminv1.ResetPaywallResponse], error) {
	if err := h.store.ClearProjectPaywall(ctx, req.Msg.ProjectId, h.now()); err != nil {
		return nil, projectErr(err)
	}
	h.audit.record(ctx, entry{
		Action: ActionPaywallReset, TargetType: "paywall", TargetID: req.Msg.ProjectId,
		ProjectID: req.Msg.ProjectId, Summary: "Reset the paywall to defaults",
	})
	return connect.NewResponse(&adminv1.ResetPaywallResponse{
		Config: paywallProto(paywall.Default()),
	}), nil
}

// paywallSaveAttempts bounds mutatePaywall's compare-and-swap retries; racing
// admin edits resolve in one or two rounds, anything more is a bug.
const paywallSaveAttempts = 5

// mutatePaywall runs a read-modify-write cycle on the project's paywall config
// under optimistic concurrency, mirroring ThemeHandler.mutateTheme: mut builds
// the new config from a fresh project read, and the save is retried from
// scratch when a concurrent save lands in between (SetProjectPaywall's revision
// CAS). Returned errors are already connect-coded, as are the ones mut returns.
func (h *PaywallHandler) mutatePaywall(ctx context.Context, projectID string, mut func(p store.Project) (paywall.Config, error)) (paywall.Config, string, error) {
	for attempt := 1; ; attempt++ {
		p, err := h.store.GetProject(ctx, projectID)
		if err != nil {
			return paywall.Config{}, "", projectErr(err)
		}
		c, err := mut(p)
		if err != nil {
			return paywall.Config{}, "", err
		}
		raw, err := paywall.Encode(c)
		if err != nil {
			return paywall.Config{}, "", connect.NewError(connect.CodeInternal, err)
		}
		rev := store.PaywallRevision{
			ID:        NewID(),
			ProjectID: projectID,
			Paywall:   string(raw),
			CreatedAt: h.now(),
		}
		err = h.store.SetProjectPaywall(ctx, rev, p.PaywallRevisionID)
		if errors.Is(err, store.ErrConflict) {
			if attempt < paywallSaveAttempts {
				continue
			}
			return paywall.Config{}, "", connect.NewError(connect.CodeAborted,
				errors.New("paywall changed concurrently; retry the edit"))
		}
		if err != nil {
			return paywall.Config{}, "", projectErr(err)
		}
		return c, rev.ID, nil
	}
}

// currentPaywall returns the config the project renders with and whether it is
// the built-in default (never customized, reset, or — defensively — a corrupt
// stored document).
func currentPaywall(p store.Project) (paywall.Config, bool) {
	if p.Paywall == "" {
		return paywall.Default(), true
	}
	c, err := paywall.Parse([]byte(p.Paywall))
	if err != nil {
		return paywall.Default(), true
	}
	return c, false
}

func paywallLayoutProto(layout string) adminv1.PaywallLayout {
	switch layout {
	case paywall.LayoutTiles:
		return adminv1.PaywallLayout_PAYWALL_LAYOUT_TILES
	case paywall.LayoutList:
		return adminv1.PaywallLayout_PAYWALL_LAYOUT_LIST
	case paywall.LayoutCompact:
		return adminv1.PaywallLayout_PAYWALL_LAYOUT_COMPACT
	default:
		return adminv1.PaywallLayout_PAYWALL_LAYOUT_UNSPECIFIED
	}
}

// paywallLayoutFromProto maps the proto enum to the domain string. An
// unspecified layout defaults to tiles, so a client that omits it still saves a
// valid config.
func paywallLayoutFromProto(layout adminv1.PaywallLayout) string {
	switch layout {
	case adminv1.PaywallLayout_PAYWALL_LAYOUT_LIST:
		return paywall.LayoutList
	case adminv1.PaywallLayout_PAYWALL_LAYOUT_COMPACT:
		return paywall.LayoutCompact
	default:
		return paywall.LayoutTiles
	}
}

func paywallProto(c paywall.Config) *adminv1.PaywallConfig {
	return &adminv1.PaywallConfig{
		Headline:                     c.Headline,
		Subtitle:                     c.Subtitle,
		Benefits:                     c.Benefits,
		Offering:                     c.Offering,
		HighlightedProductIdentifier: c.HighlightedIdentifier,
		Layout:                       paywallLayoutProto(c.Layout),
		Legal: &adminv1.PaywallLegal{
			TermsUrl:   c.Legal.TermsURL,
			PrivacyUrl: c.Legal.PrivacyURL,
		},
	}
}

// paywallFromProto converts the client message into the domain model (a nil
// legal sub-message becomes empty links). Values are validated by Validate.
func paywallFromProto(msg *adminv1.PaywallConfig) paywall.Config {
	c := paywall.Config{
		Version:               paywall.SchemaVersion,
		Headline:              msg.Headline,
		Subtitle:              msg.Subtitle,
		Benefits:              msg.Benefits,
		Offering:              msg.Offering,
		HighlightedIdentifier: msg.HighlightedProductIdentifier,
		Layout:                paywallLayoutFromProto(msg.Layout),
	}
	if l := msg.Legal; l != nil {
		c.Legal = paywall.Legal{TermsURL: l.TermsUrl, PrivacyURL: l.PrivacyUrl}
	}
	return c
}
