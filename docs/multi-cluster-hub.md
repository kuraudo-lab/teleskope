# Multi-cluster hub design

Date: 2026-09-10. Scope: define the intended multi-cluster shape before adding
in-cluster probes. This is a design note, not an implemented feature.

## Direction

Teleskope should keep `serve` scoped to a single cluster and add a separate hub
mode for multi-cluster aggregation.

The process model is:

```text
cluster A teleskope serve/probe ─┐
cluster B teleskope serve/probe ─┼─ remote write ─> teleskope serve-hub ─> web UI
cluster C teleskope serve/probe ─┘
```

Each cluster has its own collector process. That process owns the credentials,
network path, retry policy, and collection state for one cluster. The hub receives
published reports from those processes and presents a multi-cluster view.

This keeps the multi-cluster seam at the report envelope instead of spreading
multi-cluster scheduling, authentication, and failure handling through the
collectors.

## Command shape

`serve` remains single-cluster:

```bash
teleskope serve k8s --kube-context prod-a
teleskope serve eks --cluster prod-a --kube-context prod-a
```

When remote write is enabled, that single-cluster process also publishes its
latest report to a hub:

```bash
teleskope serve k8s --kube-context prod-a --hub-url https://hub.example
teleskope serve eks --cluster prod-a --kube-context prod-a --hub-url https://hub.example
```

The hub address can also come from an environment variable. Do not introduce a
configuration file for this first version.

The hub is a new startup mode:

```bash
teleskope serve-hub
```

`serve-hub` owns the multi-cluster web UI and report store. It should not require
direct Kubernetes or AWS credentials for the clusters it displays.

## Data interface

The central interface should be a cluster report envelope. A first version can be:

```json
{
  "cluster": {
    "id": "prod-a",
    "name": "prod-a",
    "provider": "eks",
    "region": "ap-northeast-1"
  },
  "revision": 42,
  "collectedAt": "2026-09-10T00:00:00Z",
  "snapshot": {},
  "advisor": {},
  "sources": {},
  "events": []
}
```

`cluster.id` is the stable key used by the hub. It must not depend only on a local
kubeconfig context name, because context names are machine-local labels. For EKS,
the cluster ARN is a good source for a stable identity. For generic Kubernetes,
the first version can accept an explicit `--cluster-id`, with a documented fallback
derived from the current context/server when the user does not provide one.

The existing single-cluster snapshot remains useful. The envelope wraps it with
identity, revision, freshness, source health, and event data needed by the hub.

## Remote write

Remote write should publish completed report revisions from the single-cluster
process to the hub. Opening browsers against either side must not trigger scans.

Initial behavior:

- Push the latest complete envelope after each successful or partially successful
  collection attempt.
- Include source state so the hub can show fresh, stale, partial, and failed
  cluster states without reinterpreting collector internals.
- Retry failed writes without blocking local collection.
- Treat remote write as best-effort in the first version: local `serve` continues
  to work if the hub is unavailable.
- Keep authentication minimal at first, but leave the HTTP seam capable of adding
  a token later.

The hub should accept idempotent writes by `(cluster.id, revision)` or an
equivalent monotonic revision key. If the same revision is received again, it
should not create duplicate events or duplicate history entries.

## Hub responsibilities

`serve-hub` should:

- Receive remote writes from one or more single-cluster Teleskope processes.
- Store the latest envelope per cluster in memory for the first implementation.
- Expose a web UI based on the current embedded report UI where practical.
- Provide a cluster selector for single-cluster drilldown.
- Provide aggregated views for fleet-level inventory and health.
- Export data as JSON or Markdown in a format compatible with existing scan
  reports, extended to represent multiple clusters.

The first hub does not need durable storage. Restarting the hub can clear live
state. Durable history can be added later behind the same report-envelope seam.

## UI direction

Reuse the current report UI rather than creating a separate frontend. The current
sections map well to a selected-cluster view. The hub should add:

- A cluster selector.
- An aggregated overview across all clusters.
- Aggregated health/source status.
- Cluster-scoped Events, with the option to view all clusters' events together.

The selected-cluster path should preserve the existing report navigation as much
as possible. Aggregated views should start small: counts, freshness, source
states, advisor summaries, and high-signal inventory summaries are enough for the
first version.

## Explicit non-goals for the first version

- Do not make `serve` handle multiple clusters directly.
- Do not add a configuration file.
- Do not require the hub to connect to Kubernetes API servers or AWS APIs.
- Do not introduce watch/informer mode as part of the hub design.
- Do not build durable storage or historical analytics before the live envelope
  path is working.

## Implementation order

1. Define the cluster report envelope and stable cluster identity rules.
2. Adapt the live state model so it can store envelopes keyed by cluster id.
3. Add remote write flags and environment variable handling to single-cluster
   `serve`.
4. Add `serve-hub` with an in-memory envelope store and remote-write endpoint.
5. Extend the embedded UI with a cluster selector and minimal aggregated views.
6. Extend JSON and Markdown export to support a multi-cluster bundle while keeping
   single-cluster scan output compatible.

## Open questions

- What should the environment variable name be for the hub URL?
- Should hub writes require a shared token in the first implementation, or should
  local/private-network deployments be the initial assumption?
- How much event history should the hub retain per cluster in memory?
