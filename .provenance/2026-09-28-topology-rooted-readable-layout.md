# Topology Rooted Readable Layout Provenance

## Trigger

The user reported that dense topology views produce heavily overlapping relationship lines and that the current placement still reads as three rigid invisible rows. They requested a readable hierarchy whose number of levels follows the actual graph from its root objects.

## Scope

- `docs/topology-scope-reborn-design.md`
- The Overview graph layout and SVG relationship routing in `internal/report/report.html`
- Generated recorded demo reports and report regression tests

## Conversation Summary

The accepted Scope-inspired node and inspector treatment remains unchanged. This refinement replaces fixed semantic rows with relationship-derived depth, keeps the top-to-bottom mental model, and gives every relationship an independently routed visual path. The goal is an organic rooted hierarchy, not a force-directed cloud and not a return to swimlanes.

## Design Diff

`docs/topology-scope-reborn-design.md` now specifies SCC condensation, virtual-root depth, deterministic branch placement, within-level crossing reduction, bounded stable staggering, distinct relationship ports and routes, and acceptance criteria for coincident-edge elimination.

## Decisions

- Collapse strongly connected components into a condensation DAG before deriving levels. Source SCCs attach to one non-rendered, layout-only virtual root; longest-path depth on that DAG exposes every meaningful level without ambiguous cycle traversal.
- Keep nodes inside one SCC at the same depth with deterministic local ordering. Treat disconnected objects as single-node source SCCs, and place nodes reachable from multiple sources once using all parent barycenters.
- Use resource semantics only as a secondary ordering hint when evidence is incomplete; never manufacture relationships to obtain a preferred visual hierarchy.
- Allocate horizontal space by component and descendant span, then apply deterministic barycentric ordering and a small stable stagger. This preserves hierarchy without recreating swimlanes.
- Assign every edge distinct endpoint ports and a deterministic route offset. Eliminate identical paths and long coincident trunks; minimize unavoidable crossings in non-planar graphs.
- Preserve the existing Graph/Table fallback, focus transition, inspector, pan/zoom, filters, evidence styling, accessibility contract, and label-clearance behavior.

## Rejected Alternatives

- Fixed resource-type rows: they flatten real relationship depth and recreate the rejected swimlane mental model.
- Unconstrained force simulation: it drifts between revisions, weakens top-to-bottom causality, and reduces reproducibility in recorded reports.
- Edge bundling into shared trunks: it reduces ink but makes individual evidence-backed relationships ambiguous and recreates the reported overlap.
- Hiding relationships by default: it conceals evidence rather than improving its presentation.

## Constraints

- The embedded report remains dependency-free vanilla HTML, CSS, and SVG.
- No node probe or synthetic observed-traffic relationship is introduced.
- Layout and routing must be deterministic for the same graph revision.
- Labels, secondary text, status markers, and metric badges remain clear of relationship lines.
- Table mode remains the accessible and high-complexity source of truth.

## Evaluation Plan

- Add focused report contract tests for root-derived depth and distinct edge routing helpers.
- Use the recorded EKS fixture through the real `serve snapshot` path.
- Verify that visible depth equals the condensation-DAG virtual-root-to-leaf depth, SCC members receive deterministic placement, multi-root descendants are not duplicated, node bounds do not intersect, distinct edges do not emit identical path data, and relationship paths do not intersect label rectangles.
- Exercise namespace filters, selection/focus, hover emphasis, Graph/Table switching, fit, zoom, pan, theme, and console health.
- Run focused report tests, `make check`, `make build`, and `git diff --check`.
- Obtain independent subagent acceptance before the implementation commit.

## Open Questions

- None.

## Links

- Design file: `docs/topology-scope-reborn-design.md`
- Related issue: `#49`
- Preceding topology implementation: `a4fd5a1`
