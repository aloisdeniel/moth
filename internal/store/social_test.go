package store

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func googleIdentity(u User, subject string) Identity {
	return Identity{
		ID: "gid-" + u.ID, ProjectID: u.ProjectID, UserID: u.ID,
		Provider: IdentityProviderGoogle, ProviderSubject: subject,
		ProviderEmail: u.Email, CreatedAt: u.CreatedAt.Add(time.Second),
	}
}

func TestIdentityLookupIsProjectScoped(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, p2 := twoProjects(t, s)

	u := testUser(p1, "u1", "user@example.com")
	if err := s.CreateUser(ctx, u, passwordIdentity(u), googleIdentity(u, "goog-sub")); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetIdentity(ctx, p1, IdentityProviderGoogle, "goog-sub")
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != u.ID || got.ProviderEmail != u.Email {
		t.Fatalf("GetIdentity = %+v, want user %s email %s", got, u.ID, u.Email)
	}
	if _, err := s.GetIdentity(ctx, p2, IdentityProviderGoogle, "goog-sub"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project GetIdentity: got %v, want ErrNotFound", err)
	}
	if _, err := s.GetIdentity(ctx, p1, IdentityProviderApple, "goog-sub"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong provider GetIdentity: got %v, want ErrNotFound", err)
	}

	// The same provider subject in another project is an independent
	// identity (no cross-project conflict).
	u2 := testUser(p2, "u2", "user@example.com")
	if err := s.CreateUser(ctx, u2, googleIdentity(u2, "goog-sub")); err != nil {
		t.Fatalf("same subject in second project must be independent: %v", err)
	}
}

func TestIdentityDuplicateSubjectConflicts(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, _ := twoProjects(t, s)

	u1 := testUser(p1, "u1", "a@example.com")
	if err := s.CreateUser(ctx, u1, googleIdentity(u1, "sub")); err != nil {
		t.Fatal(err)
	}
	u2 := testUser(p1, "u2", "b@example.com")
	if err := s.CreateUser(ctx, u2); err != nil {
		t.Fatal(err)
	}
	dup := googleIdentity(u2, "sub")
	if err := s.CreateIdentity(ctx, dup); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate (provider, subject): got %v, want ErrConflict", err)
	}
}

func TestListAndUnlinkUserIdentities(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, p2 := twoProjects(t, s)

	u := testUser(p1, "u1", "user@example.com")
	if err := s.CreateUser(ctx, u, passwordIdentity(u), googleIdentity(u, "goog-sub")); err != nil {
		t.Fatal(err)
	}

	ids, err := s.ListUserIdentities(ctx, p1, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0].Provider != IdentityProviderPassword || ids[1].Provider != IdentityProviderGoogle {
		t.Fatalf("ListUserIdentities = %+v, want [password google]", ids)
	}
	if got, err := s.ListUserIdentities(ctx, p2, u.ID); err != nil || len(got) != 0 {
		t.Fatalf("cross-project ListUserIdentities = %v, %v; want empty", got, err)
	}

	// Unlink is per provider; a second unlink reports NotFound.
	if err := s.DeleteUserIdentities(ctx, p1, u.ID, IdentityProviderGoogle); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteUserIdentities(ctx, p1, u.ID, IdentityProviderGoogle); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second unlink: got %v, want ErrNotFound", err)
	}
	ids, err = s.ListUserIdentities(ctx, p1, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0].Provider != IdentityProviderPassword {
		t.Fatalf("after unlink = %+v, want only password", ids)
	}
}

func TestIdentityAppleRefreshTokenAndProviderEmail(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, p2 := twoProjects(t, s)

	u := testUser(p1, "u1", "user@example.com")
	apple := Identity{
		ID: "aid-1", ProjectID: p1, UserID: u.ID,
		Provider: IdentityProviderApple, ProviderSubject: "apple-sub",
		ProviderEmail: "private@relay.appleid.com", CreatedAt: u.CreatedAt,
	}
	if err := s.CreateUser(ctx, u, apple); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetIdentity(ctx, p1, IdentityProviderApple, "apple-sub")
	if err != nil {
		t.Fatal(err)
	}
	if got.AppleRefreshTokenEnc != nil {
		t.Fatalf("fresh identity has refresh token %v, want nil", got.AppleRefreshTokenEnc)
	}

	enc := []byte{1, 2, 3, 4}
	if err := s.SetIdentityAppleRefreshToken(ctx, p1, apple.ID, enc); err != nil {
		t.Fatal(err)
	}
	if err := s.SetIdentityProviderEmail(ctx, p1, apple.ID, "new@relay.appleid.com"); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetIdentity(ctx, p1, IdentityProviderApple, "apple-sub")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.AppleRefreshTokenEnc, enc) || got.ProviderEmail != "new@relay.appleid.com" {
		t.Fatalf("identity did not round-trip: %+v", got)
	}

	// Cross-project writes must be impossible.
	if err := s.SetIdentityAppleRefreshToken(ctx, p2, apple.ID, enc); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project SetIdentityAppleRefreshToken: got %v, want ErrNotFound", err)
	}
	if err := s.SetIdentityProviderEmail(ctx, p2, apple.ID, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project SetIdentityProviderEmail: got %v, want ErrNotFound", err)
	}

	// Clearing stores NULL again.
	if err := s.SetIdentityAppleRefreshToken(ctx, p1, apple.ID, nil); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetIdentity(ctx, p1, IdentityProviderApple, "apple-sub")
	if err != nil {
		t.Fatal(err)
	}
	if got.AppleRefreshTokenEnc != nil {
		t.Fatalf("cleared refresh token = %v, want nil", got.AppleRefreshTokenEnc)
	}
}

func TestProviderSecrets(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, p2 := twoProjects(t, s)
	now := time.Now()

	if _, err := s.GetProviderSecret(ctx, p1, ProviderSecretApplePrivateKey); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing secret: got %v, want ErrNotFound", err)
	}
	if err := s.SetProviderSecret(ctx, p1, ProviderSecretApplePrivateKey, []byte("v1"), now); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetProviderSecret(ctx, p1, ProviderSecretApplePrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v1" {
		t.Fatalf("secret = %q, want v1", got)
	}
	if _, err := s.GetProviderSecret(ctx, p2, ProviderSecretApplePrivateKey); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project GetProviderSecret: got %v, want ErrNotFound", err)
	}

	// Set again overwrites (upsert).
	if err := s.SetProviderSecret(ctx, p1, ProviderSecretApplePrivateKey, []byte("v2"), now); err != nil {
		t.Fatal(err)
	}
	if got, _ = s.GetProviderSecret(ctx, p1, ProviderSecretApplePrivateKey); string(got) != "v2" {
		t.Fatalf("secret after upsert = %q, want v2", got)
	}

	// Two names coexist on one project.
	if err := s.SetProviderSecret(ctx, p1, ProviderSecretGoogleWebClientSecret, []byte("g"), now); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteProviderSecret(ctx, p1, ProviderSecretApplePrivateKey); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetProviderSecret(ctx, p1, ProviderSecretApplePrivateKey); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted secret: got %v, want ErrNotFound", err)
	}
	// Deleting an absent secret is a no-op.
	if err := s.DeleteProviderSecret(ctx, p1, ProviderSecretApplePrivateKey); err != nil {
		t.Fatal(err)
	}

	// Secrets die with the project.
	if err := s.DeleteProject(ctx, p1); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM project_provider_secrets WHERE project_id = 'p1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("%d provider secrets survived project deletion", n)
	}
}

func TestOAuthTokenSingleUse(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, p2 := twoProjects(t, s)
	now := time.Now()

	state := OAuthToken{
		ID: "ot1", ProjectID: p1, Purpose: OAuthTokenPurposeState, TokenHash: "h1",
		Provider: IdentityProviderGoogle, RedirectURI: "myapp://auth",
		Payload: `{"nonce":"n"}`, ExpiresAt: now.Add(10 * time.Minute), CreatedAt: now,
	}
	if err := s.CreateOAuthToken(ctx, state); err != nil {
		t.Fatal(err)
	}

	// Re-recording the same hashed value is a conflict (ID-token replay
	// rejection relies on it).
	dup := state
	dup.ID = "ot1-dup"
	if err := s.CreateOAuthToken(ctx, dup); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate token hash: got %v, want ErrConflict", err)
	}

	// Tokens are invisible from another project, another purpose.
	if _, err := s.ConsumeOAuthToken(ctx, p2, OAuthTokenPurposeState, "h1", now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project consume: got %v, want ErrNotFound", err)
	}
	if _, err := s.ConsumeOAuthToken(ctx, p1, OAuthTokenPurposeCode, "h1", now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong-purpose consume: got %v, want ErrNotFound", err)
	}

	got, err := s.ConsumeOAuthToken(ctx, p1, OAuthTokenPurposeState, "h1", now)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != state.ID || got.Provider != state.Provider ||
		got.RedirectURI != state.RedirectURI || got.Payload != state.Payload {
		t.Fatalf("consumed token = %+v, want %+v", got, state)
	}
	if got.ConsumedAt == nil {
		t.Fatal("consumed token must carry consumed_at")
	}

	// Single use: a second claim fails.
	if _, err := s.ConsumeOAuthToken(ctx, p1, OAuthTokenPurposeState, "h1", now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second consume: got %v, want ErrNotFound", err)
	}
}

func TestOAuthTokenExpiryAndCleanup(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, _ := twoProjects(t, s)
	now := time.Now()

	u := testUser(p1, "u1", "user@example.com")
	if err := s.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	expired := OAuthToken{
		ID: "ot1", ProjectID: p1, Purpose: OAuthTokenPurposeCode, TokenHash: "h1",
		Provider: IdentityProviderApple, UserID: u.ID,
		ExpiresAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Hour),
	}
	live := OAuthToken{
		ID: "ot2", ProjectID: p1, Purpose: OAuthTokenPurposeCode, TokenHash: "h2",
		Provider: IdentityProviderApple, UserID: u.ID,
		ExpiresAt: now.Add(time.Minute), CreatedAt: now,
	}
	for _, ot := range []OAuthToken{expired, live} {
		if err := s.CreateOAuthToken(ctx, ot); err != nil {
			t.Fatal(err)
		}
	}

	// Expired tokens cannot be claimed.
	if _, err := s.ConsumeOAuthToken(ctx, p1, OAuthTokenPurposeCode, "h1", now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired consume: got %v, want ErrNotFound", err)
	}
	got, err := s.ConsumeOAuthToken(ctx, p1, OAuthTokenPurposeCode, "h2", now)
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != u.ID {
		t.Fatalf("code user = %q, want %q", got.UserID, u.ID)
	}

	// Cleanup sweeps expired rows only.
	if err := s.DeleteExpiredOAuthTokens(ctx, now); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM oauth_tokens`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("%d oauth tokens after cleanup, want 1 (the live one)", n)
	}
}

func TestProviderSettingsRoundTrip(t *testing.T) {
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
	// Auto-linking defaults to enabled when the stored JSON predates the
	// setting (or never set it).
	if !got.Settings.AutoLinkEnabled() {
		t.Fatal("auto_link_verified_email must default to true")
	}

	off := false
	got.Settings.Google = GoogleProviderSettings{
		Enabled: true, WebClientID: "web-id", IOSClientID: "ios-id", AndroidClientID: "android-id",
	}
	got.Settings.Apple = AppleProviderSettings{
		Enabled: true, ServicesID: "com.example.signin", TeamID: "TEAM123",
		KeyID: "KEY123", BundleIDs: []string{"com.example.app", "com.example.app2"},
	}
	got.Settings.AutoLinkVerifiedEmail = &off
	got.Settings.RedirectSchemes = []string{"myapp"}
	got.UpdatedAt = time.Now()
	if err := s.UpdateProject(ctx, got); err != nil {
		t.Fatal(err)
	}

	back, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back.Settings, got.Settings) {
		t.Fatalf("provider settings did not round-trip:\n got %+v\nwant %+v", back.Settings, got.Settings)
	}
	if back.Settings.AutoLinkEnabled() {
		t.Fatal("explicit auto_link_verified_email=false must survive the round trip")
	}
}
