# Detail Drawer Bottom-right Placement Provenance

## Trigger

The user requested a small topology interaction refinement: the information drawer opened after clicking an object should appear in the bottom-right corner of the window while keeping its current margins and dimensions.

## Scope

This change only updates the fixed-position anchors for the embedded report detail drawer in `internal/report/report.html` and adds a focused report-shell assertion. Drawer content, size, scrolling, animation, and object-selection behavior remain unchanged.

## Decisions

- Replace the top anchor with a bottom anchor in both the base drawer rule and the later report-shell override.
- Preserve the matching existing inset at each cascade layer: 22px in the base rule and the effective 14px in the report shell.
- Preserve width, maximum viewport dimensions, padding, border radius, transform, transition, and z-index.

## Constraints

- The drawer must remain fully inside the viewport and independently scroll when its content exceeds its maximum height.
- The change must apply to topology workload details and generic object JSON details because they share the same drawer.
- No snapshot, relationship, Inspector, or backend behavior changes.

## Evaluation Plan

- Run `go test ./internal/report`, embedded JavaScript syntax validation, full `make check`, `make build`, and `git diff --check`.
- Open the Recorded report, select a topology object, and verify the drawer's computed position is 14px from the right and bottom while its current width and maximum dimensions remain unchanged.

## Links

- Related interaction provenance: `.provenance/2026-09-25-topology-selection-edges.md`
- Implementation: `internal/report/report.html`, `internal/report/report_test.go`
