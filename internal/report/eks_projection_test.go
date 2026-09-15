package report

import (
	"testing"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestBuildEKSProjectionPartialSnapshots(t *testing.T) {
	desired := int32(2)
	minSize := int32(1)
	maxSize := int32(4)
	tests := []struct {
		name   string
		snap   *inventory.Snapshot
		assert func(*testing.T, EKSProjection)
	}{
		{
			name: "insights only",
			snap: &inventory.Snapshot{EKS: inventory.EKSInventory{Insights: []inventory.EKSInsight{
				{Name: "zeta", Category: "UPGRADE_READINESS", KubernetesVersion: "1.32", Status: "WARNING", Reason: "late", Recommendation: "Use supported APIs.\n\nMore detail.", Resources: []inventory.EKSInsightResource{{Status: "WARNING"}}},
				{Name: "alpha", Category: "UPGRADE_READINESS", KubernetesVersion: "1.32", Status: "PASSING"},
			}}},
			assert: func(t *testing.T, got EKSProjection) {
				t.Helper()
				if !got.Visible {
					t.Fatal("insights-only snapshot should be visible")
				}
				if len(got.Overview) == 0 || got.Overview[0] != (FieldValueRow{Field: "Cluster", Value: "-"}) {
					t.Fatalf("overview = %#v, want explicit empty cluster row", got.Overview)
				}
				if len(got.Insights) != 2 || got.Insights[0].Name != "alpha" || got.Insights[1].Recommendation != "Use supported APIs. More detail." || got.Insights[1].AffectedResources != 1 {
					t.Fatalf("insights = %#v", got.Insights)
				}
			},
		},
		{
			name: "addons only",
			snap: &inventory.Snapshot{
				EKS: inventory.EKSInventory{
					Addons: []inventory.Addon{
						{Name: "z-addon", Version: "v2", Status: "DEGRADED", Issues: []inventory.HealthIssue{{Code: "ConfigError", Message: "bad config"}}},
						{Name: "vpc-cni", Version: "v1", Status: "ACTIVE", Namespace: "kube-system", ServiceAccountRoleARN: "arn:aws:iam::123456789012:role/cni", PodIdentityAssociations: []string{"kube-system/aws-node"}},
					},
					Insights: []inventory.EKSInsight{{
						Name:              "Addon Compatibility",
						Category:          "UPGRADE_READINESS",
						KubernetesVersion: "1.32",
						Status:            "WARNING",
						Reason:            "update required",
						Recommendation:    "Upgrade the managed add-on.",
						AddonCompatibility: []inventory.AddonCompatibility{{
							Name:               "vpc-cni",
							CompatibleVersions: []string{"v2", "v3"},
						}},
					}},
				},
			},
			assert: func(t *testing.T, got EKSProjection) {
				t.Helper()
				if !got.Visible {
					t.Fatal("add-ons-only snapshot should be visible")
				}
				if len(got.Addons) != 2 || got.Addons[0].Name != "vpc-cni" || got.Addons[0].IAM != "IRSA:arn:aws:iam::123456789012:role/cni PodIdentity=1" {
					t.Fatalf("addons = %#v", got.Addons)
				}
				if got.Addons[0].TargetKubernetes != "1.32" || got.Addons[0].CompatibleVersions != "v2,v3" || got.Addons[0].Upgrade != "status=WARNING use v2,v3 reason=update required recommendation=Upgrade the managed add-on." {
					t.Fatalf("addon compatibility = %#v", got.Addons[0])
				}
				if got.Addons[1].Issues != "ConfigError:bad config" {
					t.Fatalf("addon issues = %q", got.Addons[1].Issues)
				}
			},
		},
		{
			name: "nodegroups only",
			snap: &inventory.Snapshot{
				EKS: inventory.EKSInventory{Nodegroups: []inventory.Nodegroup{
					{Name: "workers-b", Version: "1.31"},
					{Name: "workers-a", Version: "1.31", ReleaseVersion: "1.31.1-20260901", Status: "ACTIVE", AMIType: "AL2023_x86_64_STANDARD", CapacityType: "ON_DEMAND", InstanceTypes: []string{"m7i.large"}, Subnets: []string{"subnet-a", "subnet-b"}, NodeRoleARN: "arn:aws:iam::123456789012:role/node", DesiredSize: &desired, MinSize: &minSize, MaxSize: &maxSize, LaunchTemplateName: "lt-workers", LaunchTemplateVersion: "7"},
					{Name: "custom", Version: "1.31", ReleaseVersion: "custom-20260901", Status: "ACTIVE", AMIType: "CUSTOM"},
				}},
				Kubernetes: inventory.Kubernetes{Nodes: []inventory.Node{
					{ObjectRef: inventory.ObjectRef{Name: "node-a"}, KubeletVersion: "v1.31.1-eks", OSImage: "Amazon Linux 2023", ContainerRuntime: "containerd://1.7.27", Labels: map[string]string{"eks.amazonaws.com/nodegroup": "workers-a", "topology.kubernetes.io/zone": "ap-northeast-1a", "node.kubernetes.io/instance-type": "m7i.large"}},
					{ObjectRef: inventory.ObjectRef{Name: "custom-a"}, KubeletVersion: "v1.31.1-eks", OSImage: "Custom Linux", ContainerRuntime: "containerd://1.7.27", Labels: map[string]string{"eks.amazonaws.com/nodegroup": "custom", "topology.kubernetes.io/zone": "ap-northeast-1b", "node.kubernetes.io/instance-type": "m7i.large"}},
					{ObjectRef: inventory.ObjectRef{Name: "karpenter-a"}, KubeletVersion: "v1.31.1-eks", OSImage: "Amazon Linux 2023", ContainerRuntime: "containerd://1.7.27", Labels: map[string]string{"karpenter.sh/nodepool": "spot", "topology.kubernetes.io/zone": "ap-northeast-1c", "node.kubernetes.io/instance-type": "c7g.large"}},
					{ObjectRef: inventory.ObjectRef{Name: "self-a"}, KubeletVersion: "v1.30.9", OSImage: "Ubuntu", ContainerRuntime: "containerd://1.7.20", Labels: map[string]string{"topology.kubernetes.io/zone": "ap-northeast-1a", "node.kubernetes.io/instance-type": "m5.large"}},
				}},
			},
			assert: func(t *testing.T, got EKSProjection) {
				t.Helper()
				if !got.Visible {
					t.Fatal("nodegroups-only snapshot should be visible")
				}
				workers := nodegroupRow(got.Nodegroups, "workers-a")
				if len(got.Nodegroups) != 3 || workers.Size != "desired=2 min=1 max=4" || workers.CapacityType != "ON_DEMAND" {
					t.Fatalf("nodegroups = %#v", got.Nodegroups)
				}
				if !containsField(got.Network, "Nodegroup subnets", "custom:<br>workers-a:subnet-a,subnet-b<br>workers-b:") {
					t.Fatalf("network = %#v, want nodegroup subnet projection", got.Network)
				}
				if !containsReadiness(got.NodegroupReadiness, "workers-a", "ap-northeast-1a", "observed: runtime evidence linked to managed nodegroup") {
					t.Fatalf("readiness = %#v, want workers-a observed row", got.NodegroupReadiness)
				}
				if !containsReadiness(got.NodegroupReadiness, "custom", "ap-northeast-1b", "unknown: custom AMI requires node runtime verification") {
					t.Fatalf("readiness = %#v, want custom AMI unknown row", got.NodegroupReadiness)
				}
				if !containsReadiness(got.NodegroupReadiness, "karpenter/spot", "ap-northeast-1c", "unknown: Karpenter nodepool") {
					t.Fatalf("readiness = %#v, want Karpenter unknown row", got.NodegroupReadiness)
				}
				if !containsReadiness(got.NodegroupReadiness, "self-managed/unknown", "ap-northeast-1a", "unknown: no managed nodegroup evidence") {
					t.Fatalf("readiness = %#v, want self-managed unknown row", got.NodegroupReadiness)
				}
				if !containsReadiness(got.NodegroupReadiness, "workers-b", "-", "unknown: no Kubernetes node evidence for managed nodegroup") {
					t.Fatalf("readiness = %#v, want workers-b no-node row", got.NodegroupReadiness)
				}
			},
		},
		{
			name: "no eks facts",
			snap: &inventory.Snapshot{},
			assert: func(t *testing.T, got EKSProjection) {
				t.Helper()
				if got.Visible {
					t.Fatalf("empty snapshot visible: %#v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assert(t, BuildEKSProjection(tt.snap))
		})
	}
}

func containsField(rows []FieldValueRow, field, value string) bool {
	for _, row := range rows {
		if row.Field == field && row.Value == value {
			return true
		}
	}
	return false
}

func containsReadiness(rows []EKSNodegroupReadinessRow, group, zone, readiness string) bool {
	for _, row := range rows {
		if row.Group == group && row.Zone == zone && row.Readiness == readiness {
			return true
		}
	}
	return false
}

func nodegroupRow(rows []EKSNodegroupRow, name string) EKSNodegroupRow {
	for _, row := range rows {
		if row.Name == name {
			return row
		}
	}
	return EKSNodegroupRow{}
}
