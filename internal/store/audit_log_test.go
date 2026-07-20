package store

import (
	"context"
	"testing"
	"time"
)

func TestAuditAppendAndList(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Date(2026, 7, 16, 9, 0, 0, 0, time.UTC)

	entries := []AuditEntry{
		{ID: "01A", ActorType: AuditActorCookie, ActorID: "admin1", ActorLabel: "ops@example.com",
			Action: "project.create", TargetType: "project", TargetID: "p1", ProjectID: "p1",
			Summary: "created project", IP: "203.0.113.0/24", CreatedAt: base},
		{ID: "01B", ActorType: AuditActorPAT, ActorID: "admin1", ActorLabel: "ci-token",
			Action: "user.disable", TargetType: "user", TargetID: "u9", ProjectID: "p1",
			BeforeAfter: `{"disabled":[false,true]}`, CreatedAt: base.Add(time.Minute)},
		{ID: "01C", ActorType: AuditActorSystem, ActorID: "", Action: "refresh.family_revoked",
			TargetType: "user", TargetID: "u9", ProjectID: "p2", CreatedAt: base.Add(2 * time.Minute)},
	}
	for _, e := range entries {
		if err := s.AppendAudit(ctx, e); err != nil {
			t.Fatal(err)
		}
	}

	// Default: newest first, all rows.
	all, err := s.ListAudit(ctx, AuditFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].ID != "01C" || all[2].ID != "01A" {
		t.Fatalf("want newest-first [01C 01B 01A], got %+v", ids(all))
	}
	// Nullable columns round-trip.
	if all[0].ProjectID != "p2" || all[0].BeforeAfter != "" {
		t.Fatalf("system row fields: %+v", all[0])
	}
	if all[1].BeforeAfter != `{"disabled":[false,true]}` {
		t.Fatalf("before_after not round-tripped: %q", all[1].BeforeAfter)
	}

	// Filter by project.
	p1, err := s.ListAudit(ctx, AuditFilter{ProjectID: "p1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p1) != 2 {
		t.Fatalf("project filter: want 2, got %d", len(p1))
	}
	// Filter by actor + action.
	act, err := s.ListAudit(ctx, AuditFilter{ActorID: "admin1", Action: "user.disable"})
	if err != nil {
		t.Fatal(err)
	}
	if len(act) != 1 || act[0].ID != "01B" {
		t.Fatalf("actor+action filter: %+v", ids(act))
	}
	// Time range [base+1m, base+2m): only 01B.
	rng, err := s.ListAudit(ctx, AuditFilter{From: base.Add(time.Minute), To: base.Add(2 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if len(rng) != 1 || rng[0].ID != "01B" {
		t.Fatalf("time-range filter: %+v", ids(rng))
	}

	// Keyset pagination: page size 2 then continue after the cursor.
	page1, err := s.ListAudit(ctx, AuditFilter{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || page1[0].ID != "01C" {
		t.Fatalf("page1: %+v", ids(page1))
	}
	page2, err := s.ListAudit(ctx, AuditFilter{Limit: 2, AfterID: page1[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 1 || page2[0].ID != "01A" {
		t.Fatalf("page2: %+v", ids(page2))
	}
}

func ids(entries []AuditEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.ID
	}
	return out
}
