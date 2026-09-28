# Topology Edge and Label Clearance Provenance

## Trigger

While implementing the accepted circular nodes, the user explicitly required Teleskope to handle the visual collision between relationship lines and node text.

## Scope

- `docs/topology-scope-reborn-design.md`
- Graph-mode relationship path routing and node-label styling in `internal/report/report.html`
- Recorded-fixture browser verification and report tests

## Conversation Summary

Moving labels below circular icons creates a risk that a vertical relationship line exits an icon and passes through its label or secondary text. The fix must protect legibility without returning to rectangular node cards.

## Design Diff

The formal design now requires paths to route around label zones, keep edges behind node content, and use only a compact canvas-colored text backing as a fallback mask.

## Decisions

- Anchor relationships to the icon silhouette.
- Route vertical relationships from a side of the icon until clear of the node's label zone, then continue toward the target.
- Render edges before node shapes and copy.
- Give label/secondary copy a compact canvas-colored backing so unavoidable crossings cannot show through text.
- Apply the same clearance to default, hover, selected, and focused relationships.

## Rejected Alternatives

- Allowing lines to show through text: this harms scanability and makes dense graphs look unfinished.
- Restoring a full rectangular card behind every node: this contradicts the accepted circular-node mock.
- Hiding relationships near selected nodes: this removes evidence-bearing information.

## Constraints

- Preserve the accepted circular visual contract and fixed footprint.
- Preserve relationship direction, selection, hover, evidence, Fit, and focus behavior.
- No new data provider or node probe.

## Evaluation Plan

- Inspect the default recorded-fixture graph at Fit scale for line/text collisions.
- Inspect `aws-node` focus and a selected relationship.
- Verify long truncated labels, Search dimming, and both themes.
- Run focused report tests, `make check`, `make build`, and `git diff --check`.

## Open Questions

None.

## Links

- Design: `docs/topology-scope-reborn-design.md`
- Accepted mock: `docs/mockups/topology-scope-reborn-prototype.html`
- Related issue: #49
