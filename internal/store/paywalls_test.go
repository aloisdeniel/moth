package store

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// paywallRev builds a revision with ids and timestamps whose lexical order
// matches n, mirroring UUIDv7 behavior.
func paywallRev(projectID string, n int, base time.Time) PaywallRevision {
	return PaywallRevision{
		ID:        fmt.Sprintf("rev-%03d", n),
		ProjectID: projectID,
		Paywall:   fmt.Sprintf(`{"version":1,"n":%d}`, n),
		CreatedAt: base.Add(time.Duration(n) * time.Second),
	}
}

func TestSetProjectPaywall(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}

	// A fresh project has no paywall config.
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Paywall != "" || got.PaywallRevisionID != "" {
		t.Fatalf("new project must have empty paywall, got %q / %q", got.Paywall, got.PaywallRevisionID)
	}

	rev1 := paywallRev("p1", 1, base)
	if err := s.SetProjectPaywall(ctx, rev1, ""); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Paywall != rev1.Paywall || got.PaywallRevisionID != rev1.ID {
		t.Errorf("project paywall = %q / %q, want %q / %q", got.Paywall, got.PaywallRevisionID, rev1.Paywall, rev1.ID)
	}

	// A second save replaces the current config and stacks a revision.
	rev2 := paywallRev("p1", 2, base)
	if err := s.SetProjectPaywall(ctx, rev2, rev1.ID); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.PaywallRevisionID != rev2.ID {
		t.Errorf("current revision = %q, want %q", got.PaywallRevisionID, rev2.ID)
	}

	revs, err := s.ListPaywallRevisions(ctx, "p1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 2 || revs[0].ID != rev2.ID || revs[1].ID != rev1.ID {
		t.Errorf("revisions not newest-first: %+v", revs)
	}

	stored, err := s.GetPaywallRevision(ctx, "p1", rev1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Paywall != rev1.Paywall || !stored.CreatedAt.Equal(rev1.CreatedAt) {
		t.Errorf("revision round trip mismatch: %+v", stored)
	}
}

func TestSetProjectPaywallMissingProject(t *testing.T) {
	s := openTestStore(t)
	err := s.SetProjectPaywall(context.Background(), paywallRev("nope", 1, time.Now()), "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	// The failed save must not leave an orphan revision behind.
	revs, err := s.ListPaywallRevisions(context.Background(), "nope", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 0 {
		t.Fatalf("orphan revisions after failed save: %+v", revs)
	}
}

func TestSetProjectPaywallConflict(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	rev1 := paywallRev("p1", 1, base)
	if err := s.SetProjectPaywall(ctx, rev1, ""); err != nil {
		t.Fatal(err)
	}

	// A save based on a stale read (here: the pre-rev1 state) loses the CAS
	// instead of silently overwriting rev1's document.
	err := s.SetProjectPaywall(ctx, paywallRev("p1", 2, base), "")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale save: want ErrConflict, got %v", err)
	}
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.PaywallRevisionID != rev1.ID {
		t.Errorf("current revision after conflict = %q, want %q", got.PaywallRevisionID, rev1.ID)
	}
	// The lost save must not leave an orphan revision behind either.
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM paywall_revisions WHERE project_id = 'p1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("revisions after conflict = %d, want 1", n)
	}

	// A save based on the current revision goes through.
	rev3 := paywallRev("p1", 3, base)
	if err := s.SetProjectPaywall(ctx, rev3, rev1.ID); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.PaywallRevisionID != rev3.ID {
		t.Errorf("current revision = %q, want %q", got.PaywallRevisionID, rev3.ID)
	}
}

func TestPaywallRevisionPruning(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	total := PaywallRevisionKeep + 3
	prev := ""
	for n := 1; n <= total; n++ {
		rev := paywallRev("p1", n, base)
		if err := s.SetProjectPaywall(ctx, rev, prev); err != nil {
			t.Fatal(err)
		}
		prev = rev.ID
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM paywall_revisions WHERE project_id = 'p1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != PaywallRevisionKeep {
		t.Fatalf("kept %d revisions, want %d", count, PaywallRevisionKeep)
	}
	revs, err := s.ListPaywallRevisions(ctx, "p1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != PaywallRevisionKeep {
		t.Fatalf("listed %d revisions, want %d", len(revs), PaywallRevisionKeep)
	}
	if revs[0].ID != paywallRev("p1", total, base).ID {
		t.Errorf("newest revision = %q, want %q", revs[0].ID, paywallRev("p1", total, base).ID)
	}
	if _, err := s.GetPaywallRevision(ctx, "p1", paywallRev("p1", 3, base).ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("revision at the prune boundary still readable: %v", err)
	}
	if _, err := s.GetPaywallRevision(ctx, "p1", paywallRev("p1", 4, base).ID); err != nil {
		t.Errorf("oldest kept revision unreadable: %v", err)
	}
}

func TestListPaywallRevisionsLimit(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	prev := ""
	for n := 1; n <= 5; n++ {
		rev := paywallRev("p1", n, base)
		if err := s.SetProjectPaywall(ctx, rev, prev); err != nil {
			t.Fatal(err)
		}
		prev = rev.ID
	}
	tests := []struct {
		limit, want int
	}{
		{limit: 2, want: 2},
		{limit: 0, want: 5},
		{limit: -1, want: 5},
		{limit: 100, want: 5},
	}
	for _, tc := range tests {
		revs, err := s.ListPaywallRevisions(ctx, "p1", tc.limit)
		if err != nil {
			t.Fatal(err)
		}
		if len(revs) != tc.want {
			t.Errorf("limit %d: got %d revisions, want %d", tc.limit, len(revs), tc.want)
		}
	}
}

func TestClearProjectPaywall(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	if err := s.SetProjectPaywall(ctx, paywallRev("p1", 1, base), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearProjectPaywall(ctx, "p1", base.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Paywall != "" || got.PaywallRevisionID != "" {
		t.Errorf("paywall not cleared: %q / %q", got.Paywall, got.PaywallRevisionID)
	}
	// History survives a reset so the old config stays restorable.
	revs, err := s.ListPaywallRevisions(ctx, "p1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 1 {
		t.Errorf("revision history lost on clear: %+v", revs)
	}

	if err := s.ClearProjectPaywall(ctx, "missing", base); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound for missing project, got %v", err)
	}
}

func TestGetPaywallRevisionScoping(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()

	p1, k1 := testProject("p1", "app-one")
	p2, k2 := testProject("p2", "app-two")
	if err := s.CreateProject(ctx, p1, k1); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateProject(ctx, p2, k2); err != nil {
		t.Fatal(err)
	}
	rev := paywallRev("p1", 1, base)
	if err := s.SetProjectPaywall(ctx, rev, ""); err != nil {
		t.Fatal(err)
	}
	// Revisions are project-scoped: another project cannot read them.
	if _, err := s.GetPaywallRevision(ctx, "p2", rev.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-project revision read: %v", err)
	}
	if _, err := s.GetPaywallRevision(ctx, "p1", "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound for unknown revision, got %v", err)
	}
}

func TestDeleteProjectCascadesPaywallRevisions(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	if err := s.SetProjectPaywall(ctx, paywallRev("p1", 1, time.Now().UTC()), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProject(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM paywall_revisions WHERE project_id = 'p1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("paywall revisions not cascaded on project delete: %d left", n)
	}
}
