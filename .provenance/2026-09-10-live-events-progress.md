# Live Events Progress Provenance

## Trigger

The user asked to improve the web UI by moving the live `events` section into its own navigation menu named `Events`. They also requested that collection progress should appear inside each subpage as an icon plus text, and then refined the in-progress icon to be animated with a spinner. After review, the user pointed out that the arrow-like spinner glyph looked visually tilted, so the implementation should use a cleaner dynamic indicator.

## Scope

This record covers the live web report UI in:

- `internal/report/report.html`
- `internal/report/live.js`
- `internal/report/html.go`
- `internal/report/report_test.go`

The scope is limited to live report navigation, progress status presentation, live event placement, and tests for those UI artifacts.

## Conversation Summary

The earlier live page showed event and collection details as a top-level status block. The accepted direction is to make `Events` a first-class menu item for the live web UI while keeping per-section progress visible near the content a user is viewing. The progress display should be compact, using an icon and concise text rather than a bulky event/details panel.

After the first dynamic icon pass used a rotating glyph, the user reported that the arrow style was visually wrong because the glyph appeared tilted. The chosen correction is a CSS ring spinner with no text glyph while collection is in progress. Terminal states continue to use simple text icons for success, partial/stale state, errors, and pause.

## Design Diff

- `internal/report/report.html`: add an `Events` section containing the live event list, include the `Events` navigation item only when rendering a live page, and disable namespace filtering on the events page.
- `internal/report/live.js`: create a progress row inside each report section, render live source status into those rows, move event rendering into the Events section, and add pause controls through the toolbar.
- `internal/report/html.go`: inject live-only styles for section progress rows, event list rows, and a CSS-based animated spinner.
- `internal/report/report_test.go`: verify the live Events page and progress artifacts exist, and verify the removed top-level status/details artifacts do not come back.

## Decisions

- Add `Events` only for live reports because static reports do not have the streaming event feed this menu represents.
- Keep progress text on every section so users can understand collection state without leaving the current page.
- Use a CSS border spinner for the in-progress state because it avoids font-dependent glyph alignment problems.
- Keep event rows monospaced and scrollable because the event feed is diagnostic information and can grow quickly during collection.
- Keep pause/resume as a toolbar action because it controls page updates globally rather than belonging to a single section.

## Rejected Alternatives

- Keep events in a top-level expandable status block. This was rejected because the user requested a separate `Events` menu and lighter per-page progress.
- Show collection details and events on every page. This was rejected because it would make every section noisy during collection.
- Use a rotating arrow glyph for the loading state. This was rejected after review because the glyph looked visually tilted in the UI.

## Constraints

- Live HTML is produced by injecting script and CSS into the static report template, so live-only styles and behavior need to remain compatible with `LiveHTML()`.
- The default static report should not gain a user-visible Events menu that depends on live collection events.
- The implementation should preserve same-origin live polling behavior and must not trigger scans from the browser.
- Tests should guard against regressions where the old top-level `live-status`, `live-events`, or `Collection details` UI returns.

## Evaluation Plan

- Run `go test ./...` with a writable Go cache.
- Run `git diff --check`.
- Manually inspect the HTML/JS diff to confirm event rendering is scoped to the Events section and the spinner is CSS-based.

## Open Questions

- None.

## Links

- Design files: `internal/report/report.html`, `internal/report/live.js`, `internal/report/html.go`, `internal/report/report_test.go`
- Related issues/tasks: local conversation request
