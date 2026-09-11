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
