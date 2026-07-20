package store

import (
	"context"
	"testing"
	"time"
)

func kids(keys []ProjectKey) map[string]ProjectKey {
	m := make(map[string]ProjectKey, len(keys))
	for _, k := range keys {
		m[k.Kid] = k
	}
	return m
}

func TestGracefulSigningKeyRotation(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	p, k := testProject("p1", "app")
	if err := s.CreateProject(ctx, p, k); err != nil { // active key kid-p1
		t.Fatal(err)
	}

	grace := now.Add(20 * time.Minute)
	newKey := ProjectKey{ID: "key2", ProjectID: "p1", Kid: "kid-rotated", Algorithm: "ES256",
		PublicKeyPEM: "PEM2", PrivateKeyEnc: []byte{9}, Status: ProjectKeyStatusActive, CreatedAt: now}
	if err := s.RotateSigningKey(ctx, "p1", newKey, grace, now); err != nil {
		t.Fatal(err)
	}

	// Only the new key is "active"; JWKS covers both while grace holds.
	active, err := s.ListActiveProjectKeys(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].Kid != "kid-rotated" {
		t.Fatalf("active keys after rotate: %+v", active)
	}

	jwks, err := s.ListActiveAndGraceKeys(ctx, "p1", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	km := kids(jwks)
	if len(jwks) != 2 || km["kid-p1"].Status != ProjectKeyStatusGrace || km["kid-rotated"].Status != ProjectKeyStatusActive {
		t.Fatalf("JWKS should include active + grace: %+v", jwks)
	}
	if g := km["kid-p1"]; g.RotatedAt == nil || g.NotAfter == nil || !g.NotAfter.Equal(grace) {
		t.Fatalf("grace metadata not persisted: rotated_at=%v not_after=%v", g.RotatedAt, g.NotAfter)
	}

	// After grace expires the old key drops out of the JWKS...
	jwksAfter, err := s.ListActiveAndGraceKeys(ctx, "p1", grace.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(jwksAfter) != 1 || jwksAfter[0].Kid != "kid-rotated" {
		t.Fatalf("expired grace key must leave the JWKS: %+v", jwksAfter)
	}

	// ...but the row lingers until pruned.
	pruned, err := s.PruneExpiredKeys(ctx, grace.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 1 {
		t.Fatalf("want 1 pruned key, got %d", pruned)
	}
	// Second prune is a no-op; active key never pruned.
	if n, _ := s.PruneExpiredKeys(ctx, grace.Add(time.Hour)); n != 0 {
		t.Fatalf("re-prune should remove nothing, got %d", n)
	}
	if active, _ := s.ListActiveProjectKeys(ctx, "p1"); len(active) != 1 {
		t.Fatalf("active key must survive pruning: %+v", active)
	}
}

// TestHardResetAfterRotationRetiresGraceKeys guards the invariant that a hard
// reset invalidates EVERY issued token even when a graceful rotation left a
// grace-period key behind: that grace key must not survive in the JWKS to keep
// validating tokens signed by it.
func TestHardResetAfterRotationRetiresGraceKeys(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	p, _ := testProject("p1", "app") // active key kid-p1
	if err := s.CreateProject(ctx, p, mustKey("kid-p1")); err != nil {
		t.Fatal(err)
	}
	// Rotate: kid-p1 → grace (long window), kid-rot → active.
	if err := s.RotateSigningKey(ctx, "p1", mustKey("kid-rot"), now.Add(time.Hour), now); err != nil {
		t.Fatal(err)
	}
	// Hard reset must retire BOTH the active and the grace key.
	if err := s.ResetProjectSigningKey(ctx, "p1", mustKey("kid-fresh"), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	jwks, err := s.ListActiveAndGraceKeys(ctx, "p1", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(jwks) != 1 || jwks[0].Kid != "kid-fresh" {
		t.Fatalf("hard reset must leave only the fresh key in the JWKS, got %+v", jwks)
	}
}

func mustKey(kid string) ProjectKey {
	return ProjectKey{ID: "id-" + kid, ProjectID: "p1", Kid: kid, Algorithm: "ES256",
		PublicKeyPEM: "PEM-" + kid, PrivateKeyEnc: []byte{9}, Status: ProjectKeyStatusActive,
		CreatedAt: time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)}
}

func TestRotationPreservesRefreshTokens(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	p, k := testProject("p1", "app")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateUser(ctx, User{ID: "u1", ProjectID: "p1", Email: "u@example.com",
		CustomClaims: "{}", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	rt := RefreshToken{ID: "rt1", ProjectID: "p1", UserID: "u1", TokenHash: "rth", FamilyID: "f1",
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now}
	if err := s.CreateRefreshToken(ctx, rt); err != nil {
		t.Fatal(err)
	}

	newKey := ProjectKey{ID: "key2", ProjectID: "p1", Kid: "kid-rot", Algorithm: "ES256",
		PublicKeyPEM: "PEM2", PrivateKeyEnc: []byte{9}, Status: ProjectKeyStatusActive, CreatedAt: now}
	if err := s.RotateSigningKey(ctx, "p1", newKey, now.Add(time.Hour), now); err != nil {
		t.Fatal(err)
	}

	// Unlike ResetProjectSigningKey, a graceful rotate keeps refresh tokens
	// live so nobody is signed out.
	got, err := s.GetRefreshToken(ctx, "p1", "rth")
	if err != nil {
		t.Fatal(err)
	}
	if got.RevokedAt != nil {
		t.Fatalf("rotation must not revoke refresh tokens: %+v", got.RevokedAt)
	}
}
