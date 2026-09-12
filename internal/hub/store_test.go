package hub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"github.com/kuraudo-lab/teleskope/internal/live"
)

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
	if drilldown.Code != http.StatusOK || !strings.Contains(drilldown.Body.String(), "Teleskope cluster report") {
		t.Fatalf("drilldown = %d: %s", drilldown.Code, drilldown.Body.String())
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
	}
}
