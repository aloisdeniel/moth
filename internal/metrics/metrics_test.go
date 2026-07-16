package metrics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
)

func exposition(r *Registry) string {
	var b strings.Builder
	r.Render(&b)
	return b.String()
}

// metricLine finds a sample line by exact "name{labels}" prefix and returns
// its value token.
func metricValue(t *testing.T, text, prefix string) string {
	t.Helper()
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix+" ") {
			return strings.TrimPrefix(line, prefix+" ")
		}
	}
	t.Fatalf("no line with prefix %q in:\n%s", prefix, text)
	return ""
}

func TestCountersExposition(t *testing.T) {
	r := New()
	r.IncAuthAttempt("success")
	r.IncAuthAttempt("success")
	r.IncAuthAttempt("failure")
	r.AddEventBufferDrops(3)
	r.AddEventBufferDrops(0) // no-op
	r.IncRollupRun("success")

	text := exposition(r)

	if got := metricValue(t, text, `moth_auth_attempts_total{result="success"}`); got != "2" {
		t.Fatalf("success count = %q, want 2", got)
	}
	if got := metricValue(t, text, `moth_auth_attempts_total{result="failure"}`); got != "1" {
		t.Fatalf("failure count = %q, want 1", got)
	}
	if got := metricValue(t, text, `moth_event_buffer_drops_total`); got != "3" {
		t.Fatalf("drops = %q, want 3", got)
	}
	if got := metricValue(t, text, `moth_rollup_runs_total{status="success"}`); got != "1" {
		t.Fatalf("rollup = %q, want 1", got)
	}
	// HELP/TYPE metadata present.
	if !strings.Contains(text, "# TYPE moth_auth_attempts_total counter") {
		t.Fatal("missing TYPE line")
	}
}

func TestHistogramExposition(t *testing.T) {
	r := New()
	r.ObserveRPC("/svc/Method", "ok", 0.03)
	r.ObserveRPC("/svc/Method", "ok", 0.2)

	text := exposition(r)

	if !strings.Contains(text, "# TYPE moth_rpc_duration_seconds histogram") {
		t.Fatal("missing histogram TYPE")
	}
	// 0.03 and 0.2 are both <= 0.25 and <= +Inf; only 0.2 exceeds 0.05.
	if got := metricValue(t, text, `moth_rpc_duration_seconds_bucket{procedure="/svc/Method",le="0.05"}`); got != "1" {
		t.Fatalf("le=0.05 bucket = %q, want 1", got)
	}
	if got := metricValue(t, text, `moth_rpc_duration_seconds_bucket{procedure="/svc/Method",le="0.25"}`); got != "2" {
		t.Fatalf("le=0.25 bucket = %q, want 2", got)
	}
	if got := metricValue(t, text, `moth_rpc_duration_seconds_bucket{procedure="/svc/Method",le="+Inf"}`); got != "2" {
		t.Fatalf("le=+Inf bucket = %q, want 2", got)
	}
	if got := metricValue(t, text, `moth_rpc_duration_seconds_count{procedure="/svc/Method"}`); got != "2" {
		t.Fatalf("count = %q, want 2", got)
	}
	if got := metricValue(t, text, `moth_rpc_requests_total{procedure="/svc/Method",code="ok"}`); got != "2" {
		t.Fatalf("request total = %q, want 2", got)
	}
}

func TestInterceptorRecords(t *testing.T) {
	r := New()
	const proc = "/moth.test.v1.Svc/Do"
	// A real connect roundtrip: connect.AnyRequest cannot be faked (it has an
	// unexported method), so drive the interceptor through an actual handler.
	handler := connect.NewUnaryHandler(proc,
		func(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("nope"))
		},
		connect.WithInterceptors(r.Interceptor()),
	)
	mux := http.NewServeMux()
	mux.Handle(proc, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := connect.NewClient[emptypb.Empty, emptypb.Empty](srv.Client(), srv.URL+proc)
	_, err := client.CallUnary(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	if err == nil {
		t.Fatal("expected error to propagate")
	}
	text := exposition(r)
	if got := metricValue(t, text, `moth_rpc_requests_total{procedure="`+proc+`",code="unauthenticated"}`); got != "1" {
		t.Fatalf("interceptor did not record error code: %q\n%s", got, text)
	}
}

func TestHandler(t *testing.T) {
	r := New()
	r.IncAuthAttempt("success")
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "moth_auth_attempts_total") {
		t.Fatal("handler body missing metric")
	}
}

func TestLabelEscaping(t *testing.T) {
	r := New()
	r.ObserveRPC(`/svc/"weird"`, "ok", 0.01)
	text := exposition(r)
	if !strings.Contains(text, `procedure="/svc/\"weird\""`) {
		t.Fatalf("label value not escaped:\n%s", text)
	}
}
