package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
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
