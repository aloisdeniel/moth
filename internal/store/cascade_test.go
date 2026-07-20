package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// NewTestID returns a fresh UUIDv7 string for seeding rows.
func NewTestID(t *testing.T) string {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	return id.String()
}

// countRows returns the number of rows of table scoped to a column=value.
func countRows(t *testing.T, s *Store, table, col, val string) int {
	t.Helper()
	var n int
	//nolint:gosec // table/col are test-controlled literals, not user input.
	if err := s.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM `+table+` WHERE `+col+` = ?`, val).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// seedCascadeFixture creates a project with one active key and one user that
// owns a row in every table that references users or projects, so a delete
// can be checked for orphans.
func seedCascadeFixture(t *testing.T, s *Store) (projectID, userID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	projectID = NewTestID(t)
	userID = NewTestID(t)

	project := Project{
		ID: projectID, Name: "Demo", Slug: "demo-" + projectID[:8],
		PublishableKey: "pk_" + projectID, SecretKeyHash: "h_" + projectID,
		Settings: DefaultProjectSettings(), CreatedAt: now, UpdatedAt: now,
	}
	key := ProjectKey{
		ID: NewTestID(t), ProjectID: projectID, Kid: "kid-" + projectID,
		Algorithm: "ES256", PublicKeyPEM: "pem", PrivateKeyEnc: []byte("enc"),
		Status: ProjectKeyStatusActive, CreatedAt: now,
	}
	if err := s.CreateProject(ctx, project, key); err != nil {
		t.Fatal(err)
	}
	user := User{
		ID: userID, ProjectID: projectID, Email: "u@example.com",
		PasswordHash: "h", CustomClaims: "{}", CreatedAt: now, UpdatedAt: now,
	}
	identity := Identity{
		ID: NewTestID(t), ProjectID: projectID, UserID: userID,
		Provider: IdentityProviderPassword, ProviderSubject: userID, CreatedAt: now,
	}
	if err := s.CreateUser(ctx, user, identity); err != nil {
		t.Fatal(err)
	}

	ts := formatTime(now)
	exec := func(q string, args ...any) {
		if _, err := s.db.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	exec(`INSERT INTO refresh_tokens (id, project_id, user_id, token_hash, family_id, expires_at, created_at)
	      VALUES (?, ?, ?, ?, ?, ?, ?)`,
		NewTestID(t), projectID, userID, "rt_"+userID, NewTestID(t), ts, ts)
	exec(`INSERT INTO email_tokens (id, project_id, user_id, purpose, token_hash, expires_at, created_at)
	      VALUES (?, ?, ?, ?, ?, ?, ?)`,
		NewTestID(t), projectID, userID, "verify", "et_"+userID, ts, ts)
	exec(`INSERT INTO oauth_tokens (id, project_id, purpose, token_hash, provider, user_id, expires_at, created_at)
	      VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		NewTestID(t), projectID, "code", "ot_"+userID, "google", userID, ts, ts)
	exec(`INSERT INTO events (id, project_id, user_id, type, created_at) VALUES (?, ?, ?, ?, ?)`,
		NewTestID(t), projectID, userID, "login", ts)
	exec(`INSERT INTO project_provider_secrets (project_id, name, secret_enc, updated_at) VALUES (?, ?, ?, ?)`,
		projectID, "google_web_client_secret", []byte("enc"), ts)
	exec(`INSERT INTO daily_stats (project_id, date) VALUES (?, ?)`, projectID, "2026-07-16")
	return projectID, userID
}

func TestDeleteUserCascadesNoOrphans(t *testing.T) {
	s := openTestStore(t)
	projectID, userID := seedCascadeFixture(t, s)

	if err := s.DeleteUser(context.Background(), projectID, userID); err != nil {
		t.Fatal(err)
	}
	for _, tbl := range []string{"identities", "refresh_tokens", "email_tokens", "oauth_tokens"} {
		if n := countRows(t, s, tbl, "user_id", userID); n != 0 {
			t.Errorf("%s left %d orphan rows after user delete", tbl, n)
		}
	}
	// The project and its independent rows survive a user delete.
	if n := countRows(t, s, "projects", "id", projectID); n != 1 {
		t.Errorf("user delete removed the project")
	}
	if n := countRows(t, s, "project_provider_secrets", "project_id", projectID); n != 1 {
		t.Errorf("user delete removed project provider secrets")
	}
}

func TestDeleteProjectCascadesNoOrphans(t *testing.T) {
	s := openTestStore(t)
	projectID, userID := seedCascadeFixture(t, s)

	if err := s.DeleteProject(context.Background(), projectID); err != nil {
		t.Fatal(err)
	}
	checks := []struct{ table, col, val string }{
		{"project_keys", "project_id", projectID},
		{"users", "project_id", projectID},
		{"identities", "project_id", projectID},
		{"refresh_tokens", "project_id", projectID},
		{"email_tokens", "project_id", projectID},
		{"oauth_tokens", "project_id", projectID},
		{"events", "project_id", projectID},
		{"project_provider_secrets", "project_id", projectID},
		{"daily_stats", "project_id", projectID},
		{"identities", "user_id", userID},
	}
	for _, c := range checks {
		if n := countRows(t, s, c.table, c.col, c.val); n != 0 {
			t.Errorf("%s left %d orphan rows after project delete", c.table, n)
		}
	}
}
