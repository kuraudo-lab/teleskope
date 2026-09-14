# Event-based Kubernetes update design

Date: 2026-09-14. Scope: design the module seam for informer/watch-based
Kubernetes updates before implementation. This is a design note, not an
implemented feature.

## Direction

Keep the existing polling collector unchanged for run-once scans and as the
default `serve` behavior. Add event-based Kubernetes freshness behind a new
watch runtime that owns list/watch clients, resource caches, reconnect state,
and debounced snapshot rebuilding.

The intended shape is:

```text
typed/dynamic watches ─> resource cache ─> snapshot builder ─┐
polling collector ───────────────────────────────────────────┤
EKS polling ─────────────────────────────────────────────────┘
                                             live publication store ─> UI / hub
```

The watch runtime should be a deep module: callers start it, receive immutable
publication candidates, and do not need to know informer lifecycle details,
resourceVersion handling, per-resource cache maps, or reconnect bookkeeping.

## Module seam

Introduce a Kubernetes watch module under `internal/k8s` rather than expanding
`internal/live.Source` to know Kubernetes-specific mechanics.

The external interface should be small:

```go
type WatchOptions struct {
    Kubeconfig string
    Context string
    Debounce time.Duration
    Progress func(format string, args ...any)
}

type WatchHealth struct {
    State string
    Coverage []inventory.CoverageItem
    LastEventAt time.Time
    LastRelistAt time.Time
    Reconnects int
    Err error
}

type WatchUpdate struct {
    Snapshot *inventory.Snapshot
    Health WatchHealth
}

type WatchRuntime interface {
    Run(ctx context.Context, publish func(WatchUpdate)) error
}
```

The concrete implementation may split internally into typed informers, dynamic
informers, a cache, and a snapshot builder, but those are internal seams. The
caller should only handle lifecycle, publication, and operational logging.

`internal/live` should later grow a publication interface that can accept both
polling attempts and watch updates. That adapter should translate `WatchHealth`
into `live.Status`; `internal/k8s` should not import `internal/live`. That work
belongs to issue #10. Issue #7 only defines the seam and status semantics.

## Relationship to polling

Polling remains authoritative for:

- `teleskope scan k8s`.
- `teleskope scan eks`.
- The default `teleskope serve k8s` and `teleskope serve eks` behavior until
  watch mode is explicitly wired.
- EKS AWS-side inventory, including cluster settings, add-ons, nodegroups,
  insights, access entries, and Pod Identity associations.
- Periodic reconciliation resources that are expensive, dynamic, or not included
  in the initial watch set.

Watch mode should reuse the existing mapping and derived-index logic wherever
possible. The watch runtime should not fork separate semantics for object refs,
workloads, running containers, running images, Gateway rows, storage rows, or
coverage. If a mapper currently only accepts a typed list item, adapt the cache
builder to call the same mapper.

## Watched resources

The first watch implementation should cover high-churn and UI-critical resources
with stable typed client-go support:

| Area | Resources | Reason |
| --- | --- | --- |
| Core identity | Namespaces, ServiceAccounts | Namespace filters and workload identity context should update quickly. |
| Core compute | Nodes, Pods | Node readiness, Pod placement, running containers, and running images are high-signal live facts. |
| Workloads | Deployments, DaemonSets, StatefulSets, ReplicaSets, Jobs, CronJobs | Workload templates and ownership drive most report relationships. |
| Networking | Services, EndpointSlices, IngressClasses, Ingresses | Routes and endpoint membership are live operational facts. |
| Storage core | PersistentVolumes, PersistentVolumeClaims, StorageClasses | Volume binding and expansion evidence should stay current. |
| Policy essentials | PodDisruptionBudgets, NetworkPolicies, ResourceQuotas, LimitRanges, HorizontalPodAutoscalers | Policy views should not require a full rescan after common changes. |

Secrets remain metadata-only. A watch implementation must never request Secret
data bodies. ConfigMaps can be watched only if the current sanitization policy is
accepted for live HTTP snapshots; otherwise they should remain polling-based or
metadata-only until a sanitization change lands.

## Polling-based resources

These resources should remain polling-based in the first event-based slice:

| Area | Resources | Reason |
| --- | --- | --- |
| Server metadata | ServerVersion, preferred APIResources | Discovery is a cluster-wide capability query, not an object stream. |
| Dynamic extensions | CustomResourceDefinitions, custom resource instances, APIServices | CRD discovery and dynamic informer lifecycle need the separate #9 design. |
| Gateway dynamic resources | GatewayClasses, Gateways, Routes, ReferenceGrants, Gateway policies | The current implementation selects versions dynamically; watch support should follow #9 or a dedicated Gateway watch slice. |
| Storage extended | CSIDrivers, CSINodes, VolumeAttachments | Lower churn and easier to reconcile with periodic lists first. |
| Runtime and admission | RuntimeClasses, MutatingWebhookConfigurations, ValidatingWebhookConfigurations | Cluster-scoped, lower churn, and acceptable as polling during the first watch runtime. |
| RBAC | Roles, RoleBindings, ClusterRoles, ClusterRoleBindings | Useful but potentially noisy; keep polling until permission/status semantics are proven on the core set. |
| EKS AWS-side | Cluster, add-ons, nodegroups, insights, access entries, Pod Identity | AWS APIs remain polling-based and independent of Kubernetes watch events. |

## Resource cache

The watch runtime owns one in-memory cache per watched resource. Cache keys use
the Kubernetes identity that already feeds `inventory.ObjectRef`:

- Cluster-scoped: `group/version/resource//name`.
- Namespaced: `group/version/resource/namespace/name`.
- Deletion removes the key only after the delete event is accepted.

The cache stores Kubernetes objects or mapped inventory objects, but the
published snapshot is always rebuilt from a cache snapshot under a lock. No HTTP
handler, hub writer, or renderer should read mutable cache state.

Derived fields such as `RunningContainers`, `RunningImages`, workload relations,
custom counts, and topology inputs are rebuilt after each debounced publication.
They are not patched incrementally by individual object handlers.

## Snapshot rebuild

Initial startup performs list-and-watch setup, waits for the watched cache set to
sync, and publishes the first usable Kubernetes snapshot. The UI is usable after
that initial list even if some watched resources are partial or polling-only
resources have not completed their next reconciliation.

Object add, update, and delete events mark the cache dirty. Dirty events enqueue
one debounced rebuild. The rebuild creates a fresh `inventory.Snapshot` with:

- `Source.Mode` set to a distinct value such as `live/watch`.
- `CollectedAt` set to the rebuild time.
- Coverage for every watched resource and every polling-based resource that is
  present in the same publication.
- Derived indexes rebuilt from the current cache state.

A quiet cluster does not become stale simply because no events arrive. Freshness
is based on watch health and last successful sync/reconnect, not object churn.

## Coverage and status semantics

Coverage remains resource-level evidence. Live source status remains source-level
health. Watch mode should use the existing state vocabulary where possible:

| Situation | Coverage | Source status | Data retention |
| --- | --- | --- | --- |
| Initial list succeeds and watch starts | `complete` for that resource | `ready` unless another resource is partial | Publish cache data. |
| Initial list is forbidden or unauthorized | `denied` for that resource | `partial` if other resources publish, otherwise `error` | Do not invent empty data. |
| Initial list times out or server is unavailable | `unavailable` for that resource | `partial` if other resources publish, otherwise `error` | Do not invent empty data. |
| Watch permission is missing after list succeeds | `partial` with reason `watch denied` | `partial` or `stale` depending on whether retained data exists | Use polling fallback only if configured. |
| Watch disconnects but reconnect is in progress | Keep last complete coverage and add watch status event | `stale` only after reconnect grace expires | Retain previous cache data. |
| resourceVersion expires (`410 Gone`) | Add status event and recovery coverage timestamp after relist | `partial` during relist, `ready` after successful relist | Retain previous data until relist succeeds. |
| Delete event is received | `complete` after rebuild | `ready` | Remove the object from cache and publish a new revision. |
| Successful empty initial list | `complete` with object count 0 | `ready` if all required resources are healthy | Publish authoritative empty list. |

Watch-specific details that the UI needs later should be separate from coverage:

- Effective mode: `poll`, `watch`, or `watch+poll`.
- Last event time.
- Last full list or relist time.
- Reconnect count.
- Last watch error.
- Resources running in polling fallback.

Issue #11 should decide how those details appear in the embedded UI and hub
drilldown.

## Reconnect and resourceVersion behavior

Use client-go informer or reflector recovery semantics rather than raw watch HTTP
loops. The implementation must handle:

- Initial list before watch.
- `HasSynced` before first publication for watched resources.
- Backoff on transport failures.
- Fresh list after `410 Gone` or equivalent resourceVersion expiration.
- Explicit status events for reconnect start, reconnect success, and reconnect
  failure.

Reconnects must not clear retained data. A watch failure changes health and may
change the live response ETag through events/status, but it should not advance the
inventory revision unless a rebuilt snapshot with usable data is published.

## CRD rediscovery

CRD and dynamic resource support is intentionally deferred to issue #9.

The #7 contract for CRD rediscovery is:

- The first watch slice keeps CRDs, APIService, Gateway dynamic resources, and
  custom resource instances on polling.
- A CRD change must eventually trigger discovery refresh before dynamic watches
  are added.
- If discovery refresh fails, existing custom-resource evidence remains visible
  as retained data with degraded coverage rather than disappearing silently.
- Dynamic informer lifetimes are bounded to selected resources; Teleskope should
  not automatically watch every discovered GVR.

## Publication handoff

Issue #10 should extract live publication semantics before #8 connects a watch
runtime. The extracted publication path should guarantee:

- Kubernetes watch events do not trigger EKS AWS collection.
- EKS polling updates do not overwrite newer Kubernetes watch cache state.
- Hub envelope revisions remain monotonic.
- Failed or stale watch updates retain prior evidence.
- Browser polling and future SSE remain read-only consumers of the shared live
  view.

The watch runtime should call that publication path with immutable snapshot
candidates. It should not write directly to hub remote-write, Advisor, report
renderers, or HTTP handlers.

## Test plan

Implementation should add tests at the module seam:

- A fake watcher publishes add, update, and delete events and advances live
  revision through a debounced rebuild.
- Missing watch permission after a successful list records partial coverage and
  does not clear retained data.
- A simulated `410 Gone` triggers relist and preserves the last snapshot until
  relist succeeds.
- EKS polling and Kubernetes watch updates merge through the same publication
  path without overwriting each other.
- Hub publish hooks see monotonic revisions only after usable publications.
- The current polling `serve` tests remain unchanged unless a new explicit
  watch-mode flag is introduced.
