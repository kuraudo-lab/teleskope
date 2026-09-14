# Dynamic Resource Watch Refresh Provenance

## Trigger

The user accepted the issue #9 implementation and asked to create a provenance commit followed by an implementation commit whose body contains `Fixed #9`.

## Scope

This record covers event-based live refresh for Kubernetes dynamic resources:

- `internal/k8s/watch.go`: dynamic client and dynamic informer integration for watch-triggered refresh.
- `internal/k8s/collector.go`: discovery fallback for environments where preferred discovery returns no resources.
- `internal/k8s/watch_test.go`: regression coverage for CRD, Gateway API, and permission-gap behavior.

## Conversation Summary

Issue #9 asks the live watch mode to keep Gateway API and custom resource visibility reliable. The existing watch runtime handled typed core resources but did not react to CRD, APIService, or Gateway API changes. Existing polling collectors already knew how to list and map CRDs, custom resource instances, APIService objects, and Gateway API resources with useful coverage records.

The implementation direction was to add dynamic informer wiring for event triggers while preserving the existing dynamic list and mapping functions as the source of snapshot data. That keeps unknown CRDs and permission gaps visible through coverage instead of silently dropping them.

## Design Diff

- Watch runtime now creates a dynamic client when constructed from kubeconfig.
- Tests can inject typed and dynamic clients through `NewWatchRuntimeWithClients`.
- Dynamic informers watch CRD, APIService, and discovered Gateway API GVRs.
- Dynamic informers trigger publication but do not become required core cache-sync gates.
- Each watch publication refreshes API discovery, extension data, CRD instances, and Gateway API data through existing collector helpers.
- API resource discovery falls back from preferred resources to all group resources when the preferred list is empty.

## Decisions

- Use dynamic informers for change notification and reuse existing collector helpers for data projection.
- Keep CRD/APIService/Gateway permission failures represented as coverage records.
- Avoid making dynamic informer sync mandatory for core-resource watch readiness, so RBAC gaps do not block live mode entirely.
- Discover Gateway API watch GVRs from API resources instead of hard-coding only one version.
- Preserve the existing debounce and publication path introduced for core-resource watch mode.

## Rejected Alternatives

- Fully mapping Gateway API objects from informer stores was rejected because it would duplicate existing collector logic and coverage fallback behavior.
- Watching every discovered CRD instance type directly was deferred because arbitrary CRD watch fan-out can be large; issue #9 can be satisfied by CRD/APIService/Gateway event triggers plus dynamic list refresh.
- Failing the whole watch update when discovery is partial was rejected because partial coverage is more useful than losing all current data.

## Constraints

- Event-based watch mode must retain the existing snapshot schema.
- Permission gaps must remain visible as coverage.
- Gateway API versions vary by cluster, so discovery must drive watch registration.
- Existing core-resource watch behavior and tests must continue to pass.
- The change should remain compatible with embedded UI and hub publication paths.

## Evaluation Plan

- Run `go test ./internal/k8s`.
- Run full `make check`.
- Run `make build`.
- Run `git diff --check`.
- Manually verify with `bin/teleskope serve k8s --watch` against a cluster with CRDs or Gateway API resources by creating/deleting a CRD or Gateway object and observing live refresh.

## Open Questions

- None.

## Links

- Related issue: #9
- Design files: `internal/k8s/watch.go`, `internal/k8s/collector.go`
