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
	Kubernetes    Kubernetes     `json:"kubernetes"`
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
	Insights                []EKSInsight             `json:"insights,omitempty"`
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

// EKSInsight records an EKS insight, including upgrade and rollback readiness findings.
type EKSInsight struct {
	ID                 string               `json:"id,omitempty"`
	Name               string               `json:"name,omitempty"`
	Category           string               `json:"category,omitempty"`
	KubernetesVersion  string               `json:"kubernetesVersion,omitempty"`
	Status             string               `json:"status,omitempty"`
	Reason             string               `json:"reason,omitempty"`
	Description        string               `json:"description,omitempty"`
	Recommendation     string               `json:"recommendation,omitempty"`
	LastRefreshTime    *time.Time           `json:"lastRefreshTime,omitempty"`
	LastTransitionTime *time.Time           `json:"lastTransitionTime,omitempty"`
	AdditionalInfo     map[string]string    `json:"additionalInfo,omitempty"`
	Resources          []EKSInsightResource `json:"resources,omitempty"`
	AddonCompatibility []AddonCompatibility `json:"addonCompatibility,omitempty"`
	DeprecationDetails []DeprecationDetail  `json:"deprecationDetails,omitempty"`
}

// EKSInsightResource records a resource evaluated by an EKS insight.
type EKSInsightResource struct {
	ARN                   string `json:"arn,omitempty"`
	KubernetesResourceURI string `json:"kubernetesResourceUri,omitempty"`
	Status                string `json:"status,omitempty"`
	Reason                string `json:"reason,omitempty"`
}

// AddonCompatibility records add-on compatibility details returned by EKS insights.
type AddonCompatibility struct {
	Name               string   `json:"name,omitempty"`
	CompatibleVersions []string `json:"compatibleVersions,omitempty"`
}

// DeprecationDetail records deprecated Kubernetes API usage details returned by EKS insights.
type DeprecationDetail struct {
	Usage                          string   `json:"usage,omitempty"`
	ReplacedWith                   string   `json:"replacedWith,omitempty"`
	StartServingReplacementVersion string   `json:"startServingReplacementVersion,omitempty"`
	StopServingVersion             string   `json:"stopServingVersion,omitempty"`
	UserAgents                     []string `json:"userAgents,omitempty"`
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

// Kubernetes contains Kubernetes API inventory facts.
type Kubernetes struct {
	Context                   string                     `json:"context,omitempty"`
	Server                    string                     `json:"server,omitempty"`
	Version                   KubernetesVersion          `json:"version,omitempty"`
	APIResources              []APIResource              `json:"apiResources,omitempty"`
	CustomResourceDefinitions []CustomResourceDefinition `json:"customResourceDefinitions,omitempty"`
	CustomResourceInstances   []CustomResourceInstance   `json:"customResourceInstances,omitempty"`
	CustomResourceCounts      []CustomResourceCount      `json:"customResourceCounts,omitempty"`
	APIServices               []APIService               `json:"apiServices,omitempty"`
	Namespaces                []Namespace                `json:"namespaces,omitempty"`
	Nodes                     []Node                     `json:"nodes,omitempty"`
	ServiceAccounts           []ServiceAccount           `json:"serviceAccounts,omitempty"`
	Workloads                 []Workload                 `json:"workloads,omitempty"`
	Pods                      []Pod                      `json:"pods,omitempty"`
	RunningImages             []RunningImage             `json:"runningImages,omitempty"`
	RunningContainers         []RunningContainer         `json:"runningContainers,omitempty"`
	Services                  []Service                  `json:"services,omitempty"`
	EndpointSlices            []EndpointSlice            `json:"endpointSlices,omitempty"`
	IngressClasses            []IngressClass             `json:"ingressClasses,omitempty"`
	Ingresses                 []Ingress                  `json:"ingresses,omitempty"`
	GatewayClasses            []GatewayClass             `json:"gatewayClasses,omitempty"`
	Gateways                  []Gateway                  `json:"gateways,omitempty"`
	GatewayRoutes             []GatewayRoute             `json:"gatewayRoutes,omitempty"`
	ReferenceGrants           []ReferenceGrant           `json:"referenceGrants,omitempty"`
	GatewayPolicies           []GatewayPolicy            `json:"gatewayPolicies,omitempty"`
	StorageClasses            []StorageClass             `json:"storageClasses,omitempty"`
	PersistentVolumes         []PersistentVolume         `json:"persistentVolumes,omitempty"`
	PersistentVolumeClaims    []PersistentVolumeClaim    `json:"persistentVolumeClaims,omitempty"`
	CSIDrivers                []CSIDriver                `json:"csiDrivers,omitempty"`
	CSINodes                  []CSINode                  `json:"csiNodes,omitempty"`
	VolumeAttachments         []VolumeAttachment         `json:"volumeAttachments,omitempty"`
	RuntimeClasses            []RuntimeClass             `json:"runtimeClasses,omitempty"`
	ConfigMaps                []ConfigObject             `json:"configMaps,omitempty"`
	Secrets                   []Secret                   `json:"secrets,omitempty"`
	RBAC                      RBAC                       `json:"rbac,omitempty"`
	Policies                  Policies                   `json:"policies,omitempty"`
}

// KubernetesVersion records the Kubernetes API server version.
type KubernetesVersion struct {
	GitVersion string `json:"gitVersion,omitempty"`
	Major      string `json:"major,omitempty"`
	Minor      string `json:"minor,omitempty"`
	Platform   string `json:"platform,omitempty"`
}

// APIResource records a discoverable Kubernetes resource type.
type APIResource struct {
	GroupVersion string   `json:"groupVersion"`
	Group        string   `json:"group,omitempty"`
	Version      string   `json:"version,omitempty"`
	Resource     string   `json:"resource"`
	Kind         string   `json:"kind,omitempty"`
	Namespaced   bool     `json:"namespaced"`
	Verbs        []string `json:"verbs,omitempty"`
	Categories   []string `json:"categories,omitempty"`
}

// CustomResourceDefinition records CRD registration metadata.
type CustomResourceDefinition struct {
	ObjectRef
	Group    string       `json:"group,omitempty"`
	Scope    string       `json:"scope,omitempty"`
	Kind     string       `json:"kind,omitempty"`
	Plural   string       `json:"plural,omitempty"`
	Versions []CRDVersion `json:"versions,omitempty"`
}

// CRDVersion records one served CRD version.
type CRDVersion struct {
	Name    string `json:"name"`
	Served  bool   `json:"served"`
	Storage bool   `json:"storage"`
}

// CustomResourceInstance records one CRD-backed object and its CRD type.
type CustomResourceInstance struct {
	ObjectRef
	CRDName         string            `json:"crdName,omitempty"`
	CRDGroup        string            `json:"crdGroup,omitempty"`
	CRDVersion      string            `json:"crdVersion,omitempty"`
	CRDKind         string            `json:"crdKind,omitempty"`
	CRDPlural       string            `json:"crdPlural,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	OwnerReferences []ObjectRef       `json:"ownerReferences,omitempty"`
}

// CustomResourceCount records the number of instances observed for one CRD type.
type CustomResourceCount struct {
	CRDName        string `json:"crdName,omitempty"`
	Group          string `json:"group,omitempty"`
	Version        string `json:"version,omitempty"`
	Kind           string `json:"kind,omitempty"`
	Plural         string `json:"plural,omitempty"`
	Scope          string `json:"scope,omitempty"`
	InstanceCount  int    `json:"instanceCount"`
	NamespaceCount int    `json:"namespaceCount,omitempty"`
}

// APIService records aggregated APIService registration metadata.
type APIService struct {
	ObjectRef
	Group            string `json:"group,omitempty"`
	Version          string `json:"version,omitempty"`
	ServiceNamespace string `json:"serviceNamespace,omitempty"`
	ServiceName      string `json:"serviceName,omitempty"`
	Available        string `json:"available,omitempty"`
}

// ObjectRef identifies a Kubernetes object without embedding the full object.
type ObjectRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
	UID        string `json:"uid,omitempty"`
}

// Namespace records namespace metadata and lifecycle state.
type Namespace struct {
	ObjectRef
	Phase  string            `json:"phase,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

// Node records node facts visible through the Kubernetes API.
type Node struct {
	ObjectRef
	ProviderID          string            `json:"providerId,omitempty"`
	Unschedulable       bool              `json:"unschedulable,omitempty"`
	Labels              map[string]string `json:"labels,omitempty"`
	Taints              []Taint           `json:"taints,omitempty"`
	KubeletVersion      string            `json:"kubeletVersion,omitempty"`
	ContainerRuntime    string            `json:"containerRuntime,omitempty"`
	OSImage             string            `json:"osImage,omitempty"`
	KernelVersion       string            `json:"kernelVersion,omitempty"`
	Architecture        string            `json:"architecture,omitempty"`
	OperatingSystem     string            `json:"operatingSystem,omitempty"`
	Capacity            map[string]string `json:"capacity,omitempty"`
	Allocatable         map[string]string `json:"allocatable,omitempty"`
	Ready               string            `json:"ready,omitempty"`
	NodeLocalBlindSpots []string          `json:"nodeLocalBlindSpots,omitempty"`
}

// ServiceAccount records ServiceAccount metadata and identity annotations.
type ServiceAccount struct {
	ObjectRef
	Annotations      map[string]string `json:"annotations,omitempty"`
	ImagePullSecrets []string          `json:"imagePullSecrets,omitempty"`
	Secrets          []string          `json:"secrets,omitempty"`
}

// Workload records a controller and its pod template summary.
type Workload struct {
	ObjectRef
	Replicas             *int32            `json:"replicas,omitempty"`
	ReadyReplicas        int32             `json:"readyReplicas,omitempty"`
	AvailableReplicas    int32             `json:"availableReplicas,omitempty"`
	Selector             map[string]string `json:"selector,omitempty"`
	ServiceAccountName   string            `json:"serviceAccountName,omitempty"`
	RuntimeClassName     string            `json:"runtimeClassName,omitempty"`
	NodeSelector         map[string]string `json:"nodeSelector,omitempty"`
	Containers           []Container       `json:"containers,omitempty"`
	InitContainers       []Container       `json:"initContainers,omitempty"`
	Volumes              []Volume          `json:"volumes,omitempty"`
	VolumeClaimTemplates []ObjectRef       `json:"volumeClaimTemplates,omitempty"`
	ConfigRefs           []ObjectRef       `json:"configRefs,omitempty"`
	SecretRefs           []ObjectRef       `json:"secretRefs,omitempty"`
	ImagePullSecretRefs  []ObjectRef       `json:"imagePullSecretRefs,omitempty"`
	OwnerReferences      []ObjectRef       `json:"ownerReferences,omitempty"`
}

// Pod records a running or historical Pod summary.
type Pod struct {
	ObjectRef
	Phase               string      `json:"phase,omitempty"`
	NodeName            string      `json:"nodeName,omitempty"`
	ServiceAccountName  string      `json:"serviceAccountName,omitempty"`
	RuntimeClassName    string      `json:"runtimeClassName,omitempty"`
	PodIP               string      `json:"podIp,omitempty"`
	HostIP              string      `json:"hostIp,omitempty"`
	Containers          []Container `json:"containers,omitempty"`
	InitContainers      []Container `json:"initContainers,omitempty"`
	EphemeralContainers []Container `json:"ephemeralContainers,omitempty"`
	Volumes             []Volume    `json:"volumes,omitempty"`
	OwnerReferences     []ObjectRef `json:"ownerReferences,omitempty"`
}

// Container records image and runtime facts visible in a pod template or status.
type Container struct {
	Name          string            `json:"name"`
	Image         string            `json:"image,omitempty"`
	ImageID       string            `json:"imageId,omitempty"`
	ContainerID   string            `json:"containerId,omitempty"`
	State         string            `json:"state,omitempty"`
	StartedAt     *time.Time        `json:"startedAt,omitempty"`
	Ready         *bool             `json:"ready,omitempty"`
	RestartCount  *int32            `json:"restartCount,omitempty"`
	Resources     map[string]string `json:"resources,omitempty"`
	EnvConfigRefs []ObjectRef       `json:"envConfigRefs,omitempty"`
	EnvSecretRefs []ObjectRef       `json:"envSecretRefs,omitempty"`
	VolumeMounts  []string          `json:"volumeMounts,omitempty"`
}

// RunningContainer records an actual container image observed in Running state.
type RunningContainer struct {
	Namespace        string     `json:"namespace,omitempty"`
	Pod              string     `json:"pod"`
	NodeName         string     `json:"nodeName,omitempty"`
	Container        string     `json:"container"`
	ContainerType    string     `json:"containerType,omitempty"`
	Image            string     `json:"image"`
	ImageID          string     `json:"imageId,omitempty"`
	ContainerID      string     `json:"containerId,omitempty"`
	Runtime          string     `json:"runtime,omitempty"`
	StartedAt        *time.Time `json:"startedAt,omitempty"`
	Ready            *bool      `json:"ready,omitempty"`
	RestartCount     *int32     `json:"restartCount,omitempty"`
	PodOwner         ObjectRef  `json:"podOwner,omitempty"`
	Workload         ObjectRef  `json:"workload,omitempty"`
	RuntimeClassName string     `json:"runtimeClassName,omitempty"`
	ServiceAccount   string     `json:"serviceAccount,omitempty"`
}

// RunningImage records a deduplicated image observed in Running containers.
type RunningImage struct {
	Image          string      `json:"image"`
	PodCount       int         `json:"podCount"`
	ContainerCount int         `json:"containerCount"`
	ImageIDs       []string    `json:"imageIds,omitempty"`
	Runtimes       []string    `json:"runtimes,omitempty"`
	Namespaces     []string    `json:"namespaces,omitempty"`
	Workloads      []ObjectRef `json:"workloads,omitempty"`
}

// Volume records pod volume references without reading Secret values.
type Volume struct {
	Name                  string      `json:"name"`
	Type                  string      `json:"type"`
	PersistentVolumeClaim string      `json:"persistentVolumeClaim,omitempty"`
	ConfigMap             string      `json:"configMap,omitempty"`
	Secret                string      `json:"secret,omitempty"`
	CSI                   string      `json:"csi,omitempty"`
	ProjectedRefs         []ObjectRef `json:"projectedRefs,omitempty"`
}

// Service records Service selector and endpoint-facing configuration.
type Service struct {
	ObjectRef
	Type        string            `json:"type,omitempty"`
	ClusterIP   string            `json:"clusterIp,omitempty"`
	ExternalIPs []string          `json:"externalIps,omitempty"`
	Selector    map[string]string `json:"selector,omitempty"`
	Ports       []ServicePort     `json:"ports,omitempty"`
}

// ServicePort records a service port mapping.
type ServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Port       int32  `json:"port"`
	TargetPort string `json:"targetPort,omitempty"`
	NodePort   int32  `json:"nodePort,omitempty"`
}

// EndpointSlice records endpoint backends and topology hints.
type EndpointSlice struct {
	ObjectRef
	AddressType string           `json:"addressType,omitempty"`
	ServiceName string           `json:"serviceName,omitempty"`
	Ports       []EndpointPort   `json:"ports,omitempty"`
	Endpoints   []EndpointTarget `json:"endpoints,omitempty"`
}

// EndpointPort records an EndpointSlice port.
type EndpointPort struct {
	Name     string `json:"name,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Port     *int32 `json:"port,omitempty"`
}

// EndpointTarget records an EndpointSlice endpoint target.
type EndpointTarget struct {
	Addresses []string  `json:"addresses,omitempty"`
	Ready     *bool     `json:"ready,omitempty"`
	NodeName  string    `json:"nodeName,omitempty"`
	Zone      string    `json:"zone,omitempty"`
	TargetRef ObjectRef `json:"targetRef,omitempty"`
}

// IngressClass records an IngressClass.
type IngressClass struct {
	ObjectRef
	Controller string    `json:"controller,omitempty"`
	Parameters ObjectRef `json:"parameters,omitempty"`
}

// Ingress records L7 routing references.
type Ingress struct {
	ObjectRef
	ClassName string        `json:"className,omitempty"`
	Rules     []IngressRule `json:"rules,omitempty"`
	TLSHosts  []string      `json:"tlsHosts,omitempty"`
	Backends  []ObjectRef   `json:"backends,omitempty"`
}

// IngressRule records a host/path routing rule.
type IngressRule struct {
	Host        string `json:"host,omitempty"`
	Path        string `json:"path,omitempty"`
	PathType    string `json:"pathType,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
	ServicePort string `json:"servicePort,omitempty"`
}

// GatewayClass records a Gateway API GatewayClass.
type GatewayClass struct {
	ObjectRef
	ControllerName string    `json:"controllerName,omitempty"`
	Parameters     ObjectRef `json:"parameters,omitempty"`
}

// Gateway records a Gateway API Gateway.
type Gateway struct {
	ObjectRef
	ClassName string            `json:"className,omitempty"`
	Addresses []string          `json:"addresses,omitempty"`
	Listeners []GatewayListener `json:"listeners,omitempty"`
}

// GatewayListener records a Gateway listener.
type GatewayListener struct {
	Name          string   `json:"name,omitempty"`
	Protocol      string   `json:"protocol,omitempty"`
	Port          int64    `json:"port,omitempty"`
	Hostname      string   `json:"hostname,omitempty"`
	AllowedRoutes []string `json:"allowedRoutes,omitempty"`
}

// GatewayRoute records a Gateway API route object.
type GatewayRoute struct {
	ObjectRef
	ParentRefs []GatewayParentRef `json:"parentRefs,omitempty"`
	Hostnames  []string           `json:"hostnames,omitempty"`
	Rules      []GatewayRouteRule `json:"rules,omitempty"`
}

// GatewayParentRef records a Gateway API route parent reference.
type GatewayParentRef struct {
	Group       string `json:"group,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Name        string `json:"name,omitempty"`
	SectionName string `json:"sectionName,omitempty"`
	Port        int64  `json:"port,omitempty"`
}

// GatewayRouteRule records matches and backends for a Gateway API route rule.
type GatewayRouteRule struct {
	Matches     []string    `json:"matches,omitempty"`
	BackendRefs []ObjectRef `json:"backendRefs,omitempty"`
}

// ReferenceGrant records a Gateway API ReferenceGrant.
type ReferenceGrant struct {
	ObjectRef
	From []GatewayGrantRef `json:"from,omitempty"`
	To   []GatewayGrantRef `json:"to,omitempty"`
}

// GatewayGrantRef records one side of a ReferenceGrant.
type GatewayGrantRef struct {
	Group     string `json:"group,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// GatewayPolicy records Gateway API policy attachments.
type GatewayPolicy struct {
	ObjectRef
	TargetRefs []ObjectRef `json:"targetRefs,omitempty"`
	Details    []string    `json:"details,omitempty"`
}

// StorageClass records provisioner and parameters.
type StorageClass struct {
	ObjectRef
	Provisioner          string            `json:"provisioner,omitempty"`
	Parameters           map[string]string `json:"parameters,omitempty"`
	ReclaimPolicy        string            `json:"reclaimPolicy,omitempty"`
	VolumeBindingMode    string            `json:"volumeBindingMode,omitempty"`
	AllowVolumeExpansion *bool             `json:"allowVolumeExpansion,omitempty"`
}

// PersistentVolume records PV storage facts.
type PersistentVolume struct {
	ObjectRef
	StorageClassName string     `json:"storageClassName,omitempty"`
	Capacity         string     `json:"capacity,omitempty"`
	AccessModes      []string   `json:"accessModes,omitempty"`
	VolumeMode       string     `json:"volumeMode,omitempty"`
	Phase            string     `json:"phase,omitempty"`
	ClaimRef         ObjectRef  `json:"claimRef,omitempty"`
	CSI              *CSIVolume `json:"csi,omitempty"`
}

// PersistentVolumeClaim records PVC storage facts.
type PersistentVolumeClaim struct {
	ObjectRef
	StorageClassName string   `json:"storageClassName,omitempty"`
	VolumeName       string   `json:"volumeName,omitempty"`
	RequestedStorage string   `json:"requestedStorage,omitempty"`
	AccessModes      []string `json:"accessModes,omitempty"`
	VolumeMode       string   `json:"volumeMode,omitempty"`
	Phase            string   `json:"phase,omitempty"`
}

// CSIVolume records CSI details on a PV.
type CSIVolume struct {
	Driver       string `json:"driver,omitempty"`
	VolumeHandle string `json:"volumeHandle,omitempty"`
	FSType       string `json:"fsType,omitempty"`
}

// CSIDriver records a CSIDriver object.
type CSIDriver struct {
	ObjectRef
	AttachRequired       *bool    `json:"attachRequired,omitempty"`
	PodInfoOnMount       *bool    `json:"podInfoOnMount,omitempty"`
	VolumeLifecycleModes []string `json:"volumeLifecycleModes,omitempty"`
}

// CSINode records CSI drivers installed on a node.
type CSINode struct {
	ObjectRef
	Drivers []CSINodeDriver `json:"drivers,omitempty"`
}

// CSINodeDriver records a single CSINode driver entry.
type CSINodeDriver struct {
	Name         string   `json:"name"`
	NodeID       string   `json:"nodeId,omitempty"`
	TopologyKeys []string `json:"topologyKeys,omitempty"`
}

// VolumeAttachment records CSI volume attachment state.
type VolumeAttachment struct {
	ObjectRef
	Attacher    string `json:"attacher,omitempty"`
	NodeName    string `json:"nodeName,omitempty"`
	PVName      string `json:"pvName,omitempty"`
	Attached    bool   `json:"attached"`
	AttachError string `json:"attachError,omitempty"`
}

// RuntimeClass records declared runtime handlers.
type RuntimeClass struct {
	ObjectRef
	Handler string `json:"handler,omitempty"`
}

// ConfigObject records metadata for ConfigMaps and similar config resources.
type ConfigObject struct {
	ObjectRef
	Data       map[string]string `json:"data,omitempty"`
	BinaryKeys []string          `json:"binaryKeys,omitempty"`
}

// Secret records Secret metadata without data values.
type Secret struct {
	ObjectRef
	Type string   `json:"type,omitempty"`
	Keys []string `json:"keys,omitempty"`
}

// RBAC records role and binding counts with object names.
type RBAC struct {
	Roles               []ObjectRef `json:"roles,omitempty"`
	RoleBindings        []ObjectRef `json:"roleBindings,omitempty"`
	ClusterRoles        []ObjectRef `json:"clusterRoles,omitempty"`
	ClusterRoleBindings []ObjectRef `json:"clusterRoleBindings,omitempty"`
}

// Policies records policy-like Kubernetes resources.
type Policies struct {
	HorizontalPodAutoscalers []ObjectRef `json:"horizontalPodAutoscalers,omitempty"`
	PodDisruptionBudgets     []ObjectRef `json:"podDisruptionBudgets,omitempty"`
	NetworkPolicies          []ObjectRef `json:"networkPolicies,omitempty"`
	ResourceQuotas           []ObjectRef `json:"resourceQuotas,omitempty"`
	LimitRanges              []ObjectRef `json:"limitRanges,omitempty"`
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
