package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func testPushDevice(projectID, userID, id string) PushDevice {
	now := time.Now()
	return PushDevice{
		ID: id, ProjectID: projectID, UserID: userID,
		Target: PushTargetAPNs, Token: "tok-" + id, DeviceID: "dev-" + id,
		Permission: PushPermissionGranted,
		Platform:   "ios", Model: "iPhone15,2", OSVersion: "18.1",
		AppVersion: "1.2.3", Locale: "fr-FR",
		CreatedAt: now, UpdatedAt: now, LastSeenAt: now,
	}
}

// seedPushUsers creates two projects with two users each and returns
// (p1, p2); users are "u1".."u4" (u1/u2 in p1, u3/u4 in p2).
func seedPushUsers(t *testing.T, s *Store) (string, string) {
	t.Helper()
	ctx := context.Background()
	p1, p2 := twoProjects(t, s)
	for i, pid := range []string{p1, p1, p2, p2} {
		id := []string{"u1", "u2", "u3", "u4"}[i]
		u := testUser(pid, id, id+"@example.com")
		if err := s.CreateUser(ctx, u, passwordIdentity(u)); err != nil {
			t.Fatal(err)
		}
	}
	return p1, p2
}

// TestUpsertPushDeviceMatrix covers the upsert matrix from the milestone-20
// acceptance criteria: fresh register, no-op re-register, same device with a
// rotated token, same token from a new device, and the same device/token
// taken over by a new user.
func TestUpsertPushDeviceMatrix(t *testing.T) {
	tests := []struct {
		name string
		next func(d PushDevice) PushDevice // mutates the second registration
		// wantRefresh: the first row survives with its id; otherwise the
		// first row must be revoked 'replaced' and a new row inserted.
		wantRefresh bool
		wantOwner   string // user owning the active row after the upsert
	}{
		{
			name: "no-op re-register refreshes in place",
			next: func(d PushDevice) PushDevice {
				d.Permission = PushPermissionDenied // permission transition round-trips
				d.AppVersion = "1.3.0"
				return d
			},
			wantRefresh: true,
			wantOwner:   "u1",
		},
		{
			name: "same device new token supersedes",
			next: func(d PushDevice) PushDevice {
				d.Token = "tok-rotated"
				return d
			},
			wantOwner: "u1",
		},
		{
			name: "same token new device supersedes",
			next: func(d PushDevice) PushDevice {
				d.DeviceID = "dev-other"
				return d
			},
			wantOwner: "u1",
		},
		{
			name: "same token and device new user takes over",
			next: func(d PushDevice) PushDevice {
				d.UserID = "u2"
				return d
			},
			wantOwner: "u2",
		},
		{
			name: "same device new target and token supersedes",
			next: func(d PushDevice) PushDevice {
				d.Target = PushTargetFCM
				d.Token = "tok-fcm"
				return d
			},
			wantOwner: "u1",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := openTestStore(t)
			ctx := context.Background()
			p1, _ := seedPushUsers(t, s)

			first := testPushDevice(p1, "u1", "d1")
			stored, err := s.UpsertPushDevice(ctx, first)
			if err != nil {
				t.Fatal(err)
			}
			if stored.ID != "d1" || stored.RevokedAt != nil {
				t.Fatalf("first register: %+v", stored)
			}

			second := tc.next(first)
			second.ID = "d2"
			later := time.Now().Add(time.Hour)
			second.CreatedAt, second.UpdatedAt, second.LastSeenAt = later, later, later
			got, err := s.UpsertPushDevice(ctx, second)
			if err != nil {
				t.Fatal(err)
			}

			old, err := s.GetPushDevice(ctx, p1, "d1")
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantRefresh {
				if got.ID != "d1" {
					t.Fatalf("want in-place refresh of d1, got row %s", got.ID)
				}
				if old.RevokedAt != nil {
					t.Fatalf("refreshed row must stay active: %+v", old)
				}
				if !old.LastSeenAt.After(first.LastSeenAt) {
					t.Fatalf("last_seen_at not refreshed: %v", old.LastSeenAt)
				}
				if old.Permission != second.Permission || old.AppVersion != second.AppVersion {
					t.Fatalf("metadata not refreshed: %+v", old)
				}
				if !old.CreatedAt.Equal(stored.CreatedAt) {
					t.Fatalf("created_at must be preserved: %v vs %v", old.CreatedAt, stored.CreatedAt)
				}
			} else {
				if got.ID != "d2" {
					t.Fatalf("want new row d2, got %s", got.ID)
				}
				if old.RevokedAt == nil || old.RevokeReason != PushRevokeReasonReplaced {
					t.Fatalf("old row must be revoked 'replaced': %+v", old)
				}
			}

			// Exactly one active row remains, owned by wantOwner, and never
			// duplicated in the project listing.
			all, err := s.ListActivePushDevices(ctx, p1, PushDevicePage{Limit: 10})
			if err != nil {
				t.Fatal(err)
			}
			if len(all) != 1 {
				t.Fatalf("want 1 active row, got %d", len(all))
			}
			if all[0].UserID != tc.wantOwner {
				t.Fatalf("active row owner = %s, want %s", all[0].UserID, tc.wantOwner)
			}

			// The takeover case: the first user's list no longer shows it.
			if tc.wantOwner != "u1" {
				mine, err := s.ListActivePushDevicesByUser(ctx, p1, "u1")
				if err != nil {
					t.Fatal(err)
				}
				if len(mine) != 0 {
					t.Fatalf("u1 must have no active devices, got %d", len(mine))
				}
			}
		})
	}
}

// TestUpsertPushDeviceConcurrent hammers UpsertPushDevice — the per-launch
// hot path — from parallel goroutines with non-colliding registrations.
// Regression test: with deferred transactions the SELECT-then-write upsert
// failed its snapshot upgrade under concurrent writers (SQLITE_BUSY /
// SQLITE_BUSY_SNAPSHOT bypass busy_timeout) and RegisterDevice sporadically
// returned 500; _txlock=immediate in the DSN makes writers queue instead.
func TestUpsertPushDeviceConcurrent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, _ := seedPushUsers(t, s)

	const workers, perWorker = 16, 25
	errs := make(chan error, workers*perWorker)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				// Distinct ids → distinct token/device_id via testPushDevice,
				// so no upsert collides with another and every row must land.
				d := testPushDevice(p1, "u1", fmt.Sprintf("d-%02d-%02d", w, i))
				if _, err := s.UpsertPushDevice(ctx, d); err != nil {
					errs <- fmt.Errorf("worker %d upsert %d: %w", w, i, err)
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	counts, err := s.CountActivePushDevicesByTarget(ctx, p1)
	if err != nil {
		t.Fatal(err)
	}
	if counts[PushTargetAPNs] != workers*perWorker {
		t.Fatalf("want %d active rows, got %d", workers*perWorker, counts[PushTargetAPNs])
	}
}

// TestUpsertPushDeviceSupersedesTwoRows exercises the corner where the new
// registration collides with two distinct active rows — one holding its
// device_id, another holding its (target, token) — and both must be replaced.
func TestUpsertPushDeviceSupersedesTwoRows(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, _ := seedPushUsers(t, s)

	a := testPushDevice(p1, "u1", "da") // token tok-da, device dev-da
	b := testPushDevice(p1, "u2", "db") // token tok-db, device dev-db
	for _, d := range []PushDevice{a, b} {
		if _, err := s.UpsertPushDevice(ctx, d); err != nil {
			t.Fatal(err)
		}
	}

	c := testPushDevice(p1, "u1", "dc")
	c.DeviceID = a.DeviceID // collides with a's installation
	c.Token = b.Token       // and with b's credential
	got, err := s.UpsertPushDevice(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "dc" {
		t.Fatalf("want new row dc, got %s", got.ID)
	}
	for _, id := range []string{"da", "db"} {
		old, err := s.GetPushDevice(ctx, p1, id)
		if err != nil {
			t.Fatal(err)
		}
		if old.RevokedAt == nil || old.RevokeReason != PushRevokeReasonReplaced {
			t.Fatalf("row %s must be revoked 'replaced': %+v", id, old)
		}
	}
	all, err := s.ListActivePushDevices(ctx, p1, PushDevicePage{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != "dc" {
		t.Fatalf("want only dc active, got %+v", all)
	}
}

func TestRevokePushDeviceIdempotence(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, _ := seedPushUsers(t, s)
	now := time.Now()

	d := testPushDevice(p1, "u1", "d1")
	if _, err := s.UpsertPushDevice(ctx, d); err != nil {
		t.Fatal(err)
	}

	// Client unregister (signed_out), replayed.
	for i := 0; i < 2; i++ {
		if err := s.RevokePushDeviceByDeviceID(ctx, p1, "u1", d.DeviceID, PushRevokeReasonSignedOut, now); err != nil {
			t.Fatalf("revoke replay %d: %v", i, err)
		}
	}
	got, err := s.GetPushDevice(ctx, p1, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if got.RevokedAt == nil || got.RevokeReason != PushRevokeReasonSignedOut {
		t.Fatalf("want signed_out, got %+v", got)
	}

	// A replay must not overwrite the recorded reason either.
	if err := s.RevokePushDeviceByTokenOrDevice(ctx, p1, d.Token, "", PushRevokeReasonReportedInvalid, now); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetPushDevice(ctx, p1, "d1")
	if got.RevokeReason != PushRevokeReasonSignedOut {
		t.Fatalf("revoked row re-revoked: %+v", got)
	}

	// Unknown selectors are no-ops, not errors.
	if err := s.RevokePushDeviceByDeviceID(ctx, p1, "u1", "dev-unknown", PushRevokeReasonSignedOut, now); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokePushDeviceByTokenOrDevice(ctx, p1, "tok-unknown", "dev-unknown", PushRevokeReasonReportedInvalid, now); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokePushDeviceByTokenOrDevice(ctx, p1, "", "", PushRevokeReasonReportedInvalid, now); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokePushDevice(ctx, p1, "row-unknown", PushRevokeReasonAdmin, now); err != nil {
		t.Fatal(err)
	}

	// Feedback revoke by token, then by device id, on fresh rows.
	d2 := testPushDevice(p1, "u1", "d2")
	if _, err := s.UpsertPushDevice(ctx, d2); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokePushDeviceByTokenOrDevice(ctx, p1, d2.Token, "", PushRevokeReasonReportedInvalid, now); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetPushDevice(ctx, p1, "d2")
	if got.RevokedAt == nil || got.RevokeReason != PushRevokeReasonReportedInvalid {
		t.Fatalf("want reported_invalid, got %+v", got)
	}

	// Admin revoke by row id.
	d3 := testPushDevice(p1, "u2", "d3")
	if _, err := s.UpsertPushDevice(ctx, d3); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokePushDevice(ctx, p1, "d3", PushRevokeReasonAdmin, now); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetPushDevice(ctx, p1, "d3")
	if got.RevokedAt == nil || got.RevokeReason != PushRevokeReasonAdmin {
		t.Fatalf("want admin, got %+v", got)
	}

	// Revoked rows never appear in the list reads.
	for _, uid := range []string{"u1", "u2"} {
		list, err := s.ListActivePushDevicesByUser(ctx, p1, uid)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 0 {
			t.Fatalf("user %s: want no active devices, got %d", uid, len(list))
		}
	}
	all, err := s.ListActivePushDevices(ctx, p1, PushDevicePage{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Fatalf("want no active devices, got %d", len(all))
	}
}

func TestPushDevicesCrossProjectInvisibility(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, p2 := seedPushUsers(t, s)
	now := time.Now()

	d1 := testPushDevice(p1, "u1", "d1")
	if _, err := s.UpsertPushDevice(ctx, d1); err != nil {
		t.Fatal(err)
	}
	// The same credential and device id may exist independently in another
	// project — uniqueness is per project.
	d2 := testPushDevice(p2, "u3", "d2")
	d2.Token, d2.DeviceID = d1.Token, d1.DeviceID
	if _, err := s.UpsertPushDevice(ctx, d2); err != nil {
		t.Fatal(err)
	}

	// Project A's reads never see B's rows and vice versa.
	if _, err := s.GetPushDevice(ctx, p2, "d1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	list, err := s.ListActivePushDevices(ctx, p2, PushDevicePage{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "d2" {
		t.Fatalf("p2 must see only d2: %+v", list)
	}

	// Project B revoking A's credential is a no-op on A's row.
	if err := s.RevokePushDeviceByTokenOrDevice(ctx, p2, d1.Token, d1.DeviceID, PushRevokeReasonReportedInvalid, now); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetPushDevice(ctx, p1, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if got.RevokedAt != nil {
		t.Fatalf("p1's row must stay active: %+v", got)
	}
	// Re-registering in p2 must not supersede p1's row.
	if got, err = s.GetPushDevice(ctx, p1, "d1"); err != nil || got.RevokeReason != "" {
		t.Fatalf("cross-project supersede: %+v (%v)", got, err)
	}
}

func TestListActivePushDevicesPagination(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, _ := seedPushUsers(t, s)

	// d1..d5: odd ids are apns, even are fcm (testPushDevice defaults apns).
	for i := 1; i <= 5; i++ {
		d := testPushDevice(p1, "u1", "d"+string(rune('0'+i)))
		if i%2 == 0 {
			d.Target = PushTargetFCM
		}
		if _, err := s.UpsertPushDevice(ctx, d); err != nil {
			t.Fatal(err)
		}
	}

	// Newest first, keyset pages.
	page1, err := s.ListActivePushDevices(ctx, p1, PushDevicePage{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || page1[0].ID != "d5" || page1[1].ID != "d4" {
		t.Fatalf("page1: %+v", page1)
	}
	page2, err := s.ListActivePushDevices(ctx, p1, PushDevicePage{Limit: 2, AfterID: page1[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 || page2[0].ID != "d3" || page2[1].ID != "d2" {
		t.Fatalf("page2: %+v", page2)
	}

	// Target filter.
	fcm, err := s.ListActivePushDevices(ctx, p1, PushDevicePage{Limit: 10, Target: PushTargetFCM})
	if err != nil {
		t.Fatal(err)
	}
	if len(fcm) != 2 {
		t.Fatalf("want 2 fcm rows, got %d", len(fcm))
	}
	for _, d := range fcm {
		if d.Target != PushTargetFCM {
			t.Fatalf("target filter leaked: %+v", d)
		}
	}

	counts, err := s.CountActivePushDevicesByTarget(ctx, p1)
	if err != nil {
		t.Fatal(err)
	}
	if counts[PushTargetAPNs] != 3 || counts[PushTargetFCM] != 2 {
		t.Fatalf("counts: %+v", counts)
	}
}

func TestRevokeStalePushDevices(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, p2 := seedPushUsers(t, s)
	now := time.Now()

	stale := testPushDevice(p1, "u1", "d-stale")
	stale.LastSeenAt = now.Add(-100 * 24 * time.Hour)
	fresh := testPushDevice(p1, "u2", "d-fresh")
	fresh.LastSeenAt = now.Add(-10 * 24 * time.Hour)
	staleP2 := testPushDevice(p2, "u3", "d-stale-p2")
	staleP2.LastSeenAt = now.Add(-100 * 24 * time.Hour)
	for _, d := range []PushDevice{stale, fresh, staleP2} {
		if _, err := s.UpsertPushDevice(ctx, d); err != nil {
			t.Fatal(err)
		}
	}

	cutoff := now.Add(-90 * 24 * time.Hour)
	n, err := s.RevokeStalePushDevices(ctx, cutoff, now)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 { // the sweep is project-agnostic
		t.Fatalf("want 2 revoked, got %d", n)
	}
	for pid, id := range map[string]string{p1: "d-stale", p2: "d-stale-p2"} {
		got, err := s.GetPushDevice(ctx, pid, id)
		if err != nil {
			t.Fatal(err)
		}
		if got.RevokedAt == nil || got.RevokeReason != PushRevokeReasonStale {
			t.Fatalf("want stale, got %+v", got)
		}
	}
	got, err := s.GetPushDevice(ctx, p1, "d-fresh")
	if err != nil {
		t.Fatal(err)
	}
	if got.RevokedAt != nil {
		t.Fatalf("fresh row swept: %+v", got)
	}

	// Idempotent on replay.
	n, err = s.RevokeStalePushDevices(ctx, cutoff, now)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("second sweep must revoke nothing, got %d", n)
	}
}

func TestListPushDevicesForAdmin(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p1, _ := seedPushUsers(t, s)
	now := time.Now()

	active := testPushDevice(p1, "u1", "d-active")
	recent := testPushDevice(p1, "u1", "d-recent")
	old := testPushDevice(p1, "u1", "d-old")
	other := testPushDevice(p1, "u2", "d-other")
	for _, d := range []PushDevice{active, recent, old, other} {
		if _, err := s.UpsertPushDevice(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RevokePushDevice(ctx, p1, "d-recent", PushRevokeReasonAdmin, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokePushDevice(ctx, p1, "d-old", PushRevokeReasonSignedOut, now.Add(-60*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	list, err := s.ListPushDevicesForAdmin(ctx, p1, "u1", now.Add(-30*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	ids := make(map[string]bool, len(list))
	for _, d := range list {
		ids[d.ID] = true
	}
	if len(list) != 2 || !ids["d-active"] || !ids["d-recent"] {
		t.Fatalf("want active + recently revoked rows, got %+v", ids)
	}
}
