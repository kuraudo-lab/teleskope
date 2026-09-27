# Topology Graph And Evidence Contract

This document defines the provider-neutral topology contract shared by recorded
snapshots, live single-cluster responses, and hub envelopes. It is the boundary
between collected inventory evidence and interactive topology presentation.

## Contract Goals

- Give every node and edge a stable identity that is independent of display
  layout.
- Keep relationship meaning explicit instead of collapsing all links into one
  visual edge type.
- Preserve the origin, time window, freshness, confidence, and coverage of
  optional evidence.
- Allow the same graph document to travel through recorded, live, and hub
  paths.
- Keep declared and resolved Kubernetes/AWS relationships useful when no
  runtime connection provider is configured.

## Graph Document

A graph document has a schema version, cluster identity, revision, generation
time, nodes, edges, and graph-level coverage records. Recorded snapshots use
revision `1` for their single immutable full graph; live and hub revisions
follow their publication sequence. Arrays are emitted in stable identity order
so equal evidence produces deterministic JSON.

Nodes contain:

- `id`: canonical identity, not a renderer-specific key.
- `identity`: the unescaped provider identity segment used to verify `id`;
  this is the UID when available and otherwise a kind-qualified stable name.
- `kind`: a stable topology kind such as cluster, namespace, workload, pod,
  service, node, EKS node group, EC2 instance, or external endpoint.
- `scope`: provider, cluster, namespace, and optional account/region identity.
- `name`, `apiVersion`, and resource `uid` when available.
- `parentId` for navigational containment only.
- display metadata and evidence references; raw credentials and Secret values
  are never part of the graph.
- `pseudoKind` for `unresolved`, `external-endpoint`, or `internet` identities
  that preserve a relationship when its target is not a collected API object.

Edges contain:

- `id`, `source`, and `target`.
- `kind`: `ownership`, `declared`, `resolved`, `inferred`, or `observed`.
- `directed` and optional relation metadata.
- evidence references that explain why the edge exists.
- finding references when a deterministic rule is supported by both endpoints.

`parentId` is not a replacement for a typed relationship. It supplies stable
drill-up/drill-down navigation while edges retain their evidence semantics.

## Stable Identity

Canonical IDs are built from escaped identity segments, never from labels or
screen position:

```text
<provider>/<cluster>/<kind>/<namespace-or-_>/<name-or-uid>
```

Cluster-scoped resources use `_` as the namespace segment. External endpoints
use the evidence provider plus its normalized endpoint identity. Edge IDs are
derived from edge kind, source ID, target ID, and provider key. Duplicate IDs
are invalid.

An `unresolved` pseudo-node preserves a collected reference whose target was
not collected. An `external-endpoint` preserves a concrete address without a
Kubernetes/AWS object identity. An `internet` node is emitted only for an
explicitly public boundary, currently an EKS API endpoint with
`endpointPublicAccess`; the corresponding edge retains `publicAccessCidrs` and
does not imply unrestricted reachability or observed traffic.

## Evidence And Relationship Tiers

Evidence attached to a node or edge records:

- `provider`: independently sourced capability or evidence stream.
- `basis`: `declared`, `resolved`, `inferred`, or `observed`.
- `observedAt`, `windowStart`, and `windowEnd` where applicable.
- `sampling`: provider-supplied sampling description.
- `freshness`: `current`, `stale`, or `unknown`.
- `confidence`: `high`, `medium`, `low`, or `unknown`.
- human-readable summary and optional field-level references.

Relationship rules:

- `ownership` comes from explicit Kubernetes/AWS ownership identity.
- `declared` represents desired configuration such as selectors or references.
- `resolved` requires collected evidence that a declared target exists and
  matches.
- `inferred` is a deterministic interpretation and must identify its rule.
- `observed` is reserved for time-bounded observations from an explicit
  external evidence provider.

A Service selector therefore cannot become an `observed` connection merely
because it matches a workload or endpoint.

## Coverage

Coverage records are scoped by provider, capability, and optional namespace.
Status is one of `complete`, `partial`, `unavailable`, or `unknown`, with a
reason and observation time/window when known.

The graph always contains a `runtime-connections` coverage record. When no
external connection provider contributed evidence, its status is
`unavailable`; declared, resolved, ownership, and inferred edges remain valid.
Duplicate provider/capability/namespace coverage records are conservatively
merged before publication: the weakest status, oldest observation timestamp,
and stable union of reasons are retained.

## Transport Compatibility

- Recorded output writes the graph as a deterministic artifact alongside the
  inventory snapshot.
- Live responses include the graph produced from the exact snapshot revision
  being published.
- Hub envelopes carry that same graph; the hub does not recollect or silently
  reinterpret provider evidence. When a remote writer configures an explicit
  hub cluster ID, it projects the graph with that same ID before publication;
  the hub rejects envelope/graph identity mismatches.
- Consumers must ignore unknown fields and reject unsupported major schema
  versions.

## Incremental Updates And Resynchronization

Live and hub pages use conditional HTTP fetch. A topology endpoint accepts the
client's last applied revision as `since` and returns
`teleskope.io/topology-update/v1alpha1`:

- `full` contains a complete graph and resets the client baseline. It is used
  for first load, expired history, a future/out-of-order client revision,
  cluster/schema mismatch, or a revision collision.
- `delta` declares `baseRevision` and contains canonical node/edge upserts,
  explicit node/edge deletion IDs, and the complete coverage set for the new
  revision.
- `unchanged` confirms that the requested revision is current.

Live uses `/api/topology?since=<revision>`. Hub drilldown uses
`/api/cluster/topology?id=<cluster>&since=<revision>`. `GET` and `HEAD` are
supported and responses carry an ETag. `/api/snapshot` remains the compatible
full-state fallback and is the resynchronization path if a client cannot apply
a delta.

Updates are idempotent: applying an already accepted target revision is a
no-op. A delta is rejected unless its cluster, schema, and base revision match
the current graph. The final graph is validated after every apply, so dangling
edges or incomplete deletion sets force a full resync rather than corrupting
browser state.

Stores retain at most 16 graph revisions per live server or hub cluster. This
is a bounded replay window, not an event log. Collector publications are
coalescing boundaries; status-only changes do not fabricate topology
revisions. Stale or failed collection retains the previous graph and its
source-specific coverage.

The browser keys layout anchors by canonical node ID. Existing nodes retain
their lane rank when other nodes are inserted, filtered, or removed; selection,
filters, viewport, paused state, and drawer context survive accepted updates.
New nodes are appended to their semantic lane and deleted identities are
pruned from the bounded layout map. WebSocket/SSE transport and force-directed
motion are not required for this publication model.

### Publication And Reconnect Lifecycle

- Polling collectors remain sequential and non-overlapping. Watch-backed
  collectors debounce object churn before publishing one immutable snapshot;
  the live store increments its revision once per accepted publication.
- The browser starts its next five-second conditional fetch only after the
  previous request finishes and gives each request a ten-second timeout. This
  bounds client backpressure without a request queue. Pausing the page skips UI
  fetch/apply while background collection continues.
- A disconnect retains the last graph, selection, viewport, and evidence. The
  browser clears its snapshot ETag and retries; on return it requests a delta
  from the retained graph revision. Missing history or any apply failure uses
  the full graph already carried by `/api/snapshot`.
- Kubernetes, AWS, metrics, and external providers keep their own timestamps
  in source status, coverage, and evidence. Graph `generatedAt` identifies the
  combined publication only; it does not make older provider evidence fresh.
- Deletion becomes visible only in an accepted publication and is encoded as
  explicit edge and node IDs. Collection failure, reconnect, or permission
  regression retains prior evidence under stale/partial status instead of
  emitting deletion churn.

Protocol tests cover disconnect-equivalent expired history, future and
out-of-order revisions, duplicate apply, explicit deletion, invalid query
input, and burst retention beyond the 16-revision window. Browser acceptance
checks refresh stability and full-reset fallback against the Recorded server.

## Security Boundary

Teleskope does not implement or ship a privileged node probe. It does not
require host PID/network namespaces, container-runtime sockets, host procfs,
conntrack/debugfs, or eBPF privileges. Runtime connection evidence can enter
only through an explicitly configured external provider and must preserve that
provider's limitations.

## Acceptance

- Contract validation rejects invalid identities, dangling references,
  duplicate identities, unsupported relationship kinds, and observed edges
  without observed evidence.
- JSON round trips preserve evidence metadata.
- Equal input evidence produces byte-stable graph JSON.
- Recorded, live, and hub paths expose the same graph schema.
- Provider absence is represented as unavailable coverage rather than an empty
  graph or fabricated observed edges.
