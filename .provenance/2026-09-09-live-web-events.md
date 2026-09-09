# Live Web Events Provenance

## Trigger

After validating terminal logging for `teleskope serve`, the user asked to add
the web-side enhancement. The user also requested a small style repair so Node
and other tables no longer have vertical scrollbars and instead expand fully.

This provenance record is retrospective: the implementation had already been
made before this file was created, because the provenance request came after the
fix.

## Scope

This record covers web-facing live operational events and table layout behavior
in:

- `internal/live/live.go`
- `internal/cli/serve.go`
- `internal/report/live.js`
- `internal/report/html.go`
- `internal/report/report.html`
- `internal/live/live_test.go`
- `internal/report/report_test.go`

## Conversation Summary

The accepted design was to add a bounded in-memory event buffer and expose it
through the existing `/api/snapshot` polling envelope, rather than introduce a
new streaming endpoint in the first version. The browser should show recent
events in the existing live status area while preserving the current live page
interaction model.

The style repair should remove vertical scrolling from all table wrappers, not
only the Node table, while preserving horizontal scrolling for wide tables.

## Design Diff

- `internal/live/live.go`: add a structured `Event` type and bounded recent
  event buffer.
- `internal/live/live.go`: publish recent events in the live response envelope
  and update ETags when events change.
- `internal/cli/serve.go`: add startup, shutdown, and collector progress
  messages to the live event buffer.
- `internal/report/live.js`: render a `Recent events` section with time, source,
  level, and message.
- `internal/report/html.go`: inject compact event-list styling for the live UI.
- `internal/report/report.html`: remove vertical max-height scrolling from table
  containers and keep horizontal overflow handling.
- `internal/live/live_test.go` and `internal/report/report_test.go`: cover event
  response behavior, bounded retention, and live HTML injection.

## Decisions

- Reuse `/api/snapshot` for recent events, because the live page already polls
  that endpoint every 5 seconds and status-only changes already update ETags.
- Keep only the most recent 200 events in memory to bound response size.
- Display only the latest 80 events in the browser for a compact first version.
- Use `debug` for collector progress and `info/warn/error` for lifecycle events.
- Let tables expand vertically and keep only horizontal scrolling, matching the
  user's request for fully visible table rows.

## Rejected Alternatives

- Add Server-Sent Events immediately; it would add another endpoint and client
  state path before the simpler polling approach is proven insufficient.
- Store logs unbounded in memory; this would make long-running live serve
  sessions grow without limit.
- Special-case only the Node table; the user requested the same behavior for any
  table that had vertical scrolling.

## Constraints

- The live server remains read-only.
- Event logs must not include credentials, kubeconfig contents, token values, or
  Kubernetes Secret data.
- Web event data is memory-only and rebuilt on restart.
- Existing terminal logging should continue to work.
- Existing report pages should remain self-contained and frontend-only.

## Evaluation Plan

- Run focused tests for `internal/live`, `internal/report`, and `internal/cli`.
- Run the full Go test suite.
- Run race tests for live and CLI packages.
- Manually validate that the live page shows recent events and that table rows
  expand without vertical scrollbars.

## Open Questions

- Whether a future version should add a dedicated `/api/events` SSE endpoint for
  sub-5-second log updates.

## Links

- Design files: none
- Implementation files: `internal/live/live.go`, `internal/cli/serve.go`,
  `internal/report/live.js`, `internal/report/html.go`,
  `internal/report/report.html`, `internal/live/live_test.go`,
  `internal/report/report_test.go`
- Related issues/tasks: user-requested web live logging and table expansion
