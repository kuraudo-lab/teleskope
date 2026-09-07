# Topology Controls and Icon Provenance

## Trigger

The user asked to continue improving the static HTML report topology and to replace the placeholder icon with a PNG logo that visually belongs to the Kuraudo Lab style family.

## Scope

This record covers report UI behavior and project visual assets.

In scope:

- `internal/report/report.html`
- `internal/report/report_test.go`
- `assets/teleskope-icon.png`
- `assets/teleskope-icon-128.png`
- `assets/teleskope-icon-32.png`
- `README.md`

## Conversation Summary

The static HTML report already had an Overview topology, table sections, namespace filtering, and a generated visual report layout. The user requested topology improvements: wheel zoom, a resource-type dropdown replacing the search input, pan support, and disabling the resource filter outside Overview so it does not filter non-topology sections. The user also rejected the first hand-authored SVG logo as visually poor and asked for an image-generated PNG icon that echoes Kuraudo Lab. The Kuraudo Lab GitHub organization avatar was used as style reference: rounded 2D sticker illustration, cream outline, teal background, cloud/lab motif, and coral accents.

## Design Diff

The implementation should update the report template so the Overview topology supports wheel zoom and drag pan through a shared SVG viewport transform. The resource type dropdown should only affect topology while Overview is active; other sections should disable the dropdown and continue filtering only by namespace. The report logo and favicon should use generated PNG assets, with the full icon stored as a project asset and smaller sizes embedded in the self-contained HTML report. Tests should assert the presence of the controls and icon wiring.

## Decisions

- Keep topology interaction dependency-free inside the existing static HTML report.
- Preserve namespace filtering globally, but scope resource-type filtering to Overview topology only.
- Disable the resource-type dropdown when the active section is not Overview to avoid misleading table filtering.
- Use pointer events for pan so mouse and trackpad-like pointer devices share one implementation.
- Suppress topology click after drag movement so panning does not accidentally open detail drawers.
- Replace the previous placeholder/SVG direction with generated PNG assets matching the Kuraudo Lab avatar style family.
- Keep the report self-contained by embedding the small PNG logo and favicon as data URLs.

## Rejected Alternatives

- Keeping the search box as the second toolbar control was rejected after the user asked for resource type filtering via dropdown.
- Applying resource type filtering to all report sections was rejected after the user clarified that non-Overview sections should disable the filter and not be filtered by it.
- Keeping the hand-authored SVG logo was rejected because the user found it visually unacceptable.
- Using the earlier neon 3D generated icon was rejected because it did not match the Kuraudo Lab visual series.

## Constraints

- The static report must remain openable as a local single HTML file without external JavaScript or CSS dependencies.
- Generated scan artifact directories such as `ex-*` must not be staged or committed.
- The icon must not copy third-party marks such as AWS or Kubernetes logos.
- Existing secret redaction behavior remains unchanged.

## Evaluation Plan

- Validate generated PNG assets with `file`.
- Run the report template script through `node --check`.
- Run `go fmt ./...`.
- Run `make check`.
- Run `make build` with a writable Go cache when the sandbox blocks the default user cache.

## Open Questions

- None.

## Links

- Design files: `.provenance/2026-09-07-topology-controls-and-icon.md`
- Related implementation files: `internal/report/report.html`, `internal/report/report_test.go`, `assets/teleskope-icon.png`, `assets/teleskope-icon-128.png`, `assets/teleskope-icon-32.png`, `README.md`
- Related issues/tasks: current Codex task
