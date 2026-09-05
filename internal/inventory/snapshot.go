// Package inventory defines the provider-neutral snapshot model.
package inventory

import "time"

// Snapshot is a point-in-time inventory document.
type Snapshot struct {
	SchemaVersion string         `json:"schemaVersion"`
	CollectedAt   time.Time      `json:"collectedAt"`
	Source        Source         `json:"source"`
	AWS           AWSIdentity    `json:"aws"`
	EKS           EKSInventory   `json:"eks"`
	Coverage      []CoverageItem `json:"coverage"`
}

// Source records how the snapshot was produced.
type Source struct {
	Tool    string `json:"tool"`
	Version string `json:"version"`
	Mode    string `json:"mode"`
}

// AWSIdentity is the effective AWS identity used for collection.
type AWSIdentity struct {
	Profile   string `json:"profile,omitempty"`
	Region    string `json:"region"`
	AccountID string `json:"accountId,omitempty"`
	ARN       string `json:"arn,omitempty"`
	UserID    string `json:"userId,omitempty"`
}

// EKSInventory contains AWS-side EKS inventory facts.
type EKSInventory struct {
	Cluster                 Cluster                  `json:"cluster"`
	Addons                  []Addon                  `json:"addons"`
	Nodegroups              []Nodegroup              `json:"nodegroups"`
	AccessEntries           []AccessEntry            `json:"accessEntries"`
	PodIdentityAssociations []PodIdentityAssociation `json:"podIdentityAssociations"`
}

// Cluster summarizes the EKS cluster control plane and AWS-owned settings.
type Cluster struct {
	Name                         string             `json:"name"`
	ARN                          string             `json:"arn,omitempty"`
	Version                      string             `json:"version,omitempty"`
	PlatformVersion              string             `json:"platformVersion,omitempty"`
	Status                       string             `json:"status,omitempty"`
	Endpoint                     string             `json:"endpoint,omitempty"`
	RoleARN                      string             `json:"roleArn,omitempty"`
	CreatedAt                    *time.Time         `json:"createdAt,omitempty"`
	VPC                          VPCConfig          `json:"vpc"`
	Network                      NetworkConfig      `json:"network"`
	AccessConfig                 AccessConfig       `json:"accessConfig"`
	AutoMode                     AutoModeConfig     `json:"autoMode"`
	EnabledControlPlaneLogTypes  []string           `json:"enabledControlPlaneLogTypes,omitempty"`
	DisabledControlPlaneLogTypes []string           `json:"disabledControlPlaneLogTypes,omitempty"`
	Encryption                   []EncryptionConfig `json:"encryption,omitempty"`
	OIDCIssuer                   string             `json:"oidcIssuer,omitempty"`
	Tags                         map[string]string  `json:"tags,omitempty"`
}

// VPCConfig records cluster networking facts visible through EKS.
type VPCConfig struct {
	VPCID                  string   `json:"vpcId,omitempty"`
	SubnetIDs              []string `json:"subnetIds,omitempty"`
	SecurityGroupIDs       []string `json:"securityGroupIds,omitempty"`
	ClusterSecurityGroupID string   `json:"clusterSecurityGroupId,omitempty"`
	EndpointPublicAccess   bool     `json:"endpointPublicAccess"`
	EndpointPrivateAccess  bool     `json:"endpointPrivateAccess"`
	PublicAccessCIDRs      []string `json:"publicAccessCidrs,omitempty"`
	ControlPlaneEgressMode string   `json:"controlPlaneEgressMode,omitempty"`
}

// NetworkConfig records Kubernetes network settings exposed by EKS.
type NetworkConfig struct {
	IPFamily                     string `json:"ipFamily,omitempty"`
	ServiceIPv4CIDR              string `json:"serviceIpv4Cidr,omitempty"`
	ServiceIPv6CIDR              string `json:"serviceIpv6Cidr,omitempty"`
	AutoModeLoadBalancingEnabled *bool  `json:"autoModeLoadBalancingEnabled,omitempty"`
}

// AccessConfig records EKS authentication mode facts.
type AccessConfig struct {
	AuthenticationMode                      string `json:"authenticationMode,omitempty"`
	BootstrapClusterCreatorAdminPermissions *bool  `json:"bootstrapClusterCreatorAdminPermissions,omitempty"`
}

// AutoModeConfig records EKS Auto Mode facts visible through DescribeCluster.
type AutoModeConfig struct {
	ComputeEnabled      *bool    `json:"computeEnabled,omitempty"`
	ComputeNodePools    []string `json:"computeNodePools,omitempty"`
	ComputeNodeRoleARN  string   `json:"computeNodeRoleArn,omitempty"`
	BlockStorageEnabled *bool    `json:"blockStorageEnabled,omitempty"`
}

// EncryptionConfig records configured envelope encryption resources.
type EncryptionConfig struct {
	Resources []string `json:"resources,omitempty"`
	KeyARN    string   `json:"keyArn,omitempty"`
}

// Addon records a managed EKS add-on.
type Addon struct {
	Name                    string            `json:"name"`
	ARN                     string            `json:"arn,omitempty"`
	Version                 string            `json:"version,omitempty"`
	Status                  string            `json:"status,omitempty"`
	Owner                   string            `json:"owner,omitempty"`
	Publisher               string            `json:"publisher,omitempty"`
	Namespace               string            `json:"namespace,omitempty"`
	ServiceAccountRoleARN   string            `json:"serviceAccountRoleArn,omitempty"`
	PodIdentityAssociations []string          `json:"podIdentityAssociations,omitempty"`
	ConfigurationValues     string            `json:"configurationValues,omitempty"`
	Issues                  []HealthIssue     `json:"issues,omitempty"`
	Tags                    map[string]string `json:"tags,omitempty"`
}

// Nodegroup records an EKS managed node group.
type Nodegroup struct {
	Name                        string            `json:"name"`
	ARN                         string            `json:"arn,omitempty"`
	Version                     string            `json:"version,omitempty"`
	ReleaseVersion              string            `json:"releaseVersion,omitempty"`
	Status                      string            `json:"status,omitempty"`
	AMIType                     string            `json:"amiType,omitempty"`
	CapacityType                string            `json:"capacityType,omitempty"`
	NodeRoleARN                 string            `json:"nodeRoleArn,omitempty"`
	Subnets                     []string          `json:"subnets,omitempty"`
	InstanceTypes               []string          `json:"instanceTypes,omitempty"`
	DiskSizeGiB                 *int32            `json:"diskSizeGiB,omitempty"`
	DesiredSize                 *int32            `json:"desiredSize,omitempty"`
	MinSize                     *int32            `json:"minSize,omitempty"`
	MaxSize                     *int32            `json:"maxSize,omitempty"`
	Labels                      map[string]string `json:"labels,omitempty"`
	Taints                      []Taint           `json:"taints,omitempty"`
	AutoScalingGroups           []string          `json:"autoScalingGroups,omitempty"`
	RemoteAccessSecurityGroupID string            `json:"remoteAccessSecurityGroupId,omitempty"`
	LaunchTemplateID            string            `json:"launchTemplateId,omitempty"`
	LaunchTemplateName          string            `json:"launchTemplateName,omitempty"`
	LaunchTemplateVersion       string            `json:"launchTemplateVersion,omitempty"`
	Issues                      []HealthIssue     `json:"issues,omitempty"`
	Tags                        map[string]string `json:"tags,omitempty"`
}

// Taint records a node taint configured through EKS.
type Taint struct {
	Key    string `json:"key,omitempty"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect,omitempty"`
}

// AccessEntry records EKS API authentication entries and associated policies.
type AccessEntry struct {
	PrincipalARN     string             `json:"principalArn"`
	ARN              string             `json:"arn,omitempty"`
	Type             string             `json:"type,omitempty"`
	Username         string             `json:"username,omitempty"`
	KubernetesGroups []string           `json:"kubernetesGroups,omitempty"`
	Policies         []AssociatedPolicy `json:"policies,omitempty"`
	Tags             map[string]string  `json:"tags,omitempty"`
}

// AssociatedPolicy records an EKS access policy association.
type AssociatedPolicy struct {
	PolicyARN       string   `json:"policyArn,omitempty"`
	ScopeType       string   `json:"scopeType,omitempty"`
	ScopeNamespaces []string `json:"scopeNamespaces,omitempty"`
}

// PodIdentityAssociation records EKS Pod Identity associations.
type PodIdentityAssociation struct {
	ID                 string            `json:"id,omitempty"`
	ARN                string            `json:"arn,omitempty"`
	Namespace          string            `json:"namespace,omitempty"`
	ServiceAccount     string            `json:"serviceAccount,omitempty"`
	RoleARN            string            `json:"roleArn,omitempty"`
	TargetRoleARN      string            `json:"targetRoleArn,omitempty"`
	OwnerARN           string            `json:"ownerArn,omitempty"`
	DisableSessionTags *bool             `json:"disableSessionTags,omitempty"`
	Tags               map[string]string `json:"tags,omitempty"`
}

// HealthIssue records health issues surfaced by EKS.
type HealthIssue struct {
	Code        string   `json:"code,omitempty"`
	Message     string   `json:"message,omitempty"`
	ResourceIDs []string `json:"resourceIds,omitempty"`
}

// CoverageItem records collection completeness for a data source.
type CoverageItem struct {
	Area        string    `json:"area"`
	Resource    string    `json:"resource"`
	Status      string    `json:"status"`
	ObjectCount int       `json:"objectCount"`
	Reason      string    `json:"reason,omitempty"`
	CollectedAt time.Time `json:"collectedAt"`
}
