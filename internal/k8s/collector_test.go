package k8s

import (
	"context"
	"testing"
	"time"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func resourceMustParse(value string) resource.Quantity {
	return resource.MustParse(value)
}

func gatewayAPIListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gatewayclasses"}:                  "GatewayClassList",
		{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "gatewayclasses"}:             "GatewayClassList",
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}:                        "GatewayList",
		{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "gateways"}:                   "GatewayList",
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}:                      "HTTPRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "httproutes"}:                 "HTTPRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "grpcroutes"}:                      "GRPCRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "grpcroutes"}:                 "GRPCRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "grpcroutes"}:                "GRPCRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "tlsroutes"}:                       "TLSRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1alpha3", Resource: "tlsroutes"}:                 "TLSRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "tlsroutes"}:                 "TLSRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "tcproutes"}:                       "TCPRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "tcproutes"}:                 "TCPRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "udproutes"}:                       "UDPRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "udproutes"}:                 "UDPRouteList",
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "referencegrants"}:                 "ReferenceGrantList",
		{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "referencegrants"}:            "ReferenceGrantList",
		{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "referencegrants"}:           "ReferenceGrantList",
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "backendtlspolicies"}:              "BackendTLSPolicyList",
		{Group: "gateway.networking.k8s.io", Version: "v1alpha3", Resource: "backendtlspolicies"}:        "BackendTLSPolicyList",
		{Group: "gateway.networking.k8s.io", Version: "v1alpha1", Resource: "backendtrafficpolicies"}:    "BackendTrafficPolicyList",
		{Group: "gateway.networking.x-k8s.io", Version: "v1alpha1", Resource: "xbackendtrafficpolicies"}: "XBackendTrafficPolicyList",
	}
}

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

func TestWorkloadMappersRecordTypeSpecificStatus(t *testing.T) {
	replicas := int32(3)
	parallelism := int32(2)
	completions := int32(5)
	suspend := true

	daemonSet := mapDaemonSet(appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ops", Name: "agent"},
		Spec: appsv1.DaemonSetSpec{
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: appsv1.RollingUpdateDaemonSetStrategyType},
			Template:       corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "agent", Image: "agent:v1"}}}},
		},
		Status: appsv1.DaemonSetStatus{DesiredNumberScheduled: 3, CurrentNumberScheduled: 2, NumberReady: 2, UpdatedNumberScheduled: 1, NumberUnavailable: 1, NumberMisscheduled: 1},
	})
	if daemonSet.DesiredScheduled != 3 || daemonSet.CurrentScheduled != 2 || daemonSet.UpdatedReplicas != 1 || daemonSet.Misscheduled != 1 || daemonSet.UpdateStrategy != "RollingUpdate" {
		t.Fatalf("daemonset workload = %#v, want scheduling and update strategy details", daemonSet)
	}

	statefulSet := mapStatefulSet(appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "db", Name: "postgres"},
		Spec: appsv1.StatefulSetSpec{
			Replicas:       &replicas,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType},
			Template:       corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "db", Image: "postgres:v1"}}}},
		},
		Status: appsv1.StatefulSetStatus{ReadyReplicas: 2, AvailableReplicas: 2, UpdatedReplicas: 1},
	})
	if statefulSet.Replicas == nil || *statefulSet.Replicas != 3 || statefulSet.UpdatedReplicas != 1 || statefulSet.UpdateStrategy != "RollingUpdate" {
		t.Fatalf("statefulset workload = %#v, want replica and update details", statefulSet)
	}

	job := mapJob(batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Namespace: "jobs", Name: "migrate"},
		Spec: batchv1.JobSpec{
			Parallelism: &parallelism,
			Completions: &completions,
			Template:    corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "migrate", Image: "job:v1"}}}},
		},
		Status: batchv1.JobStatus{Active: 1, Succeeded: 3, Failed: 1},
	})
	if job.Parallelism == nil || *job.Parallelism != 2 || job.Completions == nil || *job.Completions != 5 || job.Active != 1 || job.Succeeded != 3 || job.Failed != 1 {
		t.Fatalf("job workload = %#v, want batch status details", job)
	}

	cronJob := mapCronJob(batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Namespace: "jobs", Name: "backup"},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
			Suspend:  &suspend,
			JobTemplate: batchv1.JobTemplateSpec{Spec: batchv1.JobSpec{
				Parallelism: &parallelism,
				Completions: &completions,
				Template:    corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "backup", Image: "backup:v1"}}}},
			}},
		},
		Status: batchv1.CronJobStatus{Active: []corev1.ObjectReference{{Name: "backup-1"}}},
	})
	if cronJob.Schedule != "*/5 * * * *" || cronJob.Suspend == nil || !*cronJob.Suspend || cronJob.Active != 1 {
		t.Fatalf("cronjob workload = %#v, want schedule and active job details", cronJob)
	}
}

func TestCollectRBACRecordsRuleAndSubjectDetails(t *testing.T) {
	client := kubernetesfake.NewSimpleClientset(
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Namespace: "app", Name: "reader"},
			Rules:      []rbacv1.PolicyRule{{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get", "list"}}},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Namespace: "app", Name: "readers"},
			RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: "reader"},
			Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Namespace: "app", Name: "web"}},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "view-nodes"},
			Rules:      []rbacv1.PolicyRule{{APIGroups: []string{""}, Resources: []string{"nodes"}, Verbs: []string{"get"}}},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "node-viewers"},
			RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "view-nodes"},
			Subjects:   []rbacv1.Subject{{Kind: "Group", APIGroup: "rbac.authorization.k8s.io", Name: "ops"}},
		},
	)
	kubernetes := inventory.Kubernetes{}
	var coverage []inventory.CoverageItem

	collectRBAC(context.Background(), client, &kubernetes, &coverage, time.Now().UTC())

	if len(kubernetes.RBAC.RoleDetails) != 1 || len(kubernetes.RBAC.RoleDetails[0].Rules) != 1 || kubernetes.RBAC.RoleDetails[0].Rules[0].Resources[0] != "pods" {
		t.Fatalf("role details = %#v, want rule details", kubernetes.RBAC.RoleDetails)
	}
	if len(kubernetes.RBAC.RoleBindingDetails) != 1 || kubernetes.RBAC.RoleBindingDetails[0].RoleRef.Kind != "Role" || kubernetes.RBAC.RoleBindingDetails[0].Subjects[0].Name != "web" {
		t.Fatalf("role binding details = %#v, want roleRef and subjects", kubernetes.RBAC.RoleBindingDetails)
	}
	if len(kubernetes.RBAC.ClusterRoleDetails) != 1 || len(kubernetes.RBAC.ClusterRoleBindingDetails) != 1 {
		t.Fatalf("cluster RBAC details = %#v %#v, want cluster role and binding details", kubernetes.RBAC.ClusterRoleDetails, kubernetes.RBAC.ClusterRoleBindingDetails)
	}
}

func TestCollectPoliciesRecordsPolicyDetails(t *testing.T) {
	minAvailable := intstr.FromInt32(1)
	maxUnavailable := intstr.FromString("25%")
	client := kubernetesfake.NewSimpleClientset(
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{Namespace: "app", Name: "web"},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable:   &minAvailable,
				MaxUnavailable: &maxUnavailable,
				Selector:       &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
			},
			Status: policyv1.PodDisruptionBudgetStatus{CurrentHealthy: 2, DesiredHealthy: 1, ExpectedPods: 3, DisruptionsAllowed: 1},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Namespace: "app", Name: "default-deny"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
				Ingress:     []networkingv1.NetworkPolicyIngressRule{{From: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"role": "api"}}}}}},
				Egress:      []networkingv1.NetworkPolicyEgressRule{{To: []networkingv1.NetworkPolicyPeer{{NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "platform"}}}}}},
			},
		},
		&corev1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{Namespace: "app", Name: "quota"},
			Spec:       corev1.ResourceQuotaSpec{Hard: corev1.ResourceList{corev1.ResourcePods: resourceMustParse("10")}},
			Status: corev1.ResourceQuotaStatus{
				Used: corev1.ResourceList{corev1.ResourcePods: resourceMustParse("3")},
			},
		},
		&corev1.LimitRange{
			ObjectMeta: metav1.ObjectMeta{Namespace: "app", Name: "limits"},
			Spec: corev1.LimitRangeSpec{Limits: []corev1.LimitRangeItem{{
				Type:           corev1.LimitTypeContainer,
				DefaultRequest: corev1.ResourceList{corev1.ResourceCPU: resourceMustParse("100m")},
			}}},
		},
	)
	kubernetes := inventory.Kubernetes{}
	var coverage []inventory.CoverageItem

	collectPolicies(context.Background(), client, &kubernetes, &coverage, time.Now().UTC())

	if len(kubernetes.Policies.PodDisruptionBudgetDetails) != 1 || kubernetes.Policies.PodDisruptionBudgetDetails[0].MinAvailable != "1" || kubernetes.Policies.PodDisruptionBudgetDetails[0].MaxUnavailable != "25%" {
		t.Fatalf("pdb details = %#v, want budget details", kubernetes.Policies.PodDisruptionBudgetDetails)
	}
	if len(kubernetes.Policies.NetworkPolicyDetails) != 1 || kubernetes.Policies.NetworkPolicyDetails[0].IngressPeers != 1 || kubernetes.Policies.NetworkPolicyDetails[0].EgressPeers != 1 {
		t.Fatalf("network policy details = %#v, want rule and peer counts", kubernetes.Policies.NetworkPolicyDetails)
	}
	if len(kubernetes.Policies.ResourceQuotaDetails) != 1 || kubernetes.Policies.ResourceQuotaDetails[0].Hard["pods"] != "10" || kubernetes.Policies.ResourceQuotaDetails[0].Used["pods"] != "3" {
		t.Fatalf("resource quota details = %#v, want hard and used values", kubernetes.Policies.ResourceQuotaDetails)
	}
	if len(kubernetes.Policies.LimitRangeDetails) != 1 || kubernetes.Policies.LimitRangeDetails[0].Items[0].DefaultRequest["cpu"] != "100m" {
		t.Fatalf("limit range details = %#v, want limit item details", kubernetes.Policies.LimitRangeDetails)
	}
}

func TestCollectAdmissionRecordsWebhookDetails(t *testing.T) {
	timeout := int32(7)
	fail := admissionv1.Fail
	sideEffects := admissionv1.SideEffectClassNone
	scope := admissionv1.NamespacedScope
	client := kubernetesfake.NewSimpleClientset(
		&admissionv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{Name: "injector"},
			Webhooks: []admissionv1.MutatingWebhook{{
				Name:         "injector.example.com",
				ClientConfig: admissionv1.WebhookClientConfig{Service: &admissionv1.ServiceReference{Namespace: "system", Name: "webhook"}},
				Rules: []admissionv1.RuleWithOperations{{
					Operations: []admissionv1.OperationType{admissionv1.Create},
					Rule:       admissionv1.Rule{APIGroups: []string{""}, APIVersions: []string{"v1"}, Resources: []string{"pods"}, Scope: &scope},
				}},
				FailurePolicy:           &fail,
				SideEffects:             &sideEffects,
				TimeoutSeconds:          &timeout,
				AdmissionReviewVersions: []string{"v1"},
			}},
		},
	)
	kubernetes := inventory.Kubernetes{}
	var coverage []inventory.CoverageItem

	collectAdmission(context.Background(), client, &kubernetes, &coverage, time.Now().UTC())

	if len(kubernetes.AdmissionWebhooks) != 1 || kubernetes.AdmissionWebhooks[0].Kind != "MutatingWebhookConfiguration" || len(kubernetes.AdmissionWebhooks[0].Webhooks) != 1 {
		t.Fatalf("admission webhooks = %#v, want mutating webhook config", kubernetes.AdmissionWebhooks)
	}
	webhook := kubernetes.AdmissionWebhooks[0].Webhooks[0]
	if webhook.ClientService.Name != "webhook" || webhook.FailurePolicy != "Fail" || webhook.TimeoutSeconds == nil || *webhook.TimeoutSeconds != 7 || webhook.Resources[0] != "pods" {
		t.Fatalf("webhook = %#v, want service client and rule details", webhook)
	}
}

func TestCollectGatewaysFallsBackToV1Beta1AndMapsRoutes(t *testing.T) {
	gatewayClassGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "gatewayclasses"}
	gatewayGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "gateways"}
	httpRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "httproutes"}
	grpcRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "grpcroutes"}
	tlsRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "tlsroutes"}
	tcpRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "tcproutes"}
	udpRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "udproutes"}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		gatewayAPIListKinds(),
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
		gatewayAPIListKinds(),
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

func TestCollectGatewaysFallsBackWhenHTTPRouteMissingFromDiscovery(t *testing.T) {
	httpRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		gatewayAPIListKinds(),
	)
	client.Fake.PrependReactor("list", "*", func(action ktesting.Action) (bool, runtime.Object, error) {
		switch action.GetResource() {
		case httpRouteGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.k8s.io/v1",
				"kind":       "HTTPRoute",
				"metadata":   map[string]any{"namespace": "app", "name": "web"},
				"spec":       map[string]any{"parentRefs": []any{map[string]any{"name": "public"}}},
			}}}}, nil
		default:
			return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), action.GetResource().Resource)
		}
	})
	kubernetes := inventory.Kubernetes{APIResources: []inventory.APIResource{
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gatewayclasses", Kind: "GatewayClass", Verbs: []string{"get", "list"}},
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways", Kind: "Gateway", Namespaced: true, Verbs: []string{"get", "list"}},
	}}
	var coverage []inventory.CoverageItem

	collectGateways(context.Background(), client, &kubernetes, &coverage, time.Now().UTC(), "")

	if len(kubernetes.GatewayRoutes) != 1 || kubernetes.GatewayRoutes[0].Kind != "HTTPRoute" || kubernetes.GatewayRoutes[0].Name != "web" {
		t.Fatalf("gateway routes = %#v, want fallback v1 HTTPRoute", kubernetes.GatewayRoutes)
	}
}

func TestCollectGatewaysCollectsGrantsAndPolicies(t *testing.T) {
	referenceGrantGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "referencegrants"}
	backendTLSPolicyGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "backendtlspolicies"}
	backendTrafficPolicyGVR := schema.GroupVersionResource{Group: "gateway.networking.x-k8s.io", Version: "v1alpha1", Resource: "xbackendtrafficpolicies"}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		gatewayAPIListKinds(),
	)
	client.Fake.PrependReactor("list", "*", func(action ktesting.Action) (bool, runtime.Object, error) {
		switch action.GetResource() {
		case referenceGrantGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.k8s.io/v1beta1",
				"kind":       "ReferenceGrant",
				"metadata":   map[string]any{"namespace": "backend", "name": "allow-app"},
				"spec": map[string]any{
					"from": []any{map[string]any{"group": "gateway.networking.k8s.io", "kind": "HTTPRoute", "namespace": "app"}},
					"to":   []any{map[string]any{"group": "", "kind": "Service", "name": "api"}},
				},
			}}}}, nil
		case backendTLSPolicyGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.k8s.io/v1",
				"kind":       "BackendTLSPolicy",
				"metadata":   map[string]any{"namespace": "backend", "name": "api-tls"},
				"spec": map[string]any{
					"targetRefs": []any{map[string]any{"group": "", "kind": "Service", "name": "api"}},
					"validation": map[string]any{"hostname": "api.backend.svc", "caCertificateRefs": []any{map[string]any{"name": "api-ca"}}},
				},
			}}}}, nil
		case backendTrafficPolicyGVR:
			return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]any{
				"apiVersion": "gateway.networking.x-k8s.io/v1alpha1",
				"kind":       "XBackendTrafficPolicy",
				"metadata":   map[string]any{"namespace": "backend", "name": "api-traffic"},
				"spec": map[string]any{
					"targetRef":       map[string]any{"group": "", "kind": "Service", "name": "api"},
					"retryConstraint": map[string]any{"budget": map[string]any{"percent": int64(10)}},
				},
			}}}}, nil
		default:
			return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), action.GetResource().Resource)
		}
	})
	kubernetes := inventory.Kubernetes{APIResources: []inventory.APIResource{
		{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "referencegrants", Kind: "ReferenceGrant", Namespaced: true, Verbs: []string{"get", "list"}},
		{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "backendtlspolicies", Kind: "BackendTLSPolicy", Namespaced: true, Verbs: []string{"get", "list"}},
		{Group: "gateway.networking.x-k8s.io", Version: "v1alpha1", Resource: "xbackendtrafficpolicies", Kind: "XBackendTrafficPolicy", Namespaced: true, Verbs: []string{"get", "list"}},
	}}
	var coverage []inventory.CoverageItem

	collectGateways(context.Background(), client, &kubernetes, &coverage, time.Now().UTC(), "")

	if len(kubernetes.ReferenceGrants) != 1 || kubernetes.ReferenceGrants[0].Name != "allow-app" || len(kubernetes.ReferenceGrants[0].From) != 1 || len(kubernetes.ReferenceGrants[0].To) != 1 {
		t.Fatalf("reference grants = %#v, want one grant with from/to refs", kubernetes.ReferenceGrants)
	}
	if len(kubernetes.GatewayPolicies) != 2 {
		t.Fatalf("gateway policies = %#v, want BackendTLSPolicy and XBackendTrafficPolicy", kubernetes.GatewayPolicies)
	}
	if kubernetes.GatewayPolicies[0].Kind != "BackendTLSPolicy" || len(kubernetes.GatewayPolicies[0].TargetRefs) != 1 || len(kubernetes.GatewayPolicies[0].Details) == 0 {
		t.Fatalf("first gateway policy = %#v, want backend TLS policy with target and details", kubernetes.GatewayPolicies[0])
	}
	if kubernetes.GatewayPolicies[1].Kind != "XBackendTrafficPolicy" || len(kubernetes.GatewayPolicies[1].TargetRefs) != 1 || len(kubernetes.GatewayPolicies[1].Details) == 0 {
		t.Fatalf("second gateway policy = %#v, want backend traffic policy with target and details", kubernetes.GatewayPolicies[1])
	}
}

func TestCollectGatewaysFallsBackToContextNamespaceWhenAllNamespacesDenied(t *testing.T) {
	gatewayGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}
	httpRouteGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		gatewayAPIListKinds(),
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
