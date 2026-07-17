package store

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

// NewTestID returns a fresh UUIDv7 for a revision id in the copy tests.
func newCopyID() string { return uuid.Must(uuid.NewV7()).String() }

// copyRev builds a revision whose lexical id order matches n, mirroring
// UUIDv7 behavior.
func copyRev(projectID string, n int, base time.Time) CopyRevision {
	return CopyRevision{
		ID:        fmt.Sprintf("rev-%03d", n),
		ProjectID: projectID,
		Copy:      fmt.Sprintf(`{"en":{"sign_in.title":"v%d"}}`, n),
		CreatedAt: base.Add(time.Duration(n) * time.Second),
	}
}

// stubValidator rejects the overrides when reject is non-nil.
type stubValidator struct{ reject error }

func (v stubValidator) ValidateCopyOverrides(CopyOverrides) error { return v.reject }

func newCopyProject(t *testing.T, s *Store, id string) {
	t.Helper()
	p, k := testProject(id, id+"-app")
	if err := s.CreateProject(context.Background(), p, k); err != nil {
		t.Fatal(err)
	}
}

func TestSetProjectCopy(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()
	newCopyProject(t, s, "p1")

	// A fresh project renders the bundled default (no overrides).
	o, rev, err := s.GetProjectCopy(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(o) != 0 || rev != "" {
		t.Fatalf("new project must have empty copy, got %v / %q", o, rev)
	}

	rev1 := copyRev("p1", 1, base)
	if err := s.SetProjectCopy(ctx, rev1, ""); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Copy != rev1.Copy || got.CopyRevisionID != rev1.ID {
		t.Errorf("project copy = %q / %q, want %q / %q", got.Copy, got.CopyRevisionID, rev1.Copy, rev1.ID)
	}

	rev2 := copyRev("p1", 2, base)
	if err := s.SetProjectCopy(ctx, rev2, rev1.ID); err != nil {
		t.Fatal(err)
	}
	revs, err := s.ListCopyRevisions(ctx, "p1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 2 || revs[0].ID != rev2.ID || revs[1].ID != rev1.ID {
		t.Errorf("revisions not newest-first: %+v", revs)
	}
	stored, err := s.GetCopyRevision(ctx, "p1", rev1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Copy != rev1.Copy || !stored.CreatedAt.Equal(rev1.CreatedAt) {
		t.Errorf("revision round trip mismatch: %+v", stored)
	}
}

func TestSetProjectCopyMissingProject(t *testing.T) {
	s := openTestStore(t)
	err := s.SetProjectCopy(context.Background(), copyRev("nope", 1, time.Now()), "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	revs, err := s.ListCopyRevisions(context.Background(), "nope", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 0 {
		t.Fatalf("orphan revisions after failed save: %+v", revs)
	}
}

func TestSetProjectCopyConflict(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()
	newCopyProject(t, s, "p1")

	rev1 := copyRev("p1", 1, base)
	if err := s.SetProjectCopy(ctx, rev1, ""); err != nil {
		t.Fatal(err)
	}
	// A stale save loses the CAS instead of clobbering rev1.
	if err := s.SetProjectCopy(ctx, copyRev("p1", 2, base), ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale save: want ErrConflict, got %v", err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM copy_revisions WHERE project_id = 'p1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("revisions after conflict = %d, want 1", n)
	}
}

func TestCopyRevisionPruning(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()
	newCopyProject(t, s, "p1")

	total := CopyRevisionKeep + 3
	prev := ""
	for i := 1; i <= total; i++ {
		rev := copyRev("p1", i, base)
		if err := s.SetProjectCopy(ctx, rev, prev); err != nil {
			t.Fatal(err)
		}
		prev = rev.ID
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM copy_revisions WHERE project_id = 'p1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != CopyRevisionKeep {
		t.Fatalf("kept %d revisions, want %d", count, CopyRevisionKeep)
	}
	if _, err := s.GetCopyRevision(ctx, "p1", copyRev("p1", 3, base).ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("revision at prune boundary still readable: %v", err)
	}
	if _, err := s.GetCopyRevision(ctx, "p1", copyRev("p1", 4, base).ID); err != nil {
		t.Errorf("oldest kept revision unreadable: %v", err)
	}
}

func TestClearProjectCopy(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()
	newCopyProject(t, s, "p1")

	if err := s.SetProjectCopy(ctx, copyRev("p1", 1, base), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearProjectCopy(ctx, "p1", base.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Copy != "" || got.CopyRevisionID != "" {
		t.Errorf("copy not cleared: %q / %q", got.Copy, got.CopyRevisionID)
	}
	// History survives a reset so the old copy stays restorable.
	revs, err := s.ListCopyRevisions(ctx, "p1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 1 {
		t.Errorf("revision history lost on clear: %+v", revs)
	}
	if err := s.ClearProjectCopy(ctx, "missing", base); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound for missing project, got %v", err)
	}
}

func TestGetCopyRevisionScoping(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC()
	newCopyProject(t, s, "p1")
	newCopyProject(t, s, "p2")

	rev := copyRev("p1", 1, base)
	if err := s.SetProjectCopy(ctx, rev, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetCopyRevision(ctx, "p2", rev.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-project revision read: %v", err)
	}
	if _, err := s.GetCopyRevision(ctx, "p1", "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound for unknown revision, got %v", err)
	}
}

func TestDeleteProjectCascadesCopyRevisions(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	newCopyProject(t, s, "p1")

	if err := s.SetProjectCopy(ctx, copyRev("p1", 1, time.Now().UTC()), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProject(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM copy_revisions WHERE project_id = 'p1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("copy revisions not cascaded on project delete: %d left", n)
	}
}

func TestUpdateProjectCopy(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	newCopyProject(t, s, "p1")

	tests := []struct {
		name      string
		locale    string
		values    map[string]string
		validate  CopyValidator
		wantErr   error
		wantEmpty bool // resulting revision id is "" (cleared to default)
	}{
		{name: "adds fr overrides", locale: "fr", values: map[string]string{"sign_in.title": "Connexion"}},
		{name: "empty locale rejected", locale: "", values: map[string]string{"sign_in.title": "x"}, wantErr: ErrInvalidCopy},
		{name: "validator rejects", locale: "de", values: map[string]string{"bad.key": "x"}, validate: stubValidator{reject: errors.New("unknown key bad.key")}, wantErr: ErrInvalidCopy},
		{name: "all-empty values clears the locale", locale: "fr", values: map[string]string{"sign_in.title": ""}, wantEmpty: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rev, err := s.UpdateProjectCopy(ctx, "p1", tc.locale, tc.values, newCopyID(), now, tc.validate)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantEmpty && rev != "" {
				t.Errorf("want cleared (empty revision), got %q", rev)
			}
		})
	}

	// After the sequence: fr was added then cleared, so the project is back to
	// the bundled default with no overrides.
	o, _, err := s.GetProjectCopy(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(o) != 0 {
		t.Fatalf("expected no overrides after clear, got %v", o)
	}
}

func TestUpdateProjectCopyMergeAndReset(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	newCopyProject(t, s, "p1")

	if _, err := s.UpdateProjectCopy(ctx, "p1", "fr", map[string]string{"sign_in.title": "Connexion", "sign_up.title": "Inscription"}, newCopyID(), now, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateProjectCopy(ctx, "p1", "de", map[string]string{"sign_in.title": "Anmelden"}, newCopyID(), now, nil); err != nil {
		t.Fatal(err)
	}
	o, _, err := s.GetProjectCopy(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	want := CopyOverrides{
		"fr": {"sign_in.title": "Connexion", "sign_up.title": "Inscription"},
		"de": {"sign_in.title": "Anmelden"},
	}
	if !reflect.DeepEqual(o, want) {
		t.Fatalf("overrides = %v, want %v", o, want)
	}

	// Reset one key: fr keeps sign_up.title, loses sign_in.title.
	if _, err := s.ResetCopy(ctx, "p1", "fr", "sign_in.title", newCopyID(), now); err != nil {
		t.Fatal(err)
	}
	o, _, _ = s.GetProjectCopy(ctx, "p1")
	if _, ok := o["fr"]["sign_in.title"]; ok {
		t.Errorf("fr.sign_in.title not reset: %v", o["fr"])
	}
	if o["fr"]["sign_up.title"] != "Inscription" {
		t.Errorf("fr.sign_up.title lost: %v", o["fr"])
	}

	// Reset the whole de locale.
	if _, err := s.ResetCopy(ctx, "p1", "de", "", newCopyID(), now); err != nil {
		t.Fatal(err)
	}
	o, _, _ = s.GetProjectCopy(ctx, "p1")
	if _, ok := o["de"]; ok {
		t.Errorf("de locale not reset: %v", o)
	}
}

func TestResetCopyToDefaultClears(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	newCopyProject(t, s, "p1")

	if _, err := s.UpdateProjectCopy(ctx, "p1", "fr", map[string]string{"sign_in.title": "Connexion"}, newCopyID(), now, nil); err != nil {
		t.Fatal(err)
	}
	rev, err := s.ResetCopy(ctx, "p1", "fr", "", newCopyID(), now)
	if err != nil {
		t.Fatal(err)
	}
	if rev != "" {
		t.Errorf("resetting the last override should clear to default (empty revision), got %q", rev)
	}
	got, _ := s.GetProject(ctx, "p1")
	if got.Copy != "" || got.CopyRevisionID != "" {
		t.Errorf("project not cleared: %q / %q", got.Copy, got.CopyRevisionID)
	}
}

func TestRestoreCopyRevision(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	newCopyProject(t, s, "p1")

	first, err := s.UpdateProjectCopy(ctx, "p1", "fr", map[string]string{"sign_in.title": "Connexion"}, newCopyID(), now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateProjectCopy(ctx, "p1", "fr", map[string]string{"sign_in.title": "Se connecter"}, newCopyID(), now, nil); err != nil {
		t.Fatal(err)
	}
	// Restore the first revision as a new revision.
	newRev, err := s.RestoreCopyRevision(ctx, "p1", first, newCopyID(), now)
	if err != nil {
		t.Fatal(err)
	}
	if newRev == first {
		t.Errorf("restore must create a new revision, reused %q", first)
	}
	o, _, _ := s.GetProjectCopy(ctx, "p1")
	if o["fr"]["sign_in.title"] != "Connexion" {
		t.Errorf("restore did not reinstate the old copy: %v", o)
	}
	if _, err := s.RestoreCopyRevision(ctx, "p1", "missing", newCopyID(), now); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound for missing revision, got %v", err)
	}
}

func TestListAvailableLocales(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	newCopyProject(t, s, "p1")

	// With no overrides, only the injected default locale is available.
	locales, err := s.ListAvailableLocales(ctx, "p1", "en")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(locales, []string{"en"}) {
		t.Fatalf("locales = %v, want [en]", locales)
	}

	if _, err := s.UpdateProjectCopy(ctx, "p1", "fr", map[string]string{"sign_in.title": "Connexion"}, newCopyID(), now, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateProjectCopy(ctx, "p1", "de", map[string]string{"sign_in.title": "Anmelden"}, newCopyID(), now, nil); err != nil {
		t.Fatal(err)
	}
	locales, err = s.ListAvailableLocales(ctx, "p1", "en")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(locales, []string{"de", "en", "fr"}) {
		t.Fatalf("locales = %v, want [de en fr] (sorted, default included)", locales)
	}
}
