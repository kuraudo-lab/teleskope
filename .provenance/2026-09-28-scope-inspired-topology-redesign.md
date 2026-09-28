# Scope-inspired Topology Redesign Provenance

## Trigger

The user rejected the current object-card overflow handling, horizontal swimlanes, and the unexplained Semantic view. After a primary-source review of Weave Scope and an interactive three-variant prototype, the user accepted the Scope-like redesign and requested that it be recorded formally, committed, and implemented unchanged in the product.

## Scope

- Formal product design for the embedded topology workspace.
- Weave Scope topology research and correction of the earlier force-layout characterization.
- Accepted interactive prototype used as the implementation target.
- Production changes in `internal/report` and their automated/browser verification.

## Conversation Summary

The accepted direction uses the prototype's Scope graph as the default overview and turns its focused-neighborhood variant into the normal result of selecting a node. The prototype's vertical trace is not shipped as a separate mode; it contributes only the cross-layer ordering rule `external network → workload → runtime / infrastructure`. The current semantic selector and swimlane presentation are removed. Compact glyphs, bounded labels, evidence-aware edges, contextual details, and Table fallback replace verbose topology object cards.

This record is a structured summary of the design conversation, not a raw transcript.

## Design Diff

- `docs/topology-scope-reborn-design.md` records the accepted production behavior and acceptance criteria.
- `docs/weave-scope-topology-ui-research-2026-09-28.md` provides primary-source evidence for Scope's projection, Dagre layout, focus, details, search, and Table behaviors.
- `docs/weave-scope-product-ui-research-2026-09-26.md` corrects the earlier description of Scope as force-directed.
- `docs/mockups/topology-scope-reborn-prototype.html` captures the accepted interactive visual target.

## Decisions

- Ship one Workloads projection first; do not expose nonfunctional projections.
- Use a deterministic top-to-bottom overview graph.
- Make click selection enter a centered one-hop focus state automatically.
- Keep cross-layer vertical ordering as a layout rule, not a mode or permanent lane.
- Keep nodes compact and move verbose information to the inspector.
- Preserve evidence kind, provider/source, freshness, and coverage instead of implying all edges are observed traffic.
- Retain Table fallback, filters, search, pan/zoom, live updates, theme, Events, and export.

## Rejected Alternatives

- Horizontal swimlanes: they obscure rather than explain relationships.
- Semantic view: its product meaning is unclear and duplicates filter/projection concepts.
- A separate focused-neighborhood mode: focus is an object-selection interaction, not another view users must understand.
- A permanent external/workload/infrastructure lane stack: cross-layer ordering is useful only when relevant evidence exists.
- Node probe or fabricated runtime traffic: both violate the accepted collection and evidence boundary.

## Constraints

- No privileged node probe.
- No terminal, lifecycle, or other write action.
- The embedded single-file UI must continue to support offline, live, snapshot, and hub rendering.
- Existing topology graph and incremental-update contracts remain authoritative.
- Missing runtime evidence remains unavailable or a coverage gap.
- Long unbroken identities must not alter node geometry or overlap neighbors.

## Evaluation Plan

- Update focused `internal/report` tests for the new DOM, state, and layout contracts.
- Run `make check`, `make build`, and `git diff --check` with the repository's shared Go caches.
- Serve the recorded EKS fixture through the real embedded path and verify default graph, selection/focus, long labels, search/filter, relationship evidence, Table fallback, pan/zoom, theme, Events, and export-adjacent shell behavior in a browser.
- Obtain an independent sub-agent acceptance review before the implementation commit.

## Open Questions

None for this implementation slice.

## Links

- Design: `docs/topology-scope-reborn-design.md`
- Research: `docs/weave-scope-topology-ui-research-2026-09-28.md`
- Prototype: `docs/mockups/topology-scope-reborn-prototype.html`
- Issue: https://github.com/kuraudo-lab/teleskope/issues/49
