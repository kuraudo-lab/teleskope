package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakediscovery "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestWatchRuntimePublishesInitialListAndPodEvents(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "app"}},
		watchPod("web", "Pending"),
	)
	runtime := NewWatchRuntimeWithClient(client, WatchOptions{Debounce: 10 * time.Millisecond}, "test", "https://cluster")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan WatchUpdate, 8)
	done := make(chan error, 1)
	go func() { done <- runtime.Run(ctx, func(update WatchUpdate) { updates <- update }) }()

	initial := waitWatchUpdate(t, updates, func(update WatchUpdate) bool {
		return update.Snapshot != nil && len(update.Snapshot.Kubernetes.Pods) == 1
	})
	if initial.Snapshot.Source.Mode != "live/watch" || initial.Health.State != "ready" {
		t.Fatalf("initial update = %+v", initial)
	}

	if _, err := client.CoreV1().Pods("app").Create(ctx, watchPod("api", "Running"), metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	added := waitWatchUpdate(t, updates, func(update WatchUpdate) bool {
		return update.Snapshot != nil && len(update.Snapshot.Kubernetes.Pods) == 2
	})
	if len(added.Snapshot.Kubernetes.RunningContainers) == 0 || added.Snapshot.Kubernetes.RunningContainers[0].Pod == "" {
		t.Fatalf("running containers were not rebuilt: %+v", added.Snapshot.Kubernetes.RunningContainers)
	}

	updatedPod := watchPod("api", "Succeeded")
	if _, err := client.CoreV1().Pods("app").Update(ctx, updatedPod, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	_ = waitWatchUpdate(t, updates, func(update WatchUpdate) bool {
		if update.Snapshot == nil || len(update.Snapshot.Kubernetes.Pods) != 2 {
			return false
		}
		for _, pod := range update.Snapshot.Kubernetes.Pods {
			if pod.Name == "api" && pod.Phase == "Succeeded" {
				return true
			}
		}
		return false
	})

	if err := client.CoreV1().Pods("app").Delete(ctx, "web", metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	_ = waitWatchUpdate(t, updates, func(update WatchUpdate) bool {
		if update.Snapshot == nil || len(update.Snapshot.Kubernetes.Pods) != 1 {
			return false
		}
		return update.Snapshot.Kubernetes.Pods[0].Name == "api"
	})

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("watch runtime did not stop")
	}
}

func TestWatchRuntimePublishesCRDAndGatewayEvents(t *testing.T) {
	client := fake.NewSimpleClientset()
	setWatchDiscovery(client)
	apiResources, _, err := (&kubernetesWatchRuntime{client: client}).discoverAPIResources(context.Background())
	if err != nil || len(watchedGatewayResources(apiResources)) == 0 {
		t.Fatalf("gateway discovery resources=%+v err=%v", apiResources, err)
	}
	crdGVR := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	widgetGVR := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}
	gatewayGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), watchListKinds(),
		watchCRD("widgets.example.com", "example.com", "Widget", "widgets"),
		watchCustomResource(widgetGVR, "blue"),
	)
	runtime := NewWatchRuntimeWithClients(client, dynamicClient, WatchOptions{Debounce: 10 * time.Millisecond}, "test", "", "https://cluster")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan WatchUpdate, 12)
	done := make(chan error, 1)
	go func() { done <- runtime.Run(ctx, func(update WatchUpdate) { updates <- update }) }()

	initial := waitWatchUpdate(t, updates, func(update WatchUpdate) bool {
		return update.Snapshot != nil && len(update.Snapshot.Kubernetes.CustomResourceDefinitions) == 1 && len(update.Snapshot.Kubernetes.CustomResourceInstances) == 1
	})
	if got := coverageStatus(initial.Snapshot.Coverage, "CustomResourceInstances"); got != "complete" {
		t.Fatalf("initial custom resource coverage = %q", got)
	}

	if _, err := dynamicClient.Resource(crdGVR).Create(ctx, watchCRD("gadgets.example.com", "example.com", "Gadget", "gadgets"), metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	_ = waitWatchUpdate(t, updates, func(update WatchUpdate) bool {
		return update.Snapshot != nil && len(update.Snapshot.Kubernetes.CustomResourceDefinitions) == 2 && coverageStatus(update.Snapshot.Coverage, "CustomResourceInstances") == "complete"
	})

	if _, err := dynamicClient.Resource(gatewayGVR).Namespace("app").Create(ctx, watchGateway("public"), metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	gatewayList, err := dynamicClient.Resource(gatewayGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil || len(gatewayList.Items) != 1 {
		t.Fatalf("gateway list after create len=%d err=%v", len(gatewayList.Items), err)
	}
	_ = waitWatchUpdate(t, updates, func(update WatchUpdate) bool {
		return update.Snapshot != nil && len(update.Snapshot.Kubernetes.Gateways) == 1 && update.Snapshot.Kubernetes.Gateways[0].Name == "public"
	})

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("watch runtime did not stop")
	}
}

func TestWatchRuntimeKeepsDynamicPermissionGapsInCoverage(t *testing.T) {
	client := fake.NewSimpleClientset()
	setWatchDiscovery(client)
	crdGVR := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	widgetGVR := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), watchListKinds(),
		watchCRD("widgets.example.com", "example.com", "Widget", "widgets"),
	)
	dynamicClient.PrependReactor("list", "widgets", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(widgetGVR.GroupResource(), "", fmt.Errorf("denied"))
	})
	runtime := NewWatchRuntimeWithClients(client, dynamicClient, WatchOptions{Debounce: 10 * time.Millisecond}, "test", "", "https://cluster")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan WatchUpdate, 8)
	done := make(chan error, 1)
	go func() { done <- runtime.Run(ctx, func(update WatchUpdate) { updates <- update }) }()

	initial := waitWatchUpdate(t, updates, func(update WatchUpdate) bool {
		return update.Snapshot != nil && len(update.Snapshot.Kubernetes.CustomResourceDefinitions) == 1
	})
	if got := coverageStatus(initial.Snapshot.Coverage, "CustomResourceInstances"); got != "partial" {
		t.Fatalf("custom resource coverage = %q, want partial: %+v", got, initial.Snapshot.Coverage)
	}
	if len(initial.Snapshot.Kubernetes.CustomResourceCounts) != 1 {
		t.Fatalf("custom resource counts = %+v, want type metadata retained", initial.Snapshot.Kubernetes.CustomResourceCounts)
	}

	if err := dynamicClient.Resource(crdGVR).Delete(ctx, "widgets.example.com", metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	_ = waitWatchUpdate(t, updates, func(update WatchUpdate) bool {
		return update.Snapshot != nil && len(update.Snapshot.Kubernetes.CustomResourceDefinitions) == 0 && coverageStatus(update.Snapshot.Coverage, "CustomResourceInstances") == "complete"
	})

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("watch runtime did not stop")
	}
}

func TestWatchRuntimePublishesStaleHealthOnWatchError(t *testing.T) {
	client := fake.NewSimpleClientset(watchPod("web", "Running"))
	client.PrependReactor("list", "pods", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("resource version expired")
	})
	runtime := NewWatchRuntimeWithClient(client, WatchOptions{Debounce: 10 * time.Millisecond}, "test", "https://cluster")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan WatchUpdate, 8)
	done := make(chan error, 1)
	go func() { done <- runtime.Run(ctx, func(update WatchUpdate) { updates <- update }) }()

	stale := waitWatchUpdate(t, updates, func(update WatchUpdate) bool {
		return update.Snapshot == nil && update.Health.State == "stale" && update.Health.Reconnects > 0
	})
	if stale.Health.Err == nil {
		t.Fatalf("stale update missing error: %+v", stale)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("watch runtime did not stop")
	}
}

func watchPod(name, phase string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "app", UID: types.UID("pod-" + name)},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main", Image: "example/" + name + ":v1"}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodPhase(phase)},
	}
	if phase == "Running" {
		started := metav1.NewTime(time.Now().UTC())
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
			Name:        "main",
			Image:       "example/" + name + ":v1",
			ImageID:     "example/" + name + "@sha256:test",
			ContainerID: "containerd://test-" + name,
			Ready:       true,
			State:       corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: started}},
		}}
	}
	return pod
}

func setWatchDiscovery(client *fake.Clientset) {
	discovery := client.Discovery().(*fakediscovery.FakeDiscovery)
	discovery.Resources = []*metav1.APIResourceList{
		{GroupVersion: "example.com/v1", APIResources: []metav1.APIResource{{Name: "widgets", Namespaced: true, Kind: "Widget", Verbs: []string{"list", "watch"}}}},
		{GroupVersion: "gateway.networking.k8s.io/v1", APIResources: []metav1.APIResource{
			{Name: "gatewayclasses", Kind: "GatewayClass", Verbs: []string{"list", "watch"}},
			{Name: "gateways", Namespaced: true, Kind: "Gateway", Verbs: []string{"list", "watch"}},
			{Name: "httproutes", Namespaced: true, Kind: "HTTPRoute", Verbs: []string{"list", "watch"}},
		}},
	}
}

func watchListKinds() map[schema.GroupVersionResource]string {
	out := map[schema.GroupVersionResource]string{
		{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}: "CustomResourceDefinitionList",
		{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"}:             "APIServiceList",
		{Group: "example.com", Version: "v1", Resource: "widgets"}:                            "WidgetList",
		{Group: "example.com", Version: "v1", Resource: "gadgets"}:                            "GadgetList",
	}
	for _, version := range []string{"v1", "v1beta1"} {
		out[schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: version, Resource: "gatewayclasses"}] = "GatewayClassList"
		out[schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: version, Resource: "gateways"}] = "GatewayList"
		out[schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: version, Resource: "httproutes"}] = "HTTPRouteList"
		out[schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: version, Resource: "grpcroutes"}] = "GRPCRouteList"
	}
	for _, version := range []string{"v1", "v1alpha3", "v1alpha2"} {
		out[schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: version, Resource: "tlsroutes"}] = "TLSRouteList"
	}
	for _, version := range []string{"v1", "v1alpha2"} {
		out[schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: version, Resource: "tcproutes"}] = "TCPRouteList"
		out[schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: version, Resource: "udproutes"}] = "UDPRouteList"
		out[schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: version, Resource: "referencegrants"}] = "ReferenceGrantList"
	}
	for _, version := range []string{"v1", "v1alpha3"} {
		out[schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: version, Resource: "backendtlspolicies"}] = "BackendTLSPolicyList"
	}
	out[schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1alpha1", Resource: "backendtrafficpolicies"}] = "BackendTrafficPolicyList"
	out[schema.GroupVersionResource{Group: "gateway.networking.x-k8s.io", Version: "v1alpha1", Resource: "xbackendtrafficpolicies"}] = "XBackendTrafficPolicyList"
	return out
}

func watchCRD(name, group, kind, plural string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata":   map[string]any{"name": name, "uid": name},
		"spec": map[string]any{
			"group": group,
			"scope": "Namespaced",
			"names": map[string]any{"kind": kind, "plural": plural},
			"versions": []any{map[string]any{
				"name":    "v1",
				"served":  true,
				"storage": true,
			}},
		},
	}}
}

func watchCustomResource(gvr schema.GroupVersionResource, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": gvr.Group + "/" + gvr.Version,
		"kind":       "Widget",
		"metadata":   map[string]any{"namespace": "app", "name": name, "uid": "custom-" + name},
	}}
}

func watchGateway(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "Gateway",
		"metadata":   map[string]any{"namespace": "app", "name": name, "uid": "gateway-" + name},
		"spec":       map[string]any{"gatewayClassName": "alb"},
	}}
}

func coverageStatus(items []inventory.CoverageItem, resource string) string {
	for _, item := range items {
		if item.Resource == resource {
			return item.Status
		}
	}
	return ""
}

func waitWatchUpdate(t *testing.T, updates <-chan WatchUpdate, accept func(WatchUpdate) bool) WatchUpdate {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case update := <-updates:
			if accept(update) {
				return update
			}
		case <-deadline:
			t.Fatal("timed out waiting for watch update")
		}
	}
}

var _ runtime.Object = (*corev1.Pod)(nil)
