# Topology Selection Edges Provenance

## Trigger

The user asked to reduce topology line clutter: relationships between different object types should appear only after an object is selected, and visible relationships should use right-angle orthogonal lines instead of curved paths.

## Scope

This change covers the embedded topology interaction and rendering in `internal/report/report.html`, together with focused report-shell tests. It does not change snapshot data, topology relationship discovery, lane placement, or backend collectors.

## Conversation Summary

The current swimlane topology renders every discovered relationship immediately using cubic Bézier curves. Dense snapshots therefore show visual noise before the user has chosen an object. The accepted interaction is selection-led disclosure: preserve all discovered relationships in memory, but draw only the direct edges incident to the selected topology object.

## Design Diff

- Add persistent topology selection state keyed by topology node ID.
- Render no relationship paths when no object is selected.
- When selected, render only direct incident relationships and visibly distinguish the selected and related objects.
- Route visible edges with horizontal and vertical SVG path segments, including a same-lane detour when necessary.
- Clear the relationship focus when the user clicks empty topology canvas space.
- Preserve selection across ordinary redraws when the selected node remains visible; clear it when filters/search remove that node.
- Make topology nodes keyboard-focusable and selectable with Enter or Space.
- Synchronize the Inspector selected-object summary and the visible edge count with the current selection.

## Decisions

- Keep relationship discovery unchanged so selection affects presentation, not evidence.
- Treat only directly incident edges as related; do not recursively reveal an entire connected component.
- Keep all object cards visible as context instead of hiding unrelated objects.
- Use exact orthogonal `H` and `V` path segments rather than rounded or curved connectors.
- Use border emphasis as well as color so selection is not communicated by color alone.

## Rejected Alternatives

- Do not show all edges at reduced opacity; the request is to show relationships only on selection.
- Do not reveal relationships on hover alone; hover is unavailable to keyboard and touch users and is too transient for inspection.
- Do not change lane ordering or resource relationship rules in this interaction-only slice.

## Constraints

- Search, namespace/resource filters, pan, zoom, drawer details, and the persistent Inspector must continue working.
- Selection and focus indicators must remain distinguishable in light and dark themes.
- The embedded UI must remain dependency-free and self-contained.
- Static reports and live `/api/snapshot` rendering share this implementation.

## Evaluation Plan

- Add report assertions for selection state, selected-edge filtering, orthogonal path commands, accessibility state, and Inspector synchronization.
- Run `go test ./internal/report`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check` and `make build`.
- Run embedded JavaScript syntax validation and `git diff --check`.
- In the browser, verify no lines before selection, only direct orthogonal lines after selecting a card, selection clearing on empty canvas, and working pan/zoom/detail behavior.

## Open Questions

- None for this interaction slice.

## Links

- Related topology provenance: `.provenance/2026-09-07-topology-controls-and-icon.md`
- Implementation: `internal/report/report.html`, `internal/report/report_test.go`
