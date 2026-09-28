# Scope-inspired Topology Product Design

Status: accepted for implementation on 2026-09-28.

Related issue: [#49](https://github.com/kuraudo-lab/teleskope/issues/49)

Research basis: `docs/weave-scope-topology-ui-research-2026-09-28.md`

Accepted prototype: `docs/mockups/topology-scope-reborn-prototype.html`

## Product decision

Teleskope replaces the current semantic-view and swimlane topology with a Scope-inspired relationship workspace:

- The default is a stable, top-to-bottom workload relationship graph.
- Selecting an object is a focus transition: the selected object moves to the center, its visible one-hop neighbors form a ring, unrelated objects and edges are dimmed, and the inspector opens the complete context.
- The vertical sequence `external network → workload → runtime / infrastructure` is a cross-layer ordering rule for focused paths. It is not a permanent set of lanes and is not exposed as another topology mode.
- Graph nodes are compact glyphs. Rich metadata belongs in the inspector.
- Graph remains available for every result size. Table is an explicit alternative presentation, not an automatic replacement for the topology.

The production UI implements the accepted prototype behavior, not the prototype's A/B/C chooser. Prototype A becomes the default graph; prototype C becomes the selected-object state. Prototype B contributes only its vertical cross-layer ordering rule.

## Information architecture

### Projection

The first production slice exposes `Workloads` as the only enabled projection. Future projections may be added when their data and interaction contracts exist, but disabled placeholders must not pretend to work.

The workload projection may include entry, gateway, service, workload, selected external dependency, and provider-backed attachment nodes when the current topology graph contains evidence for them. It must not manufacture runtime traffic.

### Controls

The topology workspace retains:

- Search within the current projection.
- Namespace and resource visibility filters.
- Health, system/application, and connection-state filters where backed by the existing graph contract.
- Graph/Table switch.
- Fit, zoom, and pan.
- Live update pause/status, theme, export, and Events navigation supplied by the report shell.

There is no `Semantic view`, swimlane selector, or focus checkbox. Focus is the direct result of selecting a node.

The namespace selector must not offer a choice that is guaranteed to produce an empty Workloads projection. When topology graph evidence is available, its options are the namespaces represented by displayable external, entry, service, or workload objects. Cluster namespaces that contain only leases or other resources outside this projection remain available on their relevant Kubernetes inventory pages, but are not presented as valid topology scopes. A stale or deep-linked namespace outside the current projection falls back to `All namespaces`.

### Graph node

A graph node contains only:

- Type glyph and type-specific outline.
- Bounded display label.
- Optional count for an aggregate.
- One short secondary line.
- One pinned metric or finding marker.

The accepted mock is the visual contract for graph nodes. A normal external, gateway, service, or workload node is not a rectangular card: it is a centered 52px circular icon container with its bounded label and secondary line stacked below. External nodes use a dashed blue circle, gateways use the accent outline, services use the blue outline, and workloads use the purple outline. A selected node gains the mock's accent outline and outer halo without changing its footprint. Health remains a small circular marker attached to the icon boundary. Pod/runtime and infrastructure nodes use the mock's distinct diamond and square silhouettes so type is not encoded by color alone.

The node has a fixed 120px visual footprint. Labels and secondary text are centered below the icon and use the mock's bounded widths and ellipsis. A pinned metric is a bounded badge beside the icon, matching the mock rather than becoming another text row. The complete identity remains available through the accessible name, focus/hover tooltip, and inspector title. Unbroken names such as ARNs, image digests, node names, and generated workload names must not resize or overlap neighboring nodes.

### Relationships

Edges retain the evidence distinctions already present in the topology contract. Visual treatment and the inspector must make at least these classes distinguishable when present:

- Resolved relationship.
- Declared configuration.
- Ownership or attachment.
- Observed relationship only when supplied by an explicit external provider.

Default edges are visually quiet. Hovering a node emphasizes its incident edges and direct neighbors. Selecting an edge or a relationship row exposes direction, basis, source/provider, freshness, and coverage. Declared evidence must never be described as observed traffic.

Edges must not reduce node-label legibility. Connection paths anchor to the icon shape and route around the label/secondary-text zone rather than passing vertically through it. Labels use only a compact canvas-colored backing where needed to mask a crossing line; this backing follows the text footprint and must not recreate the removed rectangular node card. Edge rendering remains behind node shapes and copy, and highlighted/selected edges must preserve the same text-clearance rule.

### Selection and focus

With no selection, the graph uses stable top-to-bottom relationship depth and keeps the inspector in an empty guidance state. Depth is derived from the visible graph's roots and directed relationships, not from a fixed list of resource-type rows.

When an object is selected:

1. The object moves to the graph center.
2. Visible incoming and outgoing one-hop neighbors are placed around it.
3. Unrelated nodes and edges remain in the graph but are dimmed.
4. Incident edges receive the stronger selected treatment and direction marker.
5. The inspector updates from the same selected object state.
6. Clearing selection restores the overview without losing filters, projection, pan/zoom intent, or live revision state.

Search follows the same mental-map principle: matches remain strong and non-matches are dimmed in Graph mode; Table mode may show only matching rows.

### Inspector

The selected-object inspector is ordered as:

1. Identity and status.
2. Context and source freshness.
3. Direct relationships.
4. Deterministic Advisor findings.
5. Metrics and value/evidence type.
6. Children and placement.
7. Selected relationship evidence.
8. Object evidence and raw data access where already supported.

The inspector is the canonical location for full labels and verbose metadata. The graph must not duplicate these sections inside nodes.

### Scale and presentation

There is no node-count or edge-count gate that changes Graph into Table. A user who selected Graph always receives a graph, including the canonical Large fixture. Complexity may disable decorative animation, suppress nonessential edge decoration, or show a non-blocking performance notice, but it must not change presentation mode or remove graph interaction. Table remains a user-selected dense evidence view; its rows open the same inspector.

## Layout rules

- Overview layout is a deterministic, top-to-bottom rooted hierarchy following relationship direction. The renderer first collapses each strongly connected component (SCC) into one layout component, producing an acyclic condensation graph. Zero-indegree SCCs are roots; multiple source SCCs attach to one layout-only virtual root that is never rendered and never added to evidence. A disconnected object is a single-node source SCC.
- Component depth is the longest path from the virtual root through the condensation DAG. Nodes inside one SCC share that depth and receive deterministic local ordering and stagger. The renderer creates as many depth levels as the condensed visible relationship graph requires; it does not clamp the graph into fixed external/service/workload rows or a fixed maximum rank.
- Resource semantics remain a secondary ordering constraint: external entry tends to precede service and workload, with runtime/infrastructure below when the collected relationships support that direction. Semantics must not invent an edge or force unrelated objects into the same row.
- Weakly connected components are laid out independently and then packed into an open canvas. Within a component, each depth level is ordered by the barycenter of connected parents and children. Source branches receive horizontal space proportional to their reachable descendants; nodes reachable from multiple roots have one position based on all parent barycenters rather than being duplicated. Components must not be flattened into one global source row.
- Zero-degree objects do not participate in the directed ranks. Like Scope, they are arranged in a stable near-square grid beside or below the connected graph according to the resulting aspect ratio. This keeps inventory visible without turning every disconnected object into another root in an invisible full-width row.
- Node separation and rank separation are minimums, not a viewport-fitting target. The canvas grows to the layout bounds; fit/zoom reveals the whole graph without shrinking the logical spacing between objects.
- Every visible relationship receives distinct source and target ports plus a deterministic route offset. Two relationships must not emit the same SVG path or share a long coincident trunk. Curves or rounded orthogonal segments may cross when the graph is non-planar, but ordering and routing must minimize crossings and must never cross a node's icon, label, or secondary-text footprint.
- Nodes with no connected relationship remain visible in the stable isolated-object grid.
- Focus layout is deterministic for the selected identity and its visible one-hop neighborhood.
- Cross-layer focus paths place external entry above workload and runtime/infrastructure below when those relationships exist.
- Live revisions reuse stable identities and existing layout state where possible to reduce visual jumps.
- No force simulation, animated drift, permanent lane backgrounds, or multi-layer object cards are introduced.

## Evidence and security boundaries

- No node probe is implemented.
- Runtime connection evidence is unavailable unless an explicitly configured external evidence provider supplies it.
- Missing provider coverage is shown as unavailable or a coverage gap, never inferred from declaration alone.
- Kubernetes and AWS inventory remain read-only evidence.
- Terminal, container lifecycle controls, and other write operations are outside this design.

## State and compatibility

The implementation may migrate or remove the old `topologyView`/semantic URL state. It must preserve meaningful state for namespace, resource/search filters, graph/table mode, selection, selected edge, pinned metric, zoom, and pan where those controls remain.

The same embedded HTML must work for offline reports, `serve snapshot`, live serve, and hub-scoped rendering. Incremental topology updates must continue to honor active drag/pause behavior.

## Acceptance criteria

- The product contains no visible semantic-view or swimlane control/text.
- Default Workloads topology renders as a stable top-to-bottom relationship graph.
- The number of visible depth levels follows the longest visible root-to-leaf relationship path instead of a fixed three-row or resource-type layout.
- Siblings and root components form a readable, deterministic hierarchy without permanent lane backgrounds or rigid full-width rows.
- Distinct visible relationships do not produce identical SVG paths or long coincident line segments; fan-out/fan-in separates at the node boundary, and routing minimizes crossings.
- Selecting different nodes recenters each selected node and synchronizes title, graph emphasis, and inspector content.
- Long names, ARNs, image digests, and metrics do not escape their node bounds or cover neighbors at supported zoom levels.
- Default external, gateway, service, and workload objects use the accepted mock's 52px circular icon treatment rather than rectangular cards; selection, health, pod/runtime, and infrastructure silhouettes match the same visual grammar.
- Full identity is keyboard- and pointer-accessible and visible in the inspector.
- Node hover emphasizes only direct neighbors and incident edges.
- Default, highlighted, and selected relationship lines do not visibly cross node labels or secondary text.
- Relationship inspection exposes direction, basis, source, freshness, and coverage.
- Search, filters, user-selected Table mode, pan/zoom, fit, selection, theme, Events, export, and live update behavior remain functional.
- Graph mode remains Graph for the Small, Medium, and Large deterministic scale fixtures; complexity never silently rewrites the selected presentation.
- Every namespace offered by the topology workspace produces at least one displayable Workloads-projection object for the current graph revision.
- The recorded EKS fixture renders through the real embedded `/api/snapshot` path and passes browser interaction checks.
- Automated report tests, `make check`, `make build`, and `git diff --check` pass.
- No node probe or synthetic observed-traffic claim is added.
