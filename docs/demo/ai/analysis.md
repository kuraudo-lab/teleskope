# Teleskope LLM analysis

Illustrative AI analysis output — hand-authored from synthetic source-demo facts; no model was called.

| Field | Value |
| --- | --- |
| Use case | compare |
| Provider | hand-authored-example |
| Model | none |
| Prompt version | demo-v1 |
| Web search | false |

## Gateway and persistent storage

- **Checkout route and storage relationships are present.** (info; observed; high)
  - Detail: HTTPRoute checkout references Gateway public and Service checkout-api. PVC orders-data is Bound to demo-orders-pv with 20Gi requested.
  - Resources: HTTPRoute/checkout/checkout, PersistentVolumeClaim/checkout/orders-data
  - Evidence: kubernetes.gatewayRoutes[0].parentRefs, kubernetes.gatewayRoutes[0].rules[0].backendRefs, kubernetes.persistentVolumeClaims[0]
- **Validate storage expansion before migration.** (warning; inferred; medium)
  - Detail: The target fixture disables expansion on demo-block; an application requiring online growth may need a different class.
  - Recommendation: Compare source and target StorageClasses, then test volume growth in a disposable environment.
  - Evidence: source.kubernetes.storageClasses[0].allowVolumeExpansion=true, target.kubernetes.storageClasses[0].allowVolumeExpansion=false

## Limitations

- Synthetic, hand-authored output example, not an actual LLM response or operational recommendation.
- Route references do not prove traffic reachability. Bound PVC status does not prove backup or restore readiness.
- EKS upgrade insights are intentionally omitted; upgrade readiness remains unknown.
