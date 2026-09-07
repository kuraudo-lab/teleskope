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
			CustomResourceDefinitions: []inventory.CustomResourceDefinition{
				{
					ObjectRef: inventory.ObjectRef{Kind: "CustomResourceDefinition", Name: "widgets.example.com"},
					Group:     "example.com",
					Scope:     "Namespaced",
					Kind:      "Widget",
					Plural:    "widgets",
					Versions:  []inventory.CRDVersion{{Name: "v1", Served: true, Storage: true}},
				},
			},
			CustomResourceCounts: []inventory.CustomResourceCount{
				{CRDName: "widgets.example.com", Group: "example.com", Version: "v1", Kind: "Widget", Plural: "widgets", Scope: "Namespaced", InstanceCount: 2, NamespaceCount: 2},
			},
			CustomResourceInstances: []inventory.CustomResourceInstance{
				{
					ObjectRef:  inventory.ObjectRef{APIVersion: "example.com/v1", Kind: "Widget", Namespace: "app", Name: "blue"},
					CRDName:    "widgets.example.com",
					CRDGroup:   "example.com",
					CRDVersion: "v1",
					CRDKind:    "Widget",
					CRDPlural:  "widgets",
				},
			},
			RunningImages: []inventory.RunningImage{
				{
					Image:          "repo/web:v1",
					PodCount:       1,
					ContainerCount: 1,
					ImageIDs:       []string{"docker-pullable://repo/web@sha256:aaaaaaaaaaaaaaaa"},
					Runtimes:       []string{"containerd"},
					Namespaces:     []string{"app"},
					Workloads:      []inventory.ObjectRef{{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"}},
				},
			},
			RunningContainers: []inventory.RunningContainer{
				{
					Namespace:     "app",
					Pod:           "web-abc",
					Container:     "web",
					ContainerType: "app",
					Image:         "repo/web:v1",
					ImageID:       "docker-pullable://repo/web@sha256:aaaaaaaaaaaaaaaa",
					NodeName:      "node-a",
					Workload:      inventory.ObjectRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"},
				},
			},
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
		"| CRD instances | 1 |",
		"### Custom resources",
		"| example.com/Widget | 2 | 2 | Namespaced | v1 | widgets |",
		"| widget/app/blue | example.com/Widget | v1 | widgets.example.com | - |",
		"| Running images | 1 |",
		"| Running containers | 1 |",
		"| repo/web:v1 (pods=1) | 1 | containerd | app | deployment/app/web | sha256:aaaaaaaaaaaa |",
		"| app | web-abc | web | app | repo/web:v1 | sha256:aaaaaaaaaaaa | node-a | deployment/app/web |",
		"| Service | service/app/web | ClusterIP | app=web | TCP/80->http |",
	} {
		if !strings.Contains(string(summary), want) {
			t.Fatalf("summary missing %q:\n%s", want, string(summary))
		}
	}
}
