package k8s

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestReusableCollectorMetadataAndDiscoveryCancellation(t *testing.T) {
	var blockVersion atomic.Bool
	blocked := make(chan struct{})
	accept := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/version":
			if blockVersion.Load() {
				close(blocked)
				<-r.Context().Done()
				return
			}
			fmt.Fprint(w, `{"major":"1","minor":"37","gitVersion":"v1.37.0"}`)
		case "/api":
			fmt.Fprint(w, `{"kind":"APIVersions","apiVersion":"v1","versions":[]}`)
		case "/apis":
			fmt.Fprint(w, `{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`)
		case "/api/v1/secrets":
			accept <- r.Header.Get("Accept")
			fmt.Fprint(w, `{"kind":"PartialObjectMetadataList","apiVersion":"meta.k8s.io/v1","items":[{"metadata":{"name":"credentials","namespace":"app","uid":"s1"}}]}`)
		default:
			w.WriteHeader(403)
			fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"Forbidden","message":"read denied","code":403}`)
		}
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "config")
	config := fmt.Sprintf("apiVersion: v1\nkind: Config\ncurrent-context: test\nclusters:\n- name: test\n  cluster:\n    server: %s\ncontexts:\n- name: test\n  context:\n    cluster: test\n    user: test\nusers:\n- name: test\n  user: {}\n", server.URL)
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := NewCollector(Options{Kubeconfig: path})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	snapshot, coverage, err := c.Collect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Secrets) != 1 || snapshot.Secrets[0].Name != "credentials" {
		t.Fatalf("secrets=%+v", snapshot.Secrets)
	}
	if header := <-accept; !strings.Contains(header, "as=PartialObjectMetadataList") {
		t.Fatalf("secret body requested: %s", header)
	}
	found := false
	for _, item := range coverage {
		if item.Resource == "Pods" && item.Status == "denied" {
			found = true
		}
	}
	if !found {
		t.Fatal("403 not represented as denied")
	}
	blockVersion.Store(true)
	attempt, cancelAttempt := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, _, err := c.Collect(attempt); done <- err }()
	select {
	case <-blocked:
	case <-time.After(time.Second):
		cancelAttempt()
		t.Fatal("version request did not begin")
	}
	cancelAttempt()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled discovery succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("discovery ignored cancellation")
	}
}
