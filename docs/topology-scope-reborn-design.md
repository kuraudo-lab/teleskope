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
- Large or filtered result sets can be inspected in Table mode without losing selection or evidence access.

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

### Graph node

A graph node contains only:

- Type glyph and type-specific outline.
- Bounded display label.
- Optional count for an aggregate.
- One short secondary line.
- One pinned metric or finding marker.

The node has a fixed visual footprint. Labels, secondary text, and metrics use bounded widths and ellipsis. The complete identity remains available through the accessible name, focus/hover tooltip, and inspector title. Unbroken names such as ARNs, image digests, node names, and generated workload names must not resize or overlap neighboring nodes.

### Relationships

Edges retain the evidence distinctions already present in the topology contract. Visual treatment and the inspector must make at least these classes distinguishable when present:

- Resolved relationship.
- Declared configuration.
- Ownership or attachment.
- Observed relationship only when supplied by an explicit external provider.

Default edges are visually quiet. Hovering a node emphasizes its incident edges and direct neighbors. Selecting an edge or a relationship row exposes direction, basis, source/provider, freshness, and coverage. Declared evidence must never be described as observed traffic.

### Selection and focus

With no selection, the graph uses stable top-to-bottom ranks and keeps the inspector in an empty guidance state.

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

### Scale fallback

The existing complexity gate remains authoritative. When the graph exceeds its supported node/edge budget, Teleskope switches to or recommends Table mode instead of shrinking nodes indefinitely. Table rows remain selectable and open the same inspector.

## Layout rules

- Overview layout is deterministic and top-to-bottom, following relationship direction.
- Nodes with no connected rank use a stable overflow/grid placement.
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
- Selecting different nodes recenters each selected node and synchronizes title, graph emphasis, and inspector content.
- Long names, ARNs, image digests, and metrics do not escape their node bounds or cover neighbors at supported zoom levels.
- Full identity is keyboard- and pointer-accessible and visible in the inspector.
- Node hover emphasizes only direct neighbors and incident edges.
- Relationship inspection exposes direction, basis, source, freshness, and coverage.
- Search, filters, Table fallback, pan/zoom, fit, selection, theme, Events, export, and live update behavior remain functional.
- The recorded EKS fixture renders through the real embedded `/api/snapshot` path and passes browser interaction checks.
- Automated report tests, `make check`, `make build`, and `git diff --check` pass.
- No node probe or synthetic observed-traffic claim is added.

