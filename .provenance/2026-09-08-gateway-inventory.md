# Gateway Inventory Provenance

## Trigger

The user requested extending the Kubernetes inventory scanner to support collecting Gateway information. This follows earlier work that made the scanner collect EKS and Kubernetes details, save raw JSON data and summaries, and render routing/topology information in human-readable, Markdown, and HTML outputs.

## Scope

This record covers Gateway API inventory collection and presentation across:

- `internal/inventory/snapshot.go`
- `internal/k8s/collector.go`
- `internal/k8s/collector_test.go`
- `internal/render/human.go`
- `internal/render/human_test.go`
- `internal/report/report.go`
- `internal/report/report.html`
- `internal/report/report_test.go`

## Conversation Summary

The scanner already collected Services, EndpointSlices, IngressClasses, Ingresses, storage, runtime, workloads, CRDs, running images, EKS-specific details, and rendered topology in static HTML. The new requirement is to include Gateway API resources so the report can show whether the cluster exposes L7/L4 traffic through Gateway API and how routes connect to Gateways and backend Services.

## Design Diff

The implementation should extend the inventory snapshot with Gateway API structures, collect them through the Kubernetes dynamic client, and render them alongside existing routing data. Gateway data should be included in raw JSON output because the default report behavior persists the snapshot. Human-readable, Markdown, and HTML outputs should show GatewayClass, Gateway, and Gateway route information in useful routing tables and topology views.

## Decisions

- Use the Kubernetes dynamic client for Gateway API resources because Gateway API objects are CRD-backed and not available through the typed client-go Kubernetes clientset.
- Collect `GatewayClass`, `Gateway`, `HTTPRoute`, `GRPCRoute`, `TLSRoute`, `TCPRoute`, and `UDPRoute` as first-class inventory data.
- Prefer `gateway.networking.k8s.io/v1` for stable resources and fall back to `v1beta1` for GatewayClass, Gateway, HTTPRoute, and GRPCRoute so older installed Gateway API versions are not silently missed.
- Keep TLSRoute, TCPRoute, and UDPRoute collection on `v1alpha2` because these route types are commonly served at that version in Gateway API installations.
- Capture Gateway listener names, protocol, port, hostname, allowed route constraints, route parentRefs, hostnames, matches, and backendRefs because these fields expose the capability and topology relationships that matter for migration assessment.
- Capture Gateway addresses from both `spec.addresses` and `status.addresses` so the report includes configured and assigned entry points.
- Treat missing Gateway API resources as zero collected objects, while permission or API failures remain coverage errors.

## Rejected Alternatives

- Do not shell out to `kubectl get gateway...`; the scanner already has a dynamic Kubernetes client and should keep the collection path inside Go.
- Do not fold Gateway API objects into generic CRD instance output only; route/gateway relationships need dedicated fields for readable reporting and topology rendering.
- Do not require Gateway API v1 only; that would miss clusters that still serve v1beta1.

## Constraints

- Secret values remain redacted; this change does not add any Secret data collection.
- Collection must work out-of-cluster using the same kubeconfig/client setup as the existing Kubernetes scanner.
- Output must remain useful in terminal, Markdown, and static HTML report modes.
- A cluster without Gateway API installed should not fail the whole scan solely because Gateway API resources are missing.
- A Kubernetes cluster connection failure should still be treated as an error by the existing scanner behavior.

## Evaluation Plan

- Run `go test ./...` to verify collector, renderer, and report behavior.
- Run `go vet ./...` through `make check`.
- Run `make build` to verify the CLI still builds.
- Extract and syntax-check the embedded HTML report JavaScript with `node --check`.
- Add tests covering Gateway API v1beta1 fallback, Gateway listener parsing, HTTPRoute parentRef parsing, backendRef parsing, and Gateway rendering in human-readable, Markdown, and HTML outputs.

## Open Questions

- Whether future versions should collect route status conditions and listener status conditions for deeper readiness and policy analysis.

## Links

- Design files: none
- Related issues/tasks: local conversation request
