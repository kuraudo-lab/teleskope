package advisor

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func capability(t *testing.T, r Report, key string) Capability {
	t.Helper()
	for _, c := range r.Capabilities {
		if c.Key == key {
			return c
		}
	}
	t.Fatalf("missing %s", key)
	return Capability{}
}

func scopedCapability(t *testing.T, r Report, key, scope string) Capability {
	t.Helper()
	for _, c := range r.Capabilities {
		if c.Key == key && c.Scope == scope {
			return c
		}
	}
	t.Fatalf("missing %s/%s", key, scope)
	return Capability{}
}

func TestUnknownIsNotUnsupported(t *testing.T) {
	for _, status := range []string{"complete", "denied", "unavailable", "partial"} {
		t.Run(status, func(t *testing.T) {
			s := &inventory.Snapshot{Coverage: []inventory.CoverageItem{{Area: "kubernetes", Resource: "StorageClasses", Status: status}}}
			c := capability(t, Analyze(s), "storage.rwx")
			if c.Assessment != "unknown" {
				t.Fatalf("empty inventory: %+v", c)
			}
			if (c.Coverage == "complete") != (status == "complete") {
				t.Fatalf("coverage: %+v", c)
			}
		})
	}
	if len(Analyze(nil).Capabilities) != 0 {
		t.Fatal("nil snapshot invented capabilities")
	}
}

func storageSnapshot() *inventory.Snapshot {
	return &inventory.Snapshot{Kubernetes: inventory.Kubernetes{
		StorageClasses:         []inventory.StorageClass{{ObjectRef: inventory.ObjectRef{Name: "shared"}, Provisioner: "unknown.example"}},
		PersistentVolumeClaims: []inventory.PersistentVolumeClaim{{ObjectRef: inventory.ObjectRef{Name: "data", Namespace: "app"}, StorageClassName: "shared", VolumeName: "pv", Phase: "Bound", AccessModes: []string{"ReadWriteMany"}}},
		PersistentVolumes:      []inventory.PersistentVolume{{ObjectRef: inventory.ObjectRef{Name: "pv"}, StorageClassName: "shared", Phase: "Bound", AccessModes: []string{"ReadWriteMany"}, ClaimRef: inventory.ObjectRef{Name: "data", Namespace: "app"}}},
	}}
}

func TestRWXRequiresMatchingBoundPair(t *testing.T) {
	for _, test := range []struct {
		name      string
		mutate    func(*inventory.Snapshot)
		supported bool
	}{
		{"bound", func(*inventory.Snapshot) {}, true},
		{"pending", func(s *inventory.Snapshot) { s.Kubernetes.PersistentVolumeClaims[0].Phase = "Pending" }, false},
		{"wrong namespace", func(s *inventory.Snapshot) { s.Kubernetes.PersistentVolumes[0].ClaimRef.Namespace = "other" }, false},
		{"wrong class", func(s *inventory.Snapshot) { s.Kubernetes.PersistentVolumes[0].StorageClassName = "other" }, false},
		{"wrong mode", func(s *inventory.Snapshot) { s.Kubernetes.PersistentVolumes[0].AccessModes = []string{"ReadWriteOnce"} }, false},
		{"missing pv", func(s *inventory.Snapshot) { s.Kubernetes.PersistentVolumes = nil }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := storageSnapshot()
			test.mutate(s)
			c := capability(t, Analyze(s), "storage.rwx")
			if (c.Assessment == "supported") != test.supported {
				t.Fatalf("%+v", c)
			}
			if test.supported && (c.Basis != "observed" || len(c.Evidence) != 3) {
				t.Fatalf("missing binding evidence: %+v", c)
			}
		})
	}
}

func TestCoexistingUnusedClassesAndDeterminism(t *testing.T) {
	s := storageSnapshot()
	s.Kubernetes.IngressClasses = []inventory.IngressClass{{ObjectRef: inventory.ObjectRef{Name: "public"}, Controller: "example/ingress"}}
	s.Kubernetes.GatewayClasses = []inventory.GatewayClass{{ObjectRef: inventory.ObjectRef{Name: "edge"}, ControllerName: "example/gateway"}}
	before, _ := json.Marshal(s)
	r := Analyze(s)
	for _, key := range []string{"networking.ingress", "networking.gateway"} {
		c := capability(t, r, key)
		if c.Basis != "declared" || c.Assessment != "supported" {
			t.Fatalf("%+v", c)
		}
	}
	after, _ := json.Marshal(s)
	if string(before) != string(after) {
		t.Fatal("mutated snapshot")
	}
	a, _ := json.Marshal(r)
	b, _ := json.Marshal(Analyze(s))
	if string(a) != string(b) {
		t.Fatal("non deterministic")
	}
	if c := capability(t, r, "storage.expansion"); c.Assessment != "unknown" || c.Evidence[0].Value != "not recorded" {
		t.Fatalf("invented expansion setting: %+v", c)
	}
	r.SetFreshness("stale")
	for _, c := range r.Capabilities {
		if c.Freshness != "stale" {
			t.Fatal("freshness not propagated")
		}
	}
	if !strings.Contains(r.Summary, "1 Ingress classes, 1 Gateway classes") {
		t.Fatal(r.Summary)
	}
}

func TestEKSAddonAdvisorFindings(t *testing.T) {
	s := &inventory.Snapshot{
		EKS: inventory.EKSInventory{
			Addons: []inventory.Addon{
				{Name: "vpc-cni", Version: "v1", Status: "ACTIVE"},
				{Name: "coredns", Version: "v1", Status: "DEGRADED", Issues: []inventory.HealthIssue{{Code: "ConfigError", Message: "bad config"}}},
			},
			Insights: []inventory.EKSInsight{{
				Name:              "Addon Compatibility",
				Category:          "UPGRADE_READINESS",
				KubernetesVersion: "1.32",
				Status:            "WARNING",
				AddonCompatibility: []inventory.AddonCompatibility{{
					Name:               "vpc-cni",
					CompatibleVersions: []string{"v2"},
				}},
			}},
		},
		Coverage: []inventory.CoverageItem{
			{Area: "eks", Resource: "Addons", Status: "complete"},
			{Area: "eks", Resource: "Insights", Status: "complete"},
		},
	}

	r := Analyze(s)
	cni := scopedCapability(t, r, "eks.addon", "vpc-cni")
	if cni.Assessment != "unknown" || cni.Basis != "aws-reported" || !strings.Contains(cni.Summary, "current version is not listed") {
		t.Fatalf("vpc-cni capability = %+v", cni)
	}
	if len(cni.Evidence) < 2 || !strings.Contains(cni.Evidence[1].Value, "target=1.32 compatible=v2 status=WARNING") {
		t.Fatalf("vpc-cni evidence = %+v", cni.Evidence)
	}
	coredns := scopedCapability(t, r, "eks.addon", "coredns")
	if coredns.Assessment != "unsupported" || !strings.Contains(coredns.Summary, "health issue") {
		t.Fatalf("coredns capability = %+v", coredns)
	}
	if !strings.Contains(r.Summary, "2 EKS managed add-ons") {
		t.Fatal(r.Summary)
	}
}
