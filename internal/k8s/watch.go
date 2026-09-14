package k8s

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/kuraudo-lab/teleskope/internal/buildinfo"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

const defaultWatchDebounce = 750 * time.Millisecond

// WatchOptions controls event-based Kubernetes inventory publication.
type WatchOptions struct {
	Kubeconfig string
	Context    string
	Debounce   time.Duration
	Progress   func(format string, args ...any)
}

// WatchHealth describes watch freshness independently from the retained data.
type WatchHealth struct {
	State        string
	Coverage     []inventory.CoverageItem
	LastEventAt  time.Time
	LastRelistAt time.Time
	Reconnects   int
	Err          error
}

// WatchUpdate is one immutable Kubernetes publication candidate.
type WatchUpdate struct {
	Snapshot *inventory.Snapshot
	Health   WatchHealth
}

// WatchRuntime runs event-based Kubernetes inventory updates.
type WatchRuntime interface {
	Run(ctx context.Context, publish func(WatchUpdate)) error
}

type kubernetesWatchRuntime struct {
	opts             WatchOptions
	client           kubernetes.Interface
	dynamic          dynamic.Interface
	contextName      string
	contextNamespace string
	server           string
}

// NewWatchRuntime initializes a watch runtime without starting informers.
func NewWatchRuntime(opts WatchOptions) (WatchRuntime, error) {
	config, contextName, contextNamespace, server, err := loadRESTConfig(Options{
		Kubeconfig: opts.Kubeconfig,
		Context:    opts.Context,
		Progress:   opts.Progress,
	})
	if err != nil {
		return nil, fmt.Errorf("connect Kubernetes cluster: %w", err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("connect Kubernetes cluster: create client: %w", err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("connect Kubernetes cluster: create dynamic client: %w", err)
	}
	return NewWatchRuntimeWithClients(client, dynamicClient, opts, contextName, contextNamespace, server), nil
}

// NewWatchRuntimeWithClient creates a watch runtime around an existing client.
func NewWatchRuntimeWithClient(client kubernetes.Interface, opts WatchOptions, contextName, server string) WatchRuntime {
	return NewWatchRuntimeWithClients(client, nil, opts, contextName, "", server)
}

// NewWatchRuntimeWithClients creates a watch runtime around existing typed and dynamic clients.
func NewWatchRuntimeWithClients(client kubernetes.Interface, dynamicClient dynamic.Interface, opts WatchOptions, contextName, contextNamespace, server string) WatchRuntime {
	if opts.Debounce <= 0 {
		opts.Debounce = defaultWatchDebounce
	}
	return &kubernetesWatchRuntime{opts: opts, client: client, dynamic: dynamicClient, contextName: contextName, contextNamespace: contextNamespace, server: server}
}

func (r *kubernetesWatchRuntime) Run(ctx context.Context, publish func(WatchUpdate)) error {
	factory := informers.NewSharedInformerFactoryWithOptions(r.client, 0)
	coreInformers := r.coreInformers(factory)
	informers := append([]watchedInformer(nil), coreInformers...)
	var dynamicFactory dynamicinformer.DynamicSharedInformerFactory
	if r.dynamic != nil {
		dynamicFactory = dynamicinformer.NewFilteredDynamicSharedInformerFactory(r.dynamic, 0, metav1.NamespaceAll, nil)
		apiResources, _, _ := r.discoverAPIResources(ctx)
		informers = append(informers, r.dynamicInformers(dynamicFactory, apiResources)...)
	}
	health := &watchHealthState{state: "loading"}
	dirty := make(chan struct{}, 1)
	var synced bool
	var syncMu sync.RWMutex

	schedule := func() {
		syncMu.RLock()
		ready := synced
		syncMu.RUnlock()
		if !ready {
			return
		}
		select {
		case dirty <- struct{}{}:
		default:
		}
	}

	for _, item := range informers {
		if err := item.informer.SetWatchErrorHandler(func(_ *cache.Reflector, err error) {
			health.recordError(err)
			progress(r.opts.Progress, "watch reconnecting resource=%s error=%v", item.resource, err)
			publish(WatchUpdate{Health: health.snapshot()})
		}); err != nil {
			return fmt.Errorf("configure watch error handler for %s: %w", item.resource, err)
		}
		_, err := item.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj any) {
				health.recordEvent()
				schedule()
			},
			UpdateFunc: func(_, _ any) {
				health.recordEvent()
				schedule()
			},
			DeleteFunc: func(any) {
				health.recordEvent()
				schedule()
			},
		})
		if err != nil {
			return fmt.Errorf("configure event handler for %s: %w", item.resource, err)
		}
	}

	factory.Start(ctx.Done())
	if dynamicFactory != nil {
		dynamicFactory.Start(ctx.Done())
	}
	progress(r.opts.Progress, "waiting for Kubernetes watch caches to sync resources=%d", len(informers))
	if !cache.WaitForCacheSync(ctx.Done(), informerSyncs(coreInformers)...) {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("Kubernetes watch caches did not sync")
	}
	health.recordRelist()
	syncMu.Lock()
	synced = true
	syncMu.Unlock()
	progress(r.opts.Progress, "Kubernetes watch caches synced")
	publish(r.buildUpdate(ctx, informers, health))

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-dirty:
			timer := time.NewTimer(r.opts.Debounce)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
			}
			publish(r.buildUpdate(ctx, informers, health))
		}
	}
}

type watchedInformer struct {
	resource    string
	informer    cache.SharedIndexInformer
	publishOnly bool
}

func (r *kubernetesWatchRuntime) coreInformers(factory informers.SharedInformerFactory) []watchedInformer {
	return []watchedInformer{
		dataInformer("Namespaces", factory.Core().V1().Namespaces().Informer()),
		dataInformer("Nodes", factory.Core().V1().Nodes().Informer()),
		dataInformer("ServiceAccounts", factory.Core().V1().ServiceAccounts().Informer()),
		dataInformer("Pods", factory.Core().V1().Pods().Informer()),
		dataInformer("Services", factory.Core().V1().Services().Informer()),
		dataInformer("Deployments", factory.Apps().V1().Deployments().Informer()),
		dataInformer("DaemonSets", factory.Apps().V1().DaemonSets().Informer()),
		dataInformer("StatefulSets", factory.Apps().V1().StatefulSets().Informer()),
		dataInformer("ReplicaSets", factory.Apps().V1().ReplicaSets().Informer()),
		dataInformer("Jobs", factory.Batch().V1().Jobs().Informer()),
		dataInformer("CronJobs", factory.Batch().V1().CronJobs().Informer()),
		dataInformer("EndpointSlices", factory.Discovery().V1().EndpointSlices().Informer()),
		dataInformer("IngressClasses", factory.Networking().V1().IngressClasses().Informer()),
		dataInformer("Ingresses", factory.Networking().V1().Ingresses().Informer()),
		dataInformer("PersistentVolumes", factory.Core().V1().PersistentVolumes().Informer()),
		dataInformer("PersistentVolumeClaims", factory.Core().V1().PersistentVolumeClaims().Informer()),
		dataInformer("StorageClasses", factory.Storage().V1().StorageClasses().Informer()),
		dataInformer("PodDisruptionBudgets", factory.Policy().V1().PodDisruptionBudgets().Informer()),
		dataInformer("NetworkPolicies", factory.Networking().V1().NetworkPolicies().Informer()),
		dataInformer("ResourceQuotas", factory.Core().V1().ResourceQuotas().Informer()),
		dataInformer("LimitRanges", factory.Core().V1().LimitRanges().Informer()),
		dataInformer("HorizontalPodAutoscalers", factory.Autoscaling().V2().HorizontalPodAutoscalers().Informer()),
	}
}

func (r *kubernetesWatchRuntime) dynamicInformers(factory dynamicinformer.DynamicSharedInformerFactory, apiResources []inventory.APIResource) []watchedInformer {
	out := []watchedInformer{
		publishOnlyInformer("CustomResourceDefinitions", factory.ForResource(schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}).Informer()),
		publishOnlyInformer("APIServices", factory.ForResource(schema.GroupVersionResource{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"}).Informer()),
	}
	for _, candidate := range watchedGatewayResources(apiResources) {
		out = append(out, publishOnlyInformer(candidate.resource, factory.ForResource(candidate.gvr).Informer()))
	}
	return out
}

func dataInformer(resource string, informer cache.SharedIndexInformer) watchedInformer {
	return watchedInformer{resource: resource, informer: informer}
}

func publishOnlyInformer(resource string, informer cache.SharedIndexInformer) watchedInformer {
	return watchedInformer{resource: resource, informer: informer, publishOnly: true}
}

type watchedGatewayResource struct {
	resource string
	gvr      schema.GroupVersionResource
}

func watchedGatewayResources(apiResources []inventory.APIResource) []watchedGatewayResource {
	seen := map[string]bool{}
	var out []watchedGatewayResource
	for _, item := range apiResources {
		if item.Group != "gateway.networking.k8s.io" && item.Group != "gateway.networking.x-k8s.io" {
			continue
		}
		if !containsString(item.Verbs, "watch") {
			continue
		}
		name := gatewayWatchResourceName(item.Resource)
		if name == "" {
			continue
		}
		key := item.Group + "/" + item.Version + "/" + item.Resource
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, watchedGatewayResource{resource: name, gvr: schema.GroupVersionResource{Group: item.Group, Version: item.Version, Resource: item.Resource}})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].resource == out[j].resource {
			return out[i].gvr.String() < out[j].gvr.String()
		}
		return out[i].resource < out[j].resource
	})
	return out
}

func gatewayWatchResourceName(resource string) string {
	switch resource {
	case "gatewayclasses":
		return "GatewayClasses"
	case "gateways":
		return "Gateways"
	case "httproutes":
		return "HTTPRoutes"
	case "grpcroutes":
		return "GRPCRoutes"
	case "tlsroutes":
		return "TLSRoutes"
	case "tcproutes":
		return "TCPRoutes"
	case "udproutes":
		return "UDPRoutes"
	case "referencegrants":
		return "ReferenceGrants"
	case "backendtlspolicies":
		return "BackendTLSPolicies"
	case "backendtrafficpolicies":
		return "BackendTrafficPolicies"
	case "xbackendtrafficpolicies":
		return "XBackendTrafficPolicies"
	default:
		return ""
	}
}

func informerSyncs(informers []watchedInformer) []cache.InformerSynced {
	out := make([]cache.InformerSynced, 0, len(informers))
	for _, item := range informers {
		out = append(out, item.informer.HasSynced)
	}
	return out
}

type watchHealthState struct {
	mu           sync.Mutex
	state        string
	lastEventAt  time.Time
	lastRelistAt time.Time
	reconnects   int
	err          error
}

func (h *watchHealthState) recordEvent() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastEventAt = time.Now().UTC()
}

func (h *watchHealthState) recordRelist() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.state = "ready"
	h.err = nil
	h.lastRelistAt = time.Now().UTC()
}

func (h *watchHealthState) recordReady() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.state = "ready"
	h.err = nil
}

func (h *watchHealthState) recordError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.state = "stale"
	h.err = err
	h.reconnects++
}

func (h *watchHealthState) snapshot() WatchHealth {
	h.mu.Lock()
	defer h.mu.Unlock()
	return WatchHealth{
		State:        h.state,
		LastEventAt:  h.lastEventAt,
		LastRelistAt: h.lastRelistAt,
		Reconnects:   h.reconnects,
		Err:          h.err,
	}
}

func (r *kubernetesWatchRuntime) buildUpdate(ctx context.Context, informers []watchedInformer, health *watchHealthState) WatchUpdate {
	now := time.Now().UTC()
	data := inventory.Kubernetes{Context: r.contextName, Server: r.server}
	coverage := make([]inventory.CoverageItem, 0, len(informers)+1)
	apiResources, apiCoverage, err := r.discoverAPIResources(ctx)
	data.APIResources = apiResources
	coverage = append(coverage, apiCoverage)
	for _, item := range informers {
		if item.publishOnly {
			continue
		}
		count := r.addInformerData(&data, item)
		coverage = append(coverage, complete("kubernetes", item.resource, count, now))
	}
	if r.dynamic != nil {
		collectExtensions(ctx, r.dynamic, &data, &coverage, now)
		collectGateways(ctx, r.dynamic, &data, &coverage, now, r.contextNamespace)
	}
	data.RunningContainers = runningContainers(data)
	data.RunningImages = runningImages(data.RunningContainers)
	coverage = append(coverage, complete("kubernetes", "RunningContainers", len(data.RunningContainers), now))
	health.recordReady()
	snapshot := &inventory.Snapshot{
		SchemaVersion: "teleskope.io/snapshot/v1alpha1",
		CollectedAt:   now,
		Source:        inventory.Source{Tool: "teleskope", Version: buildinfo.Version, Mode: "live/watch"},
		Kubernetes:    data,
		Coverage:      coverage,
	}
	out := health.snapshot()
	if err != nil {
		out.Err = err
	}
	out.Coverage = append([]inventory.CoverageItem(nil), coverage...)
	return WatchUpdate{Snapshot: snapshot, Health: out}
}

func (r *kubernetesWatchRuntime) discoverAPIResources(ctx context.Context) ([]inventory.APIResource, inventory.CoverageItem, error) {
	now := time.Now().UTC()
	apiResources, err := collectAPIResources(ctx, r.client.Discovery())
	if err != nil {
		return apiResources, partial("kubernetes", "APIResources", len(apiResources), err, now), err
	}
	return apiResources, complete("kubernetes", "APIResources", len(apiResources), now), nil
}

func (r *kubernetesWatchRuntime) addInformerData(out *inventory.Kubernetes, item watchedInformer) int {
	switch item.resource {
	case "Namespaces":
		for _, ns := range storeItems[*corev1.Namespace](item.informer.GetStore()) {
			out.Namespaces = append(out.Namespaces, inventory.Namespace{ObjectRef: objectRef("v1", "Namespace", ns.Namespace, ns.Name, ns.UID), Phase: string(ns.Status.Phase), Labels: ns.Labels})
		}
		sortNamespaces(out.Namespaces)
		return len(out.Namespaces)
	case "Nodes":
		for _, node := range storeItems[*corev1.Node](item.informer.GetStore()) {
			out.Nodes = append(out.Nodes, mapNode(*node))
		}
		sortNodes(out.Nodes)
		return len(out.Nodes)
	case "ServiceAccounts":
		for _, serviceAccount := range storeItems[*corev1.ServiceAccount](item.informer.GetStore()) {
			out.ServiceAccounts = append(out.ServiceAccounts, mapServiceAccount(*serviceAccount))
		}
		sortServiceAccounts(out.ServiceAccounts)
		return len(out.ServiceAccounts)
	case "Pods":
		for _, pod := range storeItems[*corev1.Pod](item.informer.GetStore()) {
			out.Pods = append(out.Pods, mapPod(*pod))
		}
		sortPods(out.Pods)
		return len(out.Pods)
	case "Services":
		for _, service := range storeItems[*corev1.Service](item.informer.GetStore()) {
			out.Services = append(out.Services, mapService(*service))
		}
		sortServices(out.Services)
		return len(out.Services)
	case "Deployments":
		for _, deployment := range storeItems[*appsv1.Deployment](item.informer.GetStore()) {
			out.Workloads = append(out.Workloads, mapDeployment(*deployment))
		}
		return len(storeItems[*appsv1.Deployment](item.informer.GetStore()))
	case "DaemonSets":
		for _, daemonSet := range storeItems[*appsv1.DaemonSet](item.informer.GetStore()) {
			out.Workloads = append(out.Workloads, mapDaemonSet(*daemonSet))
		}
		return len(storeItems[*appsv1.DaemonSet](item.informer.GetStore()))
	case "StatefulSets":
		for _, statefulSet := range storeItems[*appsv1.StatefulSet](item.informer.GetStore()) {
			out.Workloads = append(out.Workloads, mapStatefulSet(*statefulSet))
		}
		return len(storeItems[*appsv1.StatefulSet](item.informer.GetStore()))
	case "ReplicaSets":
		for _, replicaSet := range storeItems[*appsv1.ReplicaSet](item.informer.GetStore()) {
			out.Workloads = append(out.Workloads, mapReplicaSet(*replicaSet))
		}
		return len(storeItems[*appsv1.ReplicaSet](item.informer.GetStore()))
	case "Jobs":
		for _, job := range storeItems[*batchv1.Job](item.informer.GetStore()) {
			out.Workloads = append(out.Workloads, mapJob(*job))
		}
		return len(storeItems[*batchv1.Job](item.informer.GetStore()))
	case "CronJobs":
		for _, cronJob := range storeItems[*batchv1.CronJob](item.informer.GetStore()) {
			out.Workloads = append(out.Workloads, mapCronJob(*cronJob))
		}
		sortWorkloads(out.Workloads)
		return len(storeItems[*batchv1.CronJob](item.informer.GetStore()))
	case "EndpointSlices":
		for _, endpointSlice := range storeItems[*discoveryv1.EndpointSlice](item.informer.GetStore()) {
			out.EndpointSlices = append(out.EndpointSlices, mapEndpointSlice(*endpointSlice))
		}
		sortEndpointSlices(out.EndpointSlices)
		return len(out.EndpointSlices)
	case "IngressClasses":
		for _, class := range storeItems[*networkingv1.IngressClass](item.informer.GetStore()) {
			out.IngressClasses = append(out.IngressClasses, mapIngressClass(*class))
		}
		sortIngressClasses(out.IngressClasses)
		return len(out.IngressClasses)
	case "Ingresses":
		for _, ingress := range storeItems[*networkingv1.Ingress](item.informer.GetStore()) {
			out.Ingresses = append(out.Ingresses, mapIngress(*ingress))
		}
		sortIngresses(out.Ingresses)
		return len(out.Ingresses)
	case "PersistentVolumes":
		for _, volume := range storeItems[*corev1.PersistentVolume](item.informer.GetStore()) {
			out.PersistentVolumes = append(out.PersistentVolumes, mapPersistentVolume(*volume))
		}
		sortPersistentVolumes(out.PersistentVolumes)
		return len(out.PersistentVolumes)
	case "PersistentVolumeClaims":
		for _, claim := range storeItems[*corev1.PersistentVolumeClaim](item.informer.GetStore()) {
			out.PersistentVolumeClaims = append(out.PersistentVolumeClaims, mapPersistentVolumeClaim(*claim))
		}
		sortPersistentVolumeClaims(out.PersistentVolumeClaims)
		return len(out.PersistentVolumeClaims)
	case "StorageClasses":
		for _, class := range storeItems[*storagev1.StorageClass](item.informer.GetStore()) {
			out.StorageClasses = append(out.StorageClasses, mapStorageClass(*class))
		}
		sortStorageClasses(out.StorageClasses)
		return len(out.StorageClasses)
	case "PodDisruptionBudgets":
		for _, pdb := range storeItems[*policyv1.PodDisruptionBudget](item.informer.GetStore()) {
			out.Policies.PodDisruptionBudgets = append(out.Policies.PodDisruptionBudgets, objectRef("policy/v1", "PodDisruptionBudget", pdb.Namespace, pdb.Name, pdb.UID))
			out.Policies.PodDisruptionBudgetDetails = append(out.Policies.PodDisruptionBudgetDetails, mapPodDisruptionBudget(*pdb))
		}
		sortObjectRefs(out.Policies.PodDisruptionBudgets)
		return len(out.Policies.PodDisruptionBudgets)
	case "NetworkPolicies":
		for _, policy := range storeItems[*networkingv1.NetworkPolicy](item.informer.GetStore()) {
			out.Policies.NetworkPolicies = append(out.Policies.NetworkPolicies, objectRef("networking.k8s.io/v1", "NetworkPolicy", policy.Namespace, policy.Name, policy.UID))
			out.Policies.NetworkPolicyDetails = append(out.Policies.NetworkPolicyDetails, mapNetworkPolicy(*policy))
		}
		sortObjectRefs(out.Policies.NetworkPolicies)
		return len(out.Policies.NetworkPolicies)
	case "ResourceQuotas":
		for _, quota := range storeItems[*corev1.ResourceQuota](item.informer.GetStore()) {
			out.Policies.ResourceQuotas = append(out.Policies.ResourceQuotas, objectRef("v1", "ResourceQuota", quota.Namespace, quota.Name, quota.UID))
			out.Policies.ResourceQuotaDetails = append(out.Policies.ResourceQuotaDetails, mapResourceQuota(*quota))
		}
		sortObjectRefs(out.Policies.ResourceQuotas)
		return len(out.Policies.ResourceQuotas)
	case "LimitRanges":
		for _, limit := range storeItems[*corev1.LimitRange](item.informer.GetStore()) {
			out.Policies.LimitRanges = append(out.Policies.LimitRanges, objectRef("v1", "LimitRange", limit.Namespace, limit.Name, limit.UID))
			out.Policies.LimitRangeDetails = append(out.Policies.LimitRangeDetails, mapLimitRange(*limit))
		}
		sortObjectRefs(out.Policies.LimitRanges)
		return len(out.Policies.LimitRanges)
	case "HorizontalPodAutoscalers":
		for _, hpa := range storeItems[*autoscalingv2.HorizontalPodAutoscaler](item.informer.GetStore()) {
			out.Policies.HorizontalPodAutoscalers = append(out.Policies.HorizontalPodAutoscalers, objectRef("autoscaling/v2", "HorizontalPodAutoscaler", hpa.Namespace, hpa.Name, hpa.UID))
		}
		sortObjectRefs(out.Policies.HorizontalPodAutoscalers)
		return len(out.Policies.HorizontalPodAutoscalers)
	default:
		return 0
	}
}

func storeItems[T runtime.Object](store cache.Store) []T {
	items := store.List()
	out := make([]T, 0, len(items))
	for _, item := range items {
		if typed, ok := item.(T); ok {
			out = append(out, typed)
		}
	}
	return out
}

func sortObjectRefs(items []inventory.ObjectRef) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Namespace == items[j].Namespace {
			return items[i].Name < items[j].Name
		}
		return items[i].Namespace < items[j].Namespace
	})
}

func sortNamespaces(items []inventory.Namespace) {
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
}
func sortNodes(items []inventory.Node) {
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
}
func sortServiceAccounts(items []inventory.ServiceAccount) {
	sort.Slice(items, func(i, j int) bool { return refLess(items[i].ObjectRef, items[j].ObjectRef) })
}
func sortPods(items []inventory.Pod) {
	sort.Slice(items, func(i, j int) bool { return refLess(items[i].ObjectRef, items[j].ObjectRef) })
}
func sortServices(items []inventory.Service) {
	sort.Slice(items, func(i, j int) bool { return refLess(items[i].ObjectRef, items[j].ObjectRef) })
}
func sortWorkloads(items []inventory.Workload) {
	sort.Slice(items, func(i, j int) bool { return refLess(items[i].ObjectRef, items[j].ObjectRef) })
}
func sortEndpointSlices(items []inventory.EndpointSlice) {
	sort.Slice(items, func(i, j int) bool { return refLess(items[i].ObjectRef, items[j].ObjectRef) })
}
func sortIngressClasses(items []inventory.IngressClass) {
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
}
func sortIngresses(items []inventory.Ingress) {
	sort.Slice(items, func(i, j int) bool { return refLess(items[i].ObjectRef, items[j].ObjectRef) })
}
func sortPersistentVolumes(items []inventory.PersistentVolume) {
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
}
func sortPersistentVolumeClaims(items []inventory.PersistentVolumeClaim) {
	sort.Slice(items, func(i, j int) bool { return refLess(items[i].ObjectRef, items[j].ObjectRef) })
}
func sortStorageClasses(items []inventory.StorageClass) {
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
}

func refLess(a, b inventory.ObjectRef) bool {
	if a.Namespace == b.Namespace {
		if a.Kind == b.Kind {
			return a.Name < b.Name
		}
		return a.Kind < b.Kind
	}
	return a.Namespace < b.Namespace
}
