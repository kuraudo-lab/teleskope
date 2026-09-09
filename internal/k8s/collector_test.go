package k8s

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestRunningContainersIncludeOnlyRunningStatuses(t *testing.T) {
	started := metav1.NewTime(time.Date(2026, 9, 7, 1, 2, 3, 0, time.UTC))
	pod := mapPod(corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "app",
			Name:      "web-abc",
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "web-rs"},
			},
		},
		Spec: corev1.PodSpec{
			NodeName:           "node-a",
			ServiceAccountName: "web",
			Containers: []corev1.Container{
				{Name: "web", Image: "repo/web:v1"},
				{Name: "sidecar", Image: "repo/sidecar:v1"},
			},
			InitContainers: []corev1.Container{
				{Name: "migrate", Image: "repo/migrate:v1"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:        "web",
					Image:       "repo/web:v1",
					ImageID:     "docker-pullable://repo/web@sha256:aaaaaaaaaaaaaaaa",
					ContainerID: "containerd://web-container",
					Ready:       true,
					State:       corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: started}},
				},
				{
					Name:  "sidecar",
					Image: "repo/sidecar:v1",
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
				},
			},
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "migrate",
					Image: "repo/migrate:v1",
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Completed"}},
				},
			},
			EphemeralContainerStatuses: []corev1.ContainerStatus{
				{
					Name:        "debugger",
					Image:       "repo/debug:v1",
					ImageID:     "docker-pullable://repo/debug@sha256:bbbbbbbbbbbbbbbb",
					ContainerID: "containerd://debug-container",
					State:       corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: started}},
				},
			},
		},
	})

	kubernetes := inventory.Kubernetes{
		Workloads: []inventory.Workload{
			{
				ObjectRef:       inventory.ObjectRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"},
				OwnerReferences: nil,
			},
			{
				ObjectRef: inventory.ObjectRef{APIVersion: "apps/v1", Kind: "ReplicaSet", Namespace: "app", Name: "web-rs"},
				OwnerReferences: []inventory.ObjectRef{
					{APIVersion: "apps/v1", Kind: "Deployment", Name: "web"},
				},
			},
		},
		Pods: []inventory.Pod{pod},
	}

	got := runningContainers(kubernetes)
	if len(got) != 2 {
		t.Fatalf("runningContainers returned %d items, want 2: %#v", len(got), got)
	}
	byName := map[string]inventory.RunningContainer{}
	for _, container := range got {
		byName[container.Container] = container
	}
	if debugger := byName["debugger"]; debugger.ContainerType != "ephemeral" || debugger.Image != "repo/debug:v1" {
		t.Fatalf("debugger container = %#v, want ephemeral debugger", debugger)
	}
	if web := byName["web"]; web.ContainerType != "app" || web.Image != "repo/web:v1" || web.Runtime != "containerd" {
		t.Fatalf("web container = %#v, want running app container", web)
	}
	for _, container := range got {
		if container.Workload.Kind != "Deployment" || container.Workload.Namespace != "app" || container.Workload.Name != "web" {
			t.Fatalf("container workload = %#v, want app/web deployment", container.Workload)
		}
		if container.PodOwner.Kind != "ReplicaSet" || container.PodOwner.Namespace != "app" || container.PodOwner.Name != "web-rs" {
			t.Fatalf("container pod owner = %#v, want app/web-rs replicaset", container.PodOwner)
		}
		if container.ServiceAccount != "web" || container.NodeName != "node-a" {
			t.Fatalf("container placement = %#v, want serviceAccount web on node-a", container)
		}
	}
}

func TestRunningImagesDeduplicateByImageAndCountPods(t *testing.T) {
	containers := []inventory.RunningContainer{
		{
			Namespace: "app",
			Pod:       "web-a",
			Container: "web",
			Image:     "repo/web:v1",
			ImageID:   "docker-pullable://repo/web@sha256:aaaaaaaaaaaaaaaa",
			Runtime:   "containerd",
			Workload:  inventory.ObjectRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"},
		},
		{
			Namespace: "app",
			Pod:       "web-a",
			Container: "sidecar",
			Image:     "repo/web:v1",
			ImageID:   "docker-pullable://repo/web@sha256:aaaaaaaaaaaaaaaa",
			Runtime:   "containerd",
			Workload:  inventory.ObjectRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"},
		},
		{
			Namespace: "app",
			Pod:       "web-b",
			Container: "web",
			Image:     "repo/web:v1",
			ImageID:   "docker-pullable://repo/web@sha256:bbbbbbbbbbbbbbbb",
			Runtime:   "containerd",
			Workload:  inventory.ObjectRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"},
		},
		{
			Namespace: "ops",
			Pod:       "agent-a",
			Container: "agent",
			Image:     "repo/agent:v2",
			Runtime:   "containerd",
			Workload:  inventory.ObjectRef{APIVersion: "apps/v1", Kind: "DaemonSet", Namespace: "ops", Name: "agent"},
		},
	}

	got := runningImages(containers)
	if len(got) != 2 {
		t.Fatalf("runningImages returned %d items, want 2: %#v", len(got), got)
	}
	if got[0].Image != "repo/web:v1" || got[0].PodCount != 2 || got[0].ContainerCount != 3 {
		t.Fatalf("first running image = %#v, want repo/web:v1 with 2 pods and 3 containers", got[0])
	}
	if len(got[0].ImageIDs) != 2 {
		t.Fatalf("image IDs = %#v, want two unique IDs", got[0].ImageIDs)
	}
	if len(got[0].Workloads) != 1 || got[0].Workloads[0].Name != "web" {
		t.Fatalf("workloads = %#v, want deduped deployment/web", got[0].Workloads)
	}
	if got[1].Image != "repo/agent:v2" || got[1].PodCount != 1 || got[1].ContainerCount != 1 {
		t.Fatalf("second running image = %#v, want repo/agent:v2", got[1])
	}
}

func TestCollectCustomResourceInstancesBuildsInstanceAndTypeLists(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "WidgetList"},
		&unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "example.com/v1",
				"kind":       "Widget",
				"metadata": map[string]any{
					"namespace": "app",
					"name":      "blue",
					"uid":       "uid-blue",
					"labels":    map[string]any{"app": "blue"},
				},
			},
		},
		&unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "example.com/v1",
				"kind":       "Widget",
				"metadata": map[string]any{
					"namespace": "ops",
					"name":      "green",
					"uid":       "uid-green",
				},
			},
		},
	)
	kubernetes := inventory.Kubernetes{
		CustomResourceDefinitions: []inventory.CustomResourceDefinition{
			{
				ObjectRef: inventory.ObjectRef{Name: "widgets.example.com"},
				Group:     "example.com",
				Scope:     "Namespaced",
				Kind:      "Widget",
				Plural:    "widgets",
				Versions:  []inventory.CRDVersion{{Name: "v1", Served: true, Storage: true}},
			},
		},
	}
	var coverage []inventory.CoverageItem

	collectCustomResourceInstances(context.Background(), client, &kubernetes, &coverage, time.Now().UTC())

	if len(kubernetes.CustomResourceInstances) != 2 {
		t.Fatalf("instances = %#v, want two custom resources", kubernetes.CustomResourceInstances)
	}
	if kubernetes.CustomResourceInstances[0].CRDName != "widgets.example.com" ||
		kubernetes.CustomResourceInstances[0].CRDGroup != "example.com" ||
		kubernetes.CustomResourceInstances[0].CRDKind != "Widget" {
		t.Fatalf("first instance type metadata = %#v", kubernetes.CustomResourceInstances[0])
	}
	if len(kubernetes.CustomResourceCounts) != 1 {
		t.Fatalf("counts = %#v, want one type count", kubernetes.CustomResourceCounts)
	}
	count := kubernetes.CustomResourceCounts[0]
	if count.CRDName != "widgets.example.com" || count.InstanceCount != 2 || count.NamespaceCount != 2 {
		t.Fatalf("count = %#v, want widgets.example.com with two instances in two namespaces", count)
	}
	if len(coverage) != 1 || coverage[0].Status != "complete" || coverage[0].ObjectCount != 2 {
		t.Fatalf("coverage = %#v, want complete count 2", coverage)
	}
}

func TestWorkloadFromTemplateRecordsAppContainerImages(t *testing.T) {
	workload := mapDeployment(appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "app", Name: "web"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "web", Image: "repo/web:v1"}},
				},
			},
		},
	})

	if len(workload.Containers) != 1 || workload.Containers[0].Image != "repo/web:v1" {
		t.Fatalf("workload containers = %#v, want repo/web:v1", workload.Containers)
	}
}

func TestCollectGatewaysFallsBackToV1Beta1AndMapsRoutes(t *testing.T) {
	gatewayClassV1GVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gatewayclasses"}
	gatewayV1GVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}
	httpRouteV1GVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}
	grpcRouteV1GVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "grpcroutes"}
	gatewayClassGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "gatewayclasses"}
	gatewayGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "gateways"}
	httpRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "httproutes"}
	grpcRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "grpcroutes"}
	tlsRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "tlsroutes"}
	tcpRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "tcproutes"}
	udpRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "udproutes"}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			gatewayClassV1GVR: "GatewayClassList",
			gatewayV1GVR:      "GatewayList",
			httpRouteV1GVR:    "HTTPRouteList",
			grpcRouteV1GVR:    "GRPCRouteList",
			gatewayClassGVR:   "GatewayClassList",
			gatewayGVR:        "GatewayList",
			httpRouteGVR:      "HTTPRouteList",
			grpcRouteGVR:      "GRPCRouteList",
			tlsRouteGVR:       "TLSRouteList",
			tcpRouteGVR:       "TCPRouteList",
			udpRouteGVR:       "UDPRouteList",
		},
	)
	client.Fake.PrependReactor("list", "*", func(action ktesting.Action) (bool, runtime.Object, error) {
		resource := action.GetResource()
		if resource.Group == "gateway.networking.k8s.io" && resource.Version == "v1" {
			return true, nil, apierrors.NewNotFound(resource.GroupResource(), resource.Resource)
		}
		switch resource {
		case gatewayClassGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.k8s.io/v1beta1",
				"kind":       "GatewayClass",
				"metadata":   map[string]any{"name": "alb"},
				"spec":       map[string]any{"controllerName": "gateway.k8s.aws/alb"},
			}}}}, nil
		case gatewayGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.k8s.io/v1beta1",
				"kind":       "Gateway",
				"metadata":   map[string]any{"namespace": "app", "name": "public"},
				"spec": map[string]any{
					"gatewayClassName": "alb",
					"addresses":        []any{map[string]any{"value": "configured.example.com"}},
					"listeners": []any{map[string]any{
						"name":     "https",
						"protocol": "HTTPS",
						"port":     int64(443),
						"hostname": "example.com",
						"allowedRoutes": map[string]any{
							"namespaces": map[string]any{"from": "Same"},
							"kinds":      []any{map[string]any{"group": "gateway.networking.k8s.io", "kind": "HTTPRoute"}},
						},
					}},
				},
				"status": map[string]any{"addresses": []any{map[string]any{"value": "internal-alb.example.com"}}},
			}}}}, nil
		case httpRouteGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.k8s.io/v1beta1",
				"kind":       "HTTPRoute",
				"metadata":   map[string]any{"namespace": "app", "name": "web"},
				"spec": map[string]any{
					"hostnames":  []any{"example.com"},
					"parentRefs": []any{map[string]any{"name": "public", "sectionName": "https"}},
					"rules": []any{map[string]any{
						"matches":     []any{map[string]any{"path": map[string]any{"type": "PathPrefix", "value": "/"}}},
						"backendRefs": []any{map[string]any{"name": "web", "port": int64(80)}},
					}},
				},
			}}}}, nil
		case grpcRouteGVR, tlsRouteGVR, tcpRouteGVR, udpRouteGVR:
			return true, &unstructured.UnstructuredList{}, nil
		}
		return false, nil, nil
	})
	kubernetes := inventory.Kubernetes{}
	var coverage []inventory.CoverageItem

	collectGateways(context.Background(), client, &kubernetes, &coverage, time.Now().UTC(), "")

	if len(kubernetes.GatewayClasses) != 1 || kubernetes.GatewayClasses[0].APIVersion != "gateway.networking.k8s.io/v1beta1" || kubernetes.GatewayClasses[0].ControllerName != "gateway.k8s.aws/alb" {
		t.Fatalf("gateway classes = %#v, want v1beta1 alb", kubernetes.GatewayClasses)
	}
	if len(kubernetes.Gateways) != 1 || kubernetes.Gateways[0].ClassName != "alb" || len(kubernetes.Gateways[0].Listeners) != 1 || len(kubernetes.Gateways[0].Addresses) != 2 {
		t.Fatalf("gateways = %#v, want one alb gateway with listener and spec/status addresses", kubernetes.Gateways)
	}
	listener := kubernetes.Gateways[0].Listeners[0]
	if listener.Name != "https" || listener.Protocol != "HTTPS" || listener.Port != 443 || len(listener.AllowedRoutes) != 2 {
		t.Fatalf("listener = %#v, want https listener with allowed routes", listener)
	}
	if len(kubernetes.GatewayRoutes) != 1 || kubernetes.GatewayRoutes[0].Kind != "HTTPRoute" || kubernetes.GatewayRoutes[0].APIVersion != "gateway.networking.k8s.io/v1beta1" {
		t.Fatalf("gateway routes = %#v, want one v1beta1 HTTPRoute", kubernetes.GatewayRoutes)
	}
	route := kubernetes.GatewayRoutes[0]
	if len(route.ParentRefs) != 1 || route.ParentRefs[0].Name != "public" || route.ParentRefs[0].Namespace != "app" || route.ParentRefs[0].SectionName != "https" {
		t.Fatalf("parent refs = %#v, want app public gateway listener", route.ParentRefs)
	}
	if len(route.Rules) != 1 || len(route.Rules[0].Matches) != 1 || route.Rules[0].Matches[0] != "PathPrefix:/" {
		t.Fatalf("rules = %#v, want path prefix match", route.Rules)
	}
	if len(route.Rules[0].BackendRefs) != 1 || route.Rules[0].BackendRefs[0].Kind != "Service" || route.Rules[0].BackendRefs[0].Name != "web" {
		t.Fatalf("backend refs = %#v, want service app/web", route.Rules[0].BackendRefs)
	}
}

func TestCollectGatewaysUsesDiscoveredV1GatewayAPI(t *testing.T) {
	gatewayClassGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gatewayclasses"}
	gatewayGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}
	httpRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			gatewayClassGVR: "GatewayClassList",
			gatewayGVR:      "GatewayList",
			httpRouteGVR:    "HTTPRouteList",
		},
	)
	client.Fake.PrependReactor("list", "*", func(action ktesting.Action) (bool, runtime.Object, error) {
		switch action.GetResource() {
		case gatewayClassGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.k8s.io/v1",
				"kind":       "GatewayClass",
				"metadata":   map[string]any{"name": "alb"},
				"spec":       map[string]any{"controllerName": "gateway.k8s.aws/alb"},
			}}}}, nil
		case gatewayGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.k8s.io/v1",
				"kind":       "Gateway",
				"metadata":   map[string]any{"namespace": "app", "name": "public"},
				"spec": map[string]any{
					"gatewayClassName": "alb",
					"listeners": []any{map[string]any{
						"name":     "http",
						"protocol": "HTTP",
						"port":     int64(80),
					}},
				},
			}}}}, nil
		case httpRouteGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.k8s.io/v1",
				"kind":       "HTTPRoute",
				"metadata":   map[string]any{"namespace": "app", "name": "web"},
				"spec": map[string]any{
					"parentRefs": []any{map[string]any{"name": "public"}},
					"rules": []any{map[string]any{
						"backendRefs": []any{map[string]any{"name": "web", "port": int64(80)}},
					}},
				},
			}}}}, nil
		default:
			return true, &unstructured.UnstructuredList{}, nil
		}
	})
	kubernetes := inventory.Kubernetes{APIResources: []inventory.APIResource{
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gatewayclasses", Kind: "GatewayClass", Verbs: []string{"get", "list"}},
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways", Kind: "Gateway", Namespaced: true, Verbs: []string{"get", "list"}},
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes", Kind: "HTTPRoute", Namespaced: true, Verbs: []string{"get", "list"}},
	}}
	var coverage []inventory.CoverageItem

	collectGateways(context.Background(), client, &kubernetes, &coverage, time.Now().UTC(), "")

	if len(kubernetes.GatewayClasses) != 1 || kubernetes.GatewayClasses[0].APIVersion != "gateway.networking.k8s.io/v1" {
		t.Fatalf("gateway classes = %#v, want discovered v1 GatewayClass", kubernetes.GatewayClasses)
	}
	if len(kubernetes.Gateways) != 1 || kubernetes.Gateways[0].APIVersion != "gateway.networking.k8s.io/v1" || kubernetes.Gateways[0].ClassName != "alb" {
		t.Fatalf("gateways = %#v, want discovered v1 Gateway", kubernetes.Gateways)
	}
	if len(kubernetes.GatewayRoutes) != 1 || kubernetes.GatewayRoutes[0].Kind != "HTTPRoute" || kubernetes.GatewayRoutes[0].APIVersion != "gateway.networking.k8s.io/v1" {
		t.Fatalf("gateway routes = %#v, want discovered v1 HTTPRoute", kubernetes.GatewayRoutes)
	}
}

func TestCollectGatewaysFallsBackToContextNamespaceWhenAllNamespacesDenied(t *testing.T) {
	gatewayGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}
	httpRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			gatewayGVR:   "GatewayList",
			httpRouteGVR: "HTTPRouteList",
		},
	)
	client.Fake.PrependReactor("list", "*", func(action ktesting.Action) (bool, runtime.Object, error) {
		if action.GetNamespace() == "" {
			return true, nil, apierrors.NewForbidden(action.GetResource().GroupResource(), action.GetResource().Resource, nil)
		}
		switch action.GetResource() {
		case gatewayGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.k8s.io/v1",
				"kind":       "Gateway",
				"metadata":   map[string]any{"namespace": action.GetNamespace(), "name": "public"},
				"spec":       map[string]any{"gatewayClassName": "alb"},
			}}}}, nil
		case httpRouteGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.k8s.io/v1",
				"kind":       "HTTPRoute",
				"metadata":   map[string]any{"namespace": action.GetNamespace(), "name": "web"},
				"spec":       map[string]any{"parentRefs": []any{map[string]any{"name": "public"}}},
			}}}}, nil
		default:
			return true, &unstructured.UnstructuredList{}, nil
		}
	})
	kubernetes := inventory.Kubernetes{APIResources: []inventory.APIResource{
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways", Kind: "Gateway", Namespaced: true, Verbs: []string{"list"}},
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes", Kind: "HTTPRoute", Namespaced: true, Verbs: []string{"list"}},
	}}
	var coverage []inventory.CoverageItem

	collectGateways(context.Background(), client, &kubernetes, &coverage, time.Now().UTC(), "app")

	if len(kubernetes.Gateways) != 1 || kubernetes.Gateways[0].Namespace != "app" {
		t.Fatalf("gateways = %#v, want one gateway from context namespace", kubernetes.Gateways)
	}
	if len(kubernetes.GatewayRoutes) != 1 || kubernetes.GatewayRoutes[0].Namespace != "app" {
		t.Fatalf("gateway routes = %#v, want one route from context namespace", kubernetes.GatewayRoutes)
	}
}
