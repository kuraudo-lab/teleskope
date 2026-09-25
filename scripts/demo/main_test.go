package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kuraudo-lab/teleskope/internal/analysis"
	"github.com/kuraudo-lab/teleskope/internal/hub"
)

func TestFixtureRelationshipsAndPrivacy(t *testing.T) {
	store, err := seededHub()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"source", "target"} {
		env, _ := store.Get(name)
		s := env.Snapshot
		k := s.Kubernetes
		if s.Source.Mode != "synthetic/offline" || s.AWS.AccountID != "000000000000" || len(k.Secrets) != 0 || len(k.ConfigMaps) != 0 {
			t.Fatal("demo must contain synthetic data without secret/config values")
		}
		if len(k.GatewayRoutes) != 1 || len(k.Gateways) != 1 || len(k.Services) != 1 || len(k.PersistentVolumeClaims) != 1 || len(k.PersistentVolumes) != 1 || len(k.StorageClasses) != 1 {
			t.Fatal("missing scenario")
		}
		route := k.GatewayRoutes[0]
		pvc := k.PersistentVolumeClaims[0]
		pv := k.PersistentVolumes[0]
		if route.ParentRefs[0].Name != k.Gateways[0].Name || route.Rules[0].BackendRefs[0].Name != k.Services[0].Name || k.Gateways[0].ClassName != k.GatewayClasses[0].Name {
			t.Fatal("broken gateway chain")
		}
		if pvc.VolumeName != pv.Name || pv.ClaimRef.Name != pvc.Name || pv.ClaimRef.Namespace != pvc.Namespace || pvc.StorageClassName != k.StorageClasses[0].Name || k.Workloads[1].Volumes[0].PersistentVolumeClaim != pvc.Name {
			t.Fatal("broken storage chain")
		}
		if *k.StorageClasses[0].AllowVolumeExpansion != (name == "source") {
			t.Fatal("lost intentional expansion difference")
		}
		for _, pod := range k.Pods {
			if pod.NodeName != k.Nodes[0].Name {
				t.Fatal("unresolved node")
			}
		}
	}
	entries, _ := fixtures.ReadDir("fixtures")
	for _, entry := range entries {
		b, err := fixtures.ReadFile("fixtures/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, account := range regexp.MustCompile(`\b[0-9]{12}\b`).FindAllString(string(b), -1) {
			if account != "000000000000" {
				t.Fatalf("non-demo account in %s", entry.Name())
			}
		}
		for _, endpoint := range regexp.MustCompile(`https?://[^"\s]+`).FindAllString(string(b), -1) {
			u, err := url.Parse(endpoint)
			if err != nil || !strings.HasSuffix(u.Hostname(), ".example.invalid") {
				t.Fatalf("non-demo endpoint %s", endpoint)
			}
		}
		for _, bad := range []string{"AKIA", "ASIA", "PRIVATE KEY", "/Users/", "/home/", "kubeconfig", "password", "api_key", "apiKey", "Bearer "} {
			if bytes.Contains(b, []byte(bad)) {
				t.Fatalf("sensitive marker %q in %s", bad, entry.Name())
			}
		}
	}
	var ai analysis.Result
	if err := decode("analysis", &ai); err != nil {
		t.Fatal(err)
	}
	if ai.Provider != "hand-authored-example" || ai.Model != "none" || len(ai.Limitations) == 0 {
		t.Fatal("AI example must disclose provenance and uncertainty")
	}
}

func TestGeneratedArtifactsAreCurrent(t *testing.T) {
	out := t.TempDir()
	if err := generate(out); err != nil {
		t.Fatal(err)
	}
	committed := filepath.Join("..", "..", "docs", "demo")
	var count int
	err := filepath.WalkDir(out, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		count++
		rel, _ := filepath.Rel(out, path)
		got, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		want, err := os.ReadFile(filepath.Join(committed, rel))
		if err != nil {
			return err
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s is stale: run go run ./scripts/demo", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var savedCount int
	if err := filepath.WalkDir(committed, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			savedCount++
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if count != savedCount || count != 17 {
		t.Fatalf("generated/committed file count = %d/%d", count, savedCount)
	}
	// Validate every relative link in the public catalog, fleet, and AI pages.
	for _, name := range []string{"index.html", "hub/index.html", "ai/index.html"} {
		b, _ := os.ReadFile(filepath.Join(out, name))
		for _, m := range regexp.MustCompile(`href="([^"]+)"`).FindAllSubmatch(b, -1) {
			link := string(m[1])
			if strings.HasPrefix(link, "http") {
				continue
			}
			if _, err := os.Stat(filepath.Join(out, filepath.Dir(name), link)); err != nil {
				t.Errorf("broken %s link %s: %v", name, link, err)
			}
		}
	}
}

func TestSeededProductionHub(t *testing.T) {
	s, err := seededHub()
	if err != nil {
		t.Fatal(err)
	}
	h := previewHandler(s)
	for _, path := range []string{"/", "/api/clusters", "/clusters/source", "/api/cluster/snapshot?id=source", "/api/export/fleet.json", "/api/search?q=checkout"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != 200 {
			t.Errorf("%s = %d: %s", path, r.Code, r.Body.String())
		}
		if path == "/api/clusters" {
			var f hub.FleetResponse
			if err := json.Unmarshal(r.Body.Bytes(), &f); err != nil {
				t.Fatal(err)
			}
			if len(f.Clusters) != 2 || f.Clusters[0].State != "partial" {
				t.Fatal("fleet must contain two clusters and explain partial coverage")
			}
		}
	}
}

func TestPreviewRejectsProviderExecutionAndIngestion(t *testing.T) {
	s, err := seededHub()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/cluster/analyze?id=source", "/api/clusters"} {
		r := httptest.NewRecorder()
		previewHandler(s).ServeHTTP(r, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`)))
		if r.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s must be read-only, got %d", path, r.Code)
		}
	}
}
