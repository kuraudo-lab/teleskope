package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/analysis"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"github.com/kuraudo-lab/teleskope/internal/live"
)

type fakeAnalyzer struct {
	calls int
	got   analysis.Request
}

func (f *fakeAnalyzer) Analyze(ctx context.Context, req analysis.Request) (analysis.Result, error) {
	f.calls++
	f.got = req
	return analysis.Result{
		UseCase:       req.UseCase,
		Model:         "fake-model",
		PromptVersion: "test",
		Summary:       "hub cluster analysis",
		Sections:      []analysis.Section{{Title: "Inventory", Items: []analysis.Item{{Summary: "web workload", Severity: "info"}}}},
	}, nil
}

type hubBlockingAnalyzer struct {
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
}

func (b *hubBlockingAnalyzer) Analyze(ctx context.Context, req analysis.Request) (analysis.Result, error) {
	b.calls.Add(1)
	b.started <- struct{}{}
	select {
	case <-b.release:
		return analysis.Result{UseCase: req.UseCase, Model: "blocking", Summary: "done"}, nil
	case <-ctx.Done():
		return analysis.Result{}, ctx.Err()
	}
}

func TestDeriveClusterUsesEKSARNBeforeKubeContext(t *testing.T) {
	snapshot := &inventory.Snapshot{
		AWS: inventory.AWSIdentity{Region: "ap-northeast-1"},
		EKS: inventory.EKSInventory{Cluster: inventory.Cluster{
			Name: "prod",
			ARN:  "arn:aws:eks:ap-northeast-1:123456789012:cluster/prod",
		}},
		Kubernetes: inventory.Kubernetes{Context: "local-context", Server: "https://example.invalid"},
	}

	got := DeriveCluster(snapshot, "", "", "")

	if got.ID != snapshot.EKS.Cluster.ARN || got.Name != "prod" || got.Provider != "eks" || got.Region != "ap-northeast-1" {
		t.Fatalf("cluster = %+v", got)
	}
}

func TestDeriveClusterFallsBackToStableKubernetesServerHash(t *testing.T) {
	snapshot := &inventory.Snapshot{Kubernetes: inventory.Kubernetes{Context: "prod", Server: "https://10.0.0.1"}}

	first := DeriveCluster(snapshot, "", "", "")
	second := DeriveCluster(snapshot, "", "", "")

	if first.ID == "" || first.ID != second.ID || first.ID == "prod" {
		t.Fatalf("fallback IDs = %q and %q", first.ID, second.ID)
	}
	if first.Name != "prod" || first.Provider != "kubernetes" {
		t.Fatalf("cluster = %+v", first)
	}
}

func TestStoreAcceptsOnlyNewerClusterRevisions(t *testing.T) {
	store := &Store{}
	env := testEnvelope("prod-a", 2)

	accepted, stored, err := store.Put(env)
	if err != nil {
		t.Fatal(err)
	}
	if !accepted || stored.Revision != 2 {
		t.Fatalf("first put accepted=%v stored=%+v", accepted, stored)
	}

	accepted, stored, err = store.Put(testEnvelope("prod-a", 1))
	if err != nil {
		t.Fatal(err)
	}
	if accepted || stored.Revision != 2 {
		t.Fatalf("older put accepted=%v stored=%+v", accepted, stored)
	}
	if fleet := store.Fleet(); fleet.Revision != 1 || len(fleet.Clusters) != 1 {
		t.Fatalf("fleet = %+v", fleet)
	}
}

func TestFleetIncludesClusterEventsAndMarkdown(t *testing.T) {
	store := &Store{}
	if accepted, _, err := store.Put(testEnvelope("prod-a", 1)); err != nil || !accepted {
		t.Fatalf("put accepted=%v err=%v", accepted, err)
	}

	fleet := store.Fleet()
	if len(fleet.Events) != 1 || fleet.Clusters[0].Events != 1 {
		t.Fatalf("fleet events = %+v", fleet)
	}
	markdown := Markdown(fleet)
	for _, want := range []string{"# Teleskope fleet summary", "## Recent events", "refresh published"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown missing %q:\n%s", want, markdown)
		}
	}
}

func TestFleetCarriesWatchFreshness(t *testing.T) {
	store := &Store{}
	env := testEnvelope("prod-a", 1)
	eventAt := env.CollectedAt.Add(2 * time.Minute)
	fullSyncAt := env.CollectedAt
	env.Sources["kubernetes"] = live.Status{
		State:          "stale",
		Mode:           live.SourceModeWatch,
		LastEventAt:    &eventAt,
		LastFullSyncAt: &fullSyncAt,
		Reconnects:     3,
		Error:          "watch disconnected",
	}
	if accepted, _, err := store.Put(env); err != nil || !accepted {
		t.Fatalf("put accepted=%v err=%v", accepted, err)
	}

	fleet := store.Fleet()
	source := fleet.Clusters[0].Sources["kubernetes"]
	if fleet.Clusters[0].State != "stale" || source.Mode != live.SourceModeWatch || source.Reconnects != 3 {
		t.Fatalf("fleet source freshness = %+v", fleet.Clusters[0])
	}
	if source.LastEventAt == nil || !source.LastEventAt.Equal(eventAt) {
		t.Fatalf("last event = %v, want %v", source.LastEventAt, eventAt)
	}
	if source.LastFullSyncAt == nil || !source.LastFullSyncAt.Equal(fullSyncAt) {
		t.Fatalf("last full sync = %v, want %v", source.LastFullSyncAt, fullSyncAt)
	}

	body, _ := store.encodedFleet()
	for _, want := range []string{`"mode":"watch"`, `"lastEventAt"`, `"lastFullSyncAt"`, `"reconnects":3`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("encoded fleet missing %q: %s", want, body)
		}
	}
}

func TestHubHTMLUsesEmbeddedTeleskopeIcon(t *testing.T) {
	html := HubHTML()
	for _, want := range []string{`<img class="logo"`, `src="data:image/png;base64,`, `multi-cluster hub`, "watch reconnecting", "full resync", "reconnects", "eventKind(event)"} {
		if !strings.Contains(html, want) {
			t.Fatalf("hub html missing %q", want)
		}
	}
	if strings.Contains(html, "__TELESKOPE_ICON__") {
		t.Fatal("hub html still contains icon placeholder")
	}
}

func TestHTTPPublishesFleetAndDrilldown(t *testing.T) {
	store := &Store{}
	handler := store.Handler(HandlerOptions{Token: "secret"})

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/api/clusters", strings.NewReader(`{}`)))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized = %d", unauthorized.Code)
	}

	body, _ := json.Marshal(testEnvelope("prod-a", 1))
	req := httptest.NewRequest(http.MethodPost, "/api/clusters", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer secret")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, req)
	if created.Code != http.StatusCreated {
		t.Fatalf("post = %d: %s", created.Code, created.Body.String())
	}

	fleet := httptest.NewRecorder()
	handler.ServeHTTP(fleet, httptest.NewRequest(http.MethodGet, "/api/clusters", nil))
	if fleet.Code != http.StatusOK || !strings.Contains(fleet.Body.String(), `"clusters"`) {
		t.Fatalf("fleet = %d: %s", fleet.Code, fleet.Body.String())
	}

	drilldown := httptest.NewRecorder()
	handler.ServeHTTP(drilldown, httptest.NewRequest(http.MethodGet, "/cluster?id=prod-a", nil))
	if drilldown.Code != http.StatusOK ||
		!strings.Contains(drilldown.Body.String(), "Analyze with AI") ||
		!strings.Contains(drilldown.Body.String(), `<script id="boot-config" type="application/json">`) ||
		!strings.Contains(drilldown.Body.String(), `"/api/cluster/snapshot?id=prod-a"`) ||
		!strings.Contains(drilldown.Body.String(), `"/api/cluster/analyze?id=prod-a"`) {
		t.Fatalf("drilldown = %d: %s", drilldown.Code, drilldown.Body.String())
	}
}

func TestHTTPRemoteWriteLogsOperationalEvents(t *testing.T) {
	store := &Store{}
	var logs []string
	handler := store.Handler(HandlerOptions{
		Token: "secret",
		Log: func(format string, args ...any) {
			logs = append(logs, formatLog(format, args...))
		},
	})

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/api/clusters", strings.NewReader(`{}`)))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized = %d", unauthorized.Code)
	}

	body, _ := json.Marshal(testEnvelope("prod-a", 2))
	req := httptest.NewRequest(http.MethodPost, "/api/clusters", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer secret")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, req)
	if created.Code != http.StatusCreated {
		t.Fatalf("post = %d: %s", created.Code, created.Body.String())
	}

	staleBody, _ := json.Marshal(testEnvelope("prod-a", 1))
	staleReq := httptest.NewRequest(http.MethodPost, "/api/clusters", strings.NewReader(string(staleBody)))
	staleReq.Header.Set("Authorization", "Bearer secret")
	stale := httptest.NewRecorder()
	handler.ServeHTTP(stale, staleReq)
	if stale.Code != http.StatusOK {
		t.Fatalf("stale = %d: %s", stale.Code, stale.Body.String())
	}

	for _, want := range []string{
		"remote write unauthorized",
		"remote write accepted cluster=prod-a",
		"remote write ignored cluster=prod-a revision=1 stored_revision=2",
	} {
		if !containsLog(logs, want) {
			t.Fatalf("logs missing %q in %#v", want, logs)
		}
	}
}

func TestHTTPClusterScopedSnapshotAndAnalyze(t *testing.T) {
	store := &Store{}
	if accepted, _, err := store.Put(testEnvelope("prod-a", 3)); err != nil || !accepted {
		t.Fatalf("put accepted=%v err=%v", accepted, err)
	}
	analyzer := &fakeAnalyzer{}
	handler := store.Handler(HandlerOptions{Analyzer: analyzer})

	snapshot := httptest.NewRecorder()
	handler.ServeHTTP(snapshot, httptest.NewRequest(http.MethodGet, "/api/cluster/snapshot?id=prod-a", nil))
	if snapshot.Code != http.StatusOK || !strings.Contains(snapshot.Body.String(), `"revision":3`) {
		t.Fatalf("snapshot = %d: %s", snapshot.Code, snapshot.Body.String())
	}

	analysisResponse := httptest.NewRecorder()
	body := `{"scope":{"pageId":"overview","namespace":"default","resourceType":"workloads"},"customPrompt":"focus prod-a"}`
	handler.ServeHTTP(analysisResponse, httptest.NewRequest(http.MethodPost, "/api/cluster/analyze?id=prod-a", strings.NewReader(body)))
	if analysisResponse.Code != http.StatusOK || !strings.Contains(analysisResponse.Body.String(), "hub cluster analysis") {
		t.Fatalf("analyze = %d: %s", analysisResponse.Code, analysisResponse.Body.String())
	}
	if analyzer.calls != 1 || analyzer.got.UseCase != analysis.UseCaseScan || analyzer.got.Snapshot == nil {
		t.Fatalf("analyzer calls=%d request=%+v", analyzer.calls, analyzer.got)
	}
	if analyzer.got.Scope.Namespace != "default" || analyzer.got.CustomPrompt != "focus prod-a" {
		t.Fatalf("analysis scope not applied: %+v", analyzer.got)
	}

	cached := httptest.NewRecorder()
	handler.ServeHTTP(cached, httptest.NewRequest(http.MethodPost, "/api/cluster/analyze?id=prod-a", strings.NewReader(body)))
	if cached.Code != http.StatusOK || analyzer.calls != 1 {
		t.Fatalf("cached analyze = %d calls=%d", cached.Code, analyzer.calls)
	}
}

func TestHTTPAnalyzeAllowsDifferentClustersToRunConcurrently(t *testing.T) {
	store := &Store{}
	for _, id := range []string{"prod-a", "prod-b"} {
		if accepted, _, err := store.Put(testEnvelope(id, 3)); err != nil || !accepted {
			t.Fatalf("put %s accepted=%v err=%v", id, accepted, err)
		}
	}
	analyzer := &hubBlockingAnalyzer{started: make(chan struct{}, 2), release: make(chan struct{})}
	handler := store.Handler(HandlerOptions{Analyzer: analyzer})

	doneA := make(chan int, 1)
	go func() {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/cluster/analyze?id=prod-a", nil))
		doneA <- w.Code
	}()
	select {
	case <-analyzer.started:
	case <-time.After(time.Second):
		t.Fatal("first cluster analysis did not start")
	}

	doneB := make(chan int, 1)
	go func() {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/cluster/analyze?id=prod-b", nil))
		doneB <- w.Code
	}()
	select {
	case <-analyzer.started:
	case <-time.After(time.Second):
		t.Fatal("second cluster analysis was blocked by unrelated cluster")
	}

	close(analyzer.release)
	for name, done := range map[string]chan int{"prod-a": doneA, "prod-b": doneB} {
		select {
		case code := <-done:
			if code != http.StatusOK {
				t.Fatalf("%s analyze = %d", name, code)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s analyze did not finish", name)
		}
	}
	if analyzer.calls.Load() != 2 {
		t.Fatalf("calls = %d, want two independent cluster analyses", analyzer.calls.Load())
	}
}

func TestHTTPDrilldownAcceptsARNClusterID(t *testing.T) {
	store := &Store{}
	id := "arn:aws:eks:ap-northeast-1:123456789012:cluster/prod-a"
	if accepted, _, err := store.Put(testEnvelope(id, 1)); err != nil || !accepted {
		t.Fatalf("put accepted=%v err=%v", accepted, err)
	}

	w := httptest.NewRecorder()
	handler := store.Handler(HandlerOptions{})
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/cluster?id=arn%3Aaws%3Aeks%3Aap-northeast-1%3A123456789012%3Acluster%2Fprod-a", nil))

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Teleskope cluster report") {
		t.Fatalf("drilldown = %d: %s", w.Code, w.Body.String())
	}
}

func formatLog(format string, args ...any) string {
	return strings.TrimSpace(fmt.Sprintf(format, args...))
}

func containsLog(logs []string, want string) bool {
	for _, log := range logs {
		if strings.Contains(log, want) {
			return true
		}
	}
	return false
}

func testEnvelope(id string, revision uint64) Envelope {
	now := time.Date(2026, 9, 12, 1, 2, 3, 0, time.UTC)
	return Envelope{
		Cluster:     Cluster{ID: id, Name: id, Provider: "kubernetes"},
		Revision:    revision,
		CollectedAt: now,
		Snapshot: &inventory.Snapshot{
			SchemaVersion: "teleskope.io/snapshot/v1alpha1",
			CollectedAt:   now,
			Source:        inventory.Source{Tool: "teleskope", Version: "test", Mode: "live/poll"},
			Kubernetes: inventory.Kubernetes{
				Context:       id,
				Nodes:         []inventory.Node{{ObjectRef: inventory.ObjectRef{Name: "node-a"}}},
				Workloads:     []inventory.Workload{{ObjectRef: inventory.ObjectRef{Name: "web"}}},
				Pods:          []inventory.Pod{{ObjectRef: inventory.ObjectRef{Name: "web-abc"}}},
				RunningImages: []inventory.RunningImage{{Image: "repo/web:v1"}},
			},
		},
		Sources: map[string]live.Status{"kubernetes": {State: "ready"}},
		Events:  []live.Event{{Sequence: revision, At: now, Source: "kubernetes", Level: "info", Message: "refresh published"}},
	}
}
