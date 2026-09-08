// Package k8s collects Kubernetes API inventory through client-go.
package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

// Options controls Kubernetes API collection.
type Options struct {
	Kubeconfig string
	Context    string
	Progress   func(format string, args ...any)
}

// Collect gathers Kubernetes inventory. Baseline connection errors are returned; resource-level collection errors are represented as coverage.
func Collect(ctx context.Context, opts Options) (inventory.Kubernetes, []inventory.CoverageItem, error) {
	now := time.Now().UTC()
	var coverage []inventory.CoverageItem

	config, contextName, server, err := loadRESTConfig(opts)
	if err != nil {
		return inventory.Kubernetes{}, nil, fmt.Errorf("connect Kubernetes cluster: %w", err)
	}
	progress(opts.Progress, "loaded kubeconfig context=%s server=%s", valueOrDefault(contextName, "-"), valueOrDefault(server, "-"))

	progress(opts.Progress, "creating Kubernetes clients")
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return inventory.Kubernetes{Context: contextName, Server: server}, nil, fmt.Errorf("connect Kubernetes cluster: create client: %w", err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return inventory.Kubernetes{Context: contextName, Server: server}, nil, fmt.Errorf("connect Kubernetes cluster: create dynamic client: %w", err)
	}
	metadataClient, err := metadata.NewForConfig(config)
	if err != nil {
		return inventory.Kubernetes{Context: contextName, Server: server}, nil, fmt.Errorf("connect Kubernetes cluster: create metadata client: %w", err)
	}

	kubernetes := inventory.Kubernetes{Context: contextName, Server: server}
	progress(opts.Progress, "checking Kubernetes API server version")
	version, err := client.Discovery().ServerVersion()
	if err != nil {
		return kubernetes, nil, fmt.Errorf("connect Kubernetes cluster: server version: %w", err)
	}
	kubernetes.Version = inventory.KubernetesVersion{
		GitVersion: version.GitVersion,
		Major:      version.Major,
		Minor:      version.Minor,
		Platform:   version.Platform,
	}
	coverage = append(coverage, complete("kubernetes", "ServerVersion", 1, now))
	progress(opts.Progress, "server version=%s platform=%s", valueOrDefault(kubernetes.Version.GitVersion, "-"), valueOrDefault(kubernetes.Version.Platform, "-"))

	progress(opts.Progress, "discovering API resources")
	apiResources, err := collectAPIResources(client.Discovery())
	kubernetes.APIResources = apiResources
	if err != nil {
		coverage = append(coverage, partial("kubernetes", "APIResources", len(apiResources), err, now))
	} else {
		coverage = append(coverage, complete("kubernetes", "APIResources", len(apiResources), now))
	}
	progress(opts.Progress, "API resources=%d", len(kubernetes.APIResources))

	progress(opts.Progress, "collecting core inventory")
	collectCore(ctx, client, metadataClient, &kubernetes, &coverage, now)
	progress(opts.Progress, "core namespaces=%d nodes=%d serviceAccounts=%d pods=%d configMaps=%d secrets=%d", len(kubernetes.Namespaces), len(kubernetes.Nodes), len(kubernetes.ServiceAccounts), len(kubernetes.Pods), len(kubernetes.ConfigMaps), len(kubernetes.Secrets))

	progress(opts.Progress, "collecting extension APIs and CRDs")
	collectExtensions(ctx, dynamicClient, &kubernetes, &coverage, now)
	progress(opts.Progress, "extensions crds=%d crInstances=%d apiServices=%d", len(kubernetes.CustomResourceDefinitions), len(kubernetes.CustomResourceInstances), len(kubernetes.APIServices))

	progress(opts.Progress, "collecting workloads")
	collectWorkloads(ctx, client, &kubernetes, &coverage, now)
	progress(opts.Progress, "workloads=%d", len(kubernetes.Workloads))

	progress(opts.Progress, "collecting networking")
	collectNetworking(ctx, client, &kubernetes, &coverage, now)
	progress(opts.Progress, "network services=%d endpointSlices=%d ingresses=%d ingressClasses=%d", len(kubernetes.Services), len(kubernetes.EndpointSlices), len(kubernetes.Ingresses), len(kubernetes.IngressClasses))

	progress(opts.Progress, "collecting storage")
	collectStorage(ctx, client, &kubernetes, &coverage, now)
	progress(opts.Progress, "storage classes=%d pv=%d pvc=%d csiDrivers=%d csiNodes=%d attachments=%d", len(kubernetes.StorageClasses), len(kubernetes.PersistentVolumes), len(kubernetes.PersistentVolumeClaims), len(kubernetes.CSIDrivers), len(kubernetes.CSINodes), len(kubernetes.VolumeAttachments))

	progress(opts.Progress, "collecting runtime classes")
	collectRuntime(ctx, client, &kubernetes, &coverage, now)
	progress(opts.Progress, "runtime classes=%d", len(kubernetes.RuntimeClasses))

	progress(opts.Progress, "collecting RBAC")
	collectRBAC(ctx, client, &kubernetes, &coverage, now)
	progress(opts.Progress, "rbac roles=%d roleBindings=%d clusterRoles=%d clusterRoleBindings=%d", len(kubernetes.RBAC.Roles), len(kubernetes.RBAC.RoleBindings), len(kubernetes.RBAC.ClusterRoles), len(kubernetes.RBAC.ClusterRoleBindings))

	progress(opts.Progress, "collecting policy objects")
	collectPolicies(ctx, client, &kubernetes, &coverage, now)
	progress(opts.Progress, "policies hpa=%d pdb=%d networkPolicies=%d resourceQuotas=%d limitRanges=%d", len(kubernetes.Policies.HorizontalPodAutoscalers), len(kubernetes.Policies.PodDisruptionBudgets), len(kubernetes.Policies.NetworkPolicies), len(kubernetes.Policies.ResourceQuotas), len(kubernetes.Policies.LimitRanges))

	progress(opts.Progress, "indexing running containers and images")
	kubernetes.RunningContainers = runningContainers(kubernetes)
	kubernetes.RunningImages = runningImages(kubernetes.RunningContainers)
	progress(opts.Progress, "running containers=%d images=%d", len(kubernetes.RunningContainers), len(kubernetes.RunningImages))

	return kubernetes, coverage, nil
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

func loadRESTConfig(opts Options) (*rest.Config, string, string, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.Kubeconfig != "" {
		rules.ExplicitPath = opts.Kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	if opts.Context != "" {
		overrides.CurrentContext = opts.Context
	}
	loading := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	raw, err := loading.RawConfig()
	if err != nil {
		return nil, "", "", fmt.Errorf("load kubeconfig: %w", err)
	}
	config, err := loading.ClientConfig()
	if err != nil {
		return nil, raw.CurrentContext, "", fmt.Errorf("build Kubernetes client config: %w", err)
	}
	contextName := raw.CurrentContext
	if opts.Context != "" {
		contextName = opts.Context
	}
	server := config.Host
	if ctxConfig, ok := raw.Contexts[contextName]; ok {
		if cluster, ok := raw.Clusters[ctxConfig.Cluster]; ok && cluster.Server != "" {
			server = cluster.Server
		}
	}
	return config, contextName, server, nil
}

func collectAPIResources(client discovery.DiscoveryInterface) ([]inventory.APIResource, error) {
	lists, err := client.ServerPreferredResources()
	resources := make([]inventory.APIResource, 0)
	for _, list := range lists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			gv = schema.GroupVersion{Version: list.GroupVersion}
		}
		for _, resource := range list.APIResources {
			resources = append(resources, inventory.APIResource{
				GroupVersion: list.GroupVersion,
				Group:        gv.Group,
				Version:      gv.Version,
				Resource:     resource.Name,
				Kind:         resource.Kind,
				Namespaced:   resource.Namespaced,
				Verbs:        resource.Verbs,
				Categories:   resource.Categories,
			})
		}
	}
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].GroupVersion == resources[j].GroupVersion {
			return resources[i].Resource < resources[j].Resource
		}
		return resources[i].GroupVersion < resources[j].GroupVersion
	})
	if err != nil && discovery.IsGroupDiscoveryFailedError(err) {
		return resources, err
	}
	return resources, err
}

func collectCore(ctx context.Context, client *kubernetes.Clientset, metadataClient metadata.Interface, out *inventory.Kubernetes, coverage *[]inventory.CoverageItem, now time.Time) {
	if namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "Namespaces", err, now))
	} else {
		out.Namespaces = make([]inventory.Namespace, 0, len(namespaces.Items))
		for _, ns := range namespaces.Items {
			out.Namespaces = append(out.Namespaces, inventory.Namespace{
				ObjectRef: objectRef("v1", "Namespace", ns.Namespace, ns.Name, ns.UID),
				Phase:     string(ns.Status.Phase),
				Labels:    ns.Labels,
			})
		}
		*coverage = append(*coverage, complete("kubernetes", "Namespaces", len(out.Namespaces), now))
	}

	if nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "Nodes", err, now))
	} else {
		out.Nodes = make([]inventory.Node, 0, len(nodes.Items))
		for _, node := range nodes.Items {
			out.Nodes = append(out.Nodes, mapNode(node))
		}
		*coverage = append(*coverage, complete("kubernetes", "Nodes", len(out.Nodes), now))
	}

	if serviceAccounts, err := client.CoreV1().ServiceAccounts("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "ServiceAccounts", err, now))
	} else {
		for _, serviceAccount := range serviceAccounts.Items {
			out.ServiceAccounts = append(out.ServiceAccounts, mapServiceAccount(serviceAccount))
		}
		*coverage = append(*coverage, complete("kubernetes", "ServiceAccounts", len(out.ServiceAccounts), now))
	}

	if pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "Pods", err, now))
	} else {
		for _, pod := range pods.Items {
			out.Pods = append(out.Pods, mapPod(pod))
		}
		*coverage = append(*coverage, complete("kubernetes", "Pods", len(out.Pods), now))
	}

	if services, err := client.CoreV1().Services("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "Services", err, now))
	} else {
		for _, service := range services.Items {
			out.Services = append(out.Services, mapService(service))
		}
		*coverage = append(*coverage, complete("kubernetes", "Services", len(out.Services), now))
	}

	if configMaps, err := client.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "ConfigMaps", err, now))
	} else {
		for _, configMap := range configMaps.Items {
			out.ConfigMaps = append(out.ConfigMaps, inventory.ConfigObject{
				ObjectRef:  objectRef("v1", "ConfigMap", configMap.Namespace, configMap.Name, configMap.UID),
				Data:       configMap.Data,
				BinaryKeys: sortedMapKeys(configMap.BinaryData),
			})
		}
		*coverage = append(*coverage, complete("kubernetes", "ConfigMaps", len(out.ConfigMaps), now))
	}

	secretGVR := schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
	if secrets, err := metadataClient.Resource(secretGVR).Namespace("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "Secrets", err, now))
	} else {
		for _, secret := range secrets.Items {
			out.Secrets = append(out.Secrets, inventory.Secret{
				ObjectRef: objectRef("v1", "Secret", secret.Namespace, secret.Name, secret.UID),
			})
		}
		*coverage = append(*coverage, complete("kubernetes", "SecretsMetadata", len(out.Secrets), now))
	}
}

func collectExtensions(ctx context.Context, client dynamic.Interface, out *inventory.Kubernetes, coverage *[]inventory.CoverageItem, now time.Time) {
	crdGVR := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	if list, err := client.Resource(crdGVR).List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "CustomResourceDefinitions", err, now))
	} else {
		for _, item := range list.Items {
			out.CustomResourceDefinitions = append(out.CustomResourceDefinitions, mapCRD(item))
		}
		*coverage = append(*coverage, complete("kubernetes", "CustomResourceDefinitions", len(out.CustomResourceDefinitions), now))
		collectCustomResourceInstances(ctx, client, out, coverage, now)
	}

	apiServiceGVR := schema.GroupVersionResource{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"}
	if list, err := client.Resource(apiServiceGVR).List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "APIServices", err, now))
	} else {
		for _, item := range list.Items {
			out.APIServices = append(out.APIServices, mapAPIService(item))
		}
		*coverage = append(*coverage, complete("kubernetes", "APIServices", len(out.APIServices), now))
	}
}

func collectCustomResourceInstances(ctx context.Context, client dynamic.Interface, out *inventory.Kubernetes, coverage *[]inventory.CoverageItem, now time.Time) {
	start := len(out.CustomResourceInstances)
	var failed []string
	for _, crd := range out.CustomResourceDefinitions {
		version := selectedCRDVersion(crd)
		if version == "" || crd.Group == "" || crd.Plural == "" {
			failed = append(failed, crd.Name+": missing group, plural, or served version")
			continue
		}
		gvr := schema.GroupVersionResource{Group: crd.Group, Version: version, Resource: crd.Plural}
		resource := client.Resource(gvr)
		var list *unstructured.UnstructuredList
		var err error
		if crd.Scope == "Namespaced" {
			list, err = resource.Namespace("").List(ctx, metav1.ListOptions{})
		} else {
			list, err = resource.List(ctx, metav1.ListOptions{})
		}
		if err != nil {
			failed = append(failed, crd.Name+": "+err.Error())
			continue
		}
		for _, item := range list.Items {
			out.CustomResourceInstances = append(out.CustomResourceInstances, mapCustomResourceInstance(item, crd, version))
		}
	}
	out.CustomResourceCounts = customResourceCounts(out.CustomResourceDefinitions, out.CustomResourceInstances)
	collected := len(out.CustomResourceInstances) - start
	switch {
	case len(out.CustomResourceDefinitions) == 0:
		*coverage = append(*coverage, complete("kubernetes", "CustomResourceInstances", 0, now))
	case len(failed) > 0:
		*coverage = append(*coverage, partial("kubernetes", "CustomResourceInstances", collected, fmt.Errorf("%s", strings.Join(failed, "; ")), now))
	default:
		*coverage = append(*coverage, complete("kubernetes", "CustomResourceInstances", collected, now))
	}
}

func collectWorkloads(ctx context.Context, client *kubernetes.Clientset, out *inventory.Kubernetes, coverage *[]inventory.CoverageItem, now time.Time) {
	start := len(out.Workloads)
	if deployments, err := client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "Deployments", err, now))
	} else {
		for _, deployment := range deployments.Items {
			out.Workloads = append(out.Workloads, mapDeployment(deployment))
		}
		*coverage = append(*coverage, complete("kubernetes", "Deployments", len(deployments.Items), now))
	}
	if daemonSets, err := client.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "DaemonSets", err, now))
	} else {
		for _, daemonSet := range daemonSets.Items {
			out.Workloads = append(out.Workloads, mapDaemonSet(daemonSet))
		}
		*coverage = append(*coverage, complete("kubernetes", "DaemonSets", len(daemonSets.Items), now))
	}
	if statefulSets, err := client.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "StatefulSets", err, now))
	} else {
		for _, statefulSet := range statefulSets.Items {
			out.Workloads = append(out.Workloads, mapStatefulSet(statefulSet))
		}
		*coverage = append(*coverage, complete("kubernetes", "StatefulSets", len(statefulSets.Items), now))
	}
	if replicaSets, err := client.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "ReplicaSets", err, now))
	} else {
		for _, replicaSet := range replicaSets.Items {
			out.Workloads = append(out.Workloads, mapReplicaSet(replicaSet))
		}
		*coverage = append(*coverage, complete("kubernetes", "ReplicaSets", len(replicaSets.Items), now))
	}
	if jobs, err := client.BatchV1().Jobs("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "Jobs", err, now))
	} else {
		for _, job := range jobs.Items {
			out.Workloads = append(out.Workloads, mapJob(job))
		}
		*coverage = append(*coverage, complete("kubernetes", "Jobs", len(jobs.Items), now))
	}
	if cronJobs, err := client.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "CronJobs", err, now))
	} else {
		for _, cronJob := range cronJobs.Items {
			out.Workloads = append(out.Workloads, mapCronJob(cronJob))
		}
		*coverage = append(*coverage, complete("kubernetes", "CronJobs", len(cronJobs.Items), now))
	}
	*coverage = append(*coverage, complete("kubernetes", "Workloads", len(out.Workloads)-start, now))
}

func collectNetworking(ctx context.Context, client *kubernetes.Clientset, out *inventory.Kubernetes, coverage *[]inventory.CoverageItem, now time.Time) {
	if endpointSlices, err := client.DiscoveryV1().EndpointSlices("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "EndpointSlices", err, now))
	} else {
		for _, endpointSlice := range endpointSlices.Items {
			out.EndpointSlices = append(out.EndpointSlices, mapEndpointSlice(endpointSlice))
		}
		*coverage = append(*coverage, complete("kubernetes", "EndpointSlices", len(out.EndpointSlices), now))
	}
	if classes, err := client.NetworkingV1().IngressClasses().List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "IngressClasses", err, now))
	} else {
		for _, class := range classes.Items {
			out.IngressClasses = append(out.IngressClasses, mapIngressClass(class))
		}
		*coverage = append(*coverage, complete("kubernetes", "IngressClasses", len(out.IngressClasses), now))
	}
	if ingresses, err := client.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "Ingresses", err, now))
	} else {
		for _, ingress := range ingresses.Items {
			out.Ingresses = append(out.Ingresses, mapIngress(ingress))
		}
		*coverage = append(*coverage, complete("kubernetes", "Ingresses", len(out.Ingresses), now))
	}
}

func collectStorage(ctx context.Context, client *kubernetes.Clientset, out *inventory.Kubernetes, coverage *[]inventory.CoverageItem, now time.Time) {
	if classes, err := client.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "StorageClasses", err, now))
	} else {
		for _, class := range classes.Items {
			out.StorageClasses = append(out.StorageClasses, mapStorageClass(class))
		}
		*coverage = append(*coverage, complete("kubernetes", "StorageClasses", len(out.StorageClasses), now))
	}
	if volumes, err := client.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "PersistentVolumes", err, now))
	} else {
		for _, volume := range volumes.Items {
			out.PersistentVolumes = append(out.PersistentVolumes, mapPersistentVolume(volume))
		}
		*coverage = append(*coverage, complete("kubernetes", "PersistentVolumes", len(out.PersistentVolumes), now))
	}
	if claims, err := client.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "PersistentVolumeClaims", err, now))
	} else {
		for _, claim := range claims.Items {
			out.PersistentVolumeClaims = append(out.PersistentVolumeClaims, mapPersistentVolumeClaim(claim))
		}
		*coverage = append(*coverage, complete("kubernetes", "PersistentVolumeClaims", len(out.PersistentVolumeClaims), now))
	}
	if drivers, err := client.StorageV1().CSIDrivers().List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "CSIDrivers", err, now))
	} else {
		for _, driver := range drivers.Items {
			out.CSIDrivers = append(out.CSIDrivers, mapCSIDriver(driver))
		}
		*coverage = append(*coverage, complete("kubernetes", "CSIDrivers", len(out.CSIDrivers), now))
	}
	if nodes, err := client.StorageV1().CSINodes().List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "CSINodes", err, now))
	} else {
		for _, node := range nodes.Items {
			out.CSINodes = append(out.CSINodes, mapCSINode(node))
		}
		*coverage = append(*coverage, complete("kubernetes", "CSINodes", len(out.CSINodes), now))
	}
	if attachments, err := client.StorageV1().VolumeAttachments().List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "VolumeAttachments", err, now))
	} else {
		for _, attachment := range attachments.Items {
			out.VolumeAttachments = append(out.VolumeAttachments, mapVolumeAttachment(attachment))
		}
		*coverage = append(*coverage, complete("kubernetes", "VolumeAttachments", len(out.VolumeAttachments), now))
	}
}

func collectRuntime(ctx context.Context, client *kubernetes.Clientset, out *inventory.Kubernetes, coverage *[]inventory.CoverageItem, now time.Time) {
	if classes, err := client.NodeV1().RuntimeClasses().List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "RuntimeClasses", err, now))
	} else {
		for _, class := range classes.Items {
			out.RuntimeClasses = append(out.RuntimeClasses, inventory.RuntimeClass{
				ObjectRef: objectRef("node.k8s.io/v1", "RuntimeClass", class.Namespace, class.Name, class.UID),
				Handler:   class.Handler,
			})
		}
		*coverage = append(*coverage, complete("kubernetes", "RuntimeClasses", len(out.RuntimeClasses), now))
	}
}

func collectRBAC(ctx context.Context, client *kubernetes.Clientset, out *inventory.Kubernetes, coverage *[]inventory.CoverageItem, now time.Time) {
	if roles, err := client.RbacV1().Roles("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "Roles", err, now))
	} else {
		out.RBAC.Roles = roleRefs(roles.Items)
		*coverage = append(*coverage, complete("kubernetes", "Roles", len(out.RBAC.Roles), now))
	}
	if bindings, err := client.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "RoleBindings", err, now))
	} else {
		out.RBAC.RoleBindings = roleBindingRefs(bindings.Items)
		*coverage = append(*coverage, complete("kubernetes", "RoleBindings", len(out.RBAC.RoleBindings), now))
	}
	if roles, err := client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "ClusterRoles", err, now))
	} else {
		out.RBAC.ClusterRoles = clusterRoleRefs(roles.Items)
		*coverage = append(*coverage, complete("kubernetes", "ClusterRoles", len(out.RBAC.ClusterRoles), now))
	}
	if bindings, err := client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "ClusterRoleBindings", err, now))
	} else {
		out.RBAC.ClusterRoleBindings = clusterRoleBindingRefs(bindings.Items)
		*coverage = append(*coverage, complete("kubernetes", "ClusterRoleBindings", len(out.RBAC.ClusterRoleBindings), now))
	}
}

func collectPolicies(ctx context.Context, client *kubernetes.Clientset, out *inventory.Kubernetes, coverage *[]inventory.CoverageItem, now time.Time) {
	if hpas, err := client.AutoscalingV2().HorizontalPodAutoscalers("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "HorizontalPodAutoscalers", err, now))
	} else {
		out.Policies.HorizontalPodAutoscalers = hpaRefs(hpas.Items)
		*coverage = append(*coverage, complete("kubernetes", "HorizontalPodAutoscalers", len(out.Policies.HorizontalPodAutoscalers), now))
	}
	if pdbs, err := client.PolicyV1().PodDisruptionBudgets("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "PodDisruptionBudgets", err, now))
	} else {
		out.Policies.PodDisruptionBudgets = pdbRefs(pdbs.Items)
		*coverage = append(*coverage, complete("kubernetes", "PodDisruptionBudgets", len(out.Policies.PodDisruptionBudgets), now))
	}
	if policies, err := client.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "NetworkPolicies", err, now))
	} else {
		out.Policies.NetworkPolicies = networkPolicyRefs(policies.Items)
		*coverage = append(*coverage, complete("kubernetes", "NetworkPolicies", len(out.Policies.NetworkPolicies), now))
	}
	if quotas, err := client.CoreV1().ResourceQuotas("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "ResourceQuotas", err, now))
	} else {
		out.Policies.ResourceQuotas = resourceQuotaRefs(quotas.Items)
		*coverage = append(*coverage, complete("kubernetes", "ResourceQuotas", len(out.Policies.ResourceQuotas), now))
	}
	if limits, err := client.CoreV1().LimitRanges("").List(ctx, metav1.ListOptions{}); err != nil {
		*coverage = append(*coverage, denied("kubernetes", "LimitRanges", err, now))
	} else {
		out.Policies.LimitRanges = limitRangeRefs(limits.Items)
		*coverage = append(*coverage, complete("kubernetes", "LimitRanges", len(out.Policies.LimitRanges), now))
	}
}

func mapNode(node corev1.Node) inventory.Node {
	out := inventory.Node{
		ObjectRef:        objectRef("v1", "Node", node.Namespace, node.Name, node.UID),
		ProviderID:       node.Spec.ProviderID,
		Unschedulable:    node.Spec.Unschedulable,
		Labels:           node.Labels,
		KubeletVersion:   node.Status.NodeInfo.KubeletVersion,
		ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
		OSImage:          node.Status.NodeInfo.OSImage,
		KernelVersion:    node.Status.NodeInfo.KernelVersion,
		Architecture:     node.Status.NodeInfo.Architecture,
		OperatingSystem:  node.Status.NodeInfo.OperatingSystem,
		Capacity:         resourceList(node.Status.Capacity),
		Allocatable:      resourceList(node.Status.Allocatable),
		NodeLocalBlindSpots: []string{
			"container runtime config",
			"runtime handlers installed on node",
			"snapshotter and registry mirror config",
			"CNI files on disk",
		},
	}
	for _, taint := range node.Spec.Taints {
		out.Taints = append(out.Taints, inventory.Taint{Key: taint.Key, Value: taint.Value, Effect: string(taint.Effect)})
	}
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			out.Ready = string(condition.Status)
			break
		}
	}
	return out
}

func mapServiceAccount(serviceAccount corev1.ServiceAccount) inventory.ServiceAccount {
	out := inventory.ServiceAccount{
		ObjectRef:   objectRef("v1", "ServiceAccount", serviceAccount.Namespace, serviceAccount.Name, serviceAccount.UID),
		Annotations: serviceAccount.Annotations,
	}
	for _, ref := range serviceAccount.ImagePullSecrets {
		out.ImagePullSecrets = append(out.ImagePullSecrets, ref.Name)
	}
	for _, ref := range serviceAccount.Secrets {
		out.Secrets = append(out.Secrets, ref.Name)
	}
	return out
}

func mapCRD(item unstructured.Unstructured) inventory.CustomResourceDefinition {
	out := inventory.CustomResourceDefinition{
		ObjectRef: objectRef("apiextensions.k8s.io/v1", "CustomResourceDefinition", item.GetNamespace(), item.GetName(), item.GetUID()),
	}
	out.Group, _, _ = unstructured.NestedString(item.Object, "spec", "group")
	out.Scope, _, _ = unstructured.NestedString(item.Object, "spec", "scope")
	out.Kind, _, _ = unstructured.NestedString(item.Object, "spec", "names", "kind")
	out.Plural, _, _ = unstructured.NestedString(item.Object, "spec", "names", "plural")
	versions, _, _ := unstructured.NestedSlice(item.Object, "spec", "versions")
	for _, raw := range versions {
		version, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name, _, _ := unstructured.NestedString(version, "name")
		served, _, _ := unstructured.NestedBool(version, "served")
		storage, _, _ := unstructured.NestedBool(version, "storage")
		out.Versions = append(out.Versions, inventory.CRDVersion{Name: name, Served: served, Storage: storage})
	}
	return out
}

func mapCustomResourceInstance(item unstructured.Unstructured, crd inventory.CustomResourceDefinition, version string) inventory.CustomResourceInstance {
	return inventory.CustomResourceInstance{
		ObjectRef:       objectRef(crd.Group+"/"+version, crd.Kind, item.GetNamespace(), item.GetName(), item.GetUID()),
		CRDName:         crd.Name,
		CRDGroup:        crd.Group,
		CRDVersion:      version,
		CRDKind:         crd.Kind,
		CRDPlural:       crd.Plural,
		Labels:          item.GetLabels(),
		OwnerReferences: ownerRefs(item.GetOwnerReferences()),
	}
}

func mapAPIService(item unstructured.Unstructured) inventory.APIService {
	out := inventory.APIService{
		ObjectRef: objectRef("apiregistration.k8s.io/v1", "APIService", item.GetNamespace(), item.GetName(), item.GetUID()),
	}
	out.Group, _, _ = unstructured.NestedString(item.Object, "spec", "group")
	out.Version, _, _ = unstructured.NestedString(item.Object, "spec", "version")
	out.ServiceNamespace, _, _ = unstructured.NestedString(item.Object, "spec", "service", "namespace")
	out.ServiceName, _, _ = unstructured.NestedString(item.Object, "spec", "service", "name")
	conditions, _, _ := unstructured.NestedSlice(item.Object, "status", "conditions")
	for _, raw := range conditions {
		condition, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		conditionType, _, _ := unstructured.NestedString(condition, "type")
		if conditionType != "Available" {
			continue
		}
		out.Available, _, _ = unstructured.NestedString(condition, "status")
		break
	}
	return out
}

func selectedCRDVersion(crd inventory.CustomResourceDefinition) string {
	for _, version := range crd.Versions {
		if version.Storage && version.Served {
			return version.Name
		}
	}
	for _, version := range crd.Versions {
		if version.Storage {
			return version.Name
		}
	}
	for _, version := range crd.Versions {
		if version.Served {
			return version.Name
		}
	}
	return ""
}

func customResourceCounts(crds []inventory.CustomResourceDefinition, instances []inventory.CustomResourceInstance) []inventory.CustomResourceCount {
	type aggregate struct {
		count      inventory.CustomResourceCount
		namespaces map[string]struct{}
	}
	byCRD := make(map[string]*aggregate, len(crds))
	for _, crd := range crds {
		version := selectedCRDVersion(crd)
		byCRD[crd.Name] = &aggregate{
			count: inventory.CustomResourceCount{
				CRDName: crd.Name,
				Group:   crd.Group,
				Version: version,
				Kind:    crd.Kind,
				Plural:  crd.Plural,
				Scope:   crd.Scope,
			},
			namespaces: map[string]struct{}{},
		}
	}
	for _, instance := range instances {
		agg, ok := byCRD[instance.CRDName]
		if !ok {
			agg = &aggregate{
				count: inventory.CustomResourceCount{
					CRDName: instance.CRDName,
					Group:   instance.CRDGroup,
					Version: instance.CRDVersion,
					Kind:    instance.CRDKind,
					Plural:  instance.CRDPlural,
				},
				namespaces: map[string]struct{}{},
			}
			byCRD[instance.CRDName] = agg
		}
		agg.count.InstanceCount++
		if instance.Namespace != "" {
			agg.namespaces[instance.Namespace] = struct{}{}
		}
	}
	counts := make([]inventory.CustomResourceCount, 0, len(byCRD))
	for _, agg := range byCRD {
		agg.count.NamespaceCount = len(agg.namespaces)
		counts = append(counts, agg.count)
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].InstanceCount != counts[j].InstanceCount {
			return counts[i].InstanceCount > counts[j].InstanceCount
		}
		if counts[i].Group != counts[j].Group {
			return counts[i].Group < counts[j].Group
		}
		return counts[i].Kind < counts[j].Kind
	})
	return counts
}

func mapPod(pod corev1.Pod) inventory.Pod {
	containers, initContainers, volumes, configRefs, secretRefs, pullSecrets := podSpecDetails(pod.Namespace, pod.Spec)
	_ = configRefs
	_ = secretRefs
	_ = pullSecrets
	out := inventory.Pod{
		ObjectRef:           objectRef("v1", "Pod", pod.Namespace, pod.Name, pod.UID),
		Phase:               string(pod.Status.Phase),
		NodeName:            pod.Spec.NodeName,
		ServiceAccountName:  pod.Spec.ServiceAccountName,
		PodIP:               pod.Status.PodIP,
		HostIP:              pod.Status.HostIP,
		Containers:          containersWithStatus(containers, pod.Status.ContainerStatuses),
		InitContainers:      containersWithStatus(initContainers, pod.Status.InitContainerStatuses),
		EphemeralContainers: containersFromStatuses("ephemeral", pod.Status.EphemeralContainerStatuses),
		Volumes:             volumes,
		OwnerReferences:     ownerRefs(pod.OwnerReferences),
	}
	if pod.Spec.RuntimeClassName != nil {
		out.RuntimeClassName = *pod.Spec.RuntimeClassName
	}
	return out
}

func mapService(service corev1.Service) inventory.Service {
	out := inventory.Service{
		ObjectRef:   objectRef("v1", "Service", service.Namespace, service.Name, service.UID),
		Type:        string(service.Spec.Type),
		ClusterIP:   service.Spec.ClusterIP,
		ExternalIPs: service.Spec.ExternalIPs,
		Selector:    service.Spec.Selector,
	}
	for _, port := range service.Spec.Ports {
		out.Ports = append(out.Ports, inventory.ServicePort{
			Name:       port.Name,
			Protocol:   string(port.Protocol),
			Port:       port.Port,
			TargetPort: intOrString(port.TargetPort),
			NodePort:   port.NodePort,
		})
	}
	return out
}

func mapDeployment(deployment appsv1.Deployment) inventory.Workload {
	out := workloadFromTemplate("apps/v1", "Deployment", deployment.Namespace, deployment.Name, deployment.UID, deployment.OwnerReferences, deployment.Spec.Template)
	out.Replicas = deployment.Spec.Replicas
	out.ReadyReplicas = deployment.Status.ReadyReplicas
	out.AvailableReplicas = deployment.Status.AvailableReplicas
	out.Selector = selectorMap(deployment.Spec.Selector)
	return out
}

func mapDaemonSet(daemonSet appsv1.DaemonSet) inventory.Workload {
	out := workloadFromTemplate("apps/v1", "DaemonSet", daemonSet.Namespace, daemonSet.Name, daemonSet.UID, daemonSet.OwnerReferences, daemonSet.Spec.Template)
	out.ReadyReplicas = daemonSet.Status.NumberReady
	out.AvailableReplicas = daemonSet.Status.NumberAvailable
	out.Selector = selectorMap(daemonSet.Spec.Selector)
	return out
}

func mapStatefulSet(statefulSet appsv1.StatefulSet) inventory.Workload {
	out := workloadFromTemplate("apps/v1", "StatefulSet", statefulSet.Namespace, statefulSet.Name, statefulSet.UID, statefulSet.OwnerReferences, statefulSet.Spec.Template)
	out.Replicas = statefulSet.Spec.Replicas
	out.ReadyReplicas = statefulSet.Status.ReadyReplicas
	out.AvailableReplicas = statefulSet.Status.AvailableReplicas
	out.Selector = selectorMap(statefulSet.Spec.Selector)
	for _, claim := range statefulSet.Spec.VolumeClaimTemplates {
		out.VolumeClaimTemplates = append(out.VolumeClaimTemplates, objectRef("v1", "PersistentVolumeClaim", statefulSet.Namespace, claim.Name, claim.UID))
	}
	return out
}

func mapReplicaSet(replicaSet appsv1.ReplicaSet) inventory.Workload {
	out := workloadFromTemplate("apps/v1", "ReplicaSet", replicaSet.Namespace, replicaSet.Name, replicaSet.UID, replicaSet.OwnerReferences, replicaSet.Spec.Template)
	out.Replicas = replicaSet.Spec.Replicas
	out.ReadyReplicas = replicaSet.Status.ReadyReplicas
	out.AvailableReplicas = replicaSet.Status.AvailableReplicas
	out.Selector = selectorMap(replicaSet.Spec.Selector)
	return out
}

func mapJob(job batchv1.Job) inventory.Workload {
	out := workloadFromTemplate("batch/v1", "Job", job.Namespace, job.Name, job.UID, job.OwnerReferences, job.Spec.Template)
	if job.Status.Ready != nil {
		out.ReadyReplicas = *job.Status.Ready
	}
	return out
}

func mapCronJob(cronJob batchv1.CronJob) inventory.Workload {
	out := workloadFromTemplate("batch/v1", "CronJob", cronJob.Namespace, cronJob.Name, cronJob.UID, cronJob.OwnerReferences, cronJob.Spec.JobTemplate.Spec.Template)
	return out
}

func workloadFromTemplate(apiVersion, kind, namespace, name string, uid any, owners []metav1.OwnerReference, template corev1.PodTemplateSpec) inventory.Workload {
	containers, initContainers, volumes, configRefs, secretRefs, pullSecrets := podSpecDetails(namespace, template.Spec)
	out := inventory.Workload{
		ObjectRef:           objectRef(apiVersion, kind, namespace, name, uid),
		ServiceAccountName:  template.Spec.ServiceAccountName,
		NodeSelector:        template.Spec.NodeSelector,
		Containers:          containers,
		InitContainers:      initContainers,
		Volumes:             volumes,
		ConfigRefs:          configRefs,
		SecretRefs:          secretRefs,
		ImagePullSecretRefs: pullSecrets,
		OwnerReferences:     ownerRefs(owners),
	}
	if template.Spec.RuntimeClassName != nil {
		out.RuntimeClassName = *template.Spec.RuntimeClassName
	}
	return out
}

func mapEndpointSlice(endpointSlice discoveryv1.EndpointSlice) inventory.EndpointSlice {
	out := inventory.EndpointSlice{
		ObjectRef:   objectRef("discovery.k8s.io/v1", "EndpointSlice", endpointSlice.Namespace, endpointSlice.Name, endpointSlice.UID),
		AddressType: string(endpointSlice.AddressType),
		ServiceName: endpointSlice.Labels[discoveryv1.LabelServiceName],
	}
	for _, port := range endpointSlice.Ports {
		out.Ports = append(out.Ports, inventory.EndpointPort{Name: stringPtr(port.Name), Protocol: protocolPtr(port.Protocol), Port: port.Port})
	}
	for _, endpoint := range endpointSlice.Endpoints {
		target := inventory.EndpointTarget{
			Addresses: endpoint.Addresses,
			Ready:     endpoint.Conditions.Ready,
			NodeName:  stringPtr(endpoint.NodeName),
			Zone:      stringPtr(endpoint.Zone),
		}
		if endpoint.TargetRef != nil {
			target.TargetRef = objectRef(endpoint.TargetRef.APIVersion, endpoint.TargetRef.Kind, endpoint.TargetRef.Namespace, endpoint.TargetRef.Name, endpoint.TargetRef.UID)
		}
		out.Endpoints = append(out.Endpoints, target)
	}
	return out
}

func mapIngressClass(class networkingv1.IngressClass) inventory.IngressClass {
	out := inventory.IngressClass{
		ObjectRef:  objectRef("networking.k8s.io/v1", "IngressClass", class.Namespace, class.Name, class.UID),
		Controller: class.Spec.Controller,
	}
	if class.Spec.Parameters != nil {
		out.Parameters = inventory.ObjectRef{
			APIVersion: stringPtr(class.Spec.Parameters.APIGroup),
			Kind:       class.Spec.Parameters.Kind,
			Namespace:  stringPtr(class.Spec.Parameters.Namespace),
			Name:       class.Spec.Parameters.Name,
		}
	}
	return out
}

func mapIngress(ingress networkingv1.Ingress) inventory.Ingress {
	out := inventory.Ingress{
		ObjectRef: objectRef("networking.k8s.io/v1", "Ingress", ingress.Namespace, ingress.Name, ingress.UID),
	}
	if ingress.Spec.IngressClassName != nil {
		out.ClassName = *ingress.Spec.IngressClassName
	}
	for _, tls := range ingress.Spec.TLS {
		out.TLSHosts = append(out.TLSHosts, tls.Hosts...)
	}
	addBackend := func(backend networkingv1.IngressBackend) {
		if backend.Service != nil {
			out.Backends = append(out.Backends, objectRef("v1", "Service", ingress.Namespace, backend.Service.Name, ""))
		}
	}
	if ingress.Spec.DefaultBackend != nil {
		addBackend(*ingress.Spec.DefaultBackend)
	}
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}
			service := path.Backend.Service
			out.Rules = append(out.Rules, inventory.IngressRule{
				Host:        rule.Host,
				Path:        path.Path,
				PathType:    pathType(path.PathType),
				ServiceName: service.Name,
				ServicePort: serviceBackendPort(service.Port),
			})
			addBackend(path.Backend)
		}
	}
	return out
}

func mapStorageClass(class storagev1.StorageClass) inventory.StorageClass {
	out := inventory.StorageClass{
		ObjectRef:            objectRef("storage.k8s.io/v1", "StorageClass", class.Namespace, class.Name, class.UID),
		Provisioner:          class.Provisioner,
		Parameters:           class.Parameters,
		AllowVolumeExpansion: class.AllowVolumeExpansion,
	}
	if class.ReclaimPolicy != nil {
		out.ReclaimPolicy = string(*class.ReclaimPolicy)
	}
	if class.VolumeBindingMode != nil {
		out.VolumeBindingMode = string(*class.VolumeBindingMode)
	}
	return out
}

func mapPersistentVolume(volume corev1.PersistentVolume) inventory.PersistentVolume {
	out := inventory.PersistentVolume{
		ObjectRef:        objectRef("v1", "PersistentVolume", volume.Namespace, volume.Name, volume.UID),
		StorageClassName: volume.Spec.StorageClassName,
		AccessModes:      accessModes(volume.Spec.AccessModes),
		Phase:            string(volume.Status.Phase),
	}
	if storage, ok := volume.Spec.Capacity[corev1.ResourceStorage]; ok {
		out.Capacity = storage.String()
	}
	if volume.Spec.VolumeMode != nil {
		out.VolumeMode = string(*volume.Spec.VolumeMode)
	}
	if volume.Spec.ClaimRef != nil {
		out.ClaimRef = objectRef(volume.Spec.ClaimRef.APIVersion, volume.Spec.ClaimRef.Kind, volume.Spec.ClaimRef.Namespace, volume.Spec.ClaimRef.Name, volume.Spec.ClaimRef.UID)
	}
	if volume.Spec.CSI != nil {
		out.CSI = &inventory.CSIVolume{Driver: volume.Spec.CSI.Driver, VolumeHandle: volume.Spec.CSI.VolumeHandle, FSType: volume.Spec.CSI.FSType}
	}
	return out
}

func mapPersistentVolumeClaim(claim corev1.PersistentVolumeClaim) inventory.PersistentVolumeClaim {
	out := inventory.PersistentVolumeClaim{
		ObjectRef:        objectRef("v1", "PersistentVolumeClaim", claim.Namespace, claim.Name, claim.UID),
		StorageClassName: stringPtr(claim.Spec.StorageClassName),
		VolumeName:       claim.Spec.VolumeName,
		AccessModes:      accessModes(claim.Spec.AccessModes),
		Phase:            string(claim.Status.Phase),
	}
	if storage, ok := claim.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		out.RequestedStorage = storage.String()
	}
	if claim.Spec.VolumeMode != nil {
		out.VolumeMode = string(*claim.Spec.VolumeMode)
	}
	return out
}

func mapCSIDriver(driver storagev1.CSIDriver) inventory.CSIDriver {
	out := inventory.CSIDriver{
		ObjectRef:      objectRef("storage.k8s.io/v1", "CSIDriver", driver.Namespace, driver.Name, driver.UID),
		AttachRequired: driver.Spec.AttachRequired,
		PodInfoOnMount: driver.Spec.PodInfoOnMount,
	}
	for _, mode := range driver.Spec.VolumeLifecycleModes {
		out.VolumeLifecycleModes = append(out.VolumeLifecycleModes, string(mode))
	}
	return out
}

func mapCSINode(node storagev1.CSINode) inventory.CSINode {
	out := inventory.CSINode{ObjectRef: objectRef("storage.k8s.io/v1", "CSINode", node.Namespace, node.Name, node.UID)}
	for _, driver := range node.Spec.Drivers {
		out.Drivers = append(out.Drivers, inventory.CSINodeDriver{Name: driver.Name, NodeID: driver.NodeID, TopologyKeys: driver.TopologyKeys})
	}
	return out
}

func mapVolumeAttachment(attachment storagev1.VolumeAttachment) inventory.VolumeAttachment {
	out := inventory.VolumeAttachment{
		ObjectRef: objectRef("storage.k8s.io/v1", "VolumeAttachment", attachment.Namespace, attachment.Name, attachment.UID),
		Attacher:  attachment.Spec.Attacher,
		NodeName:  attachment.Spec.NodeName,
		Attached:  attachment.Status.Attached,
	}
	if attachment.Spec.Source.PersistentVolumeName != nil {
		out.PVName = *attachment.Spec.Source.PersistentVolumeName
	}
	if attachment.Status.AttachError != nil {
		out.AttachError = attachment.Status.AttachError.Message
	}
	return out
}

func podSpecDetails(namespace string, spec corev1.PodSpec) ([]inventory.Container, []inventory.Container, []inventory.Volume, []inventory.ObjectRef, []inventory.ObjectRef, []inventory.ObjectRef) {
	var containers []inventory.Container
	var initContainers []inventory.Container
	var volumes []inventory.Volume
	var configRefs []inventory.ObjectRef
	var secretRefs []inventory.ObjectRef
	var pullSecrets []inventory.ObjectRef
	for _, container := range spec.Containers {
		mapped := mapContainer(namespace, container)
		containers = append(containers, mapped)
		configRefs = append(configRefs, mapped.EnvConfigRefs...)
		secretRefs = append(secretRefs, mapped.EnvSecretRefs...)
	}
	for _, container := range spec.InitContainers {
		mapped := mapContainer(namespace, container)
		initContainers = append(initContainers, mapped)
		configRefs = append(configRefs, mapped.EnvConfigRefs...)
		secretRefs = append(secretRefs, mapped.EnvSecretRefs...)
	}
	for _, volume := range spec.Volumes {
		mapped := mapVolume(namespace, volume)
		volumes = append(volumes, mapped)
		if mapped.ConfigMap != "" {
			configRefs = append(configRefs, objectRef("v1", "ConfigMap", namespace, mapped.ConfigMap, ""))
		}
		if mapped.Secret != "" {
			secretRefs = append(secretRefs, objectRef("v1", "Secret", namespace, mapped.Secret, ""))
		}
		for _, ref := range mapped.ProjectedRefs {
			if ref.Kind == "ConfigMap" {
				configRefs = append(configRefs, ref)
			}
			if ref.Kind == "Secret" {
				secretRefs = append(secretRefs, ref)
			}
		}
	}
	for _, ref := range spec.ImagePullSecrets {
		pullSecrets = append(pullSecrets, objectRef("v1", "Secret", namespace, ref.Name, ""))
	}
	return containers, initContainers, volumes, dedupeRefs(configRefs), dedupeRefs(secretRefs), dedupeRefs(pullSecrets)
}

func mapContainer(namespace string, container corev1.Container) inventory.Container {
	out := inventory.Container{
		Name:      container.Name,
		Image:     container.Image,
		Resources: containerResources(container.Resources),
	}
	for _, envFrom := range container.EnvFrom {
		if envFrom.ConfigMapRef != nil {
			out.EnvConfigRefs = append(out.EnvConfigRefs, objectRef("v1", "ConfigMap", namespace, envFrom.ConfigMapRef.Name, ""))
		}
		if envFrom.SecretRef != nil {
			out.EnvSecretRefs = append(out.EnvSecretRefs, objectRef("v1", "Secret", namespace, envFrom.SecretRef.Name, ""))
		}
	}
	for _, env := range container.Env {
		if env.ValueFrom == nil {
			continue
		}
		if env.ValueFrom.ConfigMapKeyRef != nil {
			out.EnvConfigRefs = append(out.EnvConfigRefs, objectRef("v1", "ConfigMap", namespace, env.ValueFrom.ConfigMapKeyRef.Name, ""))
		}
		if env.ValueFrom.SecretKeyRef != nil {
			out.EnvSecretRefs = append(out.EnvSecretRefs, objectRef("v1", "Secret", namespace, env.ValueFrom.SecretKeyRef.Name, ""))
		}
	}
	for _, mount := range container.VolumeMounts {
		out.VolumeMounts = append(out.VolumeMounts, mount.Name)
	}
	out.EnvConfigRefs = dedupeRefs(out.EnvConfigRefs)
	out.EnvSecretRefs = dedupeRefs(out.EnvSecretRefs)
	sort.Strings(out.VolumeMounts)
	return out
}

func mapVolume(namespace string, volume corev1.Volume) inventory.Volume {
	out := inventory.Volume{Name: volume.Name, Type: "unknown"}
	switch {
	case volume.PersistentVolumeClaim != nil:
		out.Type = "persistentVolumeClaim"
		out.PersistentVolumeClaim = volume.PersistentVolumeClaim.ClaimName
	case volume.ConfigMap != nil:
		out.Type = "configMap"
		out.ConfigMap = volume.ConfigMap.Name
	case volume.Secret != nil:
		out.Type = "secret"
		out.Secret = volume.Secret.SecretName
	case volume.CSI != nil:
		out.Type = "csi"
		out.CSI = volume.CSI.Driver
	case volume.Projected != nil:
		out.Type = "projected"
		for _, source := range volume.Projected.Sources {
			if source.ConfigMap != nil {
				out.ProjectedRefs = append(out.ProjectedRefs, objectRef("v1", "ConfigMap", namespace, source.ConfigMap.Name, ""))
			}
			if source.Secret != nil {
				out.ProjectedRefs = append(out.ProjectedRefs, objectRef("v1", "Secret", namespace, source.Secret.Name, ""))
			}
			if source.ServiceAccountToken != nil {
				out.ProjectedRefs = append(out.ProjectedRefs, objectRef("v1", "ServiceAccountToken", namespace, source.ServiceAccountToken.Path, ""))
			}
		}
	default:
		out.Type = volumeSourceType(volume)
	}
	return out
}

func containersWithStatus(containers []inventory.Container, statuses []corev1.ContainerStatus) []inventory.Container {
	byName := make(map[string]corev1.ContainerStatus, len(statuses))
	for _, status := range statuses {
		byName[status.Name] = status
	}
	for i := range containers {
		status, ok := byName[containers[i].Name]
		if !ok {
			continue
		}
		containers[i].ImageID = status.ImageID
		containers[i].ContainerID = status.ContainerID
		containers[i].State = containerState(status.State)
		containers[i].StartedAt = containerStartedAt(status.State)
		containers[i].Ready = &status.Ready
		restarts := int32(status.RestartCount)
		containers[i].RestartCount = &restarts
	}
	return containers
}

func containersFromStatuses(_ string, statuses []corev1.ContainerStatus) []inventory.Container {
	containers := make([]inventory.Container, 0, len(statuses))
	for _, status := range statuses {
		restarts := int32(status.RestartCount)
		containers = append(containers, inventory.Container{
			Name:         status.Name,
			Image:        status.Image,
			ImageID:      status.ImageID,
			ContainerID:  status.ContainerID,
			State:        containerState(status.State),
			StartedAt:    containerStartedAt(status.State),
			Ready:        &status.Ready,
			RestartCount: &restarts,
		})
	}
	return containers
}

func containerState(state corev1.ContainerState) string {
	switch {
	case state.Running != nil:
		return "running"
	case state.Waiting != nil:
		return "waiting"
	case state.Terminated != nil:
		return "terminated"
	default:
		return ""
	}
}

func containerStartedAt(state corev1.ContainerState) *time.Time {
	if state.Running == nil {
		return nil
	}
	startedAt := state.Running.StartedAt.Time
	return &startedAt
}

func runningContainers(kubernetes inventory.Kubernetes) []inventory.RunningContainer {
	workloads := workloadByOwnerKey(kubernetes.Workloads)
	out := make([]inventory.RunningContainer, 0)
	for _, pod := range kubernetes.Pods {
		podOwner := controllerOwner(pod.OwnerReferences, pod.Namespace)
		workload := resolveWorkload(podOwner, workloads, pod.Namespace)
		out = append(out, runningContainersFromPod(pod, "app", pod.Containers, podOwner, workload)...)
		out = append(out, runningContainersFromPod(pod, "init", pod.InitContainers, podOwner, workload)...)
		out = append(out, runningContainersFromPod(pod, "ephemeral", pod.EphemeralContainers, podOwner, workload)...)
	}
	sort.Slice(out, func(i, j int) bool {
		left := strings.Join([]string{out[i].Namespace, out[i].Pod, out[i].ContainerType, out[i].Container}, "\x00")
		right := strings.Join([]string{out[j].Namespace, out[j].Pod, out[j].ContainerType, out[j].Container}, "\x00")
		return left < right
	})
	return out
}

func runningImages(containers []inventory.RunningContainer) []inventory.RunningImage {
	type aggregate struct {
		item       inventory.RunningImage
		pods       map[string]struct{}
		imageIDs   map[string]struct{}
		runtimes   map[string]struct{}
		namespaces map[string]struct{}
		workloads  map[string]inventory.ObjectRef
	}
	byImage := map[string]*aggregate{}
	for _, container := range containers {
		image := container.Image
		if image == "" {
			continue
		}
		agg, ok := byImage[image]
		if !ok {
			agg = &aggregate{
				item:       inventory.RunningImage{Image: image},
				pods:       map[string]struct{}{},
				imageIDs:   map[string]struct{}{},
				runtimes:   map[string]struct{}{},
				namespaces: map[string]struct{}{},
				workloads:  map[string]inventory.ObjectRef{},
			}
			byImage[image] = agg
		}
		agg.item.ContainerCount++
		agg.pods[namespacedName(container.Namespace, container.Pod)] = struct{}{}
		if container.ImageID != "" {
			agg.imageIDs[container.ImageID] = struct{}{}
		}
		if container.Runtime != "" {
			agg.runtimes[container.Runtime] = struct{}{}
		}
		if container.Namespace != "" {
			agg.namespaces[container.Namespace] = struct{}{}
		}
		if container.Workload.Name != "" {
			agg.workloads[refKey(container.Workload)] = container.Workload
		}
	}

	out := make([]inventory.RunningImage, 0, len(byImage))
	for _, agg := range byImage {
		agg.item.PodCount = len(agg.pods)
		agg.item.ImageIDs = sortedSet(agg.imageIDs)
		agg.item.Runtimes = sortedSet(agg.runtimes)
		agg.item.Namespaces = sortedSet(agg.namespaces)
		agg.item.Workloads = sortedObjectRefs(agg.workloads)
		out = append(out, agg.item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PodCount != out[j].PodCount {
			return out[i].PodCount > out[j].PodCount
		}
		return out[i].Image < out[j].Image
	})
	return out
}

func runningContainersFromPod(pod inventory.Pod, containerType string, containers []inventory.Container, podOwner, workload inventory.ObjectRef) []inventory.RunningContainer {
	out := make([]inventory.RunningContainer, 0)
	for _, container := range containers {
		if container.State != "running" {
			continue
		}
		image := container.Image
		if image == "" {
			image = container.ImageID
		}
		out = append(out, inventory.RunningContainer{
			Namespace:        pod.Namespace,
			Pod:              pod.Name,
			NodeName:         pod.NodeName,
			Container:        container.Name,
			ContainerType:    containerType,
			Image:            image,
			ImageID:          container.ImageID,
			ContainerID:      container.ContainerID,
			Runtime:          containerRuntime(container.ContainerID),
			StartedAt:        container.StartedAt,
			Ready:            container.Ready,
			RestartCount:     container.RestartCount,
			PodOwner:         podOwner,
			Workload:         workload,
			RuntimeClassName: pod.RuntimeClassName,
			ServiceAccount:   pod.ServiceAccountName,
		})
	}
	return out
}

func workloadByOwnerKey(workloads []inventory.Workload) map[string]inventory.Workload {
	out := make(map[string]inventory.Workload, len(workloads))
	for _, workload := range workloads {
		out[ownerKey(workload.ObjectRef.Namespace, workload.ObjectRef)] = workload
	}
	return out
}

func resolveWorkload(owner inventory.ObjectRef, workloads map[string]inventory.Workload, namespace string) inventory.ObjectRef {
	if owner.Name == "" {
		return inventory.ObjectRef{}
	}
	current := owner
	if current.Namespace == "" {
		current.Namespace = namespace
	}
	for i := 0; i < 8; i++ {
		workload, ok := workloads[ownerKey(namespace, current)]
		if !ok {
			return current
		}
		if len(workload.OwnerReferences) == 0 {
			return workload.ObjectRef
		}
		next := controllerOwner(workload.OwnerReferences, workload.Namespace)
		if next.Name == "" {
			return workload.ObjectRef
		}
		current = next
		if current.Namespace == "" {
			current.Namespace = workload.Namespace
		}
	}
	return current
}

func controllerOwner(refs []inventory.ObjectRef, namespace string) inventory.ObjectRef {
	if len(refs) == 0 {
		return inventory.ObjectRef{}
	}
	ref := refs[0]
	if ref.Namespace == "" {
		ref.Namespace = namespace
	}
	return ref
}

func ownerKey(defaultNamespace string, ref inventory.ObjectRef) string {
	namespace := ref.Namespace
	if namespace == "" {
		namespace = defaultNamespace
	}
	return strings.Join([]string{strings.ToLower(ref.Kind), namespace, ref.Name}, "\x00")
}

func namespacedName(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}

func sortedSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedObjectRefs(values map[string]inventory.ObjectRef) []inventory.ObjectRef {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]inventory.ObjectRef, 0, len(keys))
	for _, key := range keys {
		out = append(out, values[key])
	}
	return out
}

func refKey(ref inventory.ObjectRef) string {
	return strings.Join([]string{ref.APIVersion, ref.Kind, ref.Namespace, ref.Name}, "\x00")
}

func containerRuntime(containerID string) string {
	if containerID == "" {
		return ""
	}
	if idx := strings.Index(containerID, "://"); idx > 0 {
		return containerID[:idx]
	}
	return ""
}

func containerResources(requirements corev1.ResourceRequirements) map[string]string {
	resources := make(map[string]string)
	for name, quantity := range requirements.Requests {
		resources["requests."+string(name)] = quantity.String()
	}
	for name, quantity := range requirements.Limits {
		resources["limits."+string(name)] = quantity.String()
	}
	if len(resources) == 0 {
		return nil
	}
	return resources
}

func resourceList(resources corev1.ResourceList) map[string]string {
	out := make(map[string]string, len(resources))
	for name, quantity := range resources {
		out[string(name)] = quantity.String()
	}
	return out
}

func metaRefs(apiVersion, kind string, items []metav1.ObjectMeta) []inventory.ObjectRef {
	refs := make([]inventory.ObjectRef, 0, len(items))
	for _, item := range items {
		refs = append(refs, objectRef(apiVersion, kind, item.Namespace, item.Name, item.UID))
	}
	return refs
}

func roleRefs(items []rbacv1.Role) []inventory.ObjectRef {
	metas := make([]metav1.ObjectMeta, 0, len(items))
	for _, item := range items {
		metas = append(metas, item.ObjectMeta)
	}
	return metaRefs("rbac.authorization.k8s.io/v1", "Role", metas)
}

func roleBindingRefs(items []rbacv1.RoleBinding) []inventory.ObjectRef {
	metas := make([]metav1.ObjectMeta, 0, len(items))
	for _, item := range items {
		metas = append(metas, item.ObjectMeta)
	}
	return metaRefs("rbac.authorization.k8s.io/v1", "RoleBinding", metas)
}

func clusterRoleRefs(items []rbacv1.ClusterRole) []inventory.ObjectRef {
	metas := make([]metav1.ObjectMeta, 0, len(items))
	for _, item := range items {
		metas = append(metas, item.ObjectMeta)
	}
	return metaRefs("rbac.authorization.k8s.io/v1", "ClusterRole", metas)
}

func clusterRoleBindingRefs(items []rbacv1.ClusterRoleBinding) []inventory.ObjectRef {
	metas := make([]metav1.ObjectMeta, 0, len(items))
	for _, item := range items {
		metas = append(metas, item.ObjectMeta)
	}
	return metaRefs("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", metas)
}

func hpaRefs(items []autoscalingv2.HorizontalPodAutoscaler) []inventory.ObjectRef {
	metas := make([]metav1.ObjectMeta, 0, len(items))
	for _, item := range items {
		metas = append(metas, item.ObjectMeta)
	}
	return metaRefs("autoscaling/v2", "HorizontalPodAutoscaler", metas)
}

func pdbRefs(items []policyv1.PodDisruptionBudget) []inventory.ObjectRef {
	metas := make([]metav1.ObjectMeta, 0, len(items))
	for _, item := range items {
		metas = append(metas, item.ObjectMeta)
	}
	return metaRefs("policy/v1", "PodDisruptionBudget", metas)
}

func networkPolicyRefs(items []networkingv1.NetworkPolicy) []inventory.ObjectRef {
	metas := make([]metav1.ObjectMeta, 0, len(items))
	for _, item := range items {
		metas = append(metas, item.ObjectMeta)
	}
	return metaRefs("networking.k8s.io/v1", "NetworkPolicy", metas)
}

func resourceQuotaRefs(items []corev1.ResourceQuota) []inventory.ObjectRef {
	metas := make([]metav1.ObjectMeta, 0, len(items))
	for _, item := range items {
		metas = append(metas, item.ObjectMeta)
	}
	return metaRefs("v1", "ResourceQuota", metas)
}

func limitRangeRefs(items []corev1.LimitRange) []inventory.ObjectRef {
	metas := make([]metav1.ObjectMeta, 0, len(items))
	for _, item := range items {
		metas = append(metas, item.ObjectMeta)
	}
	return metaRefs("v1", "LimitRange", metas)
}

func objectRef(apiVersion, kind, namespace, name string, uid any) inventory.ObjectRef {
	ref := inventory.ObjectRef{APIVersion: apiVersion, Kind: kind, Namespace: namespace, Name: name}
	switch typed := uid.(type) {
	case types.UID:
		ref.UID = string(typed)
	case string:
		ref.UID = typed
	}
	return ref
}

func ownerRefs(owners []metav1.OwnerReference) []inventory.ObjectRef {
	refs := make([]inventory.ObjectRef, 0, len(owners))
	for _, owner := range owners {
		refs = append(refs, objectRef(owner.APIVersion, owner.Kind, "", owner.Name, owner.UID))
	}
	return refs
}

func dedupeRefs(refs []inventory.ObjectRef) []inventory.ObjectRef {
	seen := make(map[string]struct{}, len(refs))
	out := make([]inventory.ObjectRef, 0, len(refs))
	for _, ref := range refs {
		key := strings.Join([]string{ref.APIVersion, ref.Kind, ref.Namespace, ref.Name}, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func selectorMap(selector *metav1.LabelSelector) map[string]string {
	if selector == nil {
		return nil
	}
	return selector.MatchLabels
}

func accessModes(modes []corev1.PersistentVolumeAccessMode) []string {
	out := make([]string, 0, len(modes))
	for _, mode := range modes {
		out = append(out, string(mode))
	}
	return out
}

func intOrString(value intstr.IntOrString) string {
	if value.Type == intstr.Int {
		return fmt.Sprintf("%d", value.IntVal)
	}
	return value.StrVal
}

func serviceBackendPort(value networkingv1.ServiceBackendPort) string {
	if value.Name != "" {
		return value.Name
	}
	return fmt.Sprintf("%d", value.Number)
}

func sortedMapKeys(maps ...any) []string {
	seen := map[string]struct{}{}
	for _, item := range maps {
		switch typed := item.(type) {
		case map[string]string:
			for key := range typed {
				seen[key] = struct{}{}
			}
		case map[string][]byte:
			for key := range typed {
				seen[key] = struct{}{}
			}
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func protocolPtr(value *corev1.Protocol) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func pathType(value *networkingv1.PathType) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func volumeSourceType(volume corev1.Volume) string {
	switch {
	case volume.EmptyDir != nil:
		return "emptyDir"
	case volume.HostPath != nil:
		return "hostPath"
	case volume.DownwardAPI != nil:
		return "downwardAPI"
	case volume.Ephemeral != nil:
		return "ephemeral"
	case volume.NFS != nil:
		return "nfs"
	case volume.AWSElasticBlockStore != nil:
		return "awsElasticBlockStore"
	case volume.AzureDisk != nil:
		return "azureDisk"
	case volume.GCEPersistentDisk != nil:
		return "gcePersistentDisk"
	default:
		return "unknown"
	}
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

func unavailable(area, resource string, err error, at time.Time) inventory.CoverageItem {
	return inventory.CoverageItem{Area: area, Resource: resource, Status: "unavailable", Reason: err.Error(), CollectedAt: at}
}
