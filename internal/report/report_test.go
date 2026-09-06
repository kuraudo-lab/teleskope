package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestWriteDirectoryCreatesRawJSONAndSummary(t *testing.T) {
	now := time.Date(2026, 9, 7, 8, 9, 10, 0, time.UTC)
	snapshot := &inventory.Snapshot{
		SchemaVersion: "teleskope.io/snapshot/v1alpha1",
		CollectedAt:   now,
		Source:        inventory.Source{Tool: "teleskope", Version: "test", Mode: "out-of-cluster/run-once"},
		Kubernetes: inventory.Kubernetes{
			Context: "prod",
			Services: []inventory.Service{
				{
					ObjectRef: inventory.ObjectRef{Kind: "Service", Namespace: "app", Name: "web"},
					Type:      "ClusterIP",
					Selector:  map[string]string{"app": "web"},
					Ports:     []inventory.ServicePort{{Protocol: "TCP", Port: 80, TargetPort: "http"}},
				},
			},
		},
		Coverage: []inventory.CoverageItem{{Area: "kubernetes", Resource: "services", Status: "complete", ObjectCount: 1, CollectedAt: now}},
	}

	artifact, err := WriteDirectory(snapshot, Options{
		BaseDir: t.TempDir(),
		Target:  "Prod Cluster",
		Now:     now,
	})
	if err != nil {
		t.Fatalf("WriteDirectory returned error: %v", err)
	}
	if !strings.HasSuffix(artifact.Dir, "prod-cluster-20260907-080910") {
		t.Fatalf("artifact dir = %q, want sanitized timestamped name", artifact.Dir)
	}
	for _, name := range []string{"snapshot.json", "source.json", "kubernetes.json", "coverage.json", "summary.md"} {
		if _, err := os.Stat(filepath.Join(artifact.Dir, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}

	summary, err := os.ReadFile(filepath.Join(artifact.Dir, "summary.md"))
	if err != nil {
		t.Fatalf("read summary.md: %v", err)
	}
	for _, want := range []string{
		"# Teleskope scan summary",
		"Target: `Prod Cluster`",
		"| Service | service/app/web | ClusterIP | app=web | TCP/80->http |",
	} {
		if !strings.Contains(string(summary), want) {
			t.Fatalf("summary missing %q:\n%s", want, string(summary))
		}
	}
}
