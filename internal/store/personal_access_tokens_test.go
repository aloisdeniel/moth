package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func createPATTestAdmin(t *testing.T, s *Store, id string) Admin {
	t.Helper()
	now := time.Now()
	a := Admin{ID: id, Email: id + "@example.com", PasswordHash: "h", CreatedAt: now, UpdatedAt: now}
	if err := s.CreateAdmin(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestPATLifecycle(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	admin := createPATTestAdmin(t, s, "a1")
	other := createPATTestAdmin(t, s, "a2")
	now := time.Now()

	expires := now.Add(24 * time.Hour)
	first := PersonalAccessToken{
		ID: "t1", AdminID: admin.ID, Name: "laptop", TokenHash: "hash-1",
		CreatedAt: now.Add(-time.Minute),
	}
	second := PersonalAccessToken{
		ID: "t2", AdminID: admin.ID, Name: "ci", TokenHash: "hash-2",
		CreatedAt: now, ExpiresAt: &expires,
	}
	for _, pat := range []PersonalAccessToken{first, second} {
		if err := s.CreatePAT(ctx, pat); err != nil {
			t.Fatal(err)
		}
	}
	// The hash is the credential; a duplicate must be rejected.
	if err := s.CreatePAT(ctx, PersonalAccessToken{
		ID: "t3", AdminID: admin.ID, Name: "dup", TokenHash: "hash-1", CreatedAt: now,
	}); err == nil {
		t.Fatal("duplicate token_hash should fail")
	}

	got, err := s.GetPATByHash(ctx, "hash-2")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "t2" || got.AdminID != admin.ID || got.Name != "ci" {
		t.Fatalf("get mismatch: %+v", got)
	}
	if got.LastUsedAt != nil || got.RevokedAt != nil {
		t.Fatalf("nullable fields should round-trip as nil: %+v", got)
	}
	if got.ExpiresAt == nil || got.ExpiresAt.Unix() != expires.Unix() {
		t.Fatalf("expires_at not round-tripped: %+v", got.ExpiresAt)
	}
	if _, err := s.GetPATByHash(ctx, "no-such-hash"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	list, err := s.ListPATs(ctx, admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != "t2" || list[1].ID != "t1" {
		t.Fatalf("want [t2 t1] newest first, got %+v", list)
	}
	if list, _ := s.ListPATs(ctx, other.ID); len(list) != 0 {
		t.Fatalf("tokens must be scoped to their admin, got %+v", list)
	}

	usedAt := now.Add(time.Second)
	if err := s.TouchPAT(ctx, "t1", usedAt); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetPATByHash(ctx, "hash-1")
	if got.LastUsedAt == nil || got.LastUsedAt.Unix() != usedAt.Unix() {
		t.Fatalf("touch not recorded: %+v", got.LastUsedAt)
	}

	// Revocation is scoped to the owning admin and not repeatable.
	revokeCases := []struct {
		name          string
		adminID, id   string
		wantErr       error
		wantRevokedAt bool
	}{
		{"wrong admin", other.ID, "t1", ErrNotFound, false},
		{"missing token", admin.ID, "nope", ErrNotFound, false},
		{"owner revokes", admin.ID, "t1", nil, true},
		{"already revoked", admin.ID, "t1", ErrNotFound, true},
	}
	for _, tc := range revokeCases {
		err := s.RevokePAT(ctx, tc.adminID, tc.id, now)
		if tc.wantErr == nil && err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
			t.Fatalf("%s: want %v, got %v", tc.name, tc.wantErr, err)
		}
		got, _ := s.GetPATByHash(ctx, "hash-1")
		if (got.RevokedAt != nil) != tc.wantRevokedAt {
			t.Fatalf("%s: revoked_at = %v, want set=%v", tc.name, got.RevokedAt, tc.wantRevokedAt)
		}
	}
	// Revoked rows stay listable until pruned.
	if list, _ := s.ListPATs(ctx, admin.ID); len(list) != 2 {
		t.Fatalf("revoked token should still list, got %+v", list)
	}
}

func TestPATUsable(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	cases := []struct {
		name string
		pat  PersonalAccessToken
		want bool
	}{
		{"no expiry", PersonalAccessToken{}, true},
		{"future expiry", PersonalAccessToken{ExpiresAt: &future}, true},
		{"expired", PersonalAccessToken{ExpiresAt: &past}, false},
		{"revoked", PersonalAccessToken{RevokedAt: &past}, false},
		{"revoked with future expiry", PersonalAccessToken{ExpiresAt: &future, RevokedAt: &now}, false},
	}
	for _, tc := range cases {
		if got := tc.pat.Usable(now); got != tc.want {
			t.Errorf("%s: Usable = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestDeleteExpiredPATs(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	admin := createPATTestAdmin(t, s, "a1")
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)

	cases := []struct {
		id        string
		expiresAt *time.Time
		revokedAt *time.Time
		keep      bool
	}{
		{"never-expires", nil, nil, true},
		{"future-expiry", &future, nil, true},
		{"expired", &past, nil, false},
		{"revoked-no-expiry", nil, &past, true}, // stays listable
		{"revoked-and-expired", &past, &past, false},
	}
	for _, tc := range cases {
		if err := s.CreatePAT(ctx, PersonalAccessToken{
			ID: tc.id, AdminID: admin.ID, Name: tc.id, TokenHash: "hash-" + tc.id,
			CreatedAt: now, ExpiresAt: tc.expiresAt, RevokedAt: tc.revokedAt,
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.DeleteExpiredPATs(ctx, now); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		_, err := s.GetPATByHash(ctx, "hash-"+tc.id)
		if tc.keep && err != nil {
			t.Errorf("%s: should survive prune, got %v", tc.id, err)
		}
		if !tc.keep && !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: should be pruned, got %v", tc.id, err)
		}
	}
}
