# Event-based Kubernetes Updates Provenance

## Trigger

Issue #7 asks Teleskope to define the module seam for informer/watch-based
Kubernetes updates before implementation. The user selected #7 as the next batch
after identifying the event/watch issues. This record is a structured summary,
not a raw transcript.

## Scope

This design record covers:

- Relationship between event-based Kubernetes sources and the existing polling
  collector.
- Watched resources versus resources that remain polling-based.
- Snapshot rebuild and resource cache ownership.
- Coverage refresh and source status behavior.
- CRD rediscovery boundaries.
- Watch reconnect and resourceVersion expiration semantics.
- Preservation of existing polling serve behavior.

Design files:

- `docs/event-based-kubernetes-updates.md`
- `docs/architecture.md`

## Conversation Summary

The next visible work after the report projection refactor is event-based
Kubernetes data freshness. The current issue set contains #7, #8, #9, #10, and
#11 for watch design, live publication, core watch implementation, dynamic
resource refresh, and UI freshness. The accepted sequence is to start with #7 so
the module seam and failure semantics are clear before implementation.

The current repository already has `internal/live.Source` for sequential polling,
`internal/k8s.Collector` for full snapshot collection, and hub remote-write over
published live responses. The design therefore keeps polling unchanged and adds a
separate watch runtime seam under `internal/k8s`, with later publication changes
owned by #10.

## Design Diff

- Added `docs/event-based-kubernetes-updates.md` to define the watch runtime
  direction, external interface shape, watched resource set, polling-only
  resource set, cache ownership, snapshot rebuild rules, coverage/status matrix,
  reconnect behavior, CRD rediscovery boundary, publication handoff, and test
  plan.
- Updated `docs/architecture.md` to link live polling architecture to the new
  event-based update design.

## Decisions

- Keep `teleskope scan k8s`, `teleskope scan eks`, and default polling `serve`
  behavior unchanged while event-based mode is introduced.
- Put the watch runtime seam under `internal/k8s` so Kubernetes-specific
  informer mechanics do not leak into `internal/live`.
- Keep `internal/live` responsible for publication, revision, source freshness,
  ETags, Advisor refresh, and hub publish hooks.
- Start watch coverage with stable, typed core resources such as Namespaces,
  Nodes, Pods, Workloads, Services, EndpointSlices, Ingresses, PVCs, PVs, and
  StorageClasses.
- Keep EKS AWS-side collection polling-based and independent from Kubernetes
  watch events.
- Defer CRDs, APIService, Gateway dynamic resources, and custom resource
  instances to #9 because dynamic informer lifecycle needs a separate bounded
  design.
- Preserve data on watch disconnects, permission failures, and resourceVersion
  expiration until a successful replacement list or rebuild is published.

## Rejected Alternatives

- Expand `internal/live.Source` into a Kubernetes-specific watch engine. This
  was rejected because it would make the live publication module know informer
  details and reduce locality.
- Replace polling collection immediately. This was rejected because #7 is a
  design issue and the current polling behavior is already useful and tested.
- Watch every discovered GVR. This was rejected because dynamic resources can be
  unbounded and require explicit discovery, permission, and cache lifecycle
  semantics.
- Treat Kubernetes Event objects as the source of truth. This was rejected
  because inventory freshness must come from list/watch object state, not
  best-effort diagnostic event records.

## Constraints

- Existing polling serve behavior must remain unchanged.
- Browser polling, future SSE, hub drilldown, and exports are read-only
  consumers of the published live response.
- Secret bodies must not be requested.
- Coverage gaps must remain visible instead of silently removing resources.
- Watch reconnects and missing watch permissions require explicit status
  behavior.
- Hub envelope revisions must remain monotonic after #10/#8.

## Evaluation Plan

- Review `docs/event-based-kubernetes-updates.md` against #7 acceptance
  criteria.
- Run repository checks even though the change is documentation-only.
- In later #10/#8 implementation, add fake watcher tests for add/update/delete,
  watch permission failures, reconnect, resourceVersion expiration, retained
  data, and publication monotonicity.

## Open Questions

None for #7. UI representation of watch freshness is intentionally left to #11,
and dynamic resource watch scope is intentionally left to #9.

## Links

- Related issue: #7
- Follow-up issues: #10, #8, #11, #9
- Design files: `docs/event-based-kubernetes-updates.md`,
  `docs/architecture.md`
