# Web UI Theme Toggle Provenance

## Trigger

The user asked to add a web UI theme toggle with two requirements: default to the system color scheme and persist a user-modified preference in localStorage.

This provenance record was requested after the implementation was already completed, so it captures the implemented intent and constraints as a structured summary rather than a pre-implementation transcript.

## Scope

This record covers the embedded Teleskope report UI and live web UI behavior:

- `internal/report/report.html`
- `internal/report/html.go`
- `internal/report/report_test.go`

## Conversation Summary

The existing UI is a single embedded HTML template with inline CSS and vanilla JavaScript. The implementation adds light and dark theme variables directly to that template, initializes from a stored preference before CSS loads, falls back to the browser system preference when no user preference exists, and exposes a toolbar button that toggles the theme.

The live UI injects additional CSS from Go, so its event row colors were moved to shared CSS variables to keep the live page consistent with the selected theme.

## Design Diff

- `internal/report/report.html`: adds early theme initialization from `localStorage`, light and dark CSS variable sets, a toolbar theme button, and JavaScript that writes the selected theme to `localStorage`.
- `internal/report/html.go`: changes live event row styling to use shared theme variables.
- `internal/report/report_test.go`: adds string-level coverage for theme initialization, persistence, and live event styling.

## Decisions

- Default theme follows `prefers-color-scheme` when no explicit preference is stored.
- User preference is stored under `teleskope.theme` with only `dark` and `light` accepted as valid values.
- The stored preference is applied on `document.documentElement.dataset.theme` before the stylesheet loads to avoid an initial wrong-theme render.
- Theme CSS uses variables so static reports and live pages share the same visual contract.

## Rejected Alternatives

- Adding a frontend build step was rejected because the existing report UI is a self-contained embedded HTML file.
- Keeping hard-coded dark UI surfaces was rejected because it would make light mode incomplete.
- Adding a reset-to-system mode was left out because the request only required dark/light switching and persistence after user modification.

## Constraints

- Offline static reports must remain self-contained.
- Live UI CSS injected from Go must respect the same theme variables.
- The implementation must tolerate browsers or contexts where `localStorage` is unavailable.
- Existing namespace and resource filters must continue to work.

## Evaluation Plan

- Run `go fmt ./internal/report`.
- Run `go test ./...` with a writable Go cache in `/private/tmp`.
- Run `git diff --check`.
- Manually review the generated HTML diff for default system theme behavior and localStorage persistence.

## Open Questions

None.

## Links

- Design files: `internal/report/report.html`, `internal/report/html.go`, `internal/report/report_test.go`
- Related issues/tasks: current Codex task
