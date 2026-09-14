package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

type EKSProjection struct {
	Visible       bool              `json:"visible"`
	Overview      []FieldValueRow   `json:"overview,omitempty"`
	Insights      []EKSInsightRow   `json:"insights,omitempty"`
	Capacity      []EKSCapacityRow  `json:"capacity,omitempty"`
	Network       []FieldValueRow   `json:"network,omitempty"`
	Security      []FieldValueRow   `json:"security,omitempty"`
	AccessEntries []AccessEntryRow  `json:"accessEntries,omitempty"`
	PodIdentities []PodIdentityRow  `json:"podIdentities,omitempty"`
	Addons        []EKSAddonRow     `json:"addons,omitempty"`
	Nodegroups    []EKSNodegroupRow `json:"nodegroups,omitempty"`
}

type FieldValueRow struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type EKSInsightRow struct {
	Name              string `json:"name"`
	Category          string `json:"category"`
	KubernetesVersion string `json:"kubernetesVersion"`
	Status            string `json:"status"`
	Reason            string `json:"reason"`
	Recommendation    string `json:"recommendation"`
	AffectedResources int    `json:"affectedResources"`
}

type EKSCapacityRow struct {
	Resource         string `json:"resource"`
	Capacity         string `json:"capacity"`
	Allocatable      string `json:"allocatable"`
	RequestedBySpecs string `json:"requestedBySpecs"`
	LimitsBySpecs    string `json:"limitsBySpecs"`
}

type AccessEntryRow struct {
	Principal string `json:"principal"`
	Type      string `json:"type"`
	User      string `json:"user"`
	Groups    string `json:"groups"`
	Policies  string `json:"policies"`
}

type PodIdentityRow struct {
	Namespace      string `json:"namespace"`
	ServiceAccount string `json:"serviceAccount"`
	Role           string `json:"role"`
	TargetRole     string `json:"targetRole"`
	Owner          string `json:"owner"`
}

type EKSAddonRow struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Status    string `json:"status"`
	Namespace string `json:"namespace"`
	IAM       string `json:"iam"`
	Issues    string `json:"issues"`
}

type EKSNodegroupRow struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Release      string `json:"release"`
	Status       string `json:"status"`
	AMI          string `json:"ami"`
	Capacity     string `json:"capacity"`
	CapacityType string `json:"capacityType"`
	Size         string `json:"size"`
	Subnets      string `json:"subnets"`
	IAM          string `json:"iam"`
	Issues       string `json:"issues"`
}

func BuildEKSProjection(snapshot *inventory.Snapshot) EKSProjection {
	if snapshot == nil || !hasEKS(snapshot) {
		return EKSProjection{}
	}
	cluster := snapshot.EKS.Cluster
	out := EKSProjection{Visible: true}
	out.Overview = []FieldValueRow{
		{"Cluster", display(cluster.Name)},
		{"Region", display(snapshot.AWS.Region)},
		{"Version", display(cluster.Version)},
		{"Platform", display(cluster.PlatformVersion)},
		{"Status", display(cluster.Status)},
		{"ARN", display(cluster.ARN)},
		{"Endpoint", fmt.Sprintf("public=%t private=%t", cluster.VPC.EndpointPublicAccess, cluster.VPC.EndpointPrivateAccess)},
		{"Auth", display(cluster.AccessConfig.AuthenticationMode) + " bootstrapCreatorAdmin=" + boolPtr(cluster.AccessConfig.BootstrapClusterCreatorAdminPermissions)},
		{"Control plane logs", "enabled=" + strings.Join(cluster.EnabledControlPlaneLogTypes, ",") + " disabled=" + strings.Join(cluster.DisabledControlPlaneLogTypes, ",")},
		{"Auto mode", "compute=" + boolPtr(cluster.AutoMode.ComputeEnabled) + " nodePools=" + strings.Join(cluster.AutoMode.ComputeNodePools, ",") + " blockStorage=" + boolPtr(cluster.AutoMode.BlockStorageEnabled)},
	}
	out.Insights = projectEKSInsights(snapshot.EKS.Insights)
	out.Capacity = projectEKSCapacity(snapshot.Kubernetes)
	out.Network = projectEKSNetwork(cluster, snapshot.EKS.Nodegroups)
	out.Security, out.AccessEntries, out.PodIdentities = projectEKSSecurity(snapshot)
	out.Addons = projectEKSAddons(snapshot.EKS.Addons)
	out.Nodegroups = projectEKSNodegroups(snapshot.EKS.Nodegroups)
	return out
}

func projectEKSInsights(insights []inventory.EKSInsight) []EKSInsightRow {
	rows := make([]EKSInsightRow, 0, len(insights))
	for _, insight := range sortedEKSInsights(insights) {
		rows = append(rows, EKSInsightRow{
			Name:              display(insight.Name),
			Category:          display(insight.Category),
			KubernetesVersion: display(insight.KubernetesVersion),
			Status:            display(insight.Status),
			Reason:            display(insight.Reason),
			Recommendation:    display(trimMarkdownText(insight.Recommendation)),
			AffectedResources: len(insight.Resources),
		})
	}
	return rows
}

func projectEKSCapacity(kubernetes inventory.Kubernetes) []EKSCapacityRow {
	if len(kubernetes.Nodes) == 0 && len(kubernetes.Pods) == 0 && len(kubernetes.Workloads) == 0 {
		return nil
	}
	capacity := nodeResourceTotals(kubernetes.Nodes)
	requests := containerResourceTotals(kubernetes)
	return []EKSCapacityRow{
		{"CPU", formatMilliCPU(capacity.CapacityCPUm), formatMilliCPU(capacity.AllocatableCPUm), formatMilliCPU(requests.RequestCPUm), formatMilliCPU(requests.LimitCPUm)},
		{"Memory", formatBytes(capacity.CapacityMemoryBytes), formatBytes(capacity.AllocatableMemoryBytes), formatBytes(requests.RequestMemoryBytes), formatBytes(requests.LimitMemoryBytes)},
		{"Pods", fmt.Sprintf("%d", capacity.CapacityPods), fmt.Sprintf("%d", capacity.AllocatablePods), "-", "-"},
	}
}

func projectEKSNetwork(cluster inventory.Cluster, nodegroups []inventory.Nodegroup) []FieldValueRow {
	if cluster.VPC.VPCID == "" && len(cluster.VPC.SubnetIDs) == 0 && cluster.Network.IPFamily == "" && len(nodegroups) == 0 {
		return nil
	}
	rows := []FieldValueRow{
		{"VPC", display(cluster.VPC.VPCID)},
		{"Cluster subnets", display(strings.Join(cluster.VPC.SubnetIDs, ","))},
		{"Cluster security groups", display(strings.Join(cluster.VPC.SecurityGroupIDs, ","))},
		{"Cluster security group", display(cluster.VPC.ClusterSecurityGroupID)},
		{"Public access CIDRs", display(strings.Join(cluster.VPC.PublicAccessCIDRs, ","))},
		{"IP family", display(cluster.Network.IPFamily)},
		{"Service CIDR", display(strings.Join(nonEmpty(cluster.Network.ServiceIPv4CIDR, cluster.Network.ServiceIPv6CIDR), ","))},
		{"Auto Mode load balancing", boolPtr(cluster.Network.AutoModeLoadBalancingEnabled)},
	}
	if len(nodegroups) > 0 {
		parts := make([]string, 0, len(nodegroups))
		for _, nodegroup := range sortedNodegroups(nodegroups) {
			parts = append(parts, nodegroup.Name+":"+strings.Join(nodegroup.Subnets, ","))
		}
		rows = append(rows, FieldValueRow{"Nodegroup subnets", display(strings.Join(parts, "<br>"))})
	}
	return rows
}

func projectEKSSecurity(snapshot *inventory.Snapshot) ([]FieldValueRow, []AccessEntryRow, []PodIdentityRow) {
	cluster := snapshot.EKS.Cluster
	if cluster.RoleARN == "" && cluster.OIDCIssuer == "" && len(snapshot.EKS.AccessEntries) == 0 && len(snapshot.EKS.PodIdentityAssociations) == 0 && len(cluster.Encryption) == 0 {
		return nil, nil, nil
	}
	summary := []FieldValueRow{
		{"Cluster role", display(cluster.RoleARN)},
		{"OIDC issuer", display(cluster.OIDCIssuer)},
		{"Encryption", encryptionValue(cluster.Encryption)},
		{"Access entries", fmt.Sprintf("%d", len(snapshot.EKS.AccessEntries))},
		{"Pod Identity associations", fmt.Sprintf("%d", len(snapshot.EKS.PodIdentityAssociations))},
	}
	access := make([]AccessEntryRow, 0, len(snapshot.EKS.AccessEntries))
	for _, entry := range sortedAccessEntries(snapshot.EKS.AccessEntries) {
		access = append(access, AccessEntryRow{
			Principal: display(entry.PrincipalARN),
			Type:      display(entry.Type),
			User:      display(entry.Username),
			Groups:    display(strings.Join(entry.KubernetesGroups, ",")),
			Policies:  accessPolicies(entry.Policies),
		})
	}
	identities := make([]PodIdentityRow, 0, len(snapshot.EKS.PodIdentityAssociations))
	for _, association := range sortedPodIdentities(snapshot.EKS.PodIdentityAssociations) {
		identities = append(identities, PodIdentityRow{
			Namespace:      display(association.Namespace),
			ServiceAccount: display(association.ServiceAccount),
			Role:           display(association.RoleARN),
			TargetRole:     display(association.TargetRoleARN),
			Owner:          display(association.OwnerARN),
		})
	}
	return summary, access, identities
}

func projectEKSAddons(addons []inventory.Addon) []EKSAddonRow {
	rows := make([]EKSAddonRow, 0, len(addons))
	for _, addon := range sortedAddons(addons) {
		rows = append(rows, EKSAddonRow{
			Name:      display(addon.Name),
			Version:   display(addon.Version),
			Status:    display(addon.Status),
			Namespace: display(addon.Namespace),
			IAM:       addonIAM(addon),
			Issues:    healthIssues(addon.Issues),
		})
	}
	return rows
}

func projectEKSNodegroups(nodegroups []inventory.Nodegroup) []EKSNodegroupRow {
	rows := make([]EKSNodegroupRow, 0, len(nodegroups))
	for _, nodegroup := range sortedNodegroups(nodegroups) {
		rows = append(rows, EKSNodegroupRow{
			Name:         display(nodegroup.Name),
			Version:      display(nodegroup.Version),
			Release:      display(nodegroup.ReleaseVersion),
			Status:       display(nodegroup.Status),
			AMI:          display(nodegroup.AMIType),
			Capacity:     display(strings.Join(nodegroup.InstanceTypes, ",")),
			CapacityType: display(nodegroup.CapacityType),
			Size:         fmt.Sprintf("desired=%s min=%s max=%s", int32Ptr(nodegroup.DesiredSize), int32Ptr(nodegroup.MinSize), int32Ptr(nodegroup.MaxSize)),
			Subnets:      display(strings.Join(nodegroup.Subnets, ",")),
			IAM:          display(nodegroup.NodeRoleARN),
			Issues:       healthIssues(nodegroup.Issues),
		})
	}
	return rows
}

func addonIAM(addon inventory.Addon) string {
	parts := []string{}
	if addon.ServiceAccountRoleARN != "" {
		parts = append(parts, "IRSA:"+addon.ServiceAccountRoleARN)
	}
	if len(addon.PodIdentityAssociations) > 0 {
		parts = append(parts, fmt.Sprintf("PodIdentity=%d", len(addon.PodIdentityAssociations)))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " ")
}

func display(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func sortedEKSInsights(items []inventory.EKSInsight) []inventory.EKSInsight {
	out := append([]inventory.EKSInsight(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		left := strings.Join([]string{out[i].Category, out[i].Status, out[i].Name}, "\x00")
		right := strings.Join([]string{out[j].Category, out[j].Status, out[j].Name}, "\x00")
		return left < right
	})
	return out
}

func sortedAddons(items []inventory.Addon) []inventory.Addon {
	out := append([]inventory.Addon(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedNodegroups(items []inventory.Nodegroup) []inventory.Nodegroup {
	out := append([]inventory.Nodegroup(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
