# Teleskope scan summary

## Advisor

0 Ingress classes, 1 Gateway classes, 1 StorageClasses, 0 custom API definitions, 0 EKS insights, 1 EKS managed add-ons and 1 EKS managed nodegroups observed. Conclusions describe configuration and collected evidence; runtime behavior is not verified. RWX bindings observed in 0 StorageClasses; 4 assessments remain unknown and 8 have incomplete coverage.

- **eks.addon / aws-ebs-csi-driver**: supported (observed; snapshot; coverage: partial). Managed add-on is active and has no recorded health issues.
- **eks.insight / cluster**: unknown (none; snapshot; coverage: partial). No EKS upgrade or readiness insights were observed in the snapshot.
- **eks.nodegroup / demo-workers**: supported (observed; snapshot; coverage: partial). Managed nodegroup has 1 observed Kubernetes node(s).
- **extensions.api / cluster**: unknown (none; snapshot; coverage: partial). No custom API definitions observed.
- **networking.gateway / demo-gateway**: supported (declared; snapshot; coverage: partial). Gateway class configured.
- **networking.ingress / cluster**: unknown (none; snapshot; coverage: partial). No Ingress class observed.
- **storage.expansion / demo-block**: unsupported (declared; snapshot; coverage: partial). StorageClass does not allow volume expansion.
- **storage.rwx / demo-block**: unknown (none; snapshot; coverage: partial). RWX support is unknown.

- Target: `target-demo`
- Collected at: `2026-09-25T00:00:00Z`
- Schema: `teleskope.io/snapshot/v1alpha1`
- Mode: `synthetic/offline`
- Coverage items: `14`

## EKS

| Field | Value |
| --- | --- |
| Cluster | target-demo |
| Region | eu-west-1 |
| Version | 1.35 |
| Platform | - |
| Status | ACTIVE |
| ARN | arn:aws:eks:eu-west-1:000000000000:cluster/target-demo |
| Endpoint | public=false private=false |
| Auth | - bootstrapCreatorAdmin=- |
| Control plane logs | enabled= disabled= |
| Auto mode | compute=- nodePools= blockStorage=- |

### EKS compute capacity and declared usage

| Resource | Capacity | Allocatable | Requested by specs | Limits by specs |
| --- | ---: | ---: | ---: | ---: |
| CPU | 2 cores | 1900m | - | - |
| Memory | 8.0 GiB | 7.0 GiB | - | - |
| Pods | 29 | 29 | - | - |

> Requested/limits are derived from Pod and workload specs; live metrics-server usage is not collected yet.

### EKS network

| Field | Value |
| --- | --- |
| VPC | - |
| Cluster subnets | - |
| Cluster security groups | - |
| Cluster security group | - |
| Public access CIDRs | - |
| IP family | - |
| Service CIDR | - |
| Auto Mode load balancing | - |
| Nodegroup subnets | demo-workers: |

### Managed add-ons

| Name | Version | Status | Namespace | Target Kubernetes | Compatible versions | Upgrade | IAM | Issues |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| aws-ebs-csi-driver | demo | ACTIVE | - | - | - | - | - | - |

### Managed nodegroups

| Name | Version | Release | Status | AMI | Capacity | Size | Subnets | IAM | Issues |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| demo-workers | 1.35 | - | ACTIVE | - | m7i.large | desired=1 min=1 max=3 | - | - | - |

### Nodegroup runtime readiness

| Group | Evidence | AZ | Expected version | Expected release | Expected AMI | Launch template | Nodes | Kubelet | OS image | Runtime | Instance types | Readiness |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| demo-workers | managed-nodegroup | - | 1.35 | - | - | - | demo-node | v1.35.0 | - | containerd://2.0.0 | - | observed: runtime evidence linked to managed nodegroup |

## Kubernetes

| Area | Count |
| --- | ---: |
| API resources | 0 |
| CRDs | 0 |
| CRD instances | 0 |
| API services | 0 |
| Namespaces | 3 |
| Nodes | 1 |
| Workloads | 2 |
| Pods | 2 |
| Running images | 2 |
| Running containers | 0 |
| Services | 1 |
| Ingresses | 0 |
| GatewayClasses | 1 |
| Gateways | 1 |
| Gateway routes | 1 |
| Admission webhooks | 0 |
| StorageClasses | 1 |
| PVCs | 1 |
| PVs | 1 |
| CSIDrivers | 1 |
| RBAC role details | 0 |
| RBAC binding details | 0 |
| NetworkPolicy details | 0 |
| ResourceQuota details | 0 |
| LimitRange details | 0 |
| PDB details | 0 |
| RuntimeClasses | 0 |

### Nodes

| Node | ProviderID | Ready | Schedulable | Kubelet | Runtime | OS | Kernel | Arch | Capacity | Allocatable | Taints | Key labels | Blind spots |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| demo-node | - | True | true | v1.35.0 | containerd://2.0.0 | - | - | - | cpu=2,memory=8Gi,pods=29 | cpu=1900m,memory=7Gi,pods=29 | - | eks.amazonaws.com/nodegroup=demo-workers | - |

### Routing

| Kind | Name | Class/Type | Route/Selector | Target/Ports |
| --- | --- | --- | --- | --- |
| GatewayClass | demo-gateway | - | - | controller=example.invalid/demo-controller |
| Gateway | gateway/checkout/public | demo-gateway | listeners=http:HTTP/80 host=shop.example.invalid allowed= | addresses=- |
| HTTPRoute | httproute/checkout/checkout | PathPrefix / | hosts=shop.example.invalid parents=Gateway/checkout/public#http | backends=service/checkout/checkout-api |
| Service | service/checkout/checkout-api | ClusterIP | app=checkout-api | TCP/80->8080 |

### Storage

| Kind | Name | Access | Mode | Size | Ref | CSI/Provisioner | Status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| StorageClass | demo-block | - | - | - | - | ebs.csi.aws.com | binding=WaitForFirstConsumer reclaim=Retain expand=false |
| PVC | persistentvolumeclaim/checkout/orders-data | ReadWriteOnce | Filesystem | 20Gi | demo-block -> demo-orders-pv | ebs.csi.aws.com | Bound |
| PV | demo-orders-pv | ReadWriteOnce | Filesystem | 20Gi | demo-block -> persistentvolumeclaim/checkout/orders-data | ebs.csi.aws.com | Bound |
| CSIDriver | ebs.csi.aws.com | - | - | - | - | ebs.csi.aws.com | attachRequired=true podInfoOnMount=- lifecycle=Persistent |

### Runtime

| Kind | Name | Runtime/Handler | Kubelet | OS/Arch | Users |
| --- | --- | --- | --- | --- | --- |
| CRI | containerd://2.0.0 | containerd://2.0.0 | - | - | nodes=1 |
| Node | demo-node | containerd://2.0.0 | v1.35.0 | -/- | - |

### Running images

| Image | Containers | Runtimes | Namespaces | Workloads | Image IDs |
| --- | ---: | --- | --- | --- | --- |
| registry.example.invalid/demo/api:v1 (pods=1) | 1 | - | checkout | deployment/checkout/checkout-api | - |
| registry.example.invalid/demo/db:v1 (pods=1) | 1 | - | checkout | statefulset/checkout/orders | - |

### Workload topology

| Workload | Images | ServiceAccount | RuntimeClass | Relations |
| --- | --- | --- | --- | --- |
| deployment/checkout/checkout-api | registry.example.invalid/demo/api:v1 | - | - | - |
| statefulset/checkout/orders | registry.example.invalid/demo/db:v1 | - | - | volume/data -> pvc/orders-data |

## Coverage

| Status | Area | Resource | Objects | Reason |
| --- | --- | --- | ---: | --- |
| complete | kubernetes | nodes | 1 | - |
| complete | kubernetes | namespaces | 3 | - |
| complete | kubernetes | deployments | 1 | - |
| complete | kubernetes | statefulsets | 1 | - |
| complete | kubernetes | pods | 2 | - |
| complete | kubernetes | services | 1 | - |
| complete | kubernetes | gatewayclasses | 1 | - |
| complete | kubernetes | gateways | 1 | - |
| complete | kubernetes | httproutes | 1 | - |
| complete | kubernetes | storageclasses | 1 | - |
| complete | kubernetes | persistentvolumes | 1 | - |
| complete | kubernetes | persistentvolumeclaims | 1 | - |
| complete | kubernetes | csidrivers | 1 | - |
| skipped | eks | eks.insight | 0 | Synthetic fixture intentionally omits upgrade insights; readiness is unknown. |
