package compare

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestAnalyzeFindsMigrationRelevantDifferences(t *testing.T) {
	source := comparisonSnapshot("source", "v1.31.2")
	target := comparisonSnapshot("target", "v1.30.9")
	target.Kubernetes.StorageClasses = []inventory.StorageClass{}
	target.Kubernetes.RuntimeClasses = []inventory.RuntimeClass{}
	target.Kubernetes.ConfigMaps = []inventory.ConfigObject{}
	target.Kubernetes.Secrets = []inventory.Secret{}
	target.Coverage = []inventory.CoverageItem{{Area: "kubernetes", Resource: "Secrets", Status: "partial", Reason: "forbidden", CollectedAt: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)}}

	report := Analyze(source, target)

	if report.SchemaVersion != "teleskope.io/comparison/v1alpha1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	for _, want := range []string{
		"Kubernetes versions differ",
		"Storage classes used or declared in source are missing from target",
		"Source workloads reference RuntimeClasses missing from target",
		"Source workloads reference ConfigMaps missing from target",
		"Source workloads reference Secrets missing from target",
	} {
		if !hasFinding(report, want) {
			t.Fatalf("missing finding containing %q: %#v", want, report.Findings)
		}
	}
	if len(report.InventoryDiffs) == 0 {
		t.Fatal("expected inventory diffs")
	}
	if len(report.CoverageWarnings) != 1 || report.CoverageWarnings[0].Resource != "Secrets" {
		t.Fatalf("coverage warnings = %#v", report.CoverageWarnings)
	}
}

func TestLoadSnapshotAcceptsReportDirectory(t *testing.T) {
	dir := t.TempDir()
	want := comparisonSnapshot("demo", "v1.31.0")
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "snapshot.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kubernetes.Context != "demo" {
		t.Fatalf("context = %q", got.Kubernetes.Context)
	}
}

func TestMarkdownRendersComparison(t *testing.T) {
	report := Analyze(comparisonSnapshot("source", "v1.31.0"), comparisonSnapshot("target", "v1.31.0"))
	md := Markdown(report)
	for _, want := range []string{"# Teleskope migration comparison", "## Clusters", "## Findings"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
}

func comparisonSnapshot(name, version string) *inventory.Snapshot {
	allowExpansion := true
	return &inventory.Snapshot{
		SchemaVersion: "teleskope.io/snapshot/v1alpha1",
		CollectedAt:   time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
		Source:        inventory.Source{Tool: "teleskope", Version: "test", Mode: "test"},
		AWS:           inventory.AWSIdentity{Region: "ap-northeast-1"},
		EKS:           inventory.EKSInventory{Cluster: inventory.Cluster{Name: name, Version: strings.TrimPrefix(version, "v")}},
		Kubernetes: inventory.Kubernetes{
			Context: name,
			Version: inventory.KubernetesVersion{GitVersion: version},
			APIResources: []inventory.APIResource{
				{GroupVersion: "v1", Version: "v1", Resource: "pods", Kind: "Pod", Namespaced: true, Verbs: []string{"get", "list"}},
				{GroupVersion: "networking.k8s.io/v1", Group: "networking.k8s.io", Version: "v1", Resource: "ingresses", Kind: "Ingress", Namespaced: true, Verbs: []string{"get", "list"}},
			},
			Namespaces: []inventory.Namespace{{ObjectRef: inventory.ObjectRef{Kind: "Namespace", Name: "app"}, Phase: "Active"}},
			ServiceAccounts: []inventory.ServiceAccount{{
				ObjectRef: inventory.ObjectRef{Kind: "ServiceAccount", Namespace: "app", Name: "default"},
			}},
			Workloads: []inventory.Workload{{
				ObjectRef:          inventory.ObjectRef{Kind: "Deployment", Namespace: "app", Name: "web"},
				ServiceAccountName: "default",
				RuntimeClassName:   "gvisor",
				ConfigRefs:         []inventory.ObjectRef{{Kind: "ConfigMap", Name: "web-config"}},
				SecretRefs:         []inventory.ObjectRef{{Kind: "Secret", Name: "web-secret"}},
				Volumes:            []inventory.Volume{{Name: "data", Type: "persistentVolumeClaim", PersistentVolumeClaim: "web-data"}},
			}},
			IngressClasses: []inventory.IngressClass{{
				ObjectRef:  inventory.ObjectRef{Kind: "IngressClass", Name: "alb"},
				Controller: "ingress.k8s.aws/alb",
			}},
			StorageClasses: []inventory.StorageClass{{
				ObjectRef:            inventory.ObjectRef{Kind: "StorageClass", Name: "gp3"},
				Provisioner:          "ebs.csi.aws.com",
				VolumeBindingMode:    "WaitForFirstConsumer",
				AllowVolumeExpansion: &allowExpansion,
			}},
			CSIDrivers: []inventory.CSIDriver{{
				ObjectRef:            inventory.ObjectRef{Kind: "CSIDriver", Name: "ebs.csi.aws.com"},
				VolumeLifecycleModes: []string{"Persistent"},
			}},
			RuntimeClasses: []inventory.RuntimeClass{{
				ObjectRef: inventory.ObjectRef{Kind: "RuntimeClass", Name: "gvisor"},
				Handler:   "runsc",
			}},
			ConfigMaps: []inventory.ConfigObject{{
				ObjectRef: inventory.ObjectRef{Kind: "ConfigMap", Namespace: "app", Name: "web-config"},
				Data:      map[string]string{"config.yaml": "..."},
			}},
			Secrets: []inventory.Secret{{
				ObjectRef: inventory.ObjectRef{Kind: "Secret", Namespace: "app", Name: "web-secret"},
				Type:      "Opaque",
				Keys:      []string{"password"},
			}},
			PersistentVolumeClaims: []inventory.PersistentVolumeClaim{{
				ObjectRef:        inventory.ObjectRef{Kind: "PersistentVolumeClaim", Namespace: "app", Name: "web-data"},
				StorageClassName: "gp3",
				AccessModes:      []string{"ReadWriteOnce"},
				Phase:            "Bound",
			}},
		},
		Coverage: []inventory.CoverageItem{
			{Area: "kubernetes", Resource: "StorageClasses", Status: "complete", ObjectCount: 1, CollectedAt: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)},
			{Area: "kubernetes", Resource: "RuntimeClasses", Status: "complete", ObjectCount: 1, CollectedAt: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)},
			{Area: "kubernetes", Resource: "ConfigMaps", Status: "complete", ObjectCount: 1, CollectedAt: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)},
			{Area: "kubernetes", Resource: "Secrets", Status: "complete", ObjectCount: 1, CollectedAt: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)},
		},
	}
}

func hasFinding(report Report, want string) bool {
	for _, finding := range report.Findings {
		if strings.Contains(finding.Summary, want) {
			return true
		}
	}
	return false
}
