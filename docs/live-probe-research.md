# Live probe research

Date: 2026-09-08. Scope: evolve the existing read-only snapshot collector into a
single Go process that supports local and in-cluster execution. This is a proposal,
not an implemented feature.

## Recommendation

Build the embedded web server and polling lifecycle first, then add informer-based
Kubernetes watches behind the same snapshot interface. Deployment location and
refresh strategy should be independent choices. Keep AWS enrichment on its own
polling schedule, including when Kubernetes uses watches.

The proposed process boundary is:

```text
Kubernetes polling / informer cache ─┐
                                   ├─ snapshot builder ─ immutable snapshot store
AWS polling ────────────────────────┘                           │
                                                   HTTP + embedded UI
```

Every browser reads the shared store. Opening more tabs must not create additional
collectors or Kubernetes watches. Publish completed snapshot revisions atomically;
HTTP handlers never observe a snapshot while a builder is mutating it.

## Current foundation

- `internal/report/html.go` already embeds `report.html` and injects snapshot JSON.
  Reuse its read-only interaction model and support both embedded offline data and
  a live snapshot endpoint; a separate frontend service/build system is unnecessary.
- `internal/k8s/collector.go` currently creates clients and performs sequential
  resource lists per collection, including custom resources discovered via CRDs.
  Polling can initially reuse collection logic, but clients should become long-lived.
- `loadRESTConfig` already uses deferred client-go loading. This is not proof of
  missing in-cluster support: that loader supports in-cluster fallback when the
  loaded configuration is empty/default. Invalid explicit configuration remains
  an error. `RawConfig` itself does not manufacture a kubeconfig context for the
  ServiceAccount path. Make the selected credential source visible and test it.
  [client-go configuration source](https://github.com/kubernetes/client-go/blob/master/tools/clientcmd/merged_client_builder.go)
- Secret collection uses the metadata client. Preserve that boundary in watches.
  ConfigMap data is currently retained; define sanitization before exposing live
  snapshots through HTTP. The current `denied` helper labels general list failures
  as denied; distinguish authorization errors, timeouts, and service failures.

## Embedded UI and serving

Use `embed.FS`, `net/http`, and the existing vanilla JavaScript UI. Go officially
supports serving embedded files and parsing templates directly from an embedded
filesystem. [Go embed](https://pkg.go.dev/embed),
[HTTP file serving](https://pkg.go.dev/net/http#FileServerFS)

Start with a same-origin `GET /api/snapshot` and a lightweight revision/status
endpoint or conditional requests. Browser polling is sufficient initially; later,
SSE can notify browsers that a newer revision is available. Neither browser polling
nor SSE determines how the server refreshes Kubernetes data. Preserve search,
filters, expanded rows, and selected topology nodes across updates.

Keep the initial server local by default. For a cluster deployment, use one
Deployment with a ServiceAccount and explicit read-only RBAC, accessed through
port-forward initially. ServiceAccount credentials are available to in-cluster API
clients; they do not replace AWS credentials for EKS enrichment.
[Kubernetes API access from a Pod](https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/)

## Polling versus watching

| Aspect | Polling | Resource watches / informers |
| --- | --- | --- |
| Fit with current code | Direct reuse of run-once collection | Requires maintained caches and change-driven rebuilding |
| Freshness | Bounded by interval plus collection time | Usually faster, subject to connectivity and rebuilding |
| Cost | Repeated full lists | Initial state plus streams; cache memory and per-resource watches |
| Permissions | Existing list/get scope | Adds watch permission for selected resources |
| Failure behavior | Retry next cycle with backoff | Reconnect, recovery lists, and cache lifecycle |

These are architectural tradeoffs, not measured benchmarks for this repository.

Recommended polling defaults are configurable intervals, jitter, request deadlines,
and at most one active collection per source. Do not queue overlapping full scans.
Start with a conservative interval such as 5 minutes, then tune using cluster size,
scan duration, throttling, and freshness requirements.

For watch mode, use client-go shared informers for built-in resources, dynamic
informers for selected custom resources, and metadata-only handling for Secrets.
Handlers should mark the view dirty and enqueue a debounced snapshot rebuild from
cache; do not launch a complete cluster scan for every Pod update. Informer caches
are eventually consistent, and ordering is not guaranteed across different
objects. The published graph is therefore not a cross-resource transactional view.
[client-go informer contract](https://raw.githubusercontent.com/kubernetes/client-go/v0.37.0/tools/cache/shared_informer.go)

Critical distinctions:

- Watch resource additions, modifications, and deletions. Kubernetes `Event`
  objects are best-effort diagnostic records and cannot be the inventory's source
  of truth. [Kubernetes Event API](https://kubernetes.io/docs/reference/kubernetes-api/events/event-v1/)
- A disconnected watch must recover. An expired resource version can produce
  `410 Gone`, requiring a fresh list and restarted watch. Use client-go recovery
  machinery rather than implementing raw watch transport.
  [Kubernetes API watch semantics](https://kubernetes.io/docs/reference/using-api/api-concepts/)
- Informer `resyncPeriod` replays objects already in the local cache to handlers;
  it does **not** periodically fetch fresh data from the API server. If deliberate
  periodic reconciliation lists are wanted, specify them as a separate policy.
  [client-go SharedInformer](https://pkg.go.dev/k8s.io/client-go/tools/cache#SharedInformer)
- Track watch errors and recovery separately from object changes. A quiet cluster
  is not stale merely because no objects changed. Initial `HasSynced` also does
  not establish that a connection remains healthy forever.
- Manage CRD additions/removals and corresponding informer lifetimes. Bound the
  resources watched; watching every discovered type has API and memory costs.
- If a resource has list permission but lacks watch permission, either use an
  explicitly configured polling fallback or expose that resource as degraded.

## Snapshot state and acceptance criteria

Keep data and collection health separate. Include revision, source identity,
collection timestamps, coverage, last successful refresh, last error, and effective
refresh mode per source/resource. On failure, retain the last successful data with
a stale/degraded marker; only a successful empty list establishes that the listed
resource set is empty. Avoid retaining stale derived relationships as current facts.

Acceptance checks for implementation:

1. Both kubeconfig and in-cluster ServiceAccount modes work with documented RBAC.
2. Failed polls retain marked old data; successful empty lists remove old objects.
3. Multiple browsers do not multiply cluster collection, and UI choices survive refresh.
4. Watch disconnects recover; missing watch permission has an explicit outcome.
5. CRD additions/removals update coverage and custom-resource collection correctly.
6. Secret bodies are never requested; HTTP snapshots respect sanitization policy.
7. Idle watches remain distinguishable from failed watches, and failed optional
   resource types do not block useful partial inventory indefinitely.

Suggested delivery order: shared snapshot store and embedded HTTP UI; polling and
explicit credential/deployment behavior; then watch caches, CRD lifecycle, recovery,
and optional browser push. Preserve existing run-once and offline report commands.
