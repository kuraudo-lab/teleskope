# Scope-Style Topology Interaction Prototype

## Question

Which interaction model best rebirths Weave Scope's relationship-first
exploration while preserving Teleskope's evidence semantics, embedded delivery,
current swimlane view, and deterministic findings?

The prototype runs against the existing Recorded EKS fixture on the current
topology route. It does not add write operations, invent runtime observations,
or change the topology contract.

## Variants

### A — Semantic Swimlanes

Evolve the current stable layout. A view switch changes the lane vocabulary
between Application, Workload, Runtime, and Infrastructure. Graph/table mode,
structured search, namespace filters, selection, and the existing contextual
drawer remain in one page.

Primary hypothesis: familiar deterministic placement and explicit layers make
cross-layer evidence easiest to understand.

### B — Scope Canvas

Use a relationship-first canvas with grouped clusters and a compact left
filter rail. Selection focuses the direct neighborhood while the complete
graph remains available by clearing focus. Table mode remains the accessible
and high-density alternative.

Primary hypothesis: spatial exploration best restores Scope's investigative
feel.

### C — Relation Explorer

Use a three-pane master/detail workspace: sortable object table, selected
object neighborhood, and contextual details. Graph navigation is local to the
selected object rather than presenting the whole cluster at once.

Primary hypothesis: an explicit result list plus bounded neighborhood gives
the fastest path from finding/search to evidence at production scale.

## Shared State Contract

Every variant exposes the same state:

- semantic view: `application`, `workload`, `runtime`, or `infrastructure`;
- presentation mode: `graph` or `table`;
- namespace, resource, system-object, and health filters;
- compound search terms for identity, labels/metadata, finding severity, and
  numeric comparisons when metrics exist;
- selected node, focus mode, and graph viewport;
- URL-restorable state using query parameters;
- live refresh without clearing a still-visible selection or viewport.

Search and filters narrow the current view; they never change evidence tier or
turn declared relationships into observed connections.

## Prototype Evaluation

Use realistic Recorded data and evaluate each variant against these tasks:

1. Start from a finding or search result and locate the affected graph object.
2. Move from a Service or workload to its Pods, Node, and EKS infrastructure.
3. Distinguish declared, resolved, ownership, inferred, and observed edges.
4. Restore the same view, filters, selection, and mode from a copied URL.
5. Complete the flow with keyboard only and a visible, unobscured focus ring.
6. Remain usable at narrow desktop width and in high-contrast/dark themes.

The winner may combine a primary structure from one variant with an isolated
affordance from another, but the production implementation must have one
coherent state model rather than shipping a permanent variant switcher.

## Acceptance Boundary

- The throwaway variants are captured outside the main branch.
- The selected interaction is rewritten into the embedded production page.
- Graph and table modes share filters and selection.
- Browser tests cover URL restoration, search/filter interaction, keyboard
  selection, and selection stability across refresh.
- The existing semantic swimlane experience remains available until a later
  migration decision explicitly removes it.
- No privileged node probe, terminal, CRUD, restart, or lifecycle controls are
  introduced.
