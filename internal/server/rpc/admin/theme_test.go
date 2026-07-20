package adminrpc

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/theme"
)

// racingStore wraps the real store and, on the first SetProjectTheme, first
// lands a competing revision (as a concurrent admin edit would) so the
// wrapped call loses the compare-and-swap — exercising mutateTheme's retry.
type racingStore struct {
	Store
	competing store.ThemeRevision
	raced     bool
}

func (s *racingStore) SetProjectTheme(ctx context.Context, rev store.ThemeRevision, prevRevisionID string) error {
	if !s.raced {
		s.raced = true
		if err := s.Store.SetProjectTheme(ctx, s.competing, prevRevisionID); err != nil {
			return err
		}
		// The original save now races against the competing revision.
	}
	return s.Store.SetProjectTheme(ctx, rev, prevRevisionID)
}

// A logo upload racing a concurrent token save must end with both changes
// applied: the retry re-reads the concurrently saved theme and re-applies
// the logo mutation instead of overwriting it with a stale snapshot.
func TestUploadLogoRetriesLostRace(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	p := store.Project{
		ID: "p1", Name: "My App", Slug: "my-app",
		PublishableKey: "pk_p1", SecretKeyHash: "hash-p1",
		CreatedAt: now, UpdatedAt: now,
	}
	k := store.ProjectKey{
		ID: "key-p1", ProjectID: "p1", Kid: "kid-p1", Algorithm: "ES256",
		PublicKeyPEM: "PEM", PrivateKeyEnc: []byte{1, 2, 3},
		Status: store.ProjectKeyStatusActive, CreatedAt: now,
	}
	if err := st.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}

	// The competing edit: a token save with a custom primary and no logo.
	competingTheme := theme.Default()
	competingTheme.Colors.Primary = "#0B57D0"
	raw, err := theme.Encode(competingTheme)
	if err != nil {
		t.Fatal(err)
	}
	racing := &racingStore{
		Store: st,
		competing: store.ThemeRevision{
			ID: "rev-concurrent", ProjectID: "p1", Theme: raw, CreatedAt: now,
		},
	}

	h := NewThemeHandler(racing, t.TempDir(), nil)
	got, err := h.UploadLogo(ctx, connect.NewRequest(&adminv1.UploadLogoRequest{
		ProjectId: "p1", Variant: adminv1.LogoVariant_LOGO_VARIANT_LIGHT,
		Data: testPNG(t, 32, 32, ""), ContentType: "image/png",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !racing.raced {
		t.Fatal("test setup: the competing save never ran")
	}

	// Both the concurrent primary and the uploaded logo survive.
	wantLogo := "/assets/p1/logo-light.png"
	if got.Msg.Theme.Logo.LightPath != wantLogo {
		t.Errorf("logo path = %q, want %q", got.Msg.Theme.Logo.LightPath, wantLogo)
	}
	if got.Msg.Theme.Colors.Primary != "#0B57D0" {
		t.Errorf("primary = %q, want the concurrently saved #0B57D0", got.Msg.Theme.Colors.Primary)
	}
	proj, err := st.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	cur, err := theme.Parse([]byte(proj.Theme))
	if err != nil {
		t.Fatal(err)
	}
	if cur.Logo.Light != wantLogo || cur.Colors.Primary != "#0B57D0" {
		t.Errorf("stored theme lost a racing change: %+v", cur)
	}
	if proj.ThemeRevisionID != got.Msg.RevisionId {
		t.Errorf("current revision = %q, want %q", proj.ThemeRevisionID, got.Msg.RevisionId)
	}
}
