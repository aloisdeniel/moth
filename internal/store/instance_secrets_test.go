package store

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestInstanceSecretRoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now()

	if _, err := s.GetInstanceSecret(ctx, InstanceSecretSMTPPassword); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing secret: want ErrNotFound, got %v", err)
	}

	ct := []byte{0x00, 0x01, 0xff, 0x7f, 0x80}
	if err := s.SetInstanceSecret(ctx, InstanceSecretSMTPPassword, ct, now); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetInstanceSecret(ctx, InstanceSecretSMTPPassword)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, ct) {
		t.Fatalf("ciphertext not round-tripped: %v", got)
	}

	// Upsert replaces.
	ct2 := []byte{0xaa, 0xbb}
	if err := s.SetInstanceSecret(ctx, InstanceSecretSMTPPassword, ct2, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetInstanceSecret(ctx, InstanceSecretSMTPPassword); !bytes.Equal(got, ct2) {
		t.Fatalf("upsert did not replace: %v", got)
	}

	if err := s.DeleteInstanceSecret(ctx, InstanceSecretSMTPPassword); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetInstanceSecret(ctx, InstanceSecretSMTPPassword); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: want ErrNotFound, got %v", err)
	}
	// Deleting an absent secret is a no-op.
	if err := s.DeleteInstanceSecret(ctx, InstanceSecretSMTPPassword); err != nil {
		t.Fatalf("delete absent secret should be no-op: %v", err)
	}
}
