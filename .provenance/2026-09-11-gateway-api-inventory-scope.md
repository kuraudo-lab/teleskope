# Gateway API Inventory Scope Provenance

## Trigger

The user clarified that teleskope must correctly collect the Gateway API surface needed for migration and capability review, including GatewayClasses, Gateways, HTTPRoutes, GRPCRoutes, TCPRoutes, UDPRoutes, ReferenceGrants, BackendTLSPolicies, and BackendTrafficPolicies.

## Scope

This record covers Kubernetes Gateway API inventory collection, snapshot schema additions, live serve partial-data detection, and report/UI rendering for the clarified resource set.

Relevant implementation surfaces:

- `internal/k8s/collector.go`
- `internal/inventory/snapshot.go`
- `internal/cli/serve.go`
- `internal/render/human.go`
- `internal/report/report.go`
- `internal/report/report.html`
- `internal/k8s/collector_test.go`

## Conversation Summary

The user first reported that teleskope could not correctly collect HTTPRoute. Investigation found that Gateway API discovery fallback was applied only when the whole `gateway.networking.k8s.io` group was absent. If discovery returned the group but omitted a specific resource such as `httproutes`, the collector skipped that resource silently.

After the initial HTTPRoute fallback fix, the user clarified the intended Gateway API inventory scope. The collector should include core Gateway API resources and the related cross-namespace grant and backend policy objects so scan, serve, and report outputs can support migration planning.

## Design Diff

No standalone design document was edited before implementation. The intended design is captured here as structured provenance:

- Treat Gateway API collection as a resource-family collector rather than only GatewayClass/Gateway/Route collection.
- Keep existing `gatewayRoutes` for HTTPRoute, GRPCRoute, TCPRoute, and UDPRoute objects because they share parent/backend routing relationships.
- Add dedicated `referenceGrants` inventory for ReferenceGrant `from`/`to` relationships.
- Add generic `gatewayPolicies` inventory for backend policy attachments, preserving actual `apiVersion` and `kind` so standard and experimental Gateway API policy names can coexist.
- Surface new inventory in terminal, markdown, and web reports.

## Decisions

- Continue using dynamic client collection for Gateway API resources because these resources can vary by installed CRD version.
- Fallback by specific resource when discovery is incomplete, instead of treating group discovery as proof that all resources were fully discovered.
- Store BackendTLSPolicy and BackendTrafficPolicy-style objects in one `gatewayPolicies` collection with actual kind/apiVersion preserved.
- Probe both standard-looking `backendtrafficpolicies.gateway.networking.k8s.io` and upstream experimental `xbackendtrafficpolicies.gateway.networking.x-k8s.io` names to tolerate implementation differences.
- Keep ReferenceGrant as its own collection because it represents cross-namespace permission relationships rather than traffic routing.

## Rejected Alternatives

- Do not rely only on CRD browsing for these resources, because they need first-class fields in scan JSON and report views.
- Do not split each Route kind into a separate top-level collection yet, because existing consumers already use `gatewayRoutes` and the route types share common shape.
- Do not add an LLM advisory layer for this change; the immediate requirement is deterministic collection and display.

## Constraints

- Preserve existing JSON compatibility by only adding optional fields.
- Continue to support clusters with incomplete API discovery or missing Gateway API CRDs.
- Treat NotFound responses during fallback probing as absence, not collection failure.
- Preserve partial live serve behavior so any successfully collected Gateway API resource can be published.
- Keep tests deterministic with fake dynamic client list-kind registration for all probed resources.

## Evaluation Plan

- Add focused collector tests for incomplete HTTPRoute discovery fallback.
- Add collector tests for ReferenceGrant, BackendTLSPolicy, and XBackendTrafficPolicy mapping.
- Run Gateway collector tests with `go test ./internal/k8s -run TestCollectGateways`.
- Run the full Go test suite with `go test ./...`.
- Run `git diff --check` before committing implementation.

## Open Questions

- Whether to add richer, typed policy detail fields later instead of the current generic policy `details` summary.
- Whether to represent BackendTrafficPolicy under a stable non-experimental kind once Gateway API finalizes that object upstream.

## Links

- Design files: `.provenance/2026-09-11-gateway-api-inventory-scope.md`
- Related issues/tasks: user-reported HTTPRoute collection gap and clarified Gateway API inventory scope in this Codex task.
