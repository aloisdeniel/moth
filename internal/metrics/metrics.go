// Package metrics is moth's lightweight instrumentation: a small in-process
// registry of counters and latency histograms and an http.Handler that
// renders them in Prometheus text exposition format (version 0.0.4).
//
// The exposition is hand-rolled rather than pulling in the Prometheus client
// library — moth exports a handful of fixed metric families, so the ~150
// lines here avoid a heavy transitive dependency tree. The format is the
// stable, widely-parsed text protocol; any Prometheus/OpenMetrics scraper
// reads it.
package metrics

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// defaultBuckets are the upper bounds (seconds) of the RPC-latency
// histogram, matching the Prometheus client defaults.
var defaultBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

// Registry holds every metric moth exports. It is safe for concurrent use.
// Construct it with New; the app increments the counters and the connect
// interceptor (Interceptor) records the RPC family.
type Registry struct {
	rpcRequests  *counterVec
	rpcDuration  *histogramVec
	authAttempts *counterVec
	eventDrops   *counterVec
	rollupRuns   *counterVec

	families []family
}

type family interface{ write(w io.Writer) }

// New builds a registry with moth's fixed metric families initialised.
func New() *Registry {
	r := &Registry{
		rpcRequests:  newCounterVec("moth_rpc_requests_total", "Total RPCs handled, by procedure and result code.", "procedure", "code"),
		rpcDuration:  newHistogramVec("moth_rpc_duration_seconds", "RPC handling latency in seconds, by procedure.", defaultBuckets, "procedure"),
		authAttempts: newCounterVec("moth_auth_attempts_total", "Authentication attempts, by result.", "result"),
		eventDrops:   newCounterVec("moth_event_buffer_drops_total", "Analytics events dropped because the async writer buffer was full."),
		rollupRuns:   newCounterVec("moth_rollup_runs_total", "Analytics rollup job runs, by status.", "status"),
	}
	r.families = []family{r.rpcRequests, r.rpcDuration, r.authAttempts, r.eventDrops, r.rollupRuns}
	return r
}

// ObserveRPC records one completed RPC: it bumps the request counter for
// (procedure, code) and observes the latency in the duration histogram.
func (r *Registry) ObserveRPC(procedure, code string, seconds float64) {
	r.rpcRequests.add(1, procedure, code)
	r.rpcDuration.observe(seconds, procedure)
}

// IncAuthAttempt records an authentication attempt. result is a low
// cardinality label such as "success" or "failure".
func (r *Registry) IncAuthAttempt(result string) { r.authAttempts.add(1, result) }

// AddEventBufferDrops records n analytics events dropped by a full buffer.
func (r *Registry) AddEventBufferDrops(n int) {
	if n > 0 {
		r.eventDrops.add(uint64(n))
	}
}

// IncRollupRun records one analytics rollup run. status is "success" or
// "error".
func (r *Registry) IncRollupRun(status string) { r.rollupRuns.add(1, status) }

// Render writes the full Prometheus text exposition to w.
func (r *Registry) Render(w io.Writer) {
	for _, f := range r.families {
		f.write(w)
	}
}

// Handler serves the exposition at, conventionally, GET /metrics.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		r.Render(w)
	})
}

// --- counter vector ---

type counterVec struct {
	name, help string
	labelNames []string
	mu         sync.Mutex
	samples    map[string]*counterSample
}

type counterSample struct {
	labelValues []string
	value       uint64
}

func newCounterVec(name, help string, labelNames ...string) *counterVec {
	return &counterVec{name: name, help: help, labelNames: labelNames, samples: map[string]*counterSample{}}
}

func (c *counterVec) add(delta uint64, labelValues ...string) {
	key := strings.Join(labelValues, "\xff")
	c.mu.Lock()
	s := c.samples[key]
	if s == nil {
		s = &counterSample{labelValues: labelValues}
		c.samples[key] = s
	}
	s.value += delta
	c.mu.Unlock()
}

func (c *counterVec) write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "# HELP %s %s\n", c.name, c.help)
	_, _ = fmt.Fprintf(w, "# TYPE %s counter\n", c.name)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range sortedKeys(c.samples) {
		s := c.samples[key]
		_, _ = fmt.Fprintf(w, "%s%s %d\n", c.name, formatLabels(c.labelNames, s.labelValues), s.value)
	}
}

// --- histogram vector ---

type histogramVec struct {
	name, help string
	labelNames []string
	buckets    []float64
	mu         sync.Mutex
	samples    map[string]*histogramSample
}

type histogramSample struct {
	labelValues []string
	counts      []uint64 // cumulative per bucket (index i counts v <= buckets[i])
	sum         float64
	count       uint64
}

func newHistogramVec(name, help string, buckets []float64, labelNames ...string) *histogramVec {
	return &histogramVec{name: name, help: help, buckets: buckets, labelNames: labelNames, samples: map[string]*histogramSample{}}
}

func (h *histogramVec) observe(v float64, labelValues ...string) {
	key := strings.Join(labelValues, "\xff")
	h.mu.Lock()
	s := h.samples[key]
	if s == nil {
		s = &histogramSample{labelValues: labelValues, counts: make([]uint64, len(h.buckets))}
		h.samples[key] = s
	}
	for i, b := range h.buckets {
		if v <= b {
			s.counts[i]++
		}
	}
	s.sum += v
	s.count++
	h.mu.Unlock()
}

func (h *histogramVec) write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "# HELP %s %s\n", h.name, h.help)
	_, _ = fmt.Fprintf(w, "# TYPE %s histogram\n", h.name)
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, key := range sortedKeys(h.samples) {
		s := h.samples[key]
		for i, b := range h.buckets {
			le := strconv.FormatFloat(b, 'g', -1, 64)
			_, _ = fmt.Fprintf(w, "%s_bucket%s %d\n", h.name, formatLabels(append(h.labelNames, "le"), append(cloneStrings(s.labelValues), le)), s.counts[i])
		}
		_, _ = fmt.Fprintf(w, "%s_bucket%s %d\n", h.name, formatLabels(append(h.labelNames, "le"), append(cloneStrings(s.labelValues), "+Inf")), s.count)
		_, _ = fmt.Fprintf(w, "%s_sum%s %s\n", h.name, formatLabels(h.labelNames, s.labelValues), strconv.FormatFloat(s.sum, 'g', -1, 64))
		_, _ = fmt.Fprintf(w, "%s_count%s %d\n", h.name, formatLabels(h.labelNames, s.labelValues), s.count)
	}
}

// --- helpers ---

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func cloneStrings(s []string) []string {
	out := make([]string, len(s))
	copy(out, s)
	return out
}

// formatLabels renders {name="value",...}, or "" when there are no labels.
func formatLabels(names, values []string) string {
	if len(names) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('{')
	for i, n := range names {
		if i > 0 {
			b.WriteByte(',')
		}
		v := ""
		if i < len(values) {
			v = values[i]
		}
		b.WriteString(n)
		b.WriteString(`="`)
		b.WriteString(escapeLabelValue(v))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}

func escapeLabelValue(v string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return r.Replace(v)
}
