package ratelimit

import (
	"testing"
	"time"
)

func TestBurstThenRefill(t *testing.T) {
	l := New(60, 3) // 1/s sustained, burst 3
	now := time.Now()

	for i := 0; i < 3; i++ {
		if !l.allow("ip", now) {
			t.Fatalf("request %d within burst denied", i)
		}
	}
	if l.allow("ip", now) {
		t.Fatal("request beyond burst allowed")
	}

	// One token refills after one second.
	if !l.allow("ip", now.Add(time.Second)) {
		t.Fatal("request after refill denied")
	}
	if l.allow("ip", now.Add(time.Second)) {
		t.Fatal("second request after single refill allowed")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	l := New(60, 1)
	now := time.Now()

	if !l.allow("a", now) {
		t.Fatal("first key denied")
	}
	if l.allow("a", now) {
		t.Fatal("first key not limited")
	}
	if !l.allow("b", now) {
		t.Fatal("second key affected by first")
	}
}
