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
			snap: &inventory.Snapshot{EKS: inventory.EKSInventory{Nodegroups: []inventory.Nodegroup{
				{Name: "workers-b", Version: "1.31"},
				{Name: "workers-a", Version: "1.31", ReleaseVersion: "1.31.1-20260901", Status: "ACTIVE", AMIType: "AL2023_x86_64_STANDARD", CapacityType: "ON_DEMAND", InstanceTypes: []string{"m7i.large"}, Subnets: []string{"subnet-a", "subnet-b"}, NodeRoleARN: "arn:aws:iam::123456789012:role/node", DesiredSize: &desired, MinSize: &minSize, MaxSize: &maxSize},
			}}},
			assert: func(t *testing.T, got EKSProjection) {
				t.Helper()
				if !got.Visible {
					t.Fatal("nodegroups-only snapshot should be visible")
				}
				if len(got.Nodegroups) != 2 || got.Nodegroups[0].Name != "workers-a" || got.Nodegroups[0].Size != "desired=2 min=1 max=4" || got.Nodegroups[0].CapacityType != "ON_DEMAND" {
					t.Fatalf("nodegroups = %#v", got.Nodegroups)
				}
				if !containsField(got.Network, "Nodegroup subnets", "workers-a:subnet-a,subnet-b<br>workers-b:") {
					t.Fatalf("network = %#v, want nodegroup subnet projection", got.Network)
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
