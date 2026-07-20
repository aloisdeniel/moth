package authrpc

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"

	"crypto/sha256"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/password"
	"github.com/aloisdeniel/moth/internal/store"
)

// lifecycleFixture is a project with one active signing key and a handler
// whose clock the test controls.
type lifecycleFixture struct {
	h       *Handler
	st      *store.Store
	master  keys.MasterKey
	project store.Project
	clock   *time.Time
}

func newLifecycleFixture(t *testing.T) *lifecycleFixture {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "moth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	master, err := keys.LoadOrCreateMasterKey(t.TempDir(), os.Getenv)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	clock := &now

	settings := store.DefaultProjectSettings()
	settings.AccessTokenTTLSeconds = 100000 // long, so grace-window tests are not masked by token expiry
	project := store.Project{
		ID:             NewID(),
		Name:           "Demo",
		Slug:           "demo",
		PublishableKey: "pk_" + NewID(),
		SecretKeyHash:  NewID(),
		Settings:       settings,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	key := newProjectKey(t, master, project.ID, now)
	if err := st.CreateProject(context.Background(), project, key); err != nil {
		t.Fatal(err)
	}
	// Reload so Settings carry the resolved defaults the store persisted.
	project, err = st.GetProject(context.Background(), project.ID)
	if err != nil {
		t.Fatal(err)
	}

	h := New(Options{
		Store:   st,
		Master:  master,
		Mailer:  mail.Console{},
		BaseURL: "http://localhost:8080",
		Now:     func() time.Time { return *clock },
	})
	return &lifecycleFixture{h: h, st: st, master: master, project: project, clock: clock}
}

func newProjectKey(t *testing.T, master keys.MasterKey, projectID string, now time.Time) store.ProjectKey {
	t.Helper()
	sk, err := keys.GenerateSigningKey(master)
	if err != nil {
		t.Fatal(err)
	}
	return store.ProjectKey{
		ID:            NewID(),
		ProjectID:     projectID,
		Kid:           sk.Kid,
		Algorithm:     sk.Algorithm,
		PublicKeyPEM:  sk.PublicKeyPEM,
		PrivateKeyEnc: sk.PrivateKeyEnc,
		Status:        store.ProjectKeyStatusActive,
		CreatedAt:     now,
	}
}

func (f *lifecycleFixture) ctx() context.Context {
	return WithProject(context.Background(), f.project)
}

// createUser inserts a user with the given (possibly foreign) password hash.
func (f *lifecycleFixture) createUser(t *testing.T, email, hash, algo string) store.User {
	t.Helper()
	now := *f.clock
	u := store.User{
		ID:           NewID(),
		ProjectID:    f.project.ID,
		Email:        email,
		PasswordHash: hash,
		PasswordAlgo: algo,
		CustomClaims: "{}",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	verified := now
	u.EmailVerifiedAt = &verified
	identity := store.Identity{
		ID:              NewID(),
		ProjectID:       f.project.ID,
		UserID:          u.ID,
		Provider:        store.IdentityProviderPassword,
		ProviderSubject: u.ID,
		CreatedAt:       now,
	}
	if err := f.st.CreateUser(f.ctx(), u, identity); err != nil {
		t.Fatal(err)
	}
	return u
}

func (f *lifecycleFixture) signIn(email, pw string) (*connect.Response[authv1.SignInResponse], error) {
	return f.h.SignIn(f.ctx(), connect.NewRequest(&authv1.SignInRequest{Email: email, Password: pw}))
}

// --- foreign password hash generators (encodings pwimport.Verify accepts) ---

func bcryptHash(t *testing.T, pw string) string {
	t.Helper()
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func scryptHash(t *testing.T, pw string) string {
	t.Helper()
	salt := []byte("0123456789abcdef")
	const ln, r, p = 14, 8, 1
	dk, err := scrypt.Key([]byte(pw), salt, 1<<ln, r, p, 32)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("$scrypt$ln=%d,r=%d,p=%d$%s$%s", ln, r, p,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(dk))
}

func pbkdf2Hash(_ *testing.T, pw string) string {
	salt := []byte("fedcba9876543210")
	const iter = 12000
	dk := pbkdf2.Key([]byte(pw), salt, iter, 32, sha256.New)
	return fmt.Sprintf("$pbkdf2-sha256$%d$%s$%s", iter,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(dk))
}

func argon2Hash(t *testing.T, pw string) string {
	t.Helper()
	// A native-format argon2id hash, imported under the coarse "argon2" tag so
	// it takes the foreign verify+rehash path rather than the native one.
	h, err := password.Hash(pw)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestSignInImportedForeignHashRehashesToNative(t *testing.T) {
	cases := []struct {
		algo string
		hash func(*testing.T, string) string
	}{
		{store.PasswordAlgoBcrypt, bcryptHash},
		{store.PasswordAlgoScrypt, scryptHash},
		{store.PasswordAlgoPBKDF2, pbkdf2Hash},
		{store.PasswordAlgoArgon2, argon2Hash},
	}
	for _, tc := range cases {
		t.Run(tc.algo, func(t *testing.T) {
			f := newLifecycleFixture(t)
			const pw = "correct horse battery staple"
			email := tc.algo + "@example.com"
			u := f.createUser(t, email, tc.hash(t, pw), tc.algo)

			// Wrong password is still rejected (and must not rehash).
			if _, err := f.signIn(email, "wrong password"); err == nil {
				t.Fatal("wrong password accepted")
			}
			after, err := f.st.GetUser(f.ctx(), f.project.ID, u.ID)
			if err != nil {
				t.Fatal(err)
			}
			if after.PasswordAlgo != tc.algo {
				t.Fatalf("wrong password mutated the stored hash: algo=%q", after.PasswordAlgo)
			}

			// First correct sign-in succeeds and transparently rehashes.
			if _, err := f.signIn(email, pw); err != nil {
				t.Fatalf("first sign-in with imported %s hash failed: %v", tc.algo, err)
			}
			after, err = f.st.GetUser(f.ctx(), f.project.ID, u.ID)
			if err != nil {
				t.Fatal(err)
			}
			if after.PasswordAlgo != store.PasswordAlgoNative {
				t.Fatalf("hash not rehashed to native: algo=%q", after.PasswordAlgo)
			}
			if !password.Verify(pw, after.PasswordHash) {
				t.Fatal("rehashed hash does not verify with the native hasher")
			}
			if after.PasswordHash == u.PasswordHash {
				t.Fatal("stored hash unchanged after rehash")
			}

			// Second sign-in now takes the native path and still works.
			if _, err := f.signIn(email, pw); err != nil {
				t.Fatalf("second (native) sign-in failed: %v", err)
			}
		})
	}
}

func TestGracefulRotationKeepsOldTokensValidUntilGraceExpires(t *testing.T) {
	f := newLifecycleFixture(t)
	ctx := f.ctx()
	now0 := *f.clock

	const pw = "correct horse battery staple"
	native, err := password.Hash(pw)
	if err != nil {
		t.Fatal(err)
	}
	f.createUser(t, "u@example.com", native, store.PasswordAlgoNative)

	resp, err := f.signIn("u@example.com", pw)
	if err != nil {
		t.Fatal(err)
	}
	access := resp.Msg.Tokens.AccessToken

	getMe := func() error {
		req := connect.NewRequest(&authv1.GetMeRequest{})
		req.Header().Set("Authorization", "Bearer "+access)
		_, err := f.h.GetMe(ctx, req)
		return err
	}
	if err := getMe(); err != nil {
		t.Fatalf("token invalid before rotation: %v", err)
	}

	// Rotate: new active key B, old key A stays in grace for one hour.
	keyB := newProjectKey(t, f.master, f.project.ID, now0)
	graceUntil := now0.Add(time.Hour)
	if err := f.st.RotateSigningKey(ctx, f.project.ID, keyB, graceUntil, now0); err != nil {
		t.Fatal(err)
	}

	// During grace the old token still verifies (A is still in the JWKS set).
	if err := getMe(); err != nil {
		t.Fatalf("old token rejected during grace: %v", err)
	}
	// A freshly minted token now uses key B and also verifies.
	resp2, err := f.signIn("u@example.com", pw)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.Msg.Tokens.AccessToken == access {
		t.Fatal("expected a new access token after rotation")
	}

	// Past not_after the old key leaves the JWKS set: the old token no longer
	// verifies even though it has not yet expired.
	*f.clock = now0.Add(90 * time.Minute)
	if err := getMe(); err == nil {
		t.Fatal("old token still valid after grace expired")
	}

	// The sweep prunes the expired grace key from the row set entirely.
	n, err := f.st.PruneExpiredKeys(ctx, *f.clock)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("prune removed %d keys, want 1", n)
	}
	remaining, err := f.st.ListActiveAndGraceKeys(ctx, f.project.ID, *f.clock)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].Kid != keyB.Kid {
		t.Fatalf("after prune JWKS should hold only key B, got %d keys", len(remaining))
	}
}

func TestHardResetInvalidatesTokensImmediately(t *testing.T) {
	f := newLifecycleFixture(t)
	ctx := f.ctx()
	now0 := *f.clock

	const pw = "correct horse battery staple"
	native, err := password.Hash(pw)
	if err != nil {
		t.Fatal(err)
	}
	f.createUser(t, "u@example.com", native, store.PasswordAlgoNative)

	resp, err := f.signIn("u@example.com", pw)
	if err != nil {
		t.Fatal(err)
	}
	access := resp.Msg.Tokens.AccessToken

	getMe := func() error {
		req := connect.NewRequest(&authv1.GetMeRequest{})
		req.Header().Set("Authorization", "Bearer "+access)
		_, err := f.h.GetMe(ctx, req)
		return err
	}
	if err := getMe(); err != nil {
		t.Fatalf("token invalid before reset: %v", err)
	}

	// Hard reset retires the old key with no grace: tokens die at once.
	keyC := newProjectKey(t, f.master, f.project.ID, now0)
	if err := f.st.ResetProjectSigningKey(ctx, f.project.ID, keyC, now0); err != nil {
		t.Fatal(err)
	}
	if err := getMe(); err == nil {
		t.Fatal("token still valid immediately after a hard signing-key reset")
	}
}
