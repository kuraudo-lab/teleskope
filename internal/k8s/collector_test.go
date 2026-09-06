package k8s

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
