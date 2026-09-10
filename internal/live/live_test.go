package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func newTestStore(t *testing.T, names ...string) *Store {
	t.Helper()
	var sources []Source
	for _, name := range names {
		sources = append(sources, Source{Name: name, Interval: time.Second, Timeout: time.Second, Collect: func(context.Context) (*inventory.Snapshot, error) { return nil, nil }})
	}
	s, err := New(sources)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func readView(t *testing.T, s *Store) response {
	t.Helper()
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/snapshot", nil))
	var out response
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func podSnapshot(count int, status string) *inventory.Snapshot {
	result := &inventory.Snapshot{CollectedAt: time.Now().UTC(), Coverage: []inventory.CoverageItem{{Area: "kubernetes", Resource: "Pods", Status: status, ObjectCount: count}}}
	if count > 0 {
		result.Kubernetes.Pods = []inventory.Pod{{ObjectRef: inventory.ObjectRef{Name: "web", Namespace: "app"}, Phase: "Running"}}
	}
	return result
}
func TestRetainOnFailureAndCoverageRegressionThenAcceptDeletion(t *testing.T) {
	s := newTestStore(t, "kubernetes")
	if got := readView(t, s); got.Snapshot != nil || got.Sources["kubernetes"].State != "loading" {
		t.Fatalf("initial: %+v", got)
	}
	s.begin("kubernetes")
	s.finish("kubernetes", nil, errors.New("offline"), time.Now())
	if got := readView(t, s); got.Snapshot != nil || got.Sources["kubernetes"].State != "error" {
		t.Fatalf("first failure: %+v", got)
	}
	first := podSnapshot(1, "complete")
	s.finish("kubernetes", first, nil, time.Now())
	first.Kubernetes.Pods[0].Name = "mutated"
	original := readView(t, s)
	if original.Snapshot.Kubernetes.Pods[0].Name != "web" {
		t.Fatal("publication aliases collector data")
	}
	for _, result := range []struct {
		snapshot *inventory.Snapshot
		err      error
	}{
		{nil, errors.New("offline")}, {nil, nil}, {podSnapshot(0, "denied"), nil},
		{podSnapshot(0, "unavailable"), nil}, {&inventory.Snapshot{}, nil},
	} {
		s.finish("kubernetes", result.snapshot, result.err, time.Now())
		got := readView(t, s)
		if got.Sources["kubernetes"].State != "stale" || got.Revision != original.Revision || len(got.Snapshot.Kubernetes.Pods) != 1 {
			t.Fatalf("failed refresh replaced data: %+v", got)
		}
		if !got.Sources["kubernetes"].LastSuccess.Equal(*original.Sources["kubernetes"].LastSuccess) {
			t.Fatal("failed refresh advanced success time")
		}
	}
	s.finish("kubernetes", podSnapshot(0, "complete"), nil, time.Now())
	got := readView(t, s)
	if len(got.Snapshot.Kubernetes.Pods) != 0 || got.Sources["kubernetes"].State != "ready" || got.Revision != original.Revision+1 {
		t.Fatalf("successful deletion: %+v", got)
	}
}
func TestPartialInitialDataAndIndependentSources(t *testing.T) {
	s := newTestStore(t, "kubernetes", "eks")
	partial := podSnapshot(1, "complete")
	partial.Coverage = append(partial.Coverage, inventory.CoverageItem{Area: "kubernetes", Resource: "Nodes", Status: "denied"})
	s.finish("kubernetes", partial, nil, time.Now())
	if readView(t, s).Sources["kubernetes"].State != "partial" {
		t.Fatal("missing partial state")
	}
	eks := &inventory.Snapshot{CollectedAt: time.Now().UTC(), EKS: inventory.EKSInventory{Cluster: inventory.Cluster{Name: "prod"}}}
	s.finish("eks", eks, nil, time.Now())
	s.finish("eks", nil, errors.New("AWS unavailable"), time.Now())
	s.finish("kubernetes", podSnapshot(0, "complete"), nil, time.Now())
	got := readView(t, s)
	if len(got.Snapshot.Kubernetes.Pods) != 0 || got.Snapshot.EKS.Cluster.Name != "prod" || got.Sources["eks"].State != "stale" {
		t.Fatalf("sources coupled: %+v", got)
	}
}
func TestHTTPReadOnlyAndConditionalRequests(t *testing.T) {
	s := newTestStore(t, "kubernetes")
	h := s.Handler()
	for _, item := range []struct {
		method, path string
		code         int
	}{{"GET", "/", 200}, {"HEAD", "/", 200}, {"GET", "/missing", 404}, {"POST", "/api/snapshot", 405}, {"HEAD", "/api/snapshot", 200}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(item.method, item.path, nil))
		if w.Code != item.code {
			t.Fatalf("%s %s: %d", item.method, item.path, w.Code)
		}
		if item.method == "HEAD" && w.Body.Len() != 0 {
			t.Fatal("HEAD returned body")
		}
		if item.path == "/" && item.method == "GET" && (!strings.Contains(w.Body.String(), "refreshLivePage") || strings.Contains(w.Body.String(), "__TELESKOPE")) {
			t.Fatal("live page is not embedded")
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/snapshot", nil))
	req := httptest.NewRequest("GET", "/api/snapshot", nil)
	req.Header.Set("If-None-Match", w.Header().Get("ETag"))
	cached := httptest.NewRecorder()
	h.ServeHTTP(cached, req)
	if cached.Code != 304 || cached.Body.Len() != 0 {
		t.Fatal("unchanged view not conditional")
	}
	s.begin("kubernetes")
	changed := httptest.NewRecorder()
	h.ServeHTTP(changed, req)
	if changed.Code != 200 {
		t.Fatal("status change did not invalidate ETag")
	}
	eventReq := httptest.NewRequest("GET", "/api/snapshot", nil)
	eventReq.Header.Set("If-None-Match", changed.Header().Get("ETag"))
	s.AddEvent("kubernetes", "info", "refresh starting")
	eventChanged := httptest.NewRecorder()
	h.ServeHTTP(eventChanged, eventReq)
	if eventChanged.Code != 200 {
		t.Fatal("event change did not invalidate ETag")
	}
	var eventView response
	if err := json.Unmarshal(eventChanged.Body.Bytes(), &eventView); err != nil {
		t.Fatal(err)
	}
	if len(eventView.Events) != 1 || eventView.Events[0].Source != "kubernetes" || eventView.Events[0].Message != "refresh starting" {
		t.Fatalf("events = %#v, want recent event in response", eventView.Events)
	}
}

func TestHTTPExportUsesCurrentSnapshot(t *testing.T) {
	s := newTestStore(t, "kubernetes")
	h := s.Handler()

	missing := httptest.NewRecorder()
	h.ServeHTTP(missing, httptest.NewRequest("GET", "/api/export/summary.md", nil))
	if missing.Code != 503 {
		t.Fatalf("export without snapshot = %d, want 503", missing.Code)
	}

	s.finish("kubernetes", podSnapshot(1, "complete"), nil, time.Now())
	for _, item := range []struct {
		path        string
		contentType string
		want        string
	}{
		{"/api/export/snapshot.json", "application/json", `"schemaVersion": "teleskope.io/snapshot/v1alpha1"`},
		{"/api/export/summary.md", "text/markdown; charset=utf-8", "# Teleskope scan summary"},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", item.path, nil))
		if w.Code != 200 {
			t.Fatalf("%s = %d: %s", item.path, w.Code, w.Body.String())
		}
		if got := w.Header().Get("Content-Type"); got != item.contentType {
			t.Fatalf("%s content type = %q, want %q", item.path, got, item.contentType)
		}
		if !strings.Contains(w.Body.String(), item.want) {
			t.Fatalf("%s missing %q:\n%s", item.path, item.want, w.Body.String())
		}
	}
}

func TestEventBufferKeepsRecentEvents(t *testing.T) {
	s := newTestStore(t, "kubernetes")

	for i := range maxEvents + 5 {
		s.AddEvent("kubernetes", "debug", "event %03d", i)
	}

	got := readView(t, s)
	if len(got.Events) != maxEvents {
		t.Fatalf("events=%d, want %d", len(got.Events), maxEvents)
	}
	if got.Events[0].Message != "event 005" || got.Events[len(got.Events)-1].Message != "event 204" {
		t.Fatalf("events range = %q..%q, want recent bounded events", got.Events[0].Message, got.Events[len(got.Events)-1].Message)
	}
}

func TestPollingSharedNonOverlappingAndCancellation(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	sources := []Source{{Name: "kubernetes", Interval: time.Millisecond, Timeout: time.Second, Collect: func(ctx context.Context) (*inventory.Snapshot, error) {
		n := calls.Add(1)
		started <- struct{}{}
		if n == 1 {
			select {
			case <-release:
				return podSnapshot(1, "complete"), nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}}}
	s, err := New(sources)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("no immediate collection")
	}
	h := s.Handler()
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 10 {
				h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/snapshot", nil))
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal("HTTP reads or interval started overlapping scans")
	}
	close(release)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("no subsequent collection")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancellation did not stop worker")
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestPollingLogsRefreshLifecycle(t *testing.T) {
	logs := make(chan string, 4)
	source := Source{
		Name:     "kubernetes",
		Interval: time.Hour,
		Timeout:  time.Second,
		Collect: func(context.Context) (*inventory.Snapshot, error) {
			return podSnapshot(1, "complete"), nil
		},
		Log: func(format string, args ...any) {
			logs <- fmt.Sprintf(format, args...)
		},
	}
	s, err := New([]Source{source})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("cancellation did not stop worker")
		}
	}()

	var got []string
	for len(got) < 2 {
		select {
		case line := <-logs:
			got = append(got, line)
		case <-time.After(time.Second):
			t.Fatalf("missing lifecycle logs: %#v", got)
		}
	}
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "[kubernetes] refresh starting") || !strings.Contains(joined, "[kubernetes] refresh published state=ready") {
		t.Fatalf("logs = %#v, want refresh start and publish", got)
	}
}

func TestAttemptTimeoutAndRetry(t *testing.T) {
	var calls atomic.Int32
	recovered := make(chan struct{})
	source := Source{Name: "kubernetes", Interval: time.Millisecond, Timeout: 10 * time.Millisecond, Collect: func(ctx context.Context) (*inventory.Snapshot, error) {
		if calls.Add(1) == 1 {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		select {
		case <-recovered:
		default:
			close(recovered)
		}
		return podSnapshot(0, "complete"), nil
	}}
	s, err := New([]Source{source})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	select {
	case <-recovered:
	case <-time.After(time.Second):
		t.Fatal("timeout did not retry")
	}
	cancel()
	<-done
}

func TestAdvisorFreshnessChangesWithoutDataRevision(t *testing.T) {
	s := newTestStore(t, "kubernetes")
	s.finish("kubernetes", podSnapshot(1, "complete"), nil, time.Now())
	before := readView(t, s)
	if before.Advisor == nil {
		t.Fatal("missing advisor")
	}
	s.finish("kubernetes", nil, errors.New("offline"), time.Now())
	after := readView(t, s)
	if after.Revision != before.Revision {
		t.Fatal("failure changed data revision")
	}
	for _, c := range after.Advisor.Capabilities {
		if c.Freshness != "stale" {
			t.Fatalf("stale capability labeled %s", c.Freshness)
		}
	}
}
