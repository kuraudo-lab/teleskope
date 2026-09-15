package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestWriteDirectoryCreatesRawJSONAndSummary(t *testing.T) {
	now := time.Date(2026, 9, 7, 8, 9, 10, 0, time.UTC)
	admin := true
	autoCompute := true
	autoBlock := false
	autoLB := true
	disk := int32(20)
	zero := int32(0)
	desired := int32(2)
	minSize := int32(1)
	maxSize := int32(4)
	snapshot := &inventory.Snapshot{
		SchemaVersion: "teleskope.io/snapshot/v1alpha1",
		CollectedAt:   now,
		Source:        inventory.Source{Tool: "teleskope", Version: "test", Mode: "out-of-cluster/run-once"},
		AWS:           inventory.AWSIdentity{Region: "ap-northeast-1", AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:user/scanner"},
		EKS: inventory.EKSInventory{
			Cluster: inventory.Cluster{
				Name:            "prod",
				ARN:             "arn:aws:eks:ap-northeast-1:123456789012:cluster/prod",
				Version:         "1.31",
				PlatformVersion: "eks.7",
				Status:          "ACTIVE",
				RoleARN:         "arn:aws:iam::123456789012:role/eks-cluster",
				VPC: inventory.VPCConfig{
					VPCID:                  "vpc-123",
					SubnetIDs:              []string{"subnet-a", "subnet-b"},
					SecurityGroupIDs:       []string{"sg-extra"},
					ClusterSecurityGroupID: "sg-cluster",
					EndpointPublicAccess:   true,
					EndpointPrivateAccess:  true,
					PublicAccessCIDRs:      []string{"203.0.113.0/24"},
				},
				Network:      inventory.NetworkConfig{IPFamily: "ipv4", ServiceIPv4CIDR: "172.20.0.0/16", AutoModeLoadBalancingEnabled: &autoLB},
				AccessConfig: inventory.AccessConfig{AuthenticationMode: "API_AND_CONFIG_MAP", BootstrapClusterCreatorAdminPermissions: &admin},
				AutoMode:     inventory.AutoModeConfig{ComputeEnabled: &autoCompute, ComputeNodePools: []string{"system"}, BlockStorageEnabled: &autoBlock},
				Encryption:   []inventory.EncryptionConfig{{Resources: []string{"secrets"}, KeyARN: "arn:aws:kms:ap-northeast-1:123456789012:key/key-id"}},
				OIDCIssuer:   "https://oidc.eks.ap-northeast-1.amazonaws.com/id/ABC",
			},
			Insights: []inventory.EKSInsight{
				{Name: "Deprecated APIs", Category: "UPGRADE_READINESS", KubernetesVersion: "1.32", Status: "WARNING", Reason: "deprecated API observed", Recommendation: "Migrate API versions.", Resources: []inventory.EKSInsightResource{{KubernetesResourceURI: "/apis/extensions/v1beta1/ingresses", Status: "WARNING"}}},
				{Name: "Addon Compatibility", Category: "UPGRADE_READINESS", KubernetesVersion: "1.32", Status: "WARNING", Reason: "addon update required", Recommendation: "Upgrade add-on before the cluster upgrade.", AddonCompatibility: []inventory.AddonCompatibility{{Name: "vpc-cni", CompatibleVersions: []string{"v1.20.0-eksbuild.1"}}}},
			},
			Addons:                  []inventory.Addon{{Name: "vpc-cni", Version: "v1.19.0-eksbuild.1", Status: "ACTIVE", Namespace: "kube-system", ServiceAccountRoleARN: "arn:aws:iam::123456789012:role/cni"}},
			Nodegroups:              []inventory.Nodegroup{{Name: "system", Version: "1.31", ReleaseVersion: "1.31.1-20260901", Status: "ACTIVE", AMIType: "AL2023_x86_64_STANDARD", CapacityType: "ON_DEMAND", NodeRoleARN: "arn:aws:iam::123456789012:role/node", Subnets: []string{"subnet-a", "subnet-b"}, InstanceTypes: []string{"m7i.large"}, DiskSizeGiB: &disk, DesiredSize: &desired, MinSize: &minSize, MaxSize: &maxSize}},
			AccessEntries:           []inventory.AccessEntry{{PrincipalARN: "arn:aws:iam::123456789012:role/admin", Type: "STANDARD", KubernetesGroups: []string{"system:masters"}, Policies: []inventory.AssociatedPolicy{{PolicyARN: "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy", ScopeType: "cluster"}}}},
			PodIdentityAssociations: []inventory.PodIdentityAssociation{{Namespace: "app", ServiceAccount: "web", RoleARN: "arn:aws:iam::123456789012:role/web"}},
		},
		Kubernetes: inventory.Kubernetes{
			Context: "prod",
			Nodes: []inventory.Node{{
				ObjectRef:        inventory.ObjectRef{Kind: "Node", Name: "node-a"},
				ProviderID:       "aws:///ap-northeast-1a/i-123",
				Ready:            "True",
				KubeletVersion:   "v1.31.1-eks",
				ContainerRuntime: "containerd://1.7.27",
				OSImage:          "Amazon Linux 2023",
				KernelVersion:    "6.1.0",
				Architecture:     "amd64",
				Capacity:         map[string]string{"cpu": "2", "memory": "8Gi", "pods": "29"},
				Allocatable:      map[string]string{"cpu": "1930m", "memory": "7600Mi", "pods": "29"},
				Labels:           map[string]string{"eks.amazonaws.com/nodegroup": "system", "node.kubernetes.io/instance-type": "m7i.large", "topology.kubernetes.io/zone": "ap-northeast-1a"},
			}},
			ServiceAccounts: []inventory.ServiceAccount{{
				ObjectRef: inventory.ObjectRef{Kind: "ServiceAccount", Namespace: "app", Name: "worker"},
			}},
			Workloads: []inventory.Workload{
				{
					ObjectRef:          inventory.ObjectRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"},
					Replicas:           &desired,
					ReadyReplicas:      1,
					Selector:           map[string]string{"app": "web"},
					ServiceAccountName: "web",
					Containers:         []inventory.Container{{Name: "web", Image: "repo/web:v1"}},
				},
				{
					ObjectRef:            inventory.ObjectRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "worker"},
					Replicas:             &zero,
					Selector:             map[string]string{"app": "worker"},
					ServiceAccountName:   "worker",
					Containers:           []inventory.Container{{Name: "worker", Image: "repo/worker:v2", EnvConfigRefs: []inventory.ObjectRef{{Kind: "ConfigMap", Namespace: "app", Name: "worker-env"}}}},
					Volumes:              []inventory.Volume{{Name: "cache", Type: "persistentVolumeClaim", PersistentVolumeClaim: "worker-cache"}, {Name: "settings", Type: "configMap", ConfigMap: "worker-config"}, {Name: "token", Type: "secret", Secret: "worker-secret"}},
					ConfigRefs:           []inventory.ObjectRef{{Kind: "ConfigMap", Namespace: "app", Name: "worker-config"}},
					SecretRefs:           []inventory.ObjectRef{{Kind: "Secret", Namespace: "app", Name: "worker-secret"}},
					ImagePullSecretRefs:  []inventory.ObjectRef{{Kind: "Secret", Namespace: "app", Name: "worker-pull"}},
					VolumeClaimTemplates: []inventory.ObjectRef{{Kind: "PersistentVolumeClaim", Namespace: "app", Name: "worker-cache-template"}},
				},
			},
			Pods: []inventory.Pod{{
				ObjectRef: inventory.ObjectRef{Kind: "Pod", Namespace: "app", Name: "web-abc"},
				Phase:     "Running",
				NodeName:  "node-a",
				Containers: []inventory.Container{{
					Name:      "web",
					Image:     "repo/web:v1",
					Resources: map[string]string{"requests.cpu": "500m", "requests.memory": "256Mi", "limits.cpu": "1", "limits.memory": "512Mi"},
				}},
			}},
			CustomResourceDefinitions: []inventory.CustomResourceDefinition{
				{
					ObjectRef: inventory.ObjectRef{Kind: "CustomResourceDefinition", Name: "widgets.example.com"},
					Group:     "example.com",
					Scope:     "Namespaced",
					Kind:      "Widget",
					Plural:    "widgets",
					Versions:  []inventory.CRDVersion{{Name: "v1", Served: true, Storage: true}},
				},
			},
			CustomResourceCounts: []inventory.CustomResourceCount{
				{CRDName: "widgets.example.com", Group: "example.com", Version: "v1", Kind: "Widget", Plural: "widgets", Scope: "Namespaced", InstanceCount: 2, NamespaceCount: 2},
			},
			CustomResourceInstances: []inventory.CustomResourceInstance{
				{
					ObjectRef:  inventory.ObjectRef{APIVersion: "example.com/v1", Kind: "Widget", Namespace: "app", Name: "blue"},
					CRDName:    "widgets.example.com",
					CRDGroup:   "example.com",
					CRDVersion: "v1",
					CRDKind:    "Widget",
					CRDPlural:  "widgets",
				},
			},
			RunningImages: []inventory.RunningImage{
				{
					Image:          "repo/web:v1",
					PodCount:       1,
					ContainerCount: 1,
					ImageIDs:       []string{"docker-pullable://repo/web@sha256:aaaaaaaaaaaaaaaa"},
					Runtimes:       []string{"containerd"},
					Namespaces:     []string{"app"},
					Workloads:      []inventory.ObjectRef{{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"}},
				},
			},
			RunningContainers: []inventory.RunningContainer{
				{
					Namespace:     "app",
					Pod:           "web-abc",
					Container:     "web",
					ContainerType: "app",
					Image:         "repo/web:v1",
					ImageID:       "docker-pullable://repo/web@sha256:aaaaaaaaaaaaaaaa",
					NodeName:      "node-a",
					Workload:      inventory.ObjectRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"},
				},
			},
			Services: []inventory.Service{
				{
					ObjectRef: inventory.ObjectRef{Kind: "Service", Namespace: "app", Name: "web"},
					Type:      "ClusterIP",
					Selector:  map[string]string{"app": "web"},
					Ports:     []inventory.ServicePort{{Protocol: "TCP", Port: 80, TargetPort: "http"}},
				},
				{
					ObjectRef: inventory.ObjectRef{Kind: "Service", Namespace: "app", Name: "worker"},
					Type:      "ClusterIP",
					Selector:  map[string]string{"app": "worker"},
					Ports:     []inventory.ServicePort{{Protocol: "TCP", Port: 9090, TargetPort: "metrics"}},
				},
			},
			GatewayClasses: []inventory.GatewayClass{{
				ObjectRef:      inventory.ObjectRef{APIVersion: "gateway.networking.k8s.io/v1", Kind: "GatewayClass", Name: "alb"},
				ControllerName: "gateway.k8s.aws/alb",
			}},
			Gateways: []inventory.Gateway{{
				ObjectRef: inventory.ObjectRef{APIVersion: "gateway.networking.k8s.io/v1", Kind: "Gateway", Namespace: "app", Name: "public"},
				ClassName: "alb",
				Addresses: []string{"internal-alb.example.com"},
				Listeners: []inventory.GatewayListener{{Name: "https", Protocol: "HTTPS", Port: 443, Hostname: "example.com", AllowedRoutes: []string{"namespaces=Same", "gateway.networking.k8s.io/HTTPRoute"}}},
			}},
			GatewayRoutes: []inventory.GatewayRoute{{
				ObjectRef:  inventory.ObjectRef{APIVersion: "gateway.networking.k8s.io/v1", Kind: "HTTPRoute", Namespace: "app", Name: "web"},
				ParentRefs: []inventory.GatewayParentRef{{Kind: "Gateway", Namespace: "app", Name: "public", SectionName: "https"}},
				Hostnames:  []string{"example.com"},
				Rules: []inventory.GatewayRouteRule{{
					Matches:     []string{"PathPrefix:/"},
					BackendRefs: []inventory.ObjectRef{{APIVersion: "v1", Kind: "Service", Namespace: "app", Name: "web"}},
				}},
			}},
			PersistentVolumeClaims: []inventory.PersistentVolumeClaim{{
				ObjectRef:        inventory.ObjectRef{Kind: "PersistentVolumeClaim", Namespace: "app", Name: "worker-cache"},
				StorageClassName: "gp3",
				RequestedStorage: "5Gi",
				Phase:            "Bound",
			}},
		},
		Coverage: []inventory.CoverageItem{{Area: "kubernetes", Resource: "services", Status: "complete", ObjectCount: 1, CollectedAt: now}},
	}

	artifact, err := WriteDirectory(snapshot, Options{
		BaseDir: t.TempDir(),
		Target:  "Prod Cluster",
		Now:     now,
	})
	if err != nil {
		t.Fatalf("WriteDirectory returned error: %v", err)
	}
	if !strings.HasSuffix(artifact.Dir, "prod-cluster-20260907-080910") {
		t.Fatalf("artifact dir = %q, want sanitized timestamped name", artifact.Dir)
	}
	for _, name := range []string{"snapshot.json", "source.json", "aws.json", "eks.json", "kubernetes.json", "coverage.json", "summary.md", "index.html"} {
		if _, err := os.Stat(filepath.Join(artifact.Dir, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}

	summary, err := os.ReadFile(filepath.Join(artifact.Dir, "summary.md"))
	if err != nil {
		t.Fatalf("read summary.md: %v", err)
	}
	for _, want := range []string{
		"# Teleskope scan summary",
		"Target: `Prod Cluster`",
		"| CRD instances | 1 |",
		"### EKS upgrade and rollback insights",
		"| Deprecated APIs | UPGRADE_READINESS | 1.32 | WARNING | deprecated API observed | Migrate API versions. | 1 |",
		"| Addon Compatibility | UPGRADE_READINESS | 1.32 | WARNING | addon update required | Upgrade add-on before the cluster upgrade. | 0 |",
		"| vpc-cni | v1.19.0-eksbuild.1 | ACTIVE | kube-system | 1.32 | v1.20.0-eksbuild.1 | status=WARNING use v1.20.0-eksbuild.1 reason=addon update required recommendation=Upgrade add-on before the cluster upgrade. | IRSA:arn:aws:iam::123456789012:role/cni | - |",
		"### EKS compute capacity and declared usage",
		"| CPU | 2 cores | 1930m | 500m | 1 cores |",
		"| Memory | 8.0 GiB | 7.4 GiB | 256.0 MiB | 512.0 MiB |",
		"### EKS network",
		"| Cluster subnets | subnet-a,subnet-b |",
		"### EKS security and identity",
		"| Cluster role | arn:aws:iam::123456789012:role/eks-cluster |",
		"### Managed nodegroups",
		"| system | 1.31 | 1.31.1-20260901 | ACTIVE | AL2023_x86_64_STANDARD | m7i.large | desired=2 min=1 max=4 | subnet-a,subnet-b | arn:aws:iam::123456789012:role/node | - |",
		"### Nodegroup runtime readiness",
		"| system | managed-nodegroup | ap-northeast-1a | 1.31 | 1.31.1-20260901 | AL2023_x86_64_STANDARD | - | node-a | v1.31.1-eks | Amazon Linux 2023 | containerd://1.7.27 | m7i.large | observed: runtime evidence linked to managed nodegroup |",
		"### Nodes",
		"| node-a | aws:///ap-northeast-1a/i-123 | True | true | v1.31.1-eks | containerd://1.7.27 | Amazon Linux 2023 | 6.1.0 | amd64 | cpu=2,memory=8Gi,pods=29 | cpu=1930m,memory=7600Mi,pods=29 | - | eks.amazonaws.com/nodegroup=system,node.kubernetes.io/instance-type=m7i.large,topology.kubernetes.io/zone=ap-northeast-1a | - |",
		"### Custom resources",
		"| example.com/Widget | 2 | 2 | Namespaced | v1 | widgets |",
		"| widget/app/blue | example.com/Widget | v1 | widgets.example.com | - |",
		"| Running images | 1 |",
		"| Running containers | 1 |",
		"| repo/web:v1 (pods=1) | 1 | containerd | app | deployment/app/web | sha256:aaaaaaaaaaaa |",
		"| app | web-abc | web | app | repo/web:v1 | sha256:aaaaaaaaaaaa | node-a | deployment/app/web |",
		"| GatewayClass | alb | - | - | controller=gateway.k8s.aws/alb |",
		"| Gateway | gateway/app/public | alb | listeners=https:HTTPS/443 host=example.com allowed=namespaces=Same,gateway.networking.k8s.io/HTTPRoute | addresses=internal-alb.example.com |",
		"| HTTPRoute | httproute/app/web | PathPrefix:/ | hosts=example.com parents=Gateway/app/public#https | backends=service/app/web |",
		"| Service | service/app/web | ClusterIP | app=web | TCP/80->http |",
	} {
		if !strings.Contains(string(summary), want) {
			t.Fatalf("summary missing %q:\n%s", want, string(summary))
		}
	}

	html, err := os.ReadFile(filepath.Join(artifact.Dir, "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	for _, want := range []string{
		"<title>Teleskope cluster report</title>",
		`rel="icon" type="image/png"`,
		`aria-hidden="true"><img src="data:image/png;base64,`,
		`<script id="boot-config" type="application/json">`,
		`<script id="snapshot-data" type="application/json">`,
		`<script id="eks-projection-data" type="application/json">`,
		`<script id="markdown-data" type="text/plain">`,
		`"mode":"offline"`,
		`"schemaVersion": "teleskope.io/snapshot/v1alpha1"`,
		`"visible":true`,
		"Target: `Prod Cluster`",
		"Running images",
		"EKS overview",
		"Upgrade / rollback insights",
		"Compute capacity and declared usage",
		"Security and identity",
		"nodesTable",
		"eksInsightsTable",
		"accessEntriesTable",
		"podIdentityTable",
		"eksProjection.overview",
		"eksProjection.insights",
		"eksProjection.capacity",
		"eksProjection.addons",
		"targetKubernetes",
		"compatibleVersions",
		"Upgrade add-on before the cluster upgrade.",
		"eksProjection.nodegroups",
		"nodegroupReadiness",
		"nodegroupReadinessTable",
		"eksProjection.nodegroupReadiness",
		"IRSA:",
		"Gateway routes",
		"gatewayClassesTable",
		"gatewaysTable",
		"gatewayRoutesTable",
		"table('gatewayClassesTable'",
		"table('gatewaysTable'",
		"table('gatewayRoutesTable'",
		`data-section="security"`,
		`data-section="policies"`,
		"rbacRolesTable",
		"rbacBindingsTable",
		"admissionWebhooksTable",
		"pdbTable",
		"networkPoliciesTable",
		"resourceQuotasTable",
		"limitRangesTable",
		"table('rbacRolesTable'",
		"table('networkPoliciesTable'",
		"table('admissionWebhooksTable'",
		"gatewayBackends",
		"Custom resources",
		"function namespacesOf",
		"resourceTypes = [",
		`data-section="events"`,
		"['events', 'Events']",
		`<select id="resourceType" aria-label="Resource type">`,
		`id="exportReport"`,
		`id="exportMenu"`,
		`data-export-format="json"`,
		`data-export-format="markdown"`,
		"function downloadText",
		"async function exportReport",
		"function toggleExportMenu",
		"bootConfig.endpoints?.exportSnapshot",
		"bootConfig.endpoints?.exportSummary",
		`id="themeToggle"`,
		`prefers-color-scheme: dark`,
		`teleskope.theme`,
		`localStorage.setItem(themeKey, next)`,
		`document.documentElement.dataset.theme = next`,
		"resourceFilter = 'all'",
		"activeSection = 'overview'",
		"currentTopologyResourceFilter",
		"resourceType').disabled = activeSection !== 'overview'",
		"showPods = topologyResourceFilter === 'pod'",
		"topologyResourceFilter === 'crd'",
		"function renderTopology",
		"topologySvg",
		"topologyViewport",
		"Wheel to zoom",
		"Drag to pan",
		"topologyDrag",
		"pointerdown",
		"pointermove",
		"applyTopologyTransform",
		"Workload lookup",
		`id="workloadSearch"`,
		`id="workloadSearchResults"`,
		"function workloadDetailHTML",
		"function renderWorkloadSearch",
		"function openWorkloadDetail",
		"data-workload-key",
		"dependency-graph",
		"repo/worker:v2",
		"worker-cache-template",
		"worker-config",
		"worker-secret",
		"worker-pull",
		"pods=${r.podCount || 0}",
		"join('<br>')",
	} {
		if !strings.Contains(string(html), want) {
			t.Fatalf("index.html missing %q", want)
		}
	}
	if strings.Contains(string(html), "<h3>Coverage</h3>") {
		t.Fatalf("index.html should not render coverage panel")
	}
}

func TestLiveHTMLIncludesRecentEvents(t *testing.T) {
	html := LiveHTML()
	for _, want := range []string{
		`<body data-live="true" class="live-loading">`,
		`<script id="boot-config" type="application/json">`,
		"Recent events",
		"live-event-list",
		"live-progress",
		"sourceFreshness",
		"source-freshness",
		"function renderSourceFreshness",
		"lastFullSyncAt",
		"watch reconnecting",
		"eventKind(event)",
		"spinning",
		"@keyframes teleskope-spin",
		"updateLiveProgress",
		"background:var(--live-event-bg)",
		"renderLiveStatus(data.sources, data.events)",
		`id="analyzeSnapshot"`,
		"Analyze with AI",
		"ai-button",
		"teleskope-bling",
		"Advisory",
		"AI analysis",
		"/api/analyze",
		"bootConfig.endpoints?.analyze",
		"bootConfig.endpoints?.snapshot",
		"bootConfig.endpoints?.exportSummary",
		"bootConfig.endpoints?.exportSnapshot",
		"function renderAnalysis",
		"analysisInFlight",
		"analysisKey === lastAnalyzedRequestKey",
		"function currentAnalysisRequest",
		"function currentAnalysisKey",
		"Analyze is available after the first scan completes.",
		"cursor:not-allowed",
		"font:inherit; font-size:14px",
		"lastAnalyzedRevision",
		"lastAnalyzedRequestKey",
		"function syncAnalyzeButtonState",
		"selectSection('advisor')",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("live HTML missing %q", want)
		}
	}
	for _, old := range []string{`id="live-status"`, `id="live-events"`, "Collection details"} {
		if strings.Contains(html, old) {
			t.Fatalf("live HTML should not include top-level live status artifact %q", old)
		}
	}
}

func TestLiveHTMLWithOptionsRendersEndpointConfig(t *testing.T) {
	html := LiveHTMLWithOptions(LiveHTMLOptions{
		SnapshotPath:       "/custom/snapshot?id=prod-a",
		AnalyzePath:        "/custom/analyze?id=prod-a",
		ExportSnapshotPath: "/custom/export/snapshot.json?id=prod-a",
		ExportSummaryPath:  "/custom/export/summary.md?id=prod-a",
	})
	for _, want := range []string{
		`"snapshot":"/custom/snapshot?id=prod-a"`,
		`"analyze":"/custom/analyze?id=prod-a"`,
		`"exportSnapshot":"/custom/export/snapshot.json?id=prod-a"`,
		`"exportSummary":"/custom/export/summary.md?id=prod-a"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("live endpoint config missing %q:\n%s", want, html)
		}
	}
	for _, old := range []string{
		`fetch('/custom/snapshot?id=prod-a'`,
		`fetch('/custom/analyze?id=prod-a'`,
	} {
		if strings.Contains(html, old) {
			t.Fatalf("endpoint was inlined into script instead of boot config: %q", old)
		}
	}
}

func TestRenderUIEscapesScriptsAndScriptData(t *testing.T) {
	snapshot := &inventory.Snapshot{}
	html, err := RenderUI(UIRenderOptions{
		Mode:     UIModeOffline,
		Snapshot: snapshot,
		Target:   `</script><script>alert(1)</script>`,
		Scripts:  []string{`console.log("</script>")`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, `</script><script>alert(1)</script>`) || strings.Contains(html, `console.log("</script>")`) {
		t.Fatalf("script content was not escaped:\n%s", html)
	}
	for _, want := range []string{`<\/script><script>alert(1)<\/script>`, `console.log("<\/script>")`} {
		if !strings.Contains(html, want) {
			t.Fatalf("escaped script content missing %q:\n%s", want, html)
		}
	}
}

func TestRenderUILiveBodyMarkerComesFromMode(t *testing.T) {
	livePage, err := RenderUI(UIRenderOptions{Mode: UIModeLive})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(livePage, `<body data-live="true" class="live-loading">`) {
		t.Fatalf("live page missing body marker")
	}
	offlinePage, err := RenderUI(UIRenderOptions{Mode: UIModeOffline, Snapshot: &inventory.Snapshot{}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(offlinePage, `data-live="true"`) || !strings.Contains(offlinePage, "<body>") {
		t.Fatalf("offline page should not include live body marker")
	}
}

func TestAdvisorArtifactAndHTML(t *testing.T) {
	s := &inventory.Snapshot{}
	artifact, err := WriteDirectory(s, Options{BaseDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(artifact.Dir, "advisor.json"))
	if err != nil || !strings.Contains(string(data), `"assessment": "unknown"`) {
		t.Fatalf("advisor artifact: %s %v", data, err)
	}
	page, err := HTML(s)
	if err != nil || strings.Contains(page, "__TELESKOPE_ADVISOR_JSON__") || strings.Contains(page, "__TELESKOPE_EKS_PROJECTION_JSON__") || !strings.Contains(page, `data-section="advisor"`) {
		t.Fatal("advisor HTML missing or not encoded")
	}
	if strings.Contains(LiveHTML(), "__TELESKOPE_ADVISOR_JSON__") {
		t.Fatal("unresolved live placeholder")
	}
	if strings.Contains(LiveHTML(), "__TELESKOPE_EKS_PROJECTION_JSON__") {
		t.Fatal("unresolved live EKS projection placeholder")
	}
	if strings.Contains(LiveHTML(), "__TELESKOPE_MARKDOWN__") {
		t.Fatal("unresolved live markdown placeholder")
	}
}
