# Topology Selection-Only Straight Edges Provenance

## Trigger

The user rejected the mixed straight-and-curved relationship rendering and
asked for all topology lines to be straight and hidden until an object is
clicked.

## Scope

The embedded topology graph presentation, focus interaction, relationship
selection state, generated demos, tests, and walkthrough acceptance.

## Conversation Summary

The open Scope-like layout remains the overview mental map, but relationship
lines are now progressive disclosure. The overview shows objects without line
noise. Selecting an object reveals all visible one-hop relationships from that
object using direct straight segments only.

## Design Diff

- `docs/topology-scope-reborn-design.md` defines selection-only edges and the
  straight-segment visual contract.
- `docs/topology-contract.md` limits graph edge interaction to relationships
  revealed by an object focus.
- `docs/scope-reborn-walkthrough.md` adds overview and focus checks.

## Decisions

- Render zero relationship lines while no object is selected.
- On object selection, render every visible incoming and outgoing one-hop
  relationship.
- Represent each relationship as one straight segment anchored at the two icon
  boundaries.
- Keep lines behind node shapes and bounded label backings.
- Keep the relationship table and Inspector as the complete accessible
  evidence surfaces.

## Rejected Alternatives

- Mixed orthogonal and curved obstacle routing: visually busy and explicitly
  rejected by the user.
- Quiet always-visible edges: still produces excessive overlap in dense
  overviews.
- Hover-only reveal: inaccessible to keyboard and touch users and does not
  establish persistent investigation context.

## Constraints

- Do not introduce node or edge count limits or automatic Table fallback.
- Preserve circular topology nodes, namespace filters, selection URL state,
  pan/zoom, Inspector evidence, and keyboard activation.
- Do not imply observed traffic for declared or resolved relationships.
- No node probe or privileged DaemonSet.

## Evaluation Plan

- Verify the Recorded overview contains nodes but zero SVG relationship lines.
- Select an object with multiple relationships and verify every visible edge is
  a single straight segment with correct endpoints and direction marker.
- Clear selection and verify all lines disappear.
- Verify pointer and keyboard object selection, namespace filtering, URL
  restoration, Table mode, and Inspector relationship rows.
- Verify the Large fixture remains Graph, pannable, and selectable.
- Run focused report tests, `make check`, `make build`, and `git diff --check`.
- Obtain independent subagent acceptance before the implementation commit.

## Open Questions

None.

## Links

- Design files: `docs/topology-scope-reborn-design.md`,
  `docs/topology-contract.md`, `docs/scope-reborn-walkthrough.md`
- Related issue: #49
