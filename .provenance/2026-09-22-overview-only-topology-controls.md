# Overview-only Topology Controls Provenance

## Trigger

The user reported that the left-hand Topology Controls pane should not remain visible on every report page. It belongs only to Overview, while the right-hand Inspector must remain visible across report pages.

## Scope

This change covers the embedded report shell layout and its focused HTML assertions. It does not change topology data, EKS/Kubernetes navigation, Inspector contents, or backend behavior.

## Conversation Summary

The current shell moves Topology Controls and Inspector outside the individual report sections. As a result, both panes persist when navigating away from Overview. The requested behavior is asymmetric: hide only Topology Controls outside Overview, preserve Inspector, and let the report content consume the released width.

## Design Diff

- Add active-section-aware layout rules for non-Overview report pages.
- Hide `.sidebar` outside Overview while retaining `.inspector`.
- Change the desktop grid from three columns to report content plus Inspector when the sidebar is absent.
- Preserve the existing Advisor exception, which intentionally uses a full-width single-column workspace.

## Decisions

- Drive the behavior from the existing `body[data-active-section]` state rather than adding duplicate JavaScript visibility state.
- Keep Inspector in the DOM and visible so inventory health, source freshness, and selected-object context remain stable.
- Allow the central report to expand instead of leaving an empty 284-pixel grid track.

## Rejected Alternatives

- Do not remove or recreate the sidebar during navigation; CSS state avoids losing controls and event handlers when returning to Overview.
- Do not hide Inspector with Topology Controls; that directly contradicts the requested persistent inspection context.
- Do not special-case every EKS and Kubernetes page separately; all non-Overview report pages share the same layout rule.

## Constraints

- Overview must retain the existing three-column layout.
- Advisor must retain its existing single-column layout with both auxiliary panes hidden.
- Responsive behavior must not create a blank grid column.
- Browser acceptance must verify both Overview and at least one non-Overview page.

## Evaluation Plan

- Add focused HTML assertions for active-section layout rules.
- Run `go test ./internal/report`.
- Run full `make check`, `make build`, JavaScript syntax validation, and `git diff --check`.
- In Recorded live UI, verify Overview shows Topology Controls and Inspector; verify EKS Network hides Topology Controls while Inspector remains visible.

## Open Questions

None.

## Links

- Implementation: `internal/report/report.html`, `internal/report/report_test.go`
- Related provenance: `.provenance/2026-09-22-eks-project-navigation-and-infrastructure-fixture.md`
