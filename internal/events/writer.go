package events

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

// BatchInserter is the narrow store surface the writer needs. The store
// package implements it in milestone 07; tests use fakes. The events slice
// is owned by the callee — the writer never reuses or mutates it after the
// call.
type BatchInserter interface {
	InsertEvents(ctx context.Context, events []Event) error
}

// Config tunes a Writer. The zero value gets sensible defaults from
// NewWriter.
type Config struct {
	// BufferSize bounds the in-flight event channel. When it is full,
	// Emit drops the event and increments Stats.Dropped — auth latency
	// is never traded for analytics completeness. Default 4096.
	BufferSize int
	// BatchSize triggers a flush once this many events are buffered.
	// Default 128.
	BatchSize int
	// FlushInterval flushes partial batches so a quiet server still
	// lands events promptly. Default 2s.
	FlushInterval time.Duration
	// SampleRates maps an event type to the fraction of its events kept
	// at Emit time, e.g. {TypeTokenRefresh: 0.1}. Types absent from the
	// map are always kept. A user's first sampled-type event of the UTC
	// day is always kept regardless of the rate: the DAU aggregation
	// counts distinct users, so purely random sampling would hide a
	// once-a-day refresher from DAU ~90% of their active days — keeping
	// the daily first preserves per-user presence while the rate still
	// thins the volume. Default samples token.refresh at
	// DefaultTokenRefreshRate.
	SampleRates map[string]float64
	// Logger receives insert-failure warnings and the rate-limited
	// lost-events warning. Default slog.Default().
	Logger *slog.Logger
	// LostLogInterval rate-limits the warning emitted when events were
	// dropped (full buffer) or failed to insert since the last report, so
	// silent analytics loss is visible in production logs. Default 1m.
	LostLogInterval time.Duration
}

// DefaultTokenRefreshRate is the default sampling rate for token.refresh
// events: 1 in 10.
const DefaultTokenRefreshRate = 0.1

// Stats is a snapshot of the writer's atomic counters.
type Stats struct {
	// Enqueued counts events accepted into the buffer.
	Enqueued uint64
	// Written counts events successfully inserted.
	Written uint64
	// Dropped counts events lost to a full buffer (or emitted after
	// Close).
	Dropped uint64
	// SampledOut counts events discarded by the sampler.
	SampledOut uint64
	// Failed counts events lost because a batch insert errored.
	Failed uint64
}

// Writer batches analytics events into a BatchInserter from a background
// goroutine. Emit never blocks. Create with NewWriter, stop with Close.
type Writer struct {
	dst    BatchInserter
	cfg    Config
	log    *slog.Logger
	ch     chan Event
	quit   chan struct{} // closed by Close to start the drain
	done   chan struct{} // closed by run when fully drained
	stop   context.CancelFunc
	runCtx context.Context

	closeOnce sync.Once
	closed    atomic.Bool

	// First-of-day bookkeeping for sampled types: seen holds
	// "type|project|user" keys already kept on seenDay (UTC).
	seenMu  sync.Mutex
	seenDay string
	seen    map[string]struct{}

	// lastLost is the dropped+failed total already reported by the
	// lost-events warning; touched only by the run goroutine.
	lastLost uint64

	enqueued   atomic.Uint64
	written    atomic.Uint64
	dropped    atomic.Uint64
	sampledOut atomic.Uint64
	failed     atomic.Uint64
}

// NewWriter starts the background flusher and returns the writer.
func NewWriter(dst BatchInserter, cfg Config) *Writer {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 4096
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 128
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 2 * time.Second
	}
	if cfg.SampleRates == nil {
		cfg.SampleRates = map[string]float64{TypeTokenRefresh: DefaultTokenRefreshRate}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.LostLogInterval <= 0 {
		cfg.LostLogInterval = time.Minute
	}
	ctx, cancel := context.WithCancel(context.Background())
	w := &Writer{
		dst:    dst,
		cfg:    cfg,
		log:    cfg.Logger,
		ch:     make(chan Event, cfg.BufferSize),
		quit:   make(chan struct{}),
		done:   make(chan struct{}),
		stop:   cancel,
		runCtx: ctx,
	}
	go w.run()
	return w
}

// Emit queues e for writing and returns immediately. It never blocks: when
// the buffer is full (e.g. the store is stalled) the event is dropped and
// counted. A zero CreatedAt is stamped with the current time. Emit is safe
// for concurrent use.
func (w *Writer) Emit(e Event) {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	if rate, ok := w.cfg.SampleRates[e.Type]; ok && !w.firstOfDay(e) && rand.Float64() >= rate {
		w.sampledOut.Add(1)
		return
	}
	if w.closed.Load() {
		w.dropped.Add(1)
		return
	}
	select {
	case w.ch <- e:
		w.enqueued.Add(1)
	default:
		w.dropped.Add(1)
	}
}

// firstOfDay reports (and records) whether e is its user's first event of
// this type on e.CreatedAt's UTC day. Such events bypass sampling so every
// active user leaves at least one refresh event per day for the DAU
// aggregation; the set resets at each UTC midnight and holds one entry per
// (type, project, active user), i.e. it is DAU-sized.
func (w *Writer) firstOfDay(e Event) bool {
	if e.UserID == "" {
		return false
	}
	day := e.CreatedAt.UTC().Format("2006-01-02")
	key := e.Type + "|" + e.ProjectID + "|" + e.UserID
	w.seenMu.Lock()
	defer w.seenMu.Unlock()
	if w.seenDay != day {
		w.seenDay = day
		w.seen = make(map[string]struct{})
	}
	if _, ok := w.seen[key]; ok {
		return false
	}
	w.seen[key] = struct{}{}
	return true
}

// Stats returns a snapshot of the counters.
func (w *Writer) Stats() Stats {
	return Stats{
		Enqueued:   w.enqueued.Load(),
		Written:    w.written.Load(),
		Dropped:    w.dropped.Load(),
		SampledOut: w.sampledOut.Load(),
		Failed:     w.failed.Load(),
	}
}

// Close stops the writer, draining buffered events until ctx expires.
// After the deadline any in-flight insert is cancelled and remaining
// events are abandoned; ctx.Err() is returned. Emit after Close drops.
// Close is idempotent and safe to call concurrently.
func (w *Writer) Close(ctx context.Context) error {
	w.closeOnce.Do(func() {
		w.closed.Store(true)
		close(w.quit)
	})
	select {
	case <-w.done:
		w.stop()
		return nil
	case <-ctx.Done():
		w.stop()
		return ctx.Err()
	}
}

// run is the background loop: it flushes on batch size, on the ticker, and
// drains everything on Close. It also periodically surfaces newly lost
// events — a counter nobody can read is not observability.
func (w *Writer) run() {
	defer close(w.done)
	ticker := time.NewTicker(w.cfg.FlushInterval)
	defer ticker.Stop()
	lostTicker := time.NewTicker(w.cfg.LostLogInterval)
	defer lostTicker.Stop()

	batch := make([]Event, 0, w.cfg.BatchSize)
	for {
		select {
		case e := <-w.ch:
			batch = append(batch, e)
			if len(batch) >= w.cfg.BatchSize {
				w.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				w.flush(batch)
				batch = batch[:0]
			}
		case <-lostTicker.C:
			w.logLost()
		case <-w.quit:
			w.drain(batch)
			w.logLost()
			return
		}
	}
}

// logLost warns once per LostLogInterval when events were dropped (full
// buffer) or failed to insert since the last report, so buffer-overflow
// drops during a store stall are visible in production logs, not only via
// Stats.
func (w *Writer) logLost() {
	lost := w.dropped.Load() + w.failed.Load()
	if lost == w.lastLost {
		return
	}
	st := w.Stats()
	w.log.Warn("events: analytics events lost",
		"new", lost-w.lastLost, "dropped", st.Dropped, "failed", st.Failed,
		"written", st.Written)
	w.lastLost = lost
}

// drain empties whatever is already buffered, then flushes the remainder.
// Events emitted concurrently with the drain may be dropped — Close means
// shutdown, not a fairness point.
func (w *Writer) drain(batch []Event) {
	for {
		select {
		case e := <-w.ch:
			batch = append(batch, e)
			if len(batch) >= w.cfg.BatchSize {
				w.flush(batch)
				batch = batch[:0]
			}
		default:
			if len(batch) > 0 {
				w.flush(batch)
			}
			return
		}
	}
}

// flush hands one batch to the store. The slice is cloned so the store may
// retain it; failures are logged and counted, never retried — analytics
// are best-effort by design.
func (w *Writer) flush(batch []Event) {
	events := make([]Event, len(batch))
	copy(events, batch)
	if err := w.dst.InsertEvents(w.runCtx, events); err != nil {
		w.failed.Add(uint64(len(events)))
		w.log.Warn("events: batch insert failed", "count", len(events), "error", err)
		return
	}
	w.written.Add(uint64(len(events)))
}
