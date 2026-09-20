package analysis

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/compare"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestBuildContextForScanIncludesWorkloadStorageAndNetwork(t *testing.T) {
	replicas := int32(2)
	snapshot := &inventory.Snapshot{
		SchemaVersion: "teleskope.io/snapshot/v1alpha1",
		CollectedAt:   time.Date(2026, 9, 11, 1, 2, 3, 0, time.UTC),
		Kubernetes: inventory.Kubernetes{
			Context: "prod",
			Version: inventory.KubernetesVersion{GitVersion: "v1.31.0"},
			Workloads: []inventory.Workload{{
				ObjectRef:          inventory.ObjectRef{Kind: "Deployment", Namespace: "kube-system", Name: "aws-load-balancer-controller", UID: "not-needed"},
				Replicas:           &replicas,
				ServiceAccountName: "aws-load-balancer-controller",
				Selector:           map[string]string{"app": "aws-load-balancer-controller"},
				Containers: []inventory.Container{{
					Name:          "controller",
					Image:         "public.ecr.aws/eks/aws-load-balancer-controller:v2.8.1",
					ImageID:       "sha256:not-needed",
					ContainerID:   "containerd://not-needed",
					EnvSecretRefs: []inventory.ObjectRef{{Name: "hidden"}},
				}},
				Volumes: []inventory.Volume{{Name: "data", Type: "PersistentVolumeClaim", PersistentVolumeClaim: "controller-data"}},
			}},
			RunningImages:          []inventory.RunningImage{{Image: "public.ecr.aws/eks/aws-load-balancer-controller:v2.8.1", Workloads: []inventory.ObjectRef{{Kind: "Deployment", Namespace: "kube-system", Name: "aws-load-balancer-controller", UID: "not-needed"}}}},
			Services:               []inventory.Service{{ObjectRef: inventory.ObjectRef{Kind: "Service", Namespace: "kube-system", Name: "webhook-service"}, Type: "ClusterIP", Selector: map[string]string{"app": "aws-load-balancer-controller"}, Ports: []inventory.ServicePort{{Protocol: "TCP", Port: 443, TargetPort: "9443"}}}},
			Ingresses:              []inventory.Ingress{{ObjectRef: inventory.ObjectRef{Kind: "Ingress", Namespace: "kube-system", Name: "controller-ingress"}, ClassName: "alb", Rules: []inventory.IngressRule{{Host: "controller.example.com", ServiceName: "webhook-service", ServicePort: "443"}}}},
			StorageClasses:         []inventory.StorageClass{{ObjectRef: inventory.ObjectRef{Kind: "StorageClass", Name: "gp3"}, Provisioner: "ebs.csi.aws.com", ReclaimPolicy: "Delete", VolumeBindingMode: "WaitForFirstConsumer"}},
			PersistentVolumeClaims: []inventory.PersistentVolumeClaim{{ObjectRef: inventory.ObjectRef{Kind: "PersistentVolumeClaim", Namespace: "kube-system", Name: "controller-data"}, StorageClassName: "gp3", VolumeName: "pv-controller", RequestedStorage: "10Gi", AccessModes: []string{"ReadWriteOnce"}, VolumeMode: "Filesystem", Phase: "Bound"}},
			PersistentVolumes:      []inventory.PersistentVolume{{ObjectRef: inventory.ObjectRef{Kind: "PersistentVolume", Name: "pv-controller"}, StorageClassName: "gp3", Capacity: "10Gi", AccessModes: []string{"ReadWriteOnce"}, VolumeMode: "Filesystem", Phase: "Bound", ClaimRef: inventory.ObjectRef{Namespace: "kube-system", Name: "controller-data"}, CSI: &inventory.CSIVolume{Driver: "ebs.csi.aws.com"}}},
			IngressClasses:         []inventory.IngressClass{{ObjectRef: inventory.ObjectRef{Name: "alb"}, Controller: "ingress.k8s.aws/alb"}},
			RBAC: inventory.RBAC{
				ClusterRoles: []inventory.ObjectRef{{Kind: "ClusterRole", Name: "cluster-admin"}},
				ClusterRoleDetails: []inventory.Role{{
					ObjectRef: inventory.ObjectRef{Kind: "ClusterRole", Name: "cluster-admin"},
					Rules:     []inventory.RBACRule{{Resources: []string{"*"}, Verbs: []string{"*"}}},
				}},
			},
		},
	}

	data, err := BuildContext(Request{UseCase: UseCaseScan, Snapshot: snapshot})
	if err != nil {
		t.Fatalf("BuildContext returned error: %v", err)
	}
	text := string(data)
	for _, want := range []string{"aws-load-balancer-controller", "public.ecr.aws/eks", "networking.ingress", "v1.31.0", "ReadWriteOnce", "ebs.csi.aws.com", "controller.example.com", "TCP/443-\\u003e9443", "coreResourceSummary"} {
		if !strings.Contains(text, want) {
			t.Fatalf("context missing %q:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"hidden", "not-needed", "ClusterRoleDetails"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("context included unwanted detail %q: %s", unwanted, text)
		}
	}
}

func TestBuildContextForCompareIncludesDeterministicReport(t *testing.T) {
	source := &inventory.Snapshot{Kubernetes: inventory.Kubernetes{Version: inventory.KubernetesVersion{GitVersion: "v1.31.0"}}}
	target := &inventory.Snapshot{Kubernetes: inventory.Kubernetes{Version: inventory.KubernetesVersion{GitVersion: "v1.30.0"}}}
	report := compare.Analyze(source, target)

	data, err := BuildContext(Request{UseCase: UseCaseCompare, Source: source, Target: target, CompareReport: &report})
	if err != nil {
		t.Fatalf("BuildContext returned error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("context is not json: %v", err)
	}
	if decoded["compareReport"] == nil {
		t.Fatalf("context did not include compareReport: %s", data)
	}
}

func TestBuildContextIncludesScopeAndFiltersScanContext(t *testing.T) {
	snapshot := &inventory.Snapshot{
		Kubernetes: inventory.Kubernetes{
			Workloads: []inventory.Workload{
				{ObjectRef: inventory.ObjectRef{Kind: "Deployment", Namespace: "payments", Name: "api"}},
				{ObjectRef: inventory.ObjectRef{Kind: "Deployment", Namespace: "platform", Name: "worker"}},
			},
			RunningImages: []inventory.RunningImage{
				{Image: "repo/api:v1", Namespaces: []string{"payments"}, Workloads: []inventory.ObjectRef{{Kind: "Deployment", Namespace: "payments", Name: "api"}}},
				{Image: "repo/worker:v1", Namespaces: []string{"platform"}, Workloads: []inventory.ObjectRef{{Kind: "Deployment", Namespace: "platform", Name: "worker"}}},
			},
		},
	}
	data, err := BuildContext(Request{
		UseCase:  UseCaseScan,
		Snapshot: snapshot,
		Scope: Scope{
			PageID:    "overview",
			Namespace: "payments",
			SelectedRefs: []inventory.ObjectRef{{
				Kind:      "Deployment",
				Namespace: "payments",
				Name:      "api",
				UID:       "not-needed",
			}},
		},
	})
	if err != nil {
		t.Fatalf("BuildContext returned error: %v", err)
	}
	text := string(data)
	for _, want := range []string{`"namespace": "payments"`, `"pageId": "overview"`, "repo/api:v1", `"name": "api"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("context missing %q:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"platform", "worker", "not-needed"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("context included out-of-scope detail %q:\n%s", unwanted, text)
		}
	}
}

func TestBuildContextIncludesEvidenceForEachScopedPage(t *testing.T) {
	snapshot := &inventory.Snapshot{
		EKS: inventory.EKSInventory{
			Cluster:    inventory.Cluster{Name: "prod"},
			Addons:     []inventory.Addon{{Name: "vpc-cni"}},
			Nodegroups: []inventory.Nodegroup{{Name: "system"}},
		},
		Kubernetes: inventory.Kubernetes{
			Nodes:     []inventory.Node{{ObjectRef: inventory.ObjectRef{Kind: "Node", Name: "node-a", UID: "remove-me"}, Ready: "True"}},
			Workloads: []inventory.Workload{{ObjectRef: inventory.ObjectRef{Kind: "Deployment", Namespace: "payments", Name: "api"}}},
			Pods:      []inventory.Pod{{ObjectRef: inventory.ObjectRef{Kind: "Pod", Namespace: "payments", Name: "api-123", UID: "remove-me"}}},
			Services: []inventory.Service{
				{ObjectRef: inventory.ObjectRef{Kind: "Service", Namespace: "payments", Name: "api"}},
				{ObjectRef: inventory.ObjectRef{Kind: "Service", Namespace: "platform", Name: "metrics"}},
			},
			PersistentVolumeClaims: []inventory.PersistentVolumeClaim{{ObjectRef: inventory.ObjectRef{Kind: "PersistentVolumeClaim", Namespace: "payments", Name: "data"}, RequestedStorage: "20Gi"}},
			StorageClasses:         []inventory.StorageClass{{ObjectRef: inventory.ObjectRef{Kind: "StorageClass", Name: "gp3"}, Provisioner: "ebs.csi.aws.com"}},
			RBAC: inventory.RBAC{
				RoleDetails:        []inventory.Role{{ObjectRef: inventory.ObjectRef{Kind: "Role", Namespace: "payments", Name: "reader"}}},
				ClusterRoleDetails: []inventory.Role{{ObjectRef: inventory.ObjectRef{Kind: "ClusterRole", Name: "view"}}},
			},
			AdmissionWebhooks: []inventory.AdmissionWebhookConfig{{ObjectRef: inventory.ObjectRef{Kind: "ValidatingWebhookConfiguration", Name: "policy"}}},
		},
	}
	tests := []struct {
		page string
		want []string
	}{
		{page: "eks", want: []string{`"eks"`, "vpc-cni", "system"}},
		{page: "nodes", want: []string{`"nodes"`, "node-a", `"ready": "True"`}},
		{page: "workloads", want: []string{`"workloads"`, "api-123", `"pods"`}},
		{page: "network", want: []string{`"network"`, `"services"`, `"name": "api"`}},
		{page: "storage", want: []string{`"storage"`, "20Gi", "ebs.csi.aws.com"}},
		{page: "security", want: []string{`"security"`, "reader", "policy"}},
	}
	for _, tt := range tests {
		t.Run(tt.page, func(t *testing.T) {
			scope := Scope{PageID: tt.page, ResourceType: tt.page, Namespace: "payments"}
			if tt.page == "network" {
				scope.SelectedRefs = []inventory.ObjectRef{{Kind: "Service", Namespace: "payments", Name: "api"}}
			}
			data, err := BuildContext(Request{UseCase: UseCaseScan, Snapshot: snapshot, Scope: scope})
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			for _, want := range tt.want {
				if !strings.Contains(text, want) {
					t.Fatalf("%s context missing %q:\n%s", tt.page, want, text)
				}
			}
			if strings.Contains(text, "remove-me") || (tt.page == "network" && strings.Contains(text, "metrics")) {
				t.Fatalf("%s context leaked UID or out-of-scope resource:\n%s", tt.page, text)
			}
		})
	}
}
