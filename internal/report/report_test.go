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
			Insights:                []inventory.EKSInsight{{Name: "Deprecated APIs", Category: "UPGRADE_READINESS", KubernetesVersion: "1.32", Status: "WARNING", Reason: "deprecated API observed", Recommendation: "Migrate API versions.", Resources: []inventory.EKSInsightResource{{KubernetesResourceURI: "/apis/extensions/v1beta1/ingresses", Status: "WARNING"}}}},
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
		"### EKS compute capacity and declared usage",
		"| CPU | 2 cores | 1930m | 500m | 1 cores |",
		"| Memory | 8.0 GiB | 7.4 GiB | 256.0 MiB | 512.0 MiB |",
		"### EKS network",
		"| Cluster subnets | subnet-a,subnet-b |",
		"### EKS security and identity",
		"| Cluster role | arn:aws:iam::123456789012:role/eks-cluster |",
		"### Managed nodegroups",
		"| system | 1.31 | 1.31.1-20260901 | ACTIVE | AL2023_x86_64_STANDARD | m7i.large | desired=2 min=1 max=4 | subnet-a,subnet-b | arn:aws:iam::123456789012:role/node | - |",
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
		`<script id="snapshot-data" type="application/json">`,
		`"schemaVersion": "teleskope.io/snapshot/v1alpha1"`,
		"Running images",
		"EKS overview",
		"Upgrade / rollback insights",
		"Compute capacity and declared usage",
		"Security and identity",
		"nodesTable",
		"eksInsightsTable",
		"accessEntriesTable",
		"podIdentityTable",
		"IRSA:",
		"nodeTotals",
		"declaredTotals",
		"quantityValue",
		"Gateway routes",
		"gatewayClassesTable",
		"gatewaysTable",
		"gatewayRoutesTable",
		"gatewayBackends",
		"Custom resources",
		"function namespacesOf",
		"resourceTypes = [",
		`<select id="resourceType" aria-label="Resource type">`,
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
