package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kuraudo-lab/teleskope/internal/live"
	"github.com/kuraudo-lab/teleskope/internal/topology"
)

func TestLiveDemoUsesRecordedFixtureAndRealUpdateContract(t *testing.T) {
	snapshot, err := loadSnapshot(filepath.Join("..", "..", defaultFixture))
	if err != nil {
		t.Fatal(err)
	}
	demo, err := newLiveDemo(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	initial, err := demo.store.View()
	if err != nil || initial.Revision != 1 || initial.Graph == nil {
		t.Fatalf("initial view = revision %d graph=%v err=%v", initial.Revision, initial.Graph != nil, err)
	}
	demo.publish(true)
	delta, err := demo.store.TopologyUpdate(1)
	if err != nil || delta.Kind != topology.UpdateDelta || delta.Revision != 2 || len(delta.UpsertNodes) == 0 {
		t.Fatalf("delta = %+v, err=%v", delta, err)
	}
	for range 17 {
		demo.publish(false)
	}
	reset, err := demo.store.TopologyUpdate(2)
	if err != nil || reset.Kind != topology.UpdateFull || reset.Graph == nil {
		t.Fatalf("reset = %+v, err=%v", reset, err)
	}
}

func TestLiveDemoHTTPHarness(t *testing.T) {
	snapshot, err := loadSnapshot(filepath.Join("..", "..", defaultFixture))
	if err != nil {
		t.Fatal(err)
	}
	demo, err := newLiveDemo(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	page := httptest.NewRecorder()
	demo.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	for _, wanted := range []string{"topologyLiveEvidence", "/demo/advance", "/demo/gap", "selectionStable", "layoutStable", "nodeProbe:false"} {
		if !strings.Contains(page.Body.String(), wanted) {
			t.Fatalf("live demo page missing %q", wanted)
		}
	}

	advance := httptest.NewRecorder()
	demo.ServeHTTP(advance, httptest.NewRequest(http.MethodPost, "/demo/advance", nil))
	if advance.Code != http.StatusNoContent {
		t.Fatalf("advance status = %d", advance.Code)
	}
	topologyResponse := httptest.NewRecorder()
	demo.ServeHTTP(topologyResponse, httptest.NewRequest(http.MethodGet, "/api/topology?since=1", nil))
	var update topology.Update
	if err := json.Unmarshal(topologyResponse.Body.Bytes(), &update); err != nil {
		t.Fatal(err)
	}
	if update.Kind != topology.UpdateDelta {
		t.Fatalf("update kind = %s", update.Kind)
	}
	view, err := demo.store.View()
	if err != nil || view.Sources["kubernetes"].Mode != live.SourceModeRecorded {
		t.Fatalf("recorded source = %+v, err=%v", view.Sources["kubernetes"], err)
	}
}
