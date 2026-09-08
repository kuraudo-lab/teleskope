package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestHumanIncludesActionableKubernetesFacts(t *testing.T) {
	allowExpansion := true
	runtimeReady := true
	snapshot := &inventory.Snapshot{
		Kubernetes: inventory.Kubernetes{
			Context: "prod",
			Version: inventory.KubernetesVersion{GitVersion: "v1.32.0"},
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
			Nodes: []inventory.Node{
				{ObjectRef: inventory.ObjectRef{Kind: "Node", Name: "ip-10-0-0-1"}, ContainerRuntime: "containerd://2.0.0", Ready: "True"},
			},
			Workloads: []inventory.Workload{
				{
					ObjectRef: inventory.ObjectRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"},
					Containers: []inventory.Container{
						{Name: "web", Image: "example/web:v1"},
					},
					Volumes: []inventory.Volume{
						{Name: "data", Type: "persistentVolumeClaim", PersistentVolumeClaim: "web-data"},
					},
					Selector:           map[string]string{"app": "web"},
					ServiceAccountName: "web",
				},
				{
					ObjectRef: inventory.ObjectRef{APIVersion: "apps/v1", Kind: "DaemonSet", Namespace: "kube-system", Name: "aws-node"},
					Containers: []inventory.Container{
						{Name: "aws-node", Image: "amazon-k8s-cni:v1.19.0"},
					},
					ConfigRefs: []inventory.ObjectRef{
						{APIVersion: "v1", Kind: "ConfigMap", Namespace: "kube-system", Name: "amazon-vpc-cni"},
					},
				},
			},
			Pods: []inventory.Pod{
				{
					ObjectRef: inventory.ObjectRef{Kind: "Pod", Namespace: "app", Name: "web-abc"},
					Phase:     "Running",
					NodeName:  "ip-10-0-0-1",
					Containers: []inventory.Container{
						{Name: "web", Image: "example/web:v1", ImageID: "docker-pullable://example/web@sha256:aaaaaaaaaaaaaaaa", ContainerID: "containerd://abc", State: "running"},
					},
				},
			},
			RunningContainers: []inventory.RunningContainer{
				{
					Namespace:     "app",
					Pod:           "web-abc",
					NodeName:      "ip-10-0-0-1",
					Container:     "web",
					ContainerType: "app",
					Image:         "example/web:v1",
					ImageID:       "docker-pullable://example/web@sha256:aaaaaaaaaaaaaaaa",
					ContainerID:   "containerd://abc",
					Runtime:       "containerd",
					Workload:      inventory.ObjectRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"},
				},
			},
			RunningImages: []inventory.RunningImage{
				{
					Image:          "example/web:v1",
					PodCount:       1,
					ContainerCount: 1,
					ImageIDs:       []string{"docker-pullable://example/web@sha256:aaaaaaaaaaaaaaaa"},
					Runtimes:       []string{"containerd"},
					Namespaces:     []string{"app"},
					Workloads:      []inventory.ObjectRef{{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "app", Name: "web"}},
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
			EndpointSlices: []inventory.EndpointSlice{
				{
					ObjectRef:   inventory.ObjectRef{Kind: "EndpointSlice", Namespace: "app", Name: "web-abc"},
					ServiceName: "web",
					Endpoints:   []inventory.EndpointTarget{{Ready: &runtimeReady}},
				},
			},
			IngressClasses: []inventory.IngressClass{
				{ObjectRef: inventory.ObjectRef{Kind: "IngressClass", Name: "alb"}, Controller: "ingress.k8s.aws/alb"},
			},
			Ingresses: []inventory.Ingress{
				{
					ObjectRef: inventory.ObjectRef{Kind: "Ingress", Namespace: "app", Name: "web"},
					ClassName: "alb",
					Rules: []inventory.IngressRule{
						{Host: "example.com", Path: "/", ServiceName: "web", ServicePort: "80"},
					},
				},
			},
			GatewayClasses: []inventory.GatewayClass{{ObjectRef: inventory.ObjectRef{APIVersion: "gateway.networking.k8s.io/v1", Kind: "GatewayClass", Name: "alb"}, ControllerName: "gateway.k8s.aws/alb"}},
			Gateways: []inventory.Gateway{{
				ObjectRef: inventory.ObjectRef{APIVersion: "gateway.networking.k8s.io/v1", Kind: "Gateway", Namespace: "app", Name: "public"},
				ClassName: "alb",
				Addresses: []string{"internal-alb.example.com"},
				Listeners: []inventory.GatewayListener{{Name: "https", Protocol: "HTTPS", Port: 443, Hostname: "example.com"}},
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
			StorageClasses: []inventory.StorageClass{
				{
					ObjectRef:            inventory.ObjectRef{Kind: "StorageClass", Name: "gp3"},
					Provisioner:          "ebs.csi.aws.com",
					VolumeBindingMode:    "WaitForFirstConsumer",
					AllowVolumeExpansion: &allowExpansion,
				},
			},
			PersistentVolumes: []inventory.PersistentVolume{
				{
					ObjectRef:        inventory.ObjectRef{Kind: "PersistentVolume", Name: "pv-web-data"},
					StorageClassName: "gp3",
					AccessModes:      []string{"ReadWriteOnce"},
					VolumeMode:       "Filesystem",
					CSI:              &inventory.CSIVolume{Driver: "ebs.csi.aws.com", VolumeHandle: "vol-123"},
				},
			},
			PersistentVolumeClaims: []inventory.PersistentVolumeClaim{
				{
					ObjectRef:        inventory.ObjectRef{Kind: "PersistentVolumeClaim", Namespace: "app", Name: "web-data"},
					StorageClassName: "gp3",
					VolumeName:       "pv-web-data",
					AccessModes:      []string{"ReadWriteOnce"},
					VolumeMode:       "Filesystem",
					RequestedStorage: "10Gi",
				},
			},
			CSIDrivers: []inventory.CSIDriver{
				{ObjectRef: inventory.ObjectRef{Kind: "CSIDriver", Name: "ebs.csi.aws.com"}},
			},
		},
	}

	var out bytes.Buffer
	if err := Human(&out, snapshot); err != nil {
		t.Fatalf("Human returned error: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"KIND           NAME        CLASS/TYPE",
		"network  services=1 endpointSlices=1 ingresses=1 classes=1 gatewayClasses=1 gateways=1 gatewayRoutes=1",
		"ingress        app/web",
		"example.com/",
		"gateway-class  alb",
		"gateway        app/public",
		"httproute      app/web",
		"Gateway/app/public#https",
		"service/app/web",
		"pvc         app/web-data     ReadWriteOnce  Filesystem  10Gi",
		"csi-driver  ebs.csi.aws.com",
		"cri   containerd://2.0.0",
		"custom-resources",
		"example.com/Widget  2          2           Namespaced  v1       widgets",
		"widget/app/blue  example.com/Widget  v1       widgets.example.com",
		"running-images",
		"example/web:v1 (pods=1)  1           containerd  app         deployment/app/web  sha256:aaaaaaaaaaaa",
		"running-containers",
		"app        web-abc  web        app   example/web:v1",
		"sha256:aaaaaaaaaaaa",
		"cni   daemonset/kube-system/aws-node",
		"deployment/app/web              volume    data    pvc/app/web-data",
		"access=ReadWriteOnce mode=Filesystem sc=gp3 pv=pv-web-data csi=ebs.csi.aws.com",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("human output missing %q:\n%s", want, text)
		}
	}
}
