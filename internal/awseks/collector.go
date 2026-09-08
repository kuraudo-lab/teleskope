// Package awseks collects EKS inventory through AWS SDK for Go v2.
package awseks

import (
	"context"
	"fmt"
	"os"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/kuraudo-lab/teleskope/internal/buildinfo"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

// Options controls EKS collection.
type Options struct {
	ClusterName string
	Profile     string
	Region      string
	Progress    func(format string, args ...any)
}

// Collect gathers AWS-side EKS inventory using the default AWS CLI config chain.
func Collect(ctx context.Context, opts Options) (*inventory.Snapshot, error) {
	if opts.ClusterName == "" {
		return nil, fmt.Errorf("cluster name is required")
	}

	cfg, err := loadConfig(ctx, opts)
	if err != nil {
		return nil, err
	}
	progress(opts.Progress, "loaded AWS config profile=%s region=%s", valueOrDefault(effectiveProfile(opts.Profile), "default"), valueOrDefault(cfg.Region, "-"))
	if cfg.Region == "" {
		return nil, fmt.Errorf("AWS region is not configured; set AWS_REGION, AWS_DEFAULT_REGION, shared config region, or --region")
	}

	now := time.Now().UTC()
	client := eks.NewFromConfig(cfg)
	snapshot := &inventory.Snapshot{
		SchemaVersion: "teleskope.io/snapshot/v1alpha1",
		CollectedAt:   now,
		Source: inventory.Source{
			Tool:    "teleskope",
			Version: buildinfo.Version,
			Mode:    "out-of-cluster/run-once",
		},
		AWS: inventory.AWSIdentity{
			Profile: effectiveProfile(opts.Profile),
			Region:  cfg.Region,
		},
	}

	progress(opts.Progress, "checking AWS caller identity")
	if err := collectCallerIdentity(ctx, cfg, snapshot, now); err != nil {
		snapshot.Coverage = append(snapshot.Coverage, denied("aws", "sts:GetCallerIdentity", err, now))
		progress(opts.Progress, "AWS caller identity unavailable: %s", err)
	} else {
		snapshot.Coverage = append(snapshot.Coverage, complete("aws", "sts:GetCallerIdentity", 1, now))
		progress(opts.Progress, "AWS caller identity account=%s", valueOrDefault(snapshot.AWS.AccountID, "-"))
	}

	progress(opts.Progress, "describing EKS cluster %s", opts.ClusterName)
	cluster, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: awssdk.String(opts.ClusterName)})
	if err != nil {
		return snapshot, fmt.Errorf("describe EKS cluster %q: %w", opts.ClusterName, err)
	}
	snapshot.EKS.Cluster = mapCluster(cluster.Cluster)
	snapshot.Coverage = append(snapshot.Coverage, complete("eks", "DescribeCluster", 1, now))
	progress(opts.Progress, "cluster status=%s version=%s platform=%s", valueOrDefault(snapshot.EKS.Cluster.Status, "-"), valueOrDefault(snapshot.EKS.Cluster.Version, "-"), valueOrDefault(snapshot.EKS.Cluster.PlatformVersion, "-"))

	progress(opts.Progress, "collecting managed add-ons")
	snapshot.EKS.Addons, err = collectAddons(ctx, client, opts.ClusterName, now, &snapshot.Coverage)
	if err != nil {
		snapshot.Coverage = append(snapshot.Coverage, partial("eks", "Addons", len(snapshot.EKS.Addons), err, now))
	}
	progress(opts.Progress, "managed add-ons=%d", len(snapshot.EKS.Addons))

	progress(opts.Progress, "collecting managed nodegroups")
	snapshot.EKS.Nodegroups, err = collectNodegroups(ctx, client, opts.ClusterName, now, &snapshot.Coverage)
	if err != nil {
		snapshot.Coverage = append(snapshot.Coverage, partial("eks", "Nodegroups", len(snapshot.EKS.Nodegroups), err, now))
	}
	progress(opts.Progress, "managed nodegroups=%d", len(snapshot.EKS.Nodegroups))

	progress(opts.Progress, "collecting upgrade and rollback insights")
	snapshot.EKS.Insights, err = collectInsights(ctx, client, opts.ClusterName, now, &snapshot.Coverage)
	if err != nil {
		snapshot.Coverage = append(snapshot.Coverage, partial("eks", "Insights", len(snapshot.EKS.Insights), err, now))
	}
	progress(opts.Progress, "insights=%d", len(snapshot.EKS.Insights))

	progress(opts.Progress, "collecting access entries")
	snapshot.EKS.AccessEntries, err = collectAccessEntries(ctx, client, opts.ClusterName, now, &snapshot.Coverage)
	if err != nil {
		snapshot.Coverage = append(snapshot.Coverage, partial("eks", "AccessEntries", len(snapshot.EKS.AccessEntries), err, now))
	}
	progress(opts.Progress, "access entries=%d", len(snapshot.EKS.AccessEntries))

	progress(opts.Progress, "collecting pod identity associations")
	snapshot.EKS.PodIdentityAssociations, err = collectPodIdentities(ctx, client, opts.ClusterName, now, &snapshot.Coverage)
	if err != nil {
		snapshot.Coverage = append(snapshot.Coverage, partial("eks", "PodIdentityAssociations", len(snapshot.EKS.PodIdentityAssociations), err, now))
	}
	progress(opts.Progress, "pod identity associations=%d", len(snapshot.EKS.PodIdentityAssociations))

	return snapshot, nil
}

func progress(fn func(format string, args ...any), format string, args ...any) {
	if fn != nil {
		fn(format, args...)
	}
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func effectiveProfile(profile string) string {
	if profile != "" {
		return profile
	}
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		return profile
	}
	return os.Getenv("AWS_DEFAULT_PROFILE")
}

func loadConfig(ctx context.Context, opts Options) (awssdk.Config, error) {
	loaders := []func(*config.LoadOptions) error{}
	if opts.Profile != "" {
		loaders = append(loaders, config.WithSharedConfigProfile(opts.Profile))
	}
	if opts.Region != "" {
		loaders = append(loaders, config.WithRegion(opts.Region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, loaders...)
	if err != nil {
		return awssdk.Config{}, fmt.Errorf("load AWS CLI config: %w", err)
	}
	return cfg, nil
}

func collectCallerIdentity(ctx context.Context, cfg awssdk.Config, snapshot *inventory.Snapshot, now time.Time) error {
	out, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return err
	}
	snapshot.AWS.AccountID = awssdk.ToString(out.Account)
	snapshot.AWS.ARN = awssdk.ToString(out.Arn)
	snapshot.AWS.UserID = awssdk.ToString(out.UserId)
	return nil
}

func collectAddons(ctx context.Context, client *eks.Client, clusterName string, now time.Time, coverage *[]inventory.CoverageItem) ([]inventory.Addon, error) {
	var addons []inventory.Addon
	paginator := eks.NewListAddonsPaginator(client, &eks.ListAddonsInput{ClusterName: awssdk.String(clusterName)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return addons, err
		}
		for _, name := range page.Addons {
			out, err := client.DescribeAddon(ctx, &eks.DescribeAddonInput{
				AddonName:   awssdk.String(name),
				ClusterName: awssdk.String(clusterName),
			})
			if err != nil {
				return addons, err
			}
			addons = append(addons, mapAddon(out.Addon))
		}
	}
	*coverage = append(*coverage, complete("eks", "Addons", len(addons), now))
	return addons, nil
}

func collectNodegroups(ctx context.Context, client *eks.Client, clusterName string, now time.Time, coverage *[]inventory.CoverageItem) ([]inventory.Nodegroup, error) {
	var nodegroups []inventory.Nodegroup
	paginator := eks.NewListNodegroupsPaginator(client, &eks.ListNodegroupsInput{ClusterName: awssdk.String(clusterName)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nodegroups, err
		}
		for _, name := range page.Nodegroups {
			out, err := client.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
				ClusterName:   awssdk.String(clusterName),
				NodegroupName: awssdk.String(name),
			})
			if err != nil {
				return nodegroups, err
			}
			nodegroups = append(nodegroups, mapNodegroup(out.Nodegroup))
		}
	}
	*coverage = append(*coverage, complete("eks", "Nodegroups", len(nodegroups), now))
	return nodegroups, nil
}

func collectInsights(ctx context.Context, client *eks.Client, clusterName string, now time.Time, coverage *[]inventory.CoverageItem) ([]inventory.EKSInsight, error) {
	var insights []inventory.EKSInsight
	paginator := eks.NewListInsightsPaginator(client, &eks.ListInsightsInput{
		ClusterName: awssdk.String(clusterName),
		Filter: &ekstypes.InsightsFilter{Categories: []ekstypes.Category{
			ekstypes.CategoryUpgradeReadiness,
			ekstypes.CategoryRollbackReadiness,
		}},
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return insights, err
		}
		for _, summary := range page.Insights {
			if summary.Id == nil {
				insights = append(insights, mapInsightSummary(summary))
				continue
			}
			out, err := client.DescribeInsight(ctx, &eks.DescribeInsightInput{
				ClusterName: awssdk.String(clusterName),
				Id:          summary.Id,
			})
			if err != nil {
				insights = append(insights, mapInsightSummary(summary))
				return insights, err
			}
			insights = append(insights, mapInsight(out.Insight, summary))
		}
	}
	*coverage = append(*coverage, complete("eks", "Insights", len(insights), now))
	return insights, nil
}

func collectAccessEntries(ctx context.Context, client *eks.Client, clusterName string, now time.Time, coverage *[]inventory.CoverageItem) ([]inventory.AccessEntry, error) {
	var entries []inventory.AccessEntry
	paginator := eks.NewListAccessEntriesPaginator(client, &eks.ListAccessEntriesInput{ClusterName: awssdk.String(clusterName)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return entries, err
		}
		for _, principal := range page.AccessEntries {
			out, err := client.DescribeAccessEntry(ctx, &eks.DescribeAccessEntryInput{
				ClusterName:  awssdk.String(clusterName),
				PrincipalArn: awssdk.String(principal),
			})
			if err != nil {
				return entries, err
			}
			entry := mapAccessEntry(out.AccessEntry)
			policies, err := collectAccessPolicies(ctx, client, clusterName, principal)
			if err != nil {
				return entries, err
			}
			entry.Policies = policies
			entries = append(entries, entry)
		}
	}
	*coverage = append(*coverage, complete("eks", "AccessEntries", len(entries), now))
	return entries, nil
}

func collectAccessPolicies(ctx context.Context, client *eks.Client, clusterName, principal string) ([]inventory.AssociatedPolicy, error) {
	var policies []inventory.AssociatedPolicy
	paginator := eks.NewListAssociatedAccessPoliciesPaginator(client, &eks.ListAssociatedAccessPoliciesInput{
		ClusterName:  awssdk.String(clusterName),
		PrincipalArn: awssdk.String(principal),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return policies, err
		}
		for _, policy := range page.AssociatedAccessPolicies {
			policies = append(policies, mapAssociatedPolicy(policy))
		}
	}
	return policies, nil
}

func collectPodIdentities(ctx context.Context, client *eks.Client, clusterName string, now time.Time, coverage *[]inventory.CoverageItem) ([]inventory.PodIdentityAssociation, error) {
	var associations []inventory.PodIdentityAssociation
	paginator := eks.NewListPodIdentityAssociationsPaginator(client, &eks.ListPodIdentityAssociationsInput{ClusterName: awssdk.String(clusterName)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return associations, err
		}
		for _, summary := range page.Associations {
			out, err := client.DescribePodIdentityAssociation(ctx, &eks.DescribePodIdentityAssociationInput{
				ClusterName:   awssdk.String(clusterName),
				AssociationId: summary.AssociationId,
			})
			if err != nil {
				return associations, err
			}
			associations = append(associations, mapPodIdentity(out.Association))
		}
	}
	*coverage = append(*coverage, complete("eks", "PodIdentityAssociations", len(associations), now))
	return associations, nil
}

func complete(area, resource string, count int, at time.Time) inventory.CoverageItem {
	return inventory.CoverageItem{Area: area, Resource: resource, Status: "complete", ObjectCount: count, CollectedAt: at}
}

func partial(area, resource string, count int, err error, at time.Time) inventory.CoverageItem {
	return inventory.CoverageItem{Area: area, Resource: resource, Status: "partial", ObjectCount: count, Reason: err.Error(), CollectedAt: at}
}

func denied(area, resource string, err error, at time.Time) inventory.CoverageItem {
	return inventory.CoverageItem{Area: area, Resource: resource, Status: "denied", Reason: err.Error(), CollectedAt: at}
}

func mapCluster(cluster *ekstypes.Cluster) inventory.Cluster {
	if cluster == nil {
		return inventory.Cluster{}
	}
	out := inventory.Cluster{
		Name:            awssdk.ToString(cluster.Name),
		ARN:             awssdk.ToString(cluster.Arn),
		Version:         awssdk.ToString(cluster.Version),
		PlatformVersion: awssdk.ToString(cluster.PlatformVersion),
		Status:          string(cluster.Status),
		Endpoint:        awssdk.ToString(cluster.Endpoint),
		RoleARN:         awssdk.ToString(cluster.RoleArn),
		CreatedAt:       cluster.CreatedAt,
		Tags:            cluster.Tags,
	}
	if cluster.ResourcesVpcConfig != nil {
		out.VPC = inventory.VPCConfig{
			VPCID:                  awssdk.ToString(cluster.ResourcesVpcConfig.VpcId),
			SubnetIDs:              cluster.ResourcesVpcConfig.SubnetIds,
			SecurityGroupIDs:       cluster.ResourcesVpcConfig.SecurityGroupIds,
			ClusterSecurityGroupID: awssdk.ToString(cluster.ResourcesVpcConfig.ClusterSecurityGroupId),
			EndpointPublicAccess:   cluster.ResourcesVpcConfig.EndpointPublicAccess,
			EndpointPrivateAccess:  cluster.ResourcesVpcConfig.EndpointPrivateAccess,
			PublicAccessCIDRs:      cluster.ResourcesVpcConfig.PublicAccessCidrs,
			ControlPlaneEgressMode: string(cluster.ResourcesVpcConfig.ControlPlaneEgressMode),
		}
	}
	if cluster.KubernetesNetworkConfig != nil {
		out.Network = inventory.NetworkConfig{
			IPFamily:        string(cluster.KubernetesNetworkConfig.IpFamily),
			ServiceIPv4CIDR: awssdk.ToString(cluster.KubernetesNetworkConfig.ServiceIpv4Cidr),
			ServiceIPv6CIDR: awssdk.ToString(cluster.KubernetesNetworkConfig.ServiceIpv6Cidr),
		}
		if cluster.KubernetesNetworkConfig.ElasticLoadBalancing != nil {
			out.Network.AutoModeLoadBalancingEnabled = cluster.KubernetesNetworkConfig.ElasticLoadBalancing.Enabled
		}
	}
	if cluster.AccessConfig != nil {
		out.AccessConfig = inventory.AccessConfig{
			AuthenticationMode:                      string(cluster.AccessConfig.AuthenticationMode),
			BootstrapClusterCreatorAdminPermissions: cluster.AccessConfig.BootstrapClusterCreatorAdminPermissions,
		}
	}
	if cluster.ComputeConfig != nil {
		out.AutoMode.ComputeEnabled = cluster.ComputeConfig.Enabled
		out.AutoMode.ComputeNodePools = cluster.ComputeConfig.NodePools
		out.AutoMode.ComputeNodeRoleARN = awssdk.ToString(cluster.ComputeConfig.NodeRoleArn)
	}
	if cluster.StorageConfig != nil && cluster.StorageConfig.BlockStorage != nil {
		out.AutoMode.BlockStorageEnabled = cluster.StorageConfig.BlockStorage.Enabled
	}
	if cluster.Identity != nil && cluster.Identity.Oidc != nil {
		out.OIDCIssuer = awssdk.ToString(cluster.Identity.Oidc.Issuer)
	}
	if cluster.Logging != nil {
		for _, logSetup := range cluster.Logging.ClusterLogging {
			for _, logType := range logSetup.Types {
				if awssdk.ToBool(logSetup.Enabled) {
					out.EnabledControlPlaneLogTypes = append(out.EnabledControlPlaneLogTypes, string(logType))
				} else {
					out.DisabledControlPlaneLogTypes = append(out.DisabledControlPlaneLogTypes, string(logType))
				}
			}
		}
	}
	for _, enc := range cluster.EncryptionConfig {
		item := inventory.EncryptionConfig{Resources: enc.Resources}
		if enc.Provider != nil {
			item.KeyARN = awssdk.ToString(enc.Provider.KeyArn)
		}
		out.Encryption = append(out.Encryption, item)
	}
	return out
}

func mapAddon(addon *ekstypes.Addon) inventory.Addon {
	if addon == nil {
		return inventory.Addon{}
	}
	out := inventory.Addon{
		Name:                    awssdk.ToString(addon.AddonName),
		ARN:                     awssdk.ToString(addon.AddonArn),
		Version:                 awssdk.ToString(addon.AddonVersion),
		Status:                  string(addon.Status),
		Owner:                   awssdk.ToString(addon.Owner),
		Publisher:               awssdk.ToString(addon.Publisher),
		ServiceAccountRoleARN:   awssdk.ToString(addon.ServiceAccountRoleArn),
		PodIdentityAssociations: addon.PodIdentityAssociations,
		ConfigurationValues:     awssdk.ToString(addon.ConfigurationValues),
		Tags:                    addon.Tags,
	}
	if addon.NamespaceConfig != nil {
		out.Namespace = awssdk.ToString(addon.NamespaceConfig.Namespace)
	}
	if addon.Health != nil {
		for _, issue := range addon.Health.Issues {
			out.Issues = append(out.Issues, inventory.HealthIssue{
				Code:        string(issue.Code),
				Message:     awssdk.ToString(issue.Message),
				ResourceIDs: issue.ResourceIds,
			})
		}
	}
	return out
}

func mapNodegroup(nodegroup *ekstypes.Nodegroup) inventory.Nodegroup {
	if nodegroup == nil {
		return inventory.Nodegroup{}
	}
	out := inventory.Nodegroup{
		Name:           awssdk.ToString(nodegroup.NodegroupName),
		ARN:            awssdk.ToString(nodegroup.NodegroupArn),
		Version:        awssdk.ToString(nodegroup.Version),
		ReleaseVersion: awssdk.ToString(nodegroup.ReleaseVersion),
		Status:         string(nodegroup.Status),
		AMIType:        string(nodegroup.AmiType),
		CapacityType:   string(nodegroup.CapacityType),
		NodeRoleARN:    awssdk.ToString(nodegroup.NodeRole),
		Subnets:        nodegroup.Subnets,
		InstanceTypes:  nodegroup.InstanceTypes,
		DiskSizeGiB:    nodegroup.DiskSize,
		Labels:         nodegroup.Labels,
		Tags:           nodegroup.Tags,
	}
	if nodegroup.ScalingConfig != nil {
		out.DesiredSize = nodegroup.ScalingConfig.DesiredSize
		out.MinSize = nodegroup.ScalingConfig.MinSize
		out.MaxSize = nodegroup.ScalingConfig.MaxSize
	}
	for _, taint := range nodegroup.Taints {
		out.Taints = append(out.Taints, inventory.Taint{
			Key:    awssdk.ToString(taint.Key),
			Value:  awssdk.ToString(taint.Value),
			Effect: string(taint.Effect),
		})
	}
	if nodegroup.Resources != nil {
		out.RemoteAccessSecurityGroupID = awssdk.ToString(nodegroup.Resources.RemoteAccessSecurityGroup)
		for _, asg := range nodegroup.Resources.AutoScalingGroups {
			out.AutoScalingGroups = append(out.AutoScalingGroups, awssdk.ToString(asg.Name))
		}
	}
	if nodegroup.LaunchTemplate != nil {
		out.LaunchTemplateID = awssdk.ToString(nodegroup.LaunchTemplate.Id)
		out.LaunchTemplateName = awssdk.ToString(nodegroup.LaunchTemplate.Name)
		out.LaunchTemplateVersion = awssdk.ToString(nodegroup.LaunchTemplate.Version)
	}
	if nodegroup.Health != nil {
		for _, issue := range nodegroup.Health.Issues {
			out.Issues = append(out.Issues, inventory.HealthIssue{
				Code:        string(issue.Code),
				Message:     awssdk.ToString(issue.Message),
				ResourceIDs: issue.ResourceIds,
			})
		}
	}
	return out
}

func mapInsightSummary(summary ekstypes.InsightSummary) inventory.EKSInsight {
	out := inventory.EKSInsight{
		ID:                 awssdk.ToString(summary.Id),
		Name:               awssdk.ToString(summary.Name),
		Category:           string(summary.Category),
		KubernetesVersion:  awssdk.ToString(summary.KubernetesVersion),
		Description:        awssdk.ToString(summary.Description),
		LastRefreshTime:    summary.LastRefreshTime,
		LastTransitionTime: summary.LastTransitionTime,
	}
	if summary.InsightStatus != nil {
		out.Status = string(summary.InsightStatus.Status)
		out.Reason = awssdk.ToString(summary.InsightStatus.Reason)
	}
	return out
}

func mapInsight(insight *ekstypes.Insight, fallback ekstypes.InsightSummary) inventory.EKSInsight {
	if insight == nil {
		return mapInsightSummary(fallback)
	}
	out := inventory.EKSInsight{
		ID:                 awssdk.ToString(insight.Id),
		Name:               awssdk.ToString(insight.Name),
		Category:           string(insight.Category),
		KubernetesVersion:  awssdk.ToString(insight.KubernetesVersion),
		Description:        awssdk.ToString(insight.Description),
		Recommendation:     awssdk.ToString(insight.Recommendation),
		LastRefreshTime:    insight.LastRefreshTime,
		LastTransitionTime: insight.LastTransitionTime,
		AdditionalInfo:     insight.AdditionalInfo,
	}
	if out.ID == "" {
		out.ID = awssdk.ToString(fallback.Id)
	}
	if out.Name == "" {
		out.Name = awssdk.ToString(fallback.Name)
	}
	if out.Category == "" {
		out.Category = string(fallback.Category)
	}
	if out.KubernetesVersion == "" {
		out.KubernetesVersion = awssdk.ToString(fallback.KubernetesVersion)
	}
	if out.Description == "" {
		out.Description = awssdk.ToString(fallback.Description)
	}
	if insight.InsightStatus != nil {
		out.Status = string(insight.InsightStatus.Status)
		out.Reason = awssdk.ToString(insight.InsightStatus.Reason)
	} else if fallback.InsightStatus != nil {
		out.Status = string(fallback.InsightStatus.Status)
		out.Reason = awssdk.ToString(fallback.InsightStatus.Reason)
	}
	for _, resource := range insight.Resources {
		item := inventory.EKSInsightResource{
			ARN:                   awssdk.ToString(resource.Arn),
			KubernetesResourceURI: awssdk.ToString(resource.KubernetesResourceUri),
		}
		if resource.InsightStatus != nil {
			item.Status = string(resource.InsightStatus.Status)
			item.Reason = awssdk.ToString(resource.InsightStatus.Reason)
		}
		out.Resources = append(out.Resources, item)
	}
	if insight.CategorySpecificSummary != nil {
		for _, detail := range insight.CategorySpecificSummary.AddonCompatibilityDetails {
			out.AddonCompatibility = append(out.AddonCompatibility, inventory.AddonCompatibility{
				Name:               awssdk.ToString(detail.Name),
				CompatibleVersions: detail.CompatibleVersions,
			})
		}
		for _, detail := range insight.CategorySpecificSummary.DeprecationDetails {
			item := inventory.DeprecationDetail{
				Usage:                          awssdk.ToString(detail.Usage),
				ReplacedWith:                   awssdk.ToString(detail.ReplacedWith),
				StartServingReplacementVersion: awssdk.ToString(detail.StartServingReplacementVersion),
				StopServingVersion:             awssdk.ToString(detail.StopServingVersion),
			}
			for _, stat := range detail.ClientStats {
				item.UserAgents = append(item.UserAgents, awssdk.ToString(stat.UserAgent))
			}
			out.DeprecationDetails = append(out.DeprecationDetails, item)
		}
	}
	return out
}

func mapAccessEntry(entry *ekstypes.AccessEntry) inventory.AccessEntry {
	if entry == nil {
		return inventory.AccessEntry{}
	}
	return inventory.AccessEntry{
		PrincipalARN:     awssdk.ToString(entry.PrincipalArn),
		ARN:              awssdk.ToString(entry.AccessEntryArn),
		Type:             awssdk.ToString(entry.Type),
		Username:         awssdk.ToString(entry.Username),
		KubernetesGroups: entry.KubernetesGroups,
		Tags:             entry.Tags,
	}
}

func mapAssociatedPolicy(policy ekstypes.AssociatedAccessPolicy) inventory.AssociatedPolicy {
	out := inventory.AssociatedPolicy{PolicyARN: awssdk.ToString(policy.PolicyArn)}
	if policy.AccessScope != nil {
		out.ScopeType = string(policy.AccessScope.Type)
		out.ScopeNamespaces = policy.AccessScope.Namespaces
	}
	return out
}

func mapPodIdentity(association *ekstypes.PodIdentityAssociation) inventory.PodIdentityAssociation {
	if association == nil {
		return inventory.PodIdentityAssociation{}
	}
	return inventory.PodIdentityAssociation{
		ID:                 awssdk.ToString(association.AssociationId),
		ARN:                awssdk.ToString(association.AssociationArn),
		Namespace:          awssdk.ToString(association.Namespace),
		ServiceAccount:     awssdk.ToString(association.ServiceAccount),
		RoleARN:            awssdk.ToString(association.RoleArn),
		TargetRoleARN:      awssdk.ToString(association.TargetRoleArn),
		OwnerARN:           awssdk.ToString(association.OwnerArn),
		DisableSessionTags: association.DisableSessionTags,
		Tags:               association.Tags,
	}
}
