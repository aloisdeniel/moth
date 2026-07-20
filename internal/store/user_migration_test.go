package store

import (
	"context"
	"testing"
	"time"
)

func TestExportImportUsersRoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)

	src, _ := testProject("src", "src-app")
	dst, _ := testProject("dst", "dst-app")
	for _, p := range []Project{src, dst} {
		_, k := testProject(p.ID, p.Slug)
		if err := s.CreateProject(ctx, p, k); err != nil {
			t.Fatal(err)
		}
	}

	verified := now.Add(-time.Hour)
	u1 := User{ID: "u1", ProjectID: "src", Email: "a@example.com", EmailVerifiedAt: &verified,
		PasswordHash: "$argon2id$native", DisplayName: "Alice", CustomClaims: `{"role":"admin"}`,
		CreatedAt: now, UpdatedAt: now}
	u2 := User{ID: "u2", ProjectID: "src", Email: "b@example.com",
		PasswordHash: "$2a$10$foreignbcrypt", PasswordAlgo: PasswordAlgoBcrypt,
		DisplayName: "Bob", CustomClaims: "{}", CreatedAt: now, UpdatedAt: now}
	if err := s.CreateUser(ctx, u1, Identity{ID: "i1", ProjectID: "src", UserID: "u1",
		Provider: IdentityProviderGoogle, ProviderSubject: "g-123", ProviderEmail: "a@example.com", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateUser(ctx, u2); err != nil {
		t.Fatal(err)
	}

	exported, err := s.ExportUsers(ctx, "src")
	if err != nil {
		t.Fatal(err)
	}
	if len(exported) != 2 {
		t.Fatalf("want 2 exported, got %d", len(exported))
	}
	// Foreign-hash marker survives the export read.
	var bob UserExport
	for _, e := range exported {
		if e.User.Email == "b@example.com" {
			bob = e
		}
	}
	if bob.User.PasswordAlgo != PasswordAlgoBcrypt || bob.User.PasswordHash != "$2a$10$foreignbcrypt" {
		t.Fatalf("foreign hash not exported: %+v", bob.User)
	}

	// Import into the destination project verbatim (new IDs, same project).
	imports := make([]UserImport, len(exported))
	for i, e := range exported {
		u := e.User
		u.ID = "d" + e.User.ID
		u.ProjectID = "dst"
		ids := make([]Identity, len(e.Identities))
		for j, id := range e.Identities {
			id.ID = "d" + id.ID
			ids[j] = id
		}
		imports[i] = UserImport{User: u, Identities: ids}
	}
	res, err := s.ImportUsers(ctx, "dst", imports, now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Imported != 2 || res.Skipped != 0 {
		t.Fatalf("import result = %+v, want 2/0", res)
	}

	got, err := s.GetUserByEmail(ctx, "dst", "b@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.PasswordAlgo != PasswordAlgoBcrypt || got.PasswordHash != "$2a$10$foreignbcrypt" {
		t.Fatalf("foreign hash not imported: %+v", got)
	}
	alice, err := s.GetUserByEmail(ctx, "dst", "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if alice.PasswordAlgo != PasswordAlgoNative || !alice.Verified() || alice.CustomClaims != `{"role":"admin"}` {
		t.Fatalf("native user not imported faithfully: %+v", alice)
	}
	idents, err := s.ListUserIdentities(ctx, "dst", alice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(idents) != 1 || idents[0].Provider != IdentityProviderGoogle || idents[0].ProviderSubject != "g-123" {
		t.Fatalf("identity not imported: %+v", idents)
	}
}

func TestImportUsersSkipsDuplicateEmail(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)

	p, k := testProject("p1", "app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateUser(ctx, User{ID: "existing", ProjectID: "p1", Email: "dup@example.com",
		CustomClaims: "{}", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}

	res, err := s.ImportUsers(ctx, "p1", []UserImport{
		{User: User{ID: "new1", Email: "dup@example.com", PasswordHash: "x", PasswordAlgo: PasswordAlgoScrypt}},
		{User: User{ID: "new2", Email: "fresh@example.com", PasswordHash: "y", PasswordAlgo: PasswordAlgoPBKDF2}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Imported != 1 || res.Skipped != 1 {
		t.Fatalf("want 1 imported / 1 skipped, got %+v", res)
	}
	// The pre-existing account was left untouched (no password overwrite).
	existing, err := s.GetUserByEmail(ctx, "p1", "dup@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if existing.ID != "existing" || existing.PasswordHash != "" {
		t.Fatalf("duplicate import clobbered the existing user: %+v", existing)
	}
}
