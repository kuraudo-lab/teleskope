# Advisory Serve UI Alignment Provenance

## Trigger

The user reviewed the newly added `teleskope advisory` web page and pointed out that its visual style differed too much from the existing `serve` UI. The requested direction is for advisory to align closely with serve and keep the two web experiences visually consistent.

## Scope

This record covers the advisory web page markup, CSS, client-side interactions, and focused tests that protect the shared visual structure.

## Conversation Summary

The earlier advisory implementation provided the required source/target snapshot upload workflow and comparison views, but used its own sidebar, panels, controls, and theme styling. The accepted adjustment is to make advisory follow the serve report layout: same app shell, brand/sidebar pattern, nav buttons, top toolbar, theme toggle behavior, export dropdown behavior, panel/card/table treatment, and responsive structure. The comparison backend and deterministic report logic remain unchanged.

## Design Diff

The implementation updates `internal/advisory/advisory.html` to adopt the same visual tokens and structural conventions as `internal/report/report.html`. It keeps advisory-specific upload inputs and comparison result sections while replacing the standalone advisory styling with the serve-style dashboard shell. `internal/advisory/advisory_test.go` gains assertions for the shared app, brand, nav, toolbar, export, and theme elements.

## Decisions

- Align advisory to serve through shared DOM and CSS conventions rather than changing Go handlers or comparison logic.
- Keep the advisory workflow centered on manual source and target snapshot uploads.
- Use one `Export` button with a dropdown menu for JSON and Markdown, matching the serve export interaction.
- Reuse the `teleskope.theme` localStorage key and system-theme fallback behavior from serve.
- Preserve advisory-specific sections: Overview, Findings, Capabilities, Inventory, Coverage, and Markdown.

## Rejected Alternatives

- Leaving advisory with an independent visual system was rejected because it makes the product feel inconsistent.
- Refactoring serve and advisory into a shared template was deferred; the current codebase keeps these pages as embedded HTML files.
- Changing the advisory backend response format was rejected because the issue is UI consistency, not compare output semantics.

## Constraints

- Existing `teleskope advisory` behavior and `/api/compare` contract must remain compatible.
- The change should remain local to advisory UI and tests.
- Theme and export behavior should remain consistent with serve.
- Tests should stay lightweight and focused on structural regressions.

## Evaluation Plan

- Run `go fmt ./internal/advisory`.
- Run `git diff --check`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache go test ./...`.
- Inspect the staged diff to confirm only advisory UI, advisory tests, and this provenance record are included.

## Open Questions

- Whether future maintenance should extract shared web UI tokens/templates so serve and advisory cannot drift.

## Links

- Design files: `internal/advisory/advisory.html`, `internal/report/report.html`
- Related issues/tasks: None
