package analysis

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/advisor"
	"github.com/kuraudo-lab/teleskope/internal/compare"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

const (
	maxWorkloads       = 80
	maxImages          = 80
	maxNetworkRefs     = 12
	maxStorageRefs     = 12
	maxCoverageItems   = 80
	maxAdvisorItems    = 80
	maxCompareFindings = 120
	maxPageItems       = 80
)

type modelContext struct {
	UseCase       UseCase         `json:"useCase"`
	WebSearch     bool            `json:"webSearch,omitempty"`
	Scope         Scope           `json:"scope,omitempty"`
	Snapshot      *scanContext    `json:"snapshot,omitempty"`
	Source        *scanContext    `json:"source,omitempty"`
	Target        *scanContext    `json:"target,omitempty"`
	CompareReport *compare.Report `json:"compareReport,omitempty"`
	Omitted       []string        `json:"omitted,omitempty"`
	GeneratedAt   time.Time       `json:"generatedAt"`
}

type scanContext struct {
	SchemaVersion       string                   `json:"schemaVersion,omitempty"`
	CollectedAt         time.Time                `json:"collectedAt,omitempty"`
	Source              inventory.Source         `json:"source,omitempty"`
	AWS                 inventory.AWSIdentity    `json:"aws,omitempty"`
	EKSCluster          inventory.Cluster        `json:"eksCluster,omitempty"`
	KubernetesContext   string                   `json:"kubernetesContext,omitempty"`
	KubernetesServer    string                   `json:"kubernetesServer,omitempty"`
	KubernetesVersion   string                   `json:"kubernetesVersion,omitempty"`
	Advisor             advisor.Report           `json:"advisor,omitempty"`
	EKS                 *eksPageContext          `json:"eks,omitempty"`
	Nodes               []inventory.Node         `json:"nodes,omitempty"`
	Workloads           []workloadContext        `json:"workloads,omitempty"`
	Pods                []inventory.Pod          `json:"pods,omitempty"`
	RunningImages       []runningImageContext    `json:"runningImages,omitempty"`
	Network             *networkPageContext      `json:"network,omitempty"`
	Storage             *storagePageContext      `json:"storage,omitempty"`
	Security            *securityPageContext     `json:"security,omitempty"`
	CoreResourceSummary coreResourceSummary      `json:"coreResourceSummary,omitempty"`
	Coverage            []inventory.CoverageItem `json:"coverage,omitempty"`
}

type eksPageContext struct {
	Cluster                 inventory.Cluster                  `json:"cluster"`
	Addons                  []inventory.Addon                  `json:"addons,omitempty"`
	Nodegroups              []inventory.Nodegroup              `json:"nodegroups,omitempty"`
	Insights                []inventory.EKSInsight             `json:"insights,omitempty"`
	AccessEntries           []inventory.AccessEntry            `json:"accessEntries,omitempty"`
	PodIdentityAssociations []inventory.PodIdentityAssociation `json:"podIdentityAssociations,omitempty"`
}

type networkPageContext struct {
	Services        []inventory.Service        `json:"services,omitempty"`
	EndpointSlices  []inventory.EndpointSlice  `json:"endpointSlices,omitempty"`
	IngressClasses  []inventory.IngressClass   `json:"ingressClasses,omitempty"`
	Ingresses       []inventory.Ingress        `json:"ingresses,omitempty"`
	GatewayClasses  []inventory.GatewayClass   `json:"gatewayClasses,omitempty"`
	Gateways        []inventory.Gateway        `json:"gateways,omitempty"`
	GatewayRoutes   []inventory.GatewayRoute   `json:"gatewayRoutes,omitempty"`
	ReferenceGrants []inventory.ReferenceGrant `json:"referenceGrants,omitempty"`
	GatewayPolicies []inventory.GatewayPolicy  `json:"gatewayPolicies,omitempty"`
}

type storagePageContext struct {
	StorageClasses         []inventory.StorageClass          `json:"storageClasses,omitempty"`
	PersistentVolumes      []inventory.PersistentVolume      `json:"persistentVolumes,omitempty"`
	PersistentVolumeClaims []inventory.PersistentVolumeClaim `json:"persistentVolumeClaims,omitempty"`
	CSIDrivers             []inventory.CSIDriver             `json:"csiDrivers,omitempty"`
	CSINodes               []inventory.CSINode               `json:"csiNodes,omitempty"`
	VolumeAttachments      []inventory.VolumeAttachment      `json:"volumeAttachments,omitempty"`
	RuntimeClasses         []inventory.RuntimeClass          `json:"runtimeClasses,omitempty"`
}

type securityPageContext struct {
	Roles             []inventory.Role                   `json:"roles,omitempty"`
	Bindings          []inventory.RoleBinding            `json:"bindings,omitempty"`
	AdmissionWebhooks []inventory.AdmissionWebhookConfig `json:"admissionWebhooks,omitempty"`
}

type workloadContext struct {
	Ref                inventory.ObjectRef   `json:"ref"`
	Replicas           *int32                `json:"replicas,omitempty"`
	ReadyReplicas      int32                 `json:"readyReplicas,omitempty"`
	AvailableReplicas  int32                 `json:"availableReplicas,omitempty"`
	Schedule           string                `json:"schedule,omitempty"`
	ServiceAccountName string                `json:"serviceAccountName,omitempty"`
	RuntimeClassName   string                `json:"runtimeClassName,omitempty"`
	Selector           map[string]string     `json:"selector,omitempty"`
	NodeSelector       map[string]string     `json:"nodeSelector,omitempty"`
	Containers         []inventory.Container `json:"containers,omitempty"`
	InitContainers     []inventory.Container `json:"initContainers,omitempty"`
	Volumes            []inventory.Volume    `json:"volumes,omitempty"`
	Storage            []storageUse          `json:"storage,omitempty"`
	Network            []networkUse          `json:"network,omitempty"`
	ConfigRefs         []inventory.ObjectRef `json:"configRefs,omitempty"`
	SecretRefs         []inventory.ObjectRef `json:"secretRefs,omitempty"`
	OwnerReferences    []inventory.ObjectRef `json:"ownerReferences,omitempty"`
}

type storageUse struct {
	VolumeName            string              `json:"volumeName,omitempty"`
	VolumeType            string              `json:"volumeType,omitempty"`
	PersistentVolumeClaim inventory.ObjectRef `json:"persistentVolumeClaim,omitempty"`
	PersistentVolume      inventory.ObjectRef `json:"persistentVolume,omitempty"`
	StorageClass          string              `json:"storageClass,omitempty"`
	Provisioner           string              `json:"provisioner,omitempty"`
	RequestedStorage      string              `json:"requestedStorage,omitempty"`
	Capacity              string              `json:"capacity,omitempty"`
	AccessModes           []string            `json:"accessModes,omitempty"`
	VolumeMode            string              `json:"volumeMode,omitempty"`
	Phase                 string              `json:"phase,omitempty"`
	CSIDriver             string              `json:"csiDriver,omitempty"`
	ReclaimPolicy         string              `json:"reclaimPolicy,omitempty"`
	VolumeBindingMode     string              `json:"volumeBindingMode,omitempty"`
}

type networkUse struct {
	Kind       string                `json:"kind"`
	Ref        inventory.ObjectRef   `json:"ref,omitempty"`
	Type       string                `json:"type,omitempty"`
	Class      string                `json:"class,omitempty"`
	Hosts      []string              `json:"hosts,omitempty"`
	Ports      []string              `json:"ports,omitempty"`
	Backends   []inventory.ObjectRef `json:"backends,omitempty"`
	ParentRefs []string              `json:"parentRefs,omitempty"`
}

type runningImageContext struct {
	Image          string                `json:"image"`
	PodCount       int                   `json:"podCount"`
	ContainerCount int                   `json:"containerCount"`
	Namespaces     []string              `json:"namespaces,omitempty"`
	Workloads      []inventory.ObjectRef `json:"workloads,omitempty"`
}

type coreResourceSummary struct {
	RBACRoles                 int `json:"rbacRoles,omitempty"`
	RBACBindings              int `json:"rbacBindings,omitempty"`
	NetworkPolicies           int `json:"networkPolicies,omitempty"`
	ResourceQuotas            int `json:"resourceQuotas,omitempty"`
	LimitRanges               int `json:"limitRanges,omitempty"`
	PodDisruptionBudgets      int `json:"podDisruptionBudgets,omitempty"`
	AdmissionWebhookConfigs   int `json:"admissionWebhookConfigs,omitempty"`
	AdmissionWebhooks         int `json:"admissionWebhooks,omitempty"`
	CustomResourceDefinitions int `json:"customResourceDefinitions,omitempty"`
	CustomResourceInstances   int `json:"customResourceInstances,omitempty"`
}

func BuildContext(req Request) ([]byte, error) {
	ctx, err := buildContext(req)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(ctx, "", "  ")
}

func buildContext(req Request) (modelContext, error) {
	out := modelContext{UseCase: req.UseCase, WebSearch: req.WebSearch, Scope: cleanScope(req.Scope), GeneratedAt: time.Now().UTC()}
	switch req.UseCase {
	case UseCaseScan:
		if req.Snapshot != nil {
			scan := buildScanContext(req.Snapshot, req.Scope, &out.Omitted)
			out.Snapshot = &scan
		}
	case UseCaseCompare, UseCaseAdvisory:
		if req.Source != nil {
			source := buildScanContext(req.Source, req.Scope, &out.Omitted)
			out.Source = &source
		}
		if req.Target != nil {
			target := buildScanContext(req.Target, req.Scope, &out.Omitted)
			out.Target = &target
		}
		if req.CompareReport != nil {
			report := *req.CompareReport
			if len(report.Findings) > maxCompareFindings {
				out.Omitted = append(out.Omitted, "compare findings truncated")
				report.Findings = report.Findings[:maxCompareFindings]
			}
			out.CompareReport = &report
		}
	default:
		return out, ErrUnknownUseCase(req.UseCase)
	}
	return out, nil
}

type ErrUnknownUseCase UseCase

func (e ErrUnknownUseCase) Error() string { return "unknown analysis use case " + string(e) }

func buildScanContext(s *inventory.Snapshot, scope Scope, omitted *[]string) scanContext {
	k := s.Kubernetes
	links := buildWorkloadLinks(k)
	workloads := append([]inventory.Workload(nil), k.Workloads...)
	sort.Slice(workloads, func(i, j int) bool { return refKey(workloads[i].ObjectRef) < refKey(workloads[j].ObjectRef) })
	var workloadItems []workloadContext
	for _, w := range take(filterWorkloads(workloads, scope), maxWorkloads, omitted, "workloads truncated") {
		key := namespacedKey(w.Namespace, w.Name)
		workloadItems = append(workloadItems, workloadContext{
			Ref: cleanRef(w.ObjectRef), Replicas: w.Replicas, ReadyReplicas: w.ReadyReplicas, AvailableReplicas: w.AvailableReplicas, Schedule: w.Schedule,
			ServiceAccountName: w.ServiceAccountName, RuntimeClassName: w.RuntimeClassName, Selector: w.Selector, NodeSelector: w.NodeSelector,
			Containers: trimContainers(w.Containers), InitContainers: trimContainers(w.InitContainers), Volumes: trimVolumes(w.Volumes), Storage: take(links.storage[key], maxStorageRefs, omitted, "workload storage references truncated"), Network: take(links.network[key], maxNetworkRefs, omitted, "workload network references truncated"),
			ConfigRefs: cleanRefs(w.ConfigRefs), SecretRefs: cleanRefs(w.SecretRefs), OwnerReferences: cleanRefs(w.OwnerReferences),
		})
	}
	analysis := advisor.Analyze(s)
	for i := range analysis.Capabilities {
		for j := range analysis.Capabilities[i].Evidence {
			analysis.Capabilities[i].Evidence[j].Resource = cleanRef(analysis.Capabilities[i].Evidence[j].Resource)
		}
	}
	if len(analysis.Capabilities) > maxAdvisorItems {
		*omitted = append(*omitted, "advisor capabilities truncated")
		analysis.Capabilities = analysis.Capabilities[:maxAdvisorItems]
	}
	out := scanContext{
		SchemaVersion: s.SchemaVersion, CollectedAt: s.CollectedAt, Source: s.Source, AWS: s.AWS, EKSCluster: s.EKS.Cluster,
		KubernetesContext: k.Context, KubernetesServer: k.Server, KubernetesVersion: k.Version.GitVersion, Advisor: analysis,
		Workloads: workloadItems, RunningImages: takeSortedImages(filterRunningImages(k.RunningImages, scope), omitted), CoreResourceSummary: summarizeCoreResources(k), Coverage: takeCoverage(k, s.Coverage, omitted),
	}
	switch scopedPageType(scope) {
	case "eks":
		out.EKS = buildEKSPageContext(s.EKS, scope, omitted)
	case "nodes":
		out.Nodes = filterNodes(k.Nodes, scope, omitted)
	case "workloads":
		out.Pods = filterPods(k.Pods, scope, omitted)
	case "network":
		out.Network = buildNetworkPageContext(k, scope, omitted)
	case "storage":
		out.Storage = buildStoragePageContext(k, scope, omitted)
	case "security":
		out.Security = buildSecurityPageContext(k, scope, omitted)
	}
	return out
}

func scopedPageType(scope Scope) string {
	value := strings.ToLower(strings.TrimSpace(scope.ResourceType))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(scope.PageID))
	}
	return value
}

func buildEKSPageContext(eks inventory.EKSInventory, scope Scope, omitted *[]string) *eksPageContext {
	selected := selectedNames(scope.SelectedRefs)
	return &eksPageContext{
		Cluster:                 eks.Cluster,
		Addons:                  take(filterNamed(eks.Addons, selected, func(v inventory.Addon) string { return v.Name }), maxPageItems, omitted, "EKS add-ons truncated"),
		Nodegroups:              take(filterNamed(eks.Nodegroups, selected, func(v inventory.Nodegroup) string { return v.Name }), maxPageItems, omitted, "EKS nodegroups truncated"),
		Insights:                take(filterNamed(eks.Insights, selected, func(v inventory.EKSInsight) string { return firstContextValue(v.Name, v.ID) }), maxPageItems, omitted, "EKS insights truncated"),
		AccessEntries:           take(filterNamed(eks.AccessEntries, selected, func(v inventory.AccessEntry) string { return firstContextValue(v.Username, v.PrincipalARN) }), maxPageItems, omitted, "EKS access entries truncated"),
		PodIdentityAssociations: take(filterNamed(eks.PodIdentityAssociations, selected, func(v inventory.PodIdentityAssociation) string { return firstContextValue(v.ServiceAccount, v.ID) }), maxPageItems, omitted, "EKS pod identity associations truncated"),
	}
}

func filterNodes(values []inventory.Node, scope Scope, omitted *[]string) []inventory.Node {
	out := make([]inventory.Node, 0, len(values))
	for _, value := range values {
		if !scopeRefMatches(value.ObjectRef, scope) {
			continue
		}
		value.ObjectRef = cleanRef(value.ObjectRef)
		out = append(out, value)
	}
	return take(out, maxPageItems, omitted, "nodes truncated")
}

func filterPods(values []inventory.Pod, scope Scope, omitted *[]string) []inventory.Pod {
	out := make([]inventory.Pod, 0, len(values))
	for _, value := range values {
		if !scopeRefMatches(value.ObjectRef, scope) {
			continue
		}
		value.ObjectRef = cleanRef(value.ObjectRef)
		value.OwnerReferences = cleanRefs(value.OwnerReferences)
		value.Containers = trimContainers(value.Containers)
		value.InitContainers = trimContainers(value.InitContainers)
		value.EphemeralContainers = trimContainers(value.EphemeralContainers)
		out = append(out, value)
	}
	return take(out, maxPageItems, omitted, "pods truncated")
}

func buildNetworkPageContext(k inventory.Kubernetes, scope Scope, omitted *[]string) *networkPageContext {
	out := &networkPageContext{
		Services:        filterRefItems(k.Services, scope, omitted, "services truncated", func(v *inventory.Service) *inventory.ObjectRef { return &v.ObjectRef }),
		EndpointSlices:  filterRefItems(k.EndpointSlices, scope, omitted, "endpoint slices truncated", func(v *inventory.EndpointSlice) *inventory.ObjectRef { return &v.ObjectRef }),
		IngressClasses:  filterRefItems(k.IngressClasses, scope, omitted, "ingress classes truncated", func(v *inventory.IngressClass) *inventory.ObjectRef { return &v.ObjectRef }),
		Ingresses:       filterRefItems(k.Ingresses, scope, omitted, "ingresses truncated", func(v *inventory.Ingress) *inventory.ObjectRef { return &v.ObjectRef }),
		GatewayClasses:  filterRefItems(k.GatewayClasses, scope, omitted, "gateway classes truncated", func(v *inventory.GatewayClass) *inventory.ObjectRef { return &v.ObjectRef }),
		Gateways:        filterRefItems(k.Gateways, scope, omitted, "gateways truncated", func(v *inventory.Gateway) *inventory.ObjectRef { return &v.ObjectRef }),
		GatewayRoutes:   filterRefItems(k.GatewayRoutes, scope, omitted, "gateway routes truncated", func(v *inventory.GatewayRoute) *inventory.ObjectRef { return &v.ObjectRef }),
		ReferenceGrants: filterRefItems(k.ReferenceGrants, scope, omitted, "reference grants truncated", func(v *inventory.ReferenceGrant) *inventory.ObjectRef { return &v.ObjectRef }),
		GatewayPolicies: filterRefItems(k.GatewayPolicies, scope, omitted, "gateway policies truncated", func(v *inventory.GatewayPolicy) *inventory.ObjectRef { return &v.ObjectRef }),
	}
	for i := range out.EndpointSlices {
		for j := range out.EndpointSlices[i].Endpoints {
			out.EndpointSlices[i].Endpoints[j].TargetRef = cleanRef(out.EndpointSlices[i].Endpoints[j].TargetRef)
		}
	}
	for i := range out.IngressClasses {
		out.IngressClasses[i].Parameters = cleanRef(out.IngressClasses[i].Parameters)
	}
	for i := range out.Ingresses {
		out.Ingresses[i].Backends = cleanRefs(out.Ingresses[i].Backends)
	}
	for i := range out.GatewayClasses {
		out.GatewayClasses[i].Parameters = cleanRef(out.GatewayClasses[i].Parameters)
	}
	for i := range out.GatewayRoutes {
		for j := range out.GatewayRoutes[i].Rules {
			out.GatewayRoutes[i].Rules[j].BackendRefs = cleanRefs(out.GatewayRoutes[i].Rules[j].BackendRefs)
		}
	}
	for i := range out.GatewayPolicies {
		out.GatewayPolicies[i].TargetRefs = cleanRefs(out.GatewayPolicies[i].TargetRefs)
	}
	return out
}

func buildStoragePageContext(k inventory.Kubernetes, scope Scope, omitted *[]string) *storagePageContext {
	out := &storagePageContext{
		StorageClasses:         filterRefItems(k.StorageClasses, scope, omitted, "storage classes truncated", func(v *inventory.StorageClass) *inventory.ObjectRef { return &v.ObjectRef }),
		PersistentVolumes:      filterRefItems(k.PersistentVolumes, scope, omitted, "persistent volumes truncated", func(v *inventory.PersistentVolume) *inventory.ObjectRef { return &v.ObjectRef }),
		PersistentVolumeClaims: filterRefItems(k.PersistentVolumeClaims, scope, omitted, "persistent volume claims truncated", func(v *inventory.PersistentVolumeClaim) *inventory.ObjectRef { return &v.ObjectRef }),
		CSIDrivers:             filterRefItems(k.CSIDrivers, scope, omitted, "CSI drivers truncated", func(v *inventory.CSIDriver) *inventory.ObjectRef { return &v.ObjectRef }),
		CSINodes:               filterRefItems(k.CSINodes, scope, omitted, "CSI nodes truncated", func(v *inventory.CSINode) *inventory.ObjectRef { return &v.ObjectRef }),
		VolumeAttachments:      filterRefItems(k.VolumeAttachments, scope, omitted, "volume attachments truncated", func(v *inventory.VolumeAttachment) *inventory.ObjectRef { return &v.ObjectRef }),
		RuntimeClasses:         filterRefItems(k.RuntimeClasses, scope, omitted, "runtime classes truncated", func(v *inventory.RuntimeClass) *inventory.ObjectRef { return &v.ObjectRef }),
	}
	for i := range out.PersistentVolumes {
		out.PersistentVolumes[i].ClaimRef = cleanRef(out.PersistentVolumes[i].ClaimRef)
	}
	return out
}

func buildSecurityPageContext(k inventory.Kubernetes, scope Scope, omitted *[]string) *securityPageContext {
	roles := append(append([]inventory.Role(nil), k.RBAC.RoleDetails...), k.RBAC.ClusterRoleDetails...)
	bindings := append(append([]inventory.RoleBinding(nil), k.RBAC.RoleBindingDetails...), k.RBAC.ClusterRoleBindingDetails...)
	out := &securityPageContext{
		Roles:             filterRefItems(roles, scope, omitted, "RBAC roles truncated", func(v *inventory.Role) *inventory.ObjectRef { return &v.ObjectRef }),
		Bindings:          filterRefItems(bindings, scope, omitted, "RBAC bindings truncated", func(v *inventory.RoleBinding) *inventory.ObjectRef { return &v.ObjectRef }),
		AdmissionWebhooks: filterRefItems(k.AdmissionWebhooks, scope, omitted, "admission webhook configurations truncated", func(v *inventory.AdmissionWebhookConfig) *inventory.ObjectRef { return &v.ObjectRef }),
	}
	for i := range out.Bindings {
		out.Bindings[i].RoleRef = cleanRef(out.Bindings[i].RoleRef)
	}
	for i := range out.AdmissionWebhooks {
		for j := range out.AdmissionWebhooks[i].Webhooks {
			out.AdmissionWebhooks[i].Webhooks[j].ClientService = cleanRef(out.AdmissionWebhooks[i].Webhooks[j].ClientService)
		}
	}
	return out
}

func filterRefItems[T any](values []T, scope Scope, omitted *[]string, message string, ref func(*T) *inventory.ObjectRef) []T {
	out := make([]T, 0, len(values))
	for _, value := range values {
		itemRef := ref(&value)
		if !scopeRefMatches(*itemRef, scope) {
			continue
		}
		*itemRef = cleanRef(*itemRef)
		out = append(out, value)
	}
	return take(out, maxPageItems, omitted, message)
}

func scopeRefMatches(ref inventory.ObjectRef, scope Scope) bool {
	if scope.Namespace != "" && ref.Namespace != "" && ref.Namespace != scope.Namespace {
		return false
	}
	if len(scope.SelectedRefs) == 0 {
		return true
	}
	selected := refSet(scope.SelectedRefs)
	return selectedMatches(selected, ref)
}

func selectedNames(refs []inventory.ObjectRef) map[string]struct{} {
	out := map[string]struct{}{}
	for _, ref := range refs {
		if ref.Name != "" {
			out[ref.Name] = struct{}{}
		}
	}
	return out
}

func filterNamed[T any](values []T, selected map[string]struct{}, name func(T) string) []T {
	if len(selected) == 0 {
		return append([]T(nil), values...)
	}
	out := make([]T, 0, len(values))
	for _, value := range values {
		if _, ok := selected[name(value)]; ok {
			out = append(out, value)
		}
	}
	return out
}

func cleanScope(scope Scope) Scope {
	scope.SelectedRefs = cleanRefs(scope.SelectedRefs)
	return scope
}

func filterWorkloads(values []inventory.Workload, scope Scope) []inventory.Workload {
	if isEmptyScope(scope) {
		return values
	}
	selected := refSet(scope.SelectedRefs)
	out := make([]inventory.Workload, 0, len(values))
	for _, value := range values {
		if scope.Namespace != "" && value.Namespace != scope.Namespace {
			continue
		}
		if len(selected) > 0 && !selectedMatches(selected, value.ObjectRef) {
			continue
		}
		if scope.ResourceType != "" && scope.ResourceType != "workload" && scope.ResourceType != "workloads" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func filterRunningImages(values []inventory.RunningImage, scope Scope) []inventory.RunningImage {
	if isEmptyScope(scope) {
		return values
	}
	out := make([]inventory.RunningImage, 0, len(values))
	for _, value := range values {
		if scope.Namespace != "" && !hasString(value.Namespaces, scope.Namespace) {
			continue
		}
		if len(scope.SelectedRefs) > 0 && !runningImageMatches(value, scope.SelectedRefs) {
			continue
		}
		if scope.ResourceType != "" && scope.ResourceType != "image" && scope.ResourceType != "images" && scope.ResourceType != "runningImages" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func isEmptyScope(scope Scope) bool {
	return scope.PageID == "" && scope.Namespace == "" && scope.ResourceType == "" && len(scope.SelectedRefs) == 0
}

func refSet(refs []inventory.ObjectRef) map[string]struct{} {
	out := map[string]struct{}{}
	for _, ref := range refs {
		out[refKey(cleanRef(ref))] = struct{}{}
	}
	return out
}

func selectedMatches(selected map[string]struct{}, ref inventory.ObjectRef) bool {
	_, ok := selected[refKey(cleanRef(ref))]
	return ok
}

func runningImageMatches(image inventory.RunningImage, refs []inventory.ObjectRef) bool {
	selected := refSet(refs)
	for _, ref := range image.Workloads {
		if selectedMatches(selected, ref) {
			return true
		}
	}
	return false
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

type workloadLinks struct {
	storage map[string][]storageUse
	network map[string][]networkUse
}

func buildWorkloadLinks(k inventory.Kubernetes) workloadLinks {
	links := workloadLinks{storage: map[string][]storageUse{}, network: map[string][]networkUse{}}
	pvcs := map[string]inventory.PersistentVolumeClaim{}
	for _, pvc := range k.PersistentVolumeClaims {
		pvcs[namespacedKey(pvc.Namespace, pvc.Name)] = pvc
	}
	pvsByClaim := map[string]inventory.PersistentVolume{}
	for _, pv := range k.PersistentVolumes {
		if pv.ClaimRef.Name != "" {
			pvsByClaim[namespacedKey(pv.ClaimRef.Namespace, pv.ClaimRef.Name)] = pv
		}
	}
	classes := map[string]inventory.StorageClass{}
	for _, class := range k.StorageClasses {
		classes[class.Name] = class
	}
	for _, workload := range k.Workloads {
		key := namespacedKey(workload.Namespace, workload.Name)
		for _, volume := range workload.Volumes {
			use := storageUse{VolumeName: volume.Name, VolumeType: volume.Type}
			if volume.PersistentVolumeClaim != "" {
				pvc, ok := pvcs[namespacedKey(workload.Namespace, volume.PersistentVolumeClaim)]
				use.PersistentVolumeClaim = inventory.ObjectRef{APIVersion: "v1", Kind: "PersistentVolumeClaim", Namespace: workload.Namespace, Name: volume.PersistentVolumeClaim}
				if ok {
					use.PersistentVolumeClaim = cleanRef(pvc.ObjectRef)
					use.StorageClass = pvc.StorageClassName
					use.RequestedStorage = pvc.RequestedStorage
					use.AccessModes = pvc.AccessModes
					use.VolumeMode = pvc.VolumeMode
					use.Phase = pvc.Phase
					if pv, exists := pvsByClaim[namespacedKey(pvc.Namespace, pvc.Name)]; exists {
						use.PersistentVolume = cleanRef(pv.ObjectRef)
						use.Capacity = pv.Capacity
						if len(use.AccessModes) == 0 {
							use.AccessModes = pv.AccessModes
						}
						if use.VolumeMode == "" {
							use.VolumeMode = pv.VolumeMode
						}
						if pv.CSI != nil {
							use.CSIDriver = pv.CSI.Driver
						}
					}
					if class, exists := classes[pvc.StorageClassName]; exists {
						use.Provisioner = class.Provisioner
						use.ReclaimPolicy = class.ReclaimPolicy
						use.VolumeBindingMode = class.VolumeBindingMode
					}
				}
			}
			links.storage[key] = append(links.storage[key], use)
		}
	}
	services := append([]inventory.Service(nil), k.Services...)
	sort.Slice(services, func(i, j int) bool { return refKey(services[i].ObjectRef) < refKey(services[j].ObjectRef) })
	serviceBackends := map[string][]string{}
	for _, service := range services {
		for _, workload := range k.Workloads {
			if service.Namespace == workload.Namespace && selectorMatches(service.Selector, workload.Selector) {
				key := namespacedKey(workload.Namespace, workload.Name)
				serviceKey := namespacedKey(service.Namespace, service.Name)
				serviceBackends[serviceKey] = append(serviceBackends[serviceKey], key)
				links.network[key] = append(links.network[key], networkUse{Kind: "Service", Ref: cleanRef(service.ObjectRef), Type: service.Type, Ports: servicePorts(service.Ports)})
			}
		}
	}
	for _, ingress := range k.Ingresses {
		for _, backend := range ingressBackends(ingress) {
			for _, workloadKey := range serviceBackends[namespacedKey(ingress.Namespace, backend.Name)] {
				links.network[workloadKey] = append(links.network[workloadKey], networkUse{Kind: "Ingress", Ref: cleanRef(ingress.ObjectRef), Class: ingress.ClassName, Hosts: ingressHosts(ingress), Backends: []inventory.ObjectRef{cleanRef(backend)}})
			}
		}
	}
	for _, route := range k.GatewayRoutes {
		for _, rule := range route.Rules {
			for _, backend := range rule.BackendRefs {
				backendNS := firstContextValue(backend.Namespace, route.Namespace)
				for _, workloadKey := range serviceBackends[namespacedKey(backendNS, backend.Name)] {
					links.network[workloadKey] = append(links.network[workloadKey], networkUse{Kind: route.Kind, Ref: cleanRef(route.ObjectRef), Hosts: route.Hostnames, Backends: []inventory.ObjectRef{cleanRef(backend)}, ParentRefs: routeParentValues(route.ParentRefs)})
				}
			}
		}
	}
	for key := range links.network {
		sort.Slice(links.network[key], func(i, j int) bool {
			return links.network[key][i].Kind+refKey(links.network[key][i].Ref) < links.network[key][j].Kind+refKey(links.network[key][j].Ref)
		})
	}
	return links
}

func trimContainers(in []inventory.Container) []inventory.Container {
	out := append([]inventory.Container(nil), in...)
	for i := range out {
		out[i].EnvSecretRefs = nil
		out[i].ImageID = ""
		out[i].ContainerID = ""
		out[i].StartedAt = nil
	}
	return out
}

func trimVolumes(in []inventory.Volume) []inventory.Volume {
	out := append([]inventory.Volume(nil), in...)
	for i := range out {
		out[i].ProjectedRefs = cleanRefs(out[i].ProjectedRefs)
	}
	return out
}

func summarizeCoreResources(k inventory.Kubernetes) coreResourceSummary {
	webhooks := 0
	for _, config := range k.AdmissionWebhooks {
		webhooks += len(config.Webhooks)
	}
	return coreResourceSummary{
		RBACRoles: len(k.RBAC.Roles) + len(k.RBAC.ClusterRoles), RBACBindings: len(k.RBAC.RoleBindings) + len(k.RBAC.ClusterRoleBindings),
		NetworkPolicies: len(k.Policies.NetworkPolicies), ResourceQuotas: len(k.Policies.ResourceQuotas), LimitRanges: len(k.Policies.LimitRanges), PodDisruptionBudgets: len(k.Policies.PodDisruptionBudgets),
		AdmissionWebhookConfigs: len(k.AdmissionWebhooks), AdmissionWebhooks: webhooks, CustomResourceDefinitions: len(k.CustomResourceDefinitions), CustomResourceInstances: len(k.CustomResourceInstances),
	}
}

func selectorMatches(serviceSelector, workloadSelector map[string]string) bool {
	if len(serviceSelector) == 0 || len(workloadSelector) == 0 {
		return false
	}
	for key, value := range serviceSelector {
		if workloadSelector[key] != value {
			return false
		}
	}
	return true
}

func servicePorts(ports []inventory.ServicePort) []string {
	out := make([]string, 0, len(ports))
	for _, port := range ports {
		value := port.Protocol
		if value == "" {
			value = "TCP"
		}
		value += "/" + intString(port.Port)
		if port.TargetPort != "" {
			value += "->" + port.TargetPort
		}
		out = append(out, value)
	}
	return out
}

func ingressBackends(ingress inventory.Ingress) []inventory.ObjectRef {
	seen := map[string]struct{}{}
	var out []inventory.ObjectRef
	for _, backend := range ingress.Backends {
		ref := cleanRef(backend)
		if ref.Namespace == "" {
			ref.Namespace = ingress.Namespace
		}
		key := refKey(ref)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, ref)
		}
	}
	for _, rule := range ingress.Rules {
		if rule.ServiceName == "" {
			continue
		}
		ref := inventory.ObjectRef{APIVersion: "v1", Kind: "Service", Namespace: ingress.Namespace, Name: rule.ServiceName}
		key := refKey(ref)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, ref)
		}
	}
	return out
}

func ingressHosts(ingress inventory.Ingress) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, rule := range ingress.Rules {
		if rule.Host == "" {
			continue
		}
		if _, ok := seen[rule.Host]; !ok {
			seen[rule.Host] = struct{}{}
			out = append(out, rule.Host)
		}
	}
	for _, host := range ingress.TLSHosts {
		if host == "" {
			continue
		}
		if _, ok := seen[host]; !ok {
			seen[host] = struct{}{}
			out = append(out, host)
		}
	}
	return out
}

func routeParentValues(values []inventory.GatewayParentRef) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strings.Join(nonEmptyContextValues(value.Kind, value.Namespace, value.Name, value.SectionName), "/"))
	}
	return out
}

func take[T any](values []T, limit int, omitted *[]string, msg string) []T {
	if len(values) > limit {
		*omitted = append(*omitted, msg)
		values = values[:limit]
	}
	return values
}

func takeCoverage(_ inventory.Kubernetes, values []inventory.CoverageItem, omitted *[]string) []inventory.CoverageItem {
	out := append([]inventory.CoverageItem(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i].Area+"/"+out[i].Resource < out[j].Area+"/"+out[j].Resource })
	return take(out, maxCoverageItems, omitted, "coverage truncated")
}

func takeSortedImages(values []inventory.RunningImage, omitted *[]string) []runningImageContext {
	out := append([]inventory.RunningImage(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i].Image < out[j].Image })
	out = take(out, maxImages, omitted, "running images truncated")
	result := make([]runningImageContext, 0, len(out))
	for _, image := range out {
		result = append(result, runningImageContext{Image: image.Image, PodCount: image.PodCount, ContainerCount: image.ContainerCount, Namespaces: image.Namespaces, Workloads: cleanRefs(image.Workloads)})
	}
	return result
}

func refKey(ref inventory.ObjectRef) string {
	return strings.Join([]string{ref.APIVersion, ref.Kind, ref.Namespace, ref.Name}, "/")
}

func namespacedKey(namespace, name string) string { return namespace + "/" + name }

func cleanRef(ref inventory.ObjectRef) inventory.ObjectRef {
	ref.UID = ""
	return ref
}

func cleanRefs(refs []inventory.ObjectRef) []inventory.ObjectRef {
	out := append([]inventory.ObjectRef(nil), refs...)
	for i := range out {
		out[i] = cleanRef(out[i])
	}
	return out
}

func firstContextValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nonEmptyContextValues(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func intString(value int32) string {
	return fmt.Sprint(value)
}
