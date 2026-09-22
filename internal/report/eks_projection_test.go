package report

import (
	"strings"
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
			name: "networking details",
			snap: &inventory.Snapshot{
				EKS: inventory.EKSInventory{
					Cluster: inventory.Cluster{
						Name: "prod",
						VPC: inventory.VPCConfig{
							VPCID:                  "vpc-123",
							SubnetIDs:              []string{"subnet-a"},
							ClusterSecurityGroupID: "sg-cluster",
							SecurityGroupIDs:       []string{"sg-extra"},
						},
					},
					Addons:     []inventory.Addon{{Name: "vpc-cni", Version: "v1.20.0", Status: "ACTIVE", Namespace: "kube-system", ConfigurationValues: `{"env":{"ENABLE_PREFIX_DELEGATION":"true"}}`}},
					Nodegroups: []inventory.Nodegroup{{Name: "system", Subnets: []string{"subnet-a", "subnet-b"}, RemoteAccessSecurityGroupID: "sg-remote"}},
				},
				Kubernetes: inventory.Kubernetes{
					Nodes: []inventory.Node{{
						ObjectRef: inventory.ObjectRef{Kind: "Node", Name: "node-a"},
						Labels: map[string]string{
							"eks.amazonaws.com/nodegroup": "system",
							"topology.kubernetes.io/zone": "ap-northeast-1a",
							"k8s.amazonaws.com/eniConfig": "az-a",
							"vpc.amazonaws.com/pod-eni":   "true",
						},
					}},
					Workloads: []inventory.Workload{{
						ObjectRef: inventory.ObjectRef{Kind: "DaemonSet", Namespace: "kube-system", Name: "aws-node"},
						Selector:  map[string]string{"k8s-app": "aws-node"},
						Containers: []inventory.Container{{
							Name:          "aws-node",
							Image:         "602401143452.dkr.ecr.ap-northeast-1.amazonaws.com/amazon-k8s-cni:v1.20.0",
							EnvConfigRefs: []inventory.ObjectRef{{Kind: "ConfigMap", Namespace: "kube-system", Name: "amazon-vpc-cni"}},
						}},
					}},
					ConfigMaps: []inventory.ConfigObject{{
						ObjectRef: inventory.ObjectRef{Kind: "ConfigMap", Namespace: "kube-system", Name: "amazon-vpc-cni"},
						Data: map[string]string{
							"ENABLE_PREFIX_DELEGATION":             "true",
							"AWS_VPC_K8S_CNI_CUSTOM_NETWORK_CFG":   "true",
							"ENI_CONFIG_LABEL_DEF":                 "k8s.amazonaws.com/eniConfig",
							"irrelevant":                           "ignored",
							"WARM_PREFIX_TARGET":                   "1",
							"AWS_VPC_K8S_CNI_EXTERNALSNAT":         "false",
							"POD_SECURITY_GROUP_ENFORCING_MODE":    "standard",
							"ENABLE_POD_ENI":                       "true",
							"AWS_VPC_K8S_CNI_RANDOMIZESNAT":        "prng",
							"AWS_VPC_K8S_CNI_NODE_PORT_SUPPORT":    "true",
							"AWS_VPC_ENI_MTU":                      "9001",
							"AWS_VPC_K8S_CNI_EXCLUDE_SNAT_CIDRS":   "10.0.0.0/8",
							"AWS_VPC_K8S_CNI_VETHPREFIX":           "eni",
							"AWS_MANAGE_ENIS_NON_SCHEDULABLE":      "false",
							"AWS_VPC_K8S_CNI_CONFIGURE_RPFILTER":   "false",
							"AWS_VPC_K8S_CNI_CONNMARK":             "128",
							"AWS_VPC_K8S_CNI_VETHPREFIX_UNRELATED": "kept",
						},
					}},
					CustomResourceInstances: []inventory.CustomResourceInstance{
						{ObjectRef: inventory.ObjectRef{Kind: "ENIConfig", Name: "az-a"}, CRDName: "eniconfigs.crd.k8s.amazonaws.com", CRDGroup: "crd.k8s.amazonaws.com", CRDVersion: "v1alpha1", CRDKind: "ENIConfig", Labels: map[string]string{"topology.kubernetes.io/zone": "ap-northeast-1a"}},
						{ObjectRef: inventory.ObjectRef{Kind: "SecurityGroupPolicy", Namespace: "app", Name: "web"}, CRDName: "securitygrouppolicies.vpcresources.k8s.aws", CRDGroup: "vpcresources.k8s.aws", CRDVersion: "v1beta1", CRDKind: "SecurityGroupPolicy"},
					},
				},
			},
			assert: func(t *testing.T, got EKSProjection) {
				t.Helper()
				for _, want := range []struct {
					area, source, contains string
				}{
					{"vpc", "eks.cluster", "security group rules are not collected"},
					{"nodegroup networking", "eks.nodegroup/system", "subnets=subnet-a,subnet-b"},
					{"node placement", "kubernetes.nodes", "zones=ap-northeast-1a"},
					{"vpc-cni", "eks.addon/vpc-cni", "ENABLE_PREFIX_DELEGATION"},
					{"vpc-cni", "kubernetes.workload/daemonset/kube-system/aws-node", "amazon-k8s-cni:v1.20.0"},
					{"vpc-cni config", "kubernetes.configmap/configmap/kube-system/amazon-vpc-cni", "AWS_VPC_K8S_CNI_CUSTOM_NETWORK_CFG=true"},
					{"custom networking", "kubernetes.eniconfig/eniconfig/az-a", "crd=eniconfigs.crd.k8s.amazonaws.com"},
					{"custom networking", "kubernetes.securitygrouppolicy/securitygrouppolicy/app/web", "custom resource spec fields are not collected"},
					{"node cni labels", "kubernetes.node/node-a", "k8s.amazonaws.com/eniConfig=az-a"},
				} {
					if !containsNetworkDetail(got.NetworkDetails, want.area, want.source, want.contains) {
						t.Fatalf("network details missing %#v in %#v", want, got.NetworkDetails)
					}
				}
			},
		},
		{
			name: "recorded infrastructure preview",
			snap: &inventory.Snapshot{
				EKS: inventory.EKSInventory{
					Nodegroups: []inventory.Nodegroup{{Name: "workers", AutoScalingGroups: []string{"asg-workers"}}},
					Infrastructure: inventory.EKSInfrastructure{
						Instances:         []inventory.EC2Instance{{InstanceID: "i-b", NodegroupName: "workers", AutoScalingGroupName: "asg-workers", KubernetesNodeName: "node-b", SubnetID: "subnet-b", VPCID: "vpc-a", SecurityGroupIDs: []string{"sg-node"}, LaunchTemplateID: "lt-a", LaunchTemplateVersion: "3"}, {InstanceID: "i-a", NodegroupName: "workers", AutoScalingGroupName: "asg-workers", KubernetesNodeName: "node-a"}},
						AutoScalingGroups: []inventory.AutoScalingGroup{{Name: "asg-workers", NodegroupName: "workers", MinSize: 1, MaxSize: 4, DesiredCapacity: 2, InstanceIDs: []string{"i-a", "i-b"}, LaunchTemplateID: "lt-a", LaunchTemplateVersion: "3"}},
						VPCs:              []inventory.VPCDetail{{VPCID: "vpc-a", CIDR: "192.0.2.0/24", DNSSupport: boolPointer(true)}},
						Subnets:           []inventory.SubnetDetail{{SubnetID: "subnet-b", VPCID: "vpc-a", CIDR: "192.0.2.64/26", AvailabilityZone: "eu-west-1b", RouteTableID: "rtb-b"}},
						SecurityGroups:    []inventory.SecurityGroupDetail{{GroupID: "sg-node", GroupName: "nodes", VPCID: "vpc-a", IngressRuleCount: 3, EgressRuleCount: 1}},
					},
				},
			},
			assert: func(t *testing.T, got EKSProjection) {
				t.Helper()
				if !got.Visible || len(got.Instances) != 2 || got.Instances[0].InstanceID != "i-a" || got.Instances[1].LaunchTemplate != "lt-a version=3" {
					t.Fatalf("instances = %#v", got.Instances)
				}
				if len(got.AutoScalingGroups) != 1 || got.AutoScalingGroups[0].Size != "desired=2 min=1 max=4" {
					t.Fatalf("auto scaling groups = %#v", got.AutoScalingGroups)
				}
				workers := nodegroupRow(got.Nodegroups, "workers")
				if workers.ASGs != "asg-workers" || workers.Instances != "i-a,i-b" {
					t.Fatalf("nodegroup associations = %#v", workers)
				}
				if len(got.VPCs) != 1 || got.VPCs[0].DNSSupport != "true" || len(got.Subnets) != 1 || got.Subnets[0].RouteTable != "rtb-b" || len(got.SecurityGroups) != 1 || got.SecurityGroups[0].Ingress != 3 {
					t.Fatalf("network infrastructure = vpcs=%#v subnets=%#v securityGroups=%#v", got.VPCs, got.Subnets, got.SecurityGroups)
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

func boolPointer(value bool) *bool { return &value }

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

func containsNetworkDetail(rows []EKSNetworkDetailRow, area, source, contains string) bool {
	for _, row := range rows {
		if row.Area != area || row.Source != source {
			continue
		}
		haystack := strings.Join([]string{row.Association, row.Evidence, row.CoverageGap}, " ")
		if strings.Contains(haystack, contains) {
			return true
		}
	}
	return false
}
