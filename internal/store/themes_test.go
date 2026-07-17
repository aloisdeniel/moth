package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// themeRev builds a revision with ids and timestamps whose lexical order
// matches n, mirroring UUIDv7 behavior.
func themeRev(projectID string, n int, base time.Time) ThemeRevision {
	return ThemeRevision{
		ID:        fmt.Sprintf("rev-%03d", n),
		ProjectID: projectID,
		Theme:     []byte(fmt.Sprintf("proto-doc-%d", n)),
		CreatedAt: base.Add(time.Duration(n) * time.Second),
	}
}

func TestSetProjectTheme(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}

	// A fresh project has no theme.
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Theme) != 0 || got.ThemeRevisionID != "" {
		t.Fatalf("new project must have empty theme, got %q / %q", got.Theme, got.ThemeRevisionID)
	}

	rev1 := themeRev("p1", 1, base)
	if err := s.SetProjectTheme(ctx, rev1, ""); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Theme, rev1.Theme) || got.ThemeRevisionID != rev1.ID {
		t.Errorf("project theme = %q / %q, want %q / %q", got.Theme, got.ThemeRevisionID, rev1.Theme, rev1.ID)
	}

	// A second save replaces the current theme and stacks a revision.
	rev2 := themeRev("p1", 2, base)
	if err := s.SetProjectTheme(ctx, rev2, rev1.ID); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ThemeRevisionID != rev2.ID {
		t.Errorf("current revision = %q, want %q", got.ThemeRevisionID, rev2.ID)
	}

	revs, err := s.ListThemeRevisions(ctx, "p1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 2 || revs[0].ID != rev2.ID || revs[1].ID != rev1.ID {
		t.Errorf("revisions not newest-first: %+v", revs)
	}

	stored, err := s.GetThemeRevision(ctx, "p1", rev1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored.Theme, rev1.Theme) || !stored.CreatedAt.Equal(rev1.CreatedAt) {
		t.Errorf("revision round trip mismatch: %+v", stored)
	}
}

func TestSetProjectThemeMissingProject(t *testing.T) {
	s := openTestStore(t)
	err := s.SetProjectTheme(context.Background(), themeRev("nope", 1, time.Now()), "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	// The failed save must not leave an orphan revision behind.
	revs, err := s.ListThemeRevisions(context.Background(), "nope", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 0 {
		t.Fatalf("orphan revisions after failed save: %+v", revs)
	}
}

func TestSetProjectThemeConflict(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	rev1 := themeRev("p1", 1, base)
	if err := s.SetProjectTheme(ctx, rev1, ""); err != nil {
		t.Fatal(err)
	}

	// A save based on a stale read (here: the pre-rev1 state) loses the CAS
	// instead of silently overwriting rev1's document.
	err := s.SetProjectTheme(ctx, themeRev("p1", 2, base), "")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale save: want ErrConflict, got %v", err)
	}
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ThemeRevisionID != rev1.ID {
		t.Errorf("current revision after conflict = %q, want %q", got.ThemeRevisionID, rev1.ID)
	}
	// The lost save must not leave an orphan revision behind either.
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM theme_revisions WHERE project_id = 'p1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("revisions after conflict = %d, want 1", n)
	}

	// A save based on the current revision goes through.
	rev3 := themeRev("p1", 3, base)
	if err := s.SetProjectTheme(ctx, rev3, rev1.ID); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ThemeRevisionID != rev3.ID {
		t.Errorf("current revision = %q, want %q", got.ThemeRevisionID, rev3.ID)
	}
}

func TestThemeRevisionPruning(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	total := ThemeRevisionKeep + 3
	prev := ""
	for n := 1; n <= total; n++ {
		rev := themeRev("p1", n, base)
		if err := s.SetProjectTheme(ctx, rev, prev); err != nil {
			t.Fatal(err)
		}
		prev = rev.ID
	}
	// Count the rows directly: ListThemeRevisions clamps its limit to
	// ThemeRevisionKeep, so it can never observe over-retention.
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM theme_revisions WHERE project_id = 'p1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != ThemeRevisionKeep {
		t.Fatalf("kept %d revisions, want %d", count, ThemeRevisionKeep)
	}
	revs, err := s.ListThemeRevisions(ctx, "p1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != ThemeRevisionKeep {
		t.Fatalf("listed %d revisions, want %d", len(revs), ThemeRevisionKeep)
	}
	// Newest survive, oldest are gone — pinned at the exact prune boundary:
	// with Keep+3 saves, revision 3 is the newest pruned and 4 the oldest
	// kept.
	if revs[0].ID != themeRev("p1", total, base).ID {
		t.Errorf("newest revision = %q, want %q", revs[0].ID, themeRev("p1", total, base).ID)
	}
	if _, err := s.GetThemeRevision(ctx, "p1", themeRev("p1", 3, base).ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("revision at the prune boundary still readable: %v", err)
	}
	if _, err := s.GetThemeRevision(ctx, "p1", themeRev("p1", 4, base).ID); err != nil {
		t.Errorf("oldest kept revision unreadable: %v", err)
	}
}

func TestListThemeRevisionsLimit(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	prev := ""
	for n := 1; n <= 5; n++ {
		rev := themeRev("p1", n, base)
		if err := s.SetProjectTheme(ctx, rev, prev); err != nil {
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
		revs, err := s.ListThemeRevisions(ctx, "p1", tc.limit)
		if err != nil {
			t.Fatal(err)
		}
		if len(revs) != tc.want {
			t.Errorf("limit %d: got %d revisions, want %d", tc.limit, len(revs), tc.want)
		}
	}
}

func TestClearProjectTheme(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	if err := s.SetProjectTheme(ctx, themeRev("p1", 1, base), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearProjectTheme(ctx, "p1", base.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Theme) != 0 || got.ThemeRevisionID != "" {
		t.Errorf("theme not cleared: %q / %q", got.Theme, got.ThemeRevisionID)
	}
	// History survives a reset so the old theme stays restorable.
	revs, err := s.ListThemeRevisions(ctx, "p1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 1 {
		t.Errorf("revision history lost on clear: %+v", revs)
	}

	if err := s.ClearProjectTheme(ctx, "missing", base); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound for missing project, got %v", err)
	}
}

func TestGetThemeRevisionScoping(t *testing.T) {
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
	rev := themeRev("p1", 1, base)
	if err := s.SetProjectTheme(ctx, rev, ""); err != nil {
		t.Fatal(err)
	}
	// Revisions are project-scoped: another project cannot read them.
	if _, err := s.GetThemeRevision(ctx, "p2", rev.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-project revision read: %v", err)
	}
	if _, err := s.GetThemeRevision(ctx, "p1", "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound for unknown revision, got %v", err)
	}
}

func TestDeleteProjectCascadesThemeRevisions(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	if err := s.SetProjectTheme(ctx, themeRev("p1", 1, time.Now().UTC()), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProject(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM theme_revisions WHERE project_id = 'p1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("theme revisions not cascaded on project delete: %d left", n)
	}
}
