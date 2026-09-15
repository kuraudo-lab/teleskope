package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

type EKSProjection struct {
	Visible            bool                       `json:"visible"`
	Overview           []FieldValueRow            `json:"overview,omitempty"`
	Insights           []EKSInsightRow            `json:"insights,omitempty"`
	Capacity           []EKSCapacityRow           `json:"capacity,omitempty"`
	Network            []FieldValueRow            `json:"network,omitempty"`
	Security           []FieldValueRow            `json:"security,omitempty"`
	AccessEntries      []AccessEntryRow           `json:"accessEntries,omitempty"`
	PodIdentities      []PodIdentityRow           `json:"podIdentities,omitempty"`
	Addons             []EKSAddonRow              `json:"addons,omitempty"`
	Nodegroups         []EKSNodegroupRow          `json:"nodegroups,omitempty"`
	NodegroupReadiness []EKSNodegroupReadinessRow `json:"nodegroupReadiness,omitempty"`
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
	Name               string `json:"name"`
	Version            string `json:"version"`
	Status             string `json:"status"`
	Namespace          string `json:"namespace"`
	TargetKubernetes   string `json:"targetKubernetes"`
	CompatibleVersions string `json:"compatibleVersions"`
	Upgrade            string `json:"upgrade"`
	IAM                string `json:"iam"`
	Issues             string `json:"issues"`
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

type EKSNodegroupReadinessRow struct {
	Group             string `json:"group"`
	Evidence          string `json:"evidence"`
	Zone              string `json:"zone"`
	ExpectedVersion   string `json:"expectedVersion"`
	ExpectedRelease   string `json:"expectedRelease"`
	ExpectedAMI       string `json:"expectedAmi"`
	LaunchTemplate    string `json:"launchTemplate"`
	Nodes             string `json:"nodes"`
	ObservedKubelet   string `json:"observedKubelet"`
	ObservedOS        string `json:"observedOs"`
	ObservedRuntime   string `json:"observedRuntime"`
	ObservedInstances string `json:"observedInstances"`
	Readiness         string `json:"readiness"`
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
	out.Addons = projectEKSAddons(snapshot.EKS.Addons, snapshot.EKS.Insights)
	out.Nodegroups = projectEKSNodegroups(snapshot.EKS.Nodegroups)
	out.NodegroupReadiness = projectEKSNodegroupReadiness(snapshot.EKS.Nodegroups, snapshot.Kubernetes.Nodes)
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

func projectEKSAddons(addons []inventory.Addon, insights []inventory.EKSInsight) []EKSAddonRow {
	compatibility := addonCompatibilityByName(insights)
	rows := make([]EKSAddonRow, 0, len(addons))
	for _, addon := range sortedAddons(addons) {
		compat := compatibility[addon.Name]
		rows = append(rows, EKSAddonRow{
			Name:               display(addon.Name),
			Version:            display(addon.Version),
			Status:             display(addon.Status),
			Namespace:          display(addon.Namespace),
			TargetKubernetes:   display(strings.Join(compat.TargetKubernetes, ",")),
			CompatibleVersions: display(strings.Join(compat.CompatibleVersions, ",")),
			Upgrade:            addonUpgrade(addon.Version, compat),
			IAM:                addonIAM(addon),
			Issues:             healthIssues(addon.Issues),
		})
	}
	return rows
}

type addonCompatibility struct {
	TargetKubernetes   []string
	CompatibleVersions []string
	Statuses           []string
	Reasons            []string
	Recommendations    []string
}

func addonCompatibilityByName(insights []inventory.EKSInsight) map[string]addonCompatibility {
	out := map[string]addonCompatibility{}
	for _, insight := range sortedEKSInsights(insights) {
		for _, item := range insight.AddonCompatibility {
			if item.Name == "" {
				continue
			}
			current := out[item.Name]
			current.TargetKubernetes = appendUnique(current.TargetKubernetes, insight.KubernetesVersion)
			current.CompatibleVersions = appendUnique(current.CompatibleVersions, item.CompatibleVersions...)
			current.Statuses = appendUnique(current.Statuses, insight.Status)
			current.Reasons = appendUnique(current.Reasons, insight.Reason)
			current.Recommendations = appendUnique(current.Recommendations, trimMarkdownText(insight.Recommendation))
			out[item.Name] = current
		}
	}
	for name, current := range out {
		sort.Strings(current.TargetKubernetes)
		sort.Strings(current.CompatibleVersions)
		sort.Strings(current.Statuses)
		sort.Strings(current.Reasons)
		sort.Strings(current.Recommendations)
		out[name] = current
	}
	return out
}

func addonUpgrade(currentVersion string, compat addonCompatibility) string {
	if len(compat.CompatibleVersions) == 0 && len(compat.Statuses) == 0 && len(compat.Reasons) == 0 {
		return "-"
	}
	parts := []string{}
	if len(compat.Statuses) > 0 {
		parts = append(parts, "status="+strings.Join(compat.Statuses, ","))
	}
	if len(compat.CompatibleVersions) > 0 {
		if has(compat.CompatibleVersions, currentVersion) {
			parts = append(parts, "current version listed compatible")
		} else {
			parts = append(parts, "use "+strings.Join(compat.CompatibleVersions, ","))
		}
	}
	if len(compat.Reasons) > 0 {
		parts = append(parts, "reason="+strings.Join(compat.Reasons, "; "))
	}
	if len(compat.Recommendations) > 0 {
		parts = append(parts, "recommendation="+strings.Join(compat.Recommendations, "; "))
	}
	return strings.Join(parts, " ")
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

type nodegroupReadinessBucket struct {
	Group             string
	Evidence          string
	Zone              string
	ObservedKubelet   string
	ObservedOS        string
	ObservedRuntime   string
	ObservedInstances string
	Nodes             []string
}

func projectEKSNodegroupReadiness(nodegroups []inventory.Nodegroup, nodes []inventory.Node) []EKSNodegroupReadinessRow {
	if len(nodegroups) == 0 && len(nodes) == 0 {
		return nil
	}
	managed := map[string]inventory.Nodegroup{}
	for _, nodegroup := range nodegroups {
		if nodegroup.Name == "" {
			continue
		}
		managed[nodegroup.Name] = nodegroup
	}
	buckets := map[string]*nodegroupReadinessBucket{}
	seenManagedNodes := map[string]int{}
	for _, node := range sortedNodes(nodes) {
		group, evidence := nodeOwnership(node, managed)
		zone := node.Labels["topology.kubernetes.io/zone"]
		instanceType := node.Labels["node.kubernetes.io/instance-type"]
		key := strings.Join([]string{group, evidence, zone, node.KubeletVersion, node.OSImage, node.ContainerRuntime, instanceType}, "\x00")
		bucket := buckets[key]
		if bucket == nil {
			bucket = &nodegroupReadinessBucket{
				Group:             group,
				Evidence:          evidence,
				Zone:              zone,
				ObservedKubelet:   node.KubeletVersion,
				ObservedOS:        node.OSImage,
				ObservedRuntime:   node.ContainerRuntime,
				ObservedInstances: instanceType,
			}
			buckets[key] = bucket
		}
		bucket.Nodes = append(bucket.Nodes, node.Name)
		if _, ok := managed[group]; ok {
			seenManagedNodes[group]++
		}
	}
	rows := make([]EKSNodegroupReadinessRow, 0, len(buckets)+len(nodegroups))
	for _, bucket := range buckets {
		nodegroup, _ := managed[bucket.Group]
		rows = append(rows, EKSNodegroupReadinessRow{
			Group:             display(bucket.Group),
			Evidence:          display(bucket.Evidence),
			Zone:              display(bucket.Zone),
			ExpectedVersion:   display(nodegroup.Version),
			ExpectedRelease:   display(nodegroup.ReleaseVersion),
			ExpectedAMI:       display(nodegroup.AMIType),
			LaunchTemplate:    display(launchTemplateValue(nodegroup)),
			Nodes:             display(strings.Join(bucket.Nodes, ",")),
			ObservedKubelet:   display(bucket.ObservedKubelet),
			ObservedOS:        display(bucket.ObservedOS),
			ObservedRuntime:   display(bucket.ObservedRuntime),
			ObservedInstances: display(bucket.ObservedInstances),
			Readiness:         nodegroupReadinessValue(nodegroup, *bucket),
		})
	}
	for _, nodegroup := range sortedNodegroups(nodegroups) {
		if seenManagedNodes[nodegroup.Name] > 0 {
			continue
		}
		rows = append(rows, EKSNodegroupReadinessRow{
			Group:           display(nodegroup.Name),
			Evidence:        "managed-nodegroup/no-node-evidence",
			Zone:            "-",
			ExpectedVersion: display(nodegroup.Version),
			ExpectedRelease: display(nodegroup.ReleaseVersion),
			ExpectedAMI:     display(nodegroup.AMIType),
			LaunchTemplate:  display(launchTemplateValue(nodegroup)),
			Nodes:           "0",
			ObservedKubelet: "-",
			ObservedOS:      "-",
			ObservedRuntime: "-",
			Readiness:       "unknown: no Kubernetes node evidence for managed nodegroup",
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		left := strings.Join([]string{rows[i].Group, rows[i].Zone, rows[i].ObservedOS, rows[i].ObservedKubelet}, "\x00")
		right := strings.Join([]string{rows[j].Group, rows[j].Zone, rows[j].ObservedOS, rows[j].ObservedKubelet}, "\x00")
		return left < right
	})
	return rows
}

func nodeOwnership(node inventory.Node, managed map[string]inventory.Nodegroup) (string, string) {
	if name := strings.TrimSpace(node.Labels["eks.amazonaws.com/nodegroup"]); name != "" {
		if _, ok := managed[name]; ok {
			return name, "managed-nodegroup"
		}
		return name, "unknown: nodegroup label has no matching EKS managed nodegroup"
	}
	if nodepool := strings.TrimSpace(node.Labels["karpenter.sh/nodepool"]); nodepool != "" {
		return "karpenter/" + nodepool, "unknown: Karpenter nodepool"
	}
	if provisioner := strings.TrimSpace(node.Labels["karpenter.sh/provisioner-name"]); provisioner != "" {
		return "karpenter/" + provisioner, "unknown: Karpenter provisioner"
	}
	if computeType := strings.TrimSpace(node.Labels["eks.amazonaws.com/compute-type"]); computeType != "" {
		return computeType, "unknown: EKS compute type without managed nodegroup evidence"
	}
	return "self-managed/unknown", "unknown: no managed nodegroup evidence"
}

func launchTemplateValue(nodegroup inventory.Nodegroup) string {
	parts := []string{}
	if nodegroup.LaunchTemplateName != "" {
		parts = append(parts, nodegroup.LaunchTemplateName)
	} else if nodegroup.LaunchTemplateID != "" {
		parts = append(parts, nodegroup.LaunchTemplateID)
	}
	if nodegroup.LaunchTemplateVersion != "" {
		parts = append(parts, "version="+nodegroup.LaunchTemplateVersion)
	}
	return strings.Join(parts, " ")
}

func nodegroupReadinessValue(nodegroup inventory.Nodegroup, bucket nodegroupReadinessBucket) string {
	if !strings.HasPrefix(bucket.Evidence, "managed-nodegroup") {
		return bucket.Evidence
	}
	if strings.Contains(strings.ToUpper(nodegroup.AMIType), "CUSTOM") {
		return "unknown: custom AMI requires node runtime verification"
	}
	if nodegroup.Status != "" && nodegroup.Status != "ACTIVE" {
		return "check: managed nodegroup status=" + nodegroup.Status
	}
	if nodegroup.Version != "" && bucket.ObservedKubelet != "" && !strings.HasPrefix(strings.TrimPrefix(bucket.ObservedKubelet, "v"), nodegroup.Version+".") && strings.TrimPrefix(bucket.ObservedKubelet, "v") != nodegroup.Version {
		return "check: kubelet differs from expected Kubernetes version"
	}
	return "observed: runtime evidence linked to managed nodegroup"
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

func appendUnique(values []string, additions ...string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			seen[value] = struct{}{}
		}
	}
	for _, value := range additions {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		values = append(values, value)
		seen[value] = struct{}{}
	}
	return values
}

func has(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
