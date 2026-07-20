package events

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// recordingStore collects batches; safe for concurrent use.
type recordingStore struct {
	mu      sync.Mutex
	batches [][]Event
}

func (s *recordingStore) InsertEvents(_ context.Context, evs []Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches = append(s.batches, evs)
	return nil
}

func (s *recordingStore) total() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, b := range s.batches {
		n += len(b)
	}
	return n
}

func (s *recordingStore) batchSizes() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	sizes := make([]int, len(s.batches))
	for i, b := range s.batches {
		sizes[i] = len(b)
	}
	return sizes
}

// stalledStore blocks every insert until the writer cancels it.
type stalledStore struct{}

func (stalledStore) InsertEvents(ctx context.Context, _ []Event) error {
	<-ctx.Done()
	return ctx.Err()
}

// failingStore rejects every insert.
type failingStore struct{}

func (failingStore) InsertEvents(context.Context, []Event) error {
	return errors.New("disk on fire")
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func testEvent(typ string) Event {
	return Event{Type: typ, ProjectID: "p1", UserID: "u1"}
}

func TestEmitNeverBlocksOnStalledStore(t *testing.T) {
	w := NewWriter(stalledStore{}, Config{
		BufferSize:    8,
		BatchSize:     4,
		FlushInterval: 10 * time.Millisecond,
	})

	const n = 10_000
	start := time.Now()
	for range n {
		w.Emit(testEvent(TypeUserLogin))
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("emitting %d events against a stalled store took %v", n, elapsed)
	}

	st := w.Stats()
	if st.Dropped == 0 {
		t.Fatal("no drops recorded despite full buffer")
	}
	if st.Enqueued+st.Dropped != n {
		t.Fatalf("enqueued %d + dropped %d != emitted %d", st.Enqueued, st.Dropped, n)
	}

	// Close cannot drain a stalled store: it must give up at the deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := w.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close = %v, want deadline exceeded", err)
	}
}

func TestFlushOnBatchSize(t *testing.T) {
	store := &recordingStore{}
	w := NewWriter(store, Config{BatchSize: 3, FlushInterval: time.Minute})
	defer w.Close(context.Background())

	for range 3 {
		w.Emit(testEvent(TypeUserSignup))
	}
	waitFor(t, "size-triggered flush", func() bool { return store.total() == 3 })
	if sizes := store.batchSizes(); len(sizes) != 1 || sizes[0] != 3 {
		t.Fatalf("batch sizes = %v, want [3]", sizes)
	}
}

func TestFlushOnInterval(t *testing.T) {
	store := &recordingStore{}
	w := NewWriter(store, Config{BatchSize: 1000, FlushInterval: 20 * time.Millisecond})
	defer w.Close(context.Background())

	w.Emit(testEvent(TypeUserLogin))
	w.Emit(testEvent(TypeEmailVerified))
	waitFor(t, "timer-triggered flush", func() bool { return store.total() == 2 })
}

func TestCloseDrains(t *testing.T) {
	store := &recordingStore{}
	w := NewWriter(store, Config{BatchSize: 1000, FlushInterval: time.Minute})

	const n = 10
	for range n {
		w.Emit(testEvent(TypeUserLogin))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := w.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := store.total(); got != n {
		t.Fatalf("store has %d events after Close, want %d", got, n)
	}
	if st := w.Stats(); st.Written != n {
		t.Fatalf("Written = %d, want %d", st.Written, n)
	}
}

func TestEmitAfterCloseDrops(t *testing.T) {
	store := &recordingStore{}
	w := NewWriter(store, Config{})
	if err := w.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	w.Emit(testEvent(TypeUserLogin))
	if st := w.Stats(); st.Dropped != 1 || st.Enqueued != 0 {
		t.Fatalf("after post-Close Emit: %+v, want 1 drop, 0 enqueued", st)
	}
}

func TestSamplerRate(t *testing.T) {
	store := &recordingStore{}
	w := NewWriter(store, Config{
		BufferSize: 20_000,
		BatchSize:  1000,
		// Default SampleRates: token.refresh at 0.1.
	})

	// One user refreshing all day: the first is kept (first-of-day), the
	// rest are sampled.
	const refreshes = 10_000
	for range refreshes {
		w.Emit(testEvent(TypeTokenRefresh))
	}
	st := w.Stats()
	if st.Dropped != 0 {
		t.Fatalf("unexpected drops: %d", st.Dropped)
	}
	// 1 + Binomial(9999, 0.1) has mean ~1001, sd ~30; ±300 is ten sigma.
	if st.Enqueued < 700 || st.Enqueued > 1300 {
		t.Fatalf("kept %d of %d token.refresh events, want ~1000", st.Enqueued, refreshes)
	}
	if st.Enqueued+st.SampledOut != refreshes {
		t.Fatalf("enqueued %d + sampled out %d != %d", st.Enqueued, st.SampledOut, refreshes)
	}

	// Unsampled types are always kept.
	before := st.Enqueued
	const logins = 500
	for range logins {
		w.Emit(testEvent(TypeUserLogin))
	}
	st = w.Stats()
	if st.Enqueued != before+logins {
		t.Fatalf("kept %d of %d user.login events, want all", st.Enqueued-before, logins)
	}

	if err := w.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// Every user's first refresh of the day bypasses sampling: the DAU
// aggregation counts distinct users, so a once-a-day refresher must always
// leave an event — random 10% sampling alone would hide them from DAU ~90%
// of their active days.
func TestSamplerKeepsFirstRefreshOfDayPerUser(t *testing.T) {
	store := &recordingStore{}
	w := NewWriter(store, Config{BufferSize: 8192, BatchSize: 8192})

	day := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	const users = 1000
	for i := range users {
		w.Emit(Event{Type: TypeTokenRefresh, ProjectID: "p1",
			UserID: fmt.Sprintf("u%d", i), CreatedAt: day})
	}
	if st := w.Stats(); st.Enqueued != users || st.SampledOut != 0 {
		t.Fatalf("first refreshes: %+v, want %d enqueued, 0 sampled out", st, users)
	}

	// The same users again the same day: back to plain sampling.
	for i := range users {
		w.Emit(Event{Type: TypeTokenRefresh, ProjectID: "p1",
			UserID: fmt.Sprintf("u%d", i), CreatedAt: day.Add(time.Hour)})
	}
	st := w.Stats()
	if st.SampledOut < users/2 {
		t.Fatalf("repeat refreshes not sampled: %+v", st)
	}

	// A new UTC day resets the first-of-day set.
	w.Emit(Event{Type: TypeTokenRefresh, ProjectID: "p1", UserID: "u0",
		CreatedAt: day.AddDate(0, 0, 1)})
	if got := w.Stats(); got.Enqueued != st.Enqueued+1 {
		t.Fatalf("next-day first refresh sampled out: %+v", got)
	}

	// Subjectless events never consult the set.
	w.Emit(Event{Type: TypeUserLoginFailed, ProjectID: "p1"})
	if got := w.Stats(); got.Enqueued != st.Enqueued+2 {
		t.Fatalf("unsampled subjectless event lost: %+v", got)
	}

	if err := w.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestConcurrentEmit(t *testing.T) {
	store := &recordingStore{}
	w := NewWriter(store, Config{
		BufferSize:    16_384,
		BatchSize:     64,
		FlushInterval: 5 * time.Millisecond,
		SampleRates:   map[string]float64{}, // keep everything
	})

	const goroutines, perGoroutine = 8, 500
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				w.Emit(testEvent(TypeUserLogin))
			}
		}()
	}
	wg.Wait()

	if err := w.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	st := w.Stats()
	const total = goroutines * perGoroutine
	if st.Enqueued+st.Dropped != total {
		t.Fatalf("enqueued %d + dropped %d != emitted %d", st.Enqueued, st.Dropped, total)
	}
	if got := store.total(); uint64(got) != st.Written || st.Written != st.Enqueued {
		t.Fatalf("store %d, written %d, enqueued %d: all should match", got, st.Written, st.Enqueued)
	}
}

func TestInsertFailuresCounted(t *testing.T) {
	w := NewWriter(failingStore{}, Config{BatchSize: 1000, FlushInterval: time.Minute})
	w.Emit(testEvent(TypeUserLogin))
	w.Emit(testEvent(TypeUserSignup))
	if err := w.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if st := w.Stats(); st.Failed != 2 || st.Written != 0 {
		t.Fatalf("stats = %+v, want 2 failed, 0 written", st)
	}
}

// warnCounter counts slog warnings by message; safe for concurrent use.
type warnCounter struct {
	mu   sync.Mutex
	msgs map[string]int
}

func (h *warnCounter) Enabled(context.Context, slog.Level) bool { return true }
func (h *warnCounter) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *warnCounter) WithGroup(string) slog.Handler            { return h }
func (h *warnCounter) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.msgs == nil {
		h.msgs = map[string]int{}
	}
	h.msgs[r.Message]++
	return nil
}

func (h *warnCounter) count(msg string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.msgs[msg]
}

// Lost events (insert failures, buffer-overflow drops) surface as a
// rate-limited log warning — the counters alone are invisible in
// production.
func TestLostEventsAreLogged(t *testing.T) {
	h := &warnCounter{}
	w := NewWriter(failingStore{}, Config{
		BatchSize:       1,
		FlushInterval:   5 * time.Millisecond,
		LostLogInterval: 5 * time.Millisecond,
		Logger:          slog.New(h),
	})
	w.Emit(testEvent(TypeUserLogin))
	waitFor(t, "lost-events warning", func() bool {
		return h.count("events: analytics events lost") >= 1
	})
	if err := w.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Nothing new lost since the last report → no further warning beyond
	// the flush-failure ones already emitted.
	n := h.count("events: analytics events lost")
	if n == 0 {
		t.Fatal("lost-events warning missing")
	}
}

func TestEmitStampsCreatedAt(t *testing.T) {
	store := &recordingStore{}
	w := NewWriter(store, Config{BatchSize: 1, FlushInterval: time.Minute})
	defer w.Close(context.Background())

	w.Emit(Event{Type: TypeUserLogin, ProjectID: "p1"})
	waitFor(t, "flush", func() bool { return store.total() == 1 })

	store.mu.Lock()
	e := store.batches[0][0]
	store.mu.Unlock()
	if e.CreatedAt.IsZero() {
		t.Fatal("CreatedAt not stamped")
	}
	if e.CreatedAt.Location() != time.UTC {
		t.Fatalf("CreatedAt in %v, want UTC", e.CreatedAt.Location())
	}
}
