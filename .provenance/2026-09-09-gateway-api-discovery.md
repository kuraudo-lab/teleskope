# Gateway API Discovery Provenance

## Trigger

The user reported that teleskope could not collect Gateway API data and identified
the affected resource family as Gateway and HTTPRoute under
`gateway.networking.k8s.io/v1`.

This provenance record is retrospective: the implementation had already been
made before this file was created, because the provenance request came after the
fix.

## Scope

This record covers Kubernetes collection of Gateway API resources in
`internal/k8s/collector.go`, plus focused tests in
`internal/k8s/collector_test.go`.

## Conversation Summary

The existing collector already had Gateway API mapping for GatewayClass, Gateway,
HTTPRoute, GRPCRoute, TLSRoute, TCPRoute, and UDPRoute. The likely failure mode
was not missing output schema, but how the collector selected and listed fixed
Gateway API GVRs against real clusters.

The fix should prefer discovered `gateway.networking.k8s.io/v1` resources when
the API server reports them, keep compatibility with older Gateway API versions,
and handle restricted read-only credentials that can list only the current
namespace.

## Design Diff

- `internal/k8s/collector.go`: retain the kube context namespace in the
  reusable collector.
- `internal/k8s/collector.go`: choose Gateway API GVR candidates from discovered
  API resources, preferring `gateway.networking.k8s.io/v1`.
- `internal/k8s/collector.go`: fall back to the current context namespace for
  namespaced Gateway API resources when all-namespace list is denied.
- `internal/k8s/collector_test.go`: add tests for discovered v1 Gateway API
  collection and context-namespace fallback.

## Decisions

- Use Kubernetes discovery as the source of truth for Gateway API availability,
  because installed Gateway API resources and served versions vary by cluster.
- Keep the existing fixed-version fallback only when the Gateway API group is not
  discovered, preserving behavior for partial discovery failures.
- Limit namespace fallback to namespaced resources and only on forbidden or
  unauthorized all-namespace list errors, so real API failures still surface in
  coverage.

## Rejected Alternatives

- Hard-code only `gateway.networking.k8s.io/v1`; this would break older clusters
  that still serve Gateway API through `v1beta1` or route extensions through
  `v1alpha2`.
- Treat every list failure as resource absence; this would hide RBAC and server
  errors that should remain visible in coverage.
- Add report-layer changes; the existing inventory and report model already had
  Gateway API fields, so the bug belonged in collection.

## Constraints

- The collector must remain read-only.
- Gateway API collection must work in both one-shot scans and the live server's
  repeated collector reuse path.
- Restricted kubeconfigs may lack cluster-wide list permissions for namespaced
  Gateway API resources.
- Existing Gateway API output shape and older-version compatibility should be
  preserved.

## Evaluation Plan

- Run focused Kubernetes collector tests.
- Run the full Go test suite.
- Verify staged implementation diff only includes Gateway API collection and
  tests.

## Open Questions

- None.

## Links

- Design files: none
- Implementation files: `internal/k8s/collector.go`,
  `internal/k8s/collector_test.go`
- Related issues/tasks: user-reported Gateway API collection gap
