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
	"k8s.io/apimachinery/pkg/runtime"
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
	opts        WatchOptions
	client      kubernetes.Interface
	contextName string
	server      string
}

// NewWatchRuntime initializes a watch runtime without starting informers.
func NewWatchRuntime(opts WatchOptions) (WatchRuntime, error) {
	config, contextName, _, server, err := loadRESTConfig(Options{
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
	return NewWatchRuntimeWithClient(client, opts, contextName, server), nil
}

// NewWatchRuntimeWithClient creates a watch runtime around an existing client.
func NewWatchRuntimeWithClient(client kubernetes.Interface, opts WatchOptions, contextName, server string) WatchRuntime {
	if opts.Debounce <= 0 {
		opts.Debounce = defaultWatchDebounce
	}
	return &kubernetesWatchRuntime{opts: opts, client: client, contextName: contextName, server: server}
}

func (r *kubernetesWatchRuntime) Run(ctx context.Context, publish func(WatchUpdate)) error {
	factory := informers.NewSharedInformerFactoryWithOptions(r.client, 0)
	informers := r.coreInformers(factory)
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
	progress(r.opts.Progress, "waiting for Kubernetes watch caches to sync resources=%d", len(informers))
	if !cache.WaitForCacheSync(ctx.Done(), informerSyncs(informers)...) {
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
	publish(r.buildUpdate(informers, health))

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
			publish(r.buildUpdate(informers, health))
		}
	}
}

type watchedInformer struct {
	resource string
	informer cache.SharedIndexInformer
}

func (r *kubernetesWatchRuntime) coreInformers(factory informers.SharedInformerFactory) []watchedInformer {
	return []watchedInformer{
		{"Namespaces", factory.Core().V1().Namespaces().Informer()},
		{"Nodes", factory.Core().V1().Nodes().Informer()},
		{"ServiceAccounts", factory.Core().V1().ServiceAccounts().Informer()},
		{"Pods", factory.Core().V1().Pods().Informer()},
		{"Services", factory.Core().V1().Services().Informer()},
		{"Deployments", factory.Apps().V1().Deployments().Informer()},
		{"DaemonSets", factory.Apps().V1().DaemonSets().Informer()},
		{"StatefulSets", factory.Apps().V1().StatefulSets().Informer()},
		{"ReplicaSets", factory.Apps().V1().ReplicaSets().Informer()},
		{"Jobs", factory.Batch().V1().Jobs().Informer()},
		{"CronJobs", factory.Batch().V1().CronJobs().Informer()},
		{"EndpointSlices", factory.Discovery().V1().EndpointSlices().Informer()},
		{"IngressClasses", factory.Networking().V1().IngressClasses().Informer()},
		{"Ingresses", factory.Networking().V1().Ingresses().Informer()},
		{"PersistentVolumes", factory.Core().V1().PersistentVolumes().Informer()},
		{"PersistentVolumeClaims", factory.Core().V1().PersistentVolumeClaims().Informer()},
		{"StorageClasses", factory.Storage().V1().StorageClasses().Informer()},
		{"PodDisruptionBudgets", factory.Policy().V1().PodDisruptionBudgets().Informer()},
		{"NetworkPolicies", factory.Networking().V1().NetworkPolicies().Informer()},
		{"ResourceQuotas", factory.Core().V1().ResourceQuotas().Informer()},
		{"LimitRanges", factory.Core().V1().LimitRanges().Informer()},
		{"HorizontalPodAutoscalers", factory.Autoscaling().V2().HorizontalPodAutoscalers().Informer()},
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

func (r *kubernetesWatchRuntime) buildUpdate(informers []watchedInformer, health *watchHealthState) WatchUpdate {
	now := time.Now().UTC()
	data := inventory.Kubernetes{Context: r.contextName, Server: r.server}
	coverage := make([]inventory.CoverageItem, 0, len(informers)+1)
	for _, item := range informers {
		count := r.addInformerData(&data, item)
		coverage = append(coverage, complete("kubernetes", item.resource, count, now))
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
	out.Coverage = append([]inventory.CoverageItem(nil), coverage...)
	return WatchUpdate{Snapshot: snapshot, Health: out}
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
