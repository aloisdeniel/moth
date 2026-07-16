package store

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// twoProjects seeds two independent projects and returns their IDs.
func twoProjects(t *testing.T, s *Store) (string, string) {
	t.Helper()
	ctx := context.Background()
	p1, k1 := testProject("p1", "app-one")
	p2, k2 := testProject("p2", "app-two")
	if err := s.CreateProject(ctx, p1, k1); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateProject(ctx, p2, k2); err != nil {
		t.Fatal(err)
	}
	return p1.ID, p2.ID
}

func testUser(projectID, id, email string) User {
	now := time.Now()
	return User{
		ID: id, ProjectID: projectID, Email: email,
		PasswordHash: "hash", CustomClaims: "{}",
		CreatedAt: now, UpdatedAt: now,
	}
}

func passwordIdentity(u User) Identity {
	return Identity{
		ID: "id-" + u.ID, ProjectID: u.ProjectID, UserID: u.ID,
		Provider: IdentityProviderPassword, ProviderSubject: u.ID,
		CreatedAt: u.CreatedAt,
	}
}

func TestUsersAreProjectScoped(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, p2 := twoProjects(t, s)

	// The same email in two projects is two unrelated accounts.
	u1 := testUser(p1, "u1", "same@example.com")
	u2 := testUser(p2, "u2", "same@example.com")
	if err := s.CreateUser(ctx, u1, passwordIdentity(u1)); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateUser(ctx, u2, passwordIdentity(u2)); err != nil {
		t.Fatalf("same email in second project must be independent: %v", err)
	}

	// Cross-project reads must be impossible.
	if _, err := s.GetUser(ctx, p2, u1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project GetUser: got %v, want ErrNotFound", err)
	}
	got, err := s.GetUserByEmail(ctx, p1, "same@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u1.ID {
		t.Fatalf("GetUserByEmail in p1 returned %s, want %s", got.ID, u1.ID)
	}

	// Cross-project writes must be impossible.
	stolen := u1
	stolen.ProjectID = p2
	stolen.DisplayName = "hijacked"
	stolen.UpdatedAt = time.Now()
	if err := s.UpdateUser(ctx, stolen); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project UpdateUser: got %v, want ErrNotFound", err)
	}
	if err := s.DeleteUser(ctx, p2, u1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project DeleteUser: got %v, want ErrNotFound", err)
	}

	users, err := s.ListUsers(ctx, p1)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].ID != u1.ID {
		t.Fatalf("ListUsers(p1) = %v, want just u1", users)
	}
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, _ := twoProjects(t, s)

	u := testUser(p1, "u1", "dup@example.com")
	if err := s.CreateUser(ctx, u, passwordIdentity(u)); err != nil {
		t.Fatal(err)
	}
	again := testUser(p1, "u9", "dup@example.com")
	if err := s.CreateUser(ctx, again); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate email: got %v, want ErrConflict", err)
	}
}

func TestDeleteUserCascades(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, _ := twoProjects(t, s)

	u := testUser(p1, "u1", "user@example.com")
	if err := s.CreateUser(ctx, u, passwordIdentity(u)); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := s.CreateRefreshToken(ctx, RefreshToken{
		ID: "rt1", ProjectID: p1, UserID: u.ID, TokenHash: "rth1",
		FamilyID: "fam1", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateEmailToken(ctx, EmailToken{
		ID: "et1", ProjectID: p1, UserID: u.ID, Purpose: EmailTokenPurposeVerify,
		TokenHash: "eth1", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteUser(ctx, p1, u.ID); err != nil {
		t.Fatal(err)
	}
	for table, query := range map[string]string{
		"identities":     `SELECT COUNT(*) FROM identities WHERE user_id = 'u1'`,
		"refresh_tokens": `SELECT COUNT(*) FROM refresh_tokens WHERE user_id = 'u1'`,
		"email_tokens":   `SELECT COUNT(*) FROM email_tokens WHERE user_id = 'u1'`,
	} {
		var n int
		if err := s.db.QueryRow(query).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s rows survived user deletion", table)
		}
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, p2 := twoProjects(t, s)

	u := testUser(p1, "u1", "user@example.com")
	if err := s.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	first := RefreshToken{
		ID: "rt1", ProjectID: p1, UserID: u.ID, TokenHash: "h1",
		FamilyID: "fam", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	if err := s.CreateRefreshToken(ctx, first); err != nil {
		t.Fatal(err)
	}

	// Tokens are invisible from another project.
	if _, err := s.GetRefreshToken(ctx, p2, "h1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project GetRefreshToken: got %v, want ErrNotFound", err)
	}

	successor := first
	successor.ID, successor.TokenHash = "rt2", "h2"
	if err := s.RotateRefreshToken(ctx, first.ID, now, successor); err != nil {
		t.Fatal(err)
	}
	old, err := s.GetRefreshToken(ctx, p1, "h1")
	if err != nil {
		t.Fatal(err)
	}
	if old.RotatedAt == nil || old.Usable(now) {
		t.Fatal("rotated token must not be usable")
	}

	// Rotating the same token twice must fail (reuse-detection hook).
	third := first
	third.ID, third.TokenHash = "rt3", "h3"
	if err := s.RotateRefreshToken(ctx, first.ID, now, third); !errors.Is(err, ErrNotFound) {
		t.Fatalf("double rotation: got %v, want ErrNotFound", err)
	}

	// Family revocation kills the successor too.
	if err := s.RevokeRefreshTokenFamily(ctx, p1, "fam", now); err != nil {
		t.Fatal(err)
	}
	cur, err := s.GetRefreshToken(ctx, p1, "h2")
	if err != nil {
		t.Fatal(err)
	}
	if cur.RevokedAt == nil || cur.Usable(now) {
		t.Fatal("family revocation must revoke the live token")
	}
}

func TestEmailTokenSingleUse(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, p2 := twoProjects(t, s)

	u := testUser(p1, "u1", "user@example.com")
	if err := s.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	et := EmailToken{
		ID: "et1", ProjectID: p1, UserID: u.ID, Purpose: EmailTokenPurposeReset,
		TokenHash: "h1", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	if err := s.CreateEmailToken(ctx, et); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetEmailToken(ctx, p2, "h1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project GetEmailToken: got %v, want ErrNotFound", err)
	}
	if err := s.ConsumeEmailToken(ctx, p1, et.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeEmailToken(ctx, p1, et.ID, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second consumption: got %v, want ErrNotFound", err)
	}
	got, err := s.GetEmailToken(ctx, p1, "h1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Usable(now) {
		t.Fatal("consumed token must not be usable")
	}
}

func TestProjectSettingsRoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	p, k := testProject("p1", "app")
	p.Settings = DefaultProjectSettings()
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Settings, DefaultProjectSettings()) {
		t.Fatalf("settings = %+v, want defaults", got.Settings)
	}

	got.Settings.RequireEmailVerification = true
	got.Settings.PasswordMinLength = 12
	got.Settings.AllowPublicSignup = false
	got.UpdatedAt = time.Now()
	if err := s.UpdateProject(ctx, got); err != nil {
		t.Fatal(err)
	}
	back, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !back.Settings.RequireEmailVerification || back.Settings.PasswordMinLength != 12 ||
		back.Settings.AllowPublicSignup {
		t.Fatalf("settings did not round-trip: %+v", back.Settings)
	}
}

func TestProjectLookupByKeys(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, _ := twoProjects(t, s)

	byPK, err := s.GetProjectByPublishableKey(ctx, "pk_p1")
	if err != nil {
		t.Fatal(err)
	}
	if byPK.ID != p1 {
		t.Fatalf("by publishable key: got %s, want %s", byPK.ID, p1)
	}
	bySK, err := s.GetProjectBySecretKeyHash(ctx, "hash-p1")
	if err != nil {
		t.Fatal(err)
	}
	if bySK.ID != p1 {
		t.Fatalf("by secret key hash: got %s, want %s", bySK.ID, p1)
	}
	if _, err := s.GetProjectByPublishableKey(ctx, "pk_nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown publishable key: got %v, want ErrNotFound", err)
	}
}
