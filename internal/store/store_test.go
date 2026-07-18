package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "moth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestMigrateFreshAndIdempotent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	// Re-running must be a no-op.
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("no migrations recorded")
	}
	for _, table := range []string{"admins", "admin_sessions", "projects", "project_keys"} {
		var count int
		err := s.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("table %s missing", table)
		}
	}
}

func testProject(id, slug string) (Project, ProjectKey) {
	now := time.Now()
	p := Project{
		ID: id, Name: "My App", Slug: slug,
		PublishableKey: "pk_" + id, SecretKeyHash: "hash-" + id,
		CreatedAt: now, UpdatedAt: now,
	}
	k := ProjectKey{
		ID: "key-" + id, ProjectID: id, Kid: "kid-" + id, Algorithm: "ES256",
		PublicKeyPEM: "PEM", PrivateKeyEnc: []byte{1, 2, 3},
		Status: ProjectKeyStatusActive, CreatedAt: now,
	}
	return p, k
}

func TestProjectCRUD(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "my-app" || got.PublishableKey != p.PublishableKey {
		t.Fatalf("get mismatch: %+v", got)
	}
	if got.CreatedAt.IsZero() || !got.CreatedAt.Equal(p.CreatedAt.UTC().Truncate(0)) && got.CreatedAt.Unix() != p.CreatedAt.Unix() {
		t.Fatalf("created_at not round-tripped: %v vs %v", got.CreatedAt, p.CreatedAt)
	}

	if _, err := s.GetProjectBySlug(ctx, "my-app"); err != nil {
		t.Fatal(err)
	}
	exists, err := s.SlugExists(ctx, "my-app")
	if err != nil || !exists {
		t.Fatalf("slug should exist: %v", err)
	}

	p2, k2 := testProject("p2", "other-app")
	if err := s.CreateProject(ctx, p2, k2); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 projects, got %d", len(list))
	}

	got.Name = "Renamed"
	got.UpdatedAt = time.Now()
	if err := s.UpdateProject(ctx, got); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetProject(ctx, "p1")
	if got.Name != "Renamed" {
		t.Fatalf("update not applied: %+v", got)
	}

	keys, err := s.ListActiveProjectKeys(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Kid != "kid-p1" || string(keys[0].PrivateKeyEnc) != "\x01\x02\x03" {
		t.Fatalf("keys mismatch: %+v", keys)
	}

	if err := s.DeleteProject(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetProject(ctx, "p1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	// Cascade removed the key.
	keys, err = s.ListActiveProjectKeys(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 0 {
		t.Fatalf("project keys should cascade-delete, got %+v", keys)
	}

	if err := s.DeleteProject(ctx, "p1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("double delete: want ErrNotFound, got %v", err)
	}
	if err := s.UpdateProject(ctx, Project{ID: "missing", UpdatedAt: time.Now()}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update missing: want ErrNotFound, got %v", err)
	}
}

func TestSetProjectProfile(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	p1, k1 := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p1, k1); err != nil {
		t.Fatal(err)
	}
	p2, k2 := testProject("p2", "other-app")
	if err := s.CreateProject(ctx, p2, k2); err != nil {
		t.Fatal(err)
	}

	// A fresh project has no profile document.
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Profile) != 0 {
		t.Fatalf("fresh project carries a profile: %v", got.Profile)
	}

	doc := []byte{0x08, 0x01} // any opaque bytes; the store never parses them
	if err := s.SetProjectProfile(ctx, "p1", doc, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got, err = s.GetProject(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if string(got.Profile) != string(doc) {
		t.Fatalf("profile not round-tripped: %v", got.Profile)
	}

	// Cross-project isolation: p2 is untouched.
	other, err := s.GetProject(ctx, "p2")
	if err != nil {
		t.Fatal(err)
	}
	if len(other.Profile) != 0 {
		t.Fatalf("profile leaked to another project: %v", other.Profile)
	}

	if err := s.SetProjectProfile(ctx, "missing", doc, time.Now()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing project: want ErrNotFound, got %v", err)
	}
}

func TestCreateProjectIsAtomic(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	p, k := testProject("p1", "my-app")
	k.Kid = "dup"
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	// Same kid → key insert fails → project insert must roll back.
	p2, k2 := testProject("p2", "other")
	k2.Kid = "dup"
	if err := s.CreateProject(ctx, p2, k2); err == nil {
		t.Fatal("expected unique violation")
	}
	if _, err := s.GetProject(ctx, "p2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("project p2 should have rolled back, got %v", err)
	}
}

func TestAdminsAndSessions(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now()

	n, err := s.CountAdmins(ctx)
	if err != nil || n != 0 {
		t.Fatalf("fresh instance should have 0 admins, got %d (%v)", n, err)
	}

	a := Admin{ID: "a1", Email: "ops@example.com", PasswordHash: "h1", CreatedAt: now, UpdatedAt: now}
	if err := s.CreateAdmin(ctx, a); err != nil {
		t.Fatal(err)
	}
	// Upsert with same email must reset the hash, not create a second row.
	a2 := Admin{ID: "a2", Email: "OPS@example.com", PasswordHash: "h2", CreatedAt: now, UpdatedAt: now}
	if err := s.UpsertAdmin(ctx, a2); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CountAdmins(ctx); n != 1 {
		t.Fatalf("upsert should not duplicate (email is NOCASE): %d admins", n)
	}
	got, err := s.GetAdminByEmail(ctx, "ops@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.PasswordHash != "h2" {
		t.Fatalf("password hash not reset: %+v", got)
	}

	sess := AdminSession{TokenHash: "th", AdminID: got.ID, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSession(ctx, "th"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteExpiredSessions(ctx, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSession(ctx, "th"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session should be gone, got %v", err)
	}
}

func TestStatePersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "moth.db")
	ctx := context.Background()

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	p, k := testProject("p1", "my-app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if err := s2.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s2.GetProject(ctx, "p1"); err != nil {
		t.Fatalf("state lost across reopen: %v", err)
	}
}
