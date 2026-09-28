# Unbounded Scope-style Topology Layout Provenance

## Trigger

The user reported that topology had become unusable after the previous readability work and explicitly rejected hard node/edge limits. They asked Teleskope to follow Weave Scope more carefully and prefer an open, attractive layout at every graph size.

## Scope

This record covers the embedded topology layout and rendering policy in `internal/report`, the deterministic scale harness, and the topology design, contract, scale, research, and walkthrough documents.

## Conversation Summary

The previous implementation forced Graph into Table above a fixed node/edge threshold and whenever sampled routing could not avoid every object footprint. Review of Weave Scope v1.13.2 confirmed that Scope uses Dagre with enlarged node footprints, explicit node/rank separation, a canvas sized to the layout bounds, an independent grid for zero-degree nodes, and cached coordinates where possible. It performs a full relayout when the graph changes materially, when relayout is explicitly requested, or when its final overlap defense detects nodes too close. Scope's complexity signal selects an initial mode and animation policy; it is not a hard stop inside the layout engine, and Scope 0.14 removed its earlier 100-node rendering limit.

The accepted correction is that Teleskope never rewrites a user's Graph selection to Table. Table remains available as an explicit alternative. Connected components use relationship-derived depth and open spacing; disconnected objects use a stable grid rather than a single global root row. Complexity may reduce animation or nonessential decoration, but the graph and its interactions remain available.

## Design Diff

- `docs/topology-scope-reborn-design.md`: replaces automatic fallback with continuous Graph availability; adds component packing, isolated-object grid, and expanding canvas rules.
- `docs/weave-scope-topology-ui-research-2026-09-28.md`: corrects the interpretation of Scope's complexity signal and records its layout spacing, zero-degree grid, cache, and overlap defense.
- `docs/topology-contract.md`: removes the fixed visible node/edge limit.
- `docs/topology-scale-performance-accessibility.md`: requires Small, Medium, and Large fixtures to remain in Graph mode.
- `docs/scope-reborn-walkthrough.md`: changes Large acceptance from automatic Table to Graph and focused-neighborhood interaction.

## Decisions

- Graph mode has no fixed node-count or edge-count gate.
- Routing conflict detection must never force Table.
- Weakly connected components are laid out independently and packed on an expanding canvas.
- Zero-degree objects use a stable near-square grid beside or below the connected graph.
- Table mode remains available only through an explicit user action.
- Complexity may disable animation and optional edge decoration, but never nodes, relationships, selection, pan/zoom, or Graph mode.

## Rejected Alternatives

- Automatic Table fallback above a fixed threshold: rejected because it makes topology unavailable precisely when investigation needs it.
- Treating every disconnected object as a source in one global depth row: rejected because it creates a rigid, extremely wide invisible row unrelated to graph structure.
- Infinite viewport fitting by shrinking glyphs and spacing: rejected because it destroys legibility; the logical canvas should expand instead.
- Reintroducing a node probe to obtain Scope's runtime traffic graph: explicitly out of scope.

## Constraints

- Preserve the self-contained embedded UI and offline/live/hub rendering paths.
- Preserve the compact circular Scope-like node visual, inspector, filters, keyboard access, theme, export, Events, and evidence boundaries.
- No node probe, privileged DaemonSet, fabricated observed traffic, or write controls.
- Identical graph input must produce deterministic positions and paths.

## Evaluation Plan

- Add a red-capable scale assertion that Large remains Graph rather than being rewritten to Table.
- Verify connected components are not flattened into one global root row and zero-degree objects form a stable grid.
- Verify graph paths remain unique and do not cross object copy in the Recorded fixture and representative cyclic/DAG fixtures.
- Run `make check`, `make build`, `git diff --check`, and the Small/Medium/Large browser harness.
- Perform keyboard, selection/focus, pan/zoom, and console checks in the real embedded Recorded path.
- Obtain independent subagent acceptance before the implementation commit.

## Open Questions

- None.

## Links

- Design files: `docs/topology-scope-reborn-design.md`, `docs/weave-scope-topology-ui-research-2026-09-28.md`, `docs/topology-contract.md`, `docs/topology-scale-performance-accessibility.md`, `docs/scope-reborn-walkthrough.md`
- Related issue: #49
- Primary reference: Weave Scope v1.13.2 `client/app/scripts/charts/nodes-layout.js`
