# Live Terminal Logging Provenance

## Trigger

The user reported that `teleskope serve` had almost no terminal logs and asked
for richer output to improve operability. The user asked to implement only the
terminal logging first and wait for manual validation before adding richer web
logs.

This provenance record is retrospective: the implementation had already been
made before this file was created, because the provenance request came after the
fix.

## Scope

This record covers terminal logging for the live polling server in
`internal/cli/serve.go` and refresh lifecycle logging in `internal/live/live.go`,
plus focused tests in `internal/live/live_test.go`.

## Conversation Summary

The existing one-shot scan path already used collector progress callbacks, and
the Kubernetes and EKS collectors already emitted detailed progress messages.
The live serve path did not wire those callbacks into stderr, so it only showed
the URL and a few explicit error messages.

The accepted first step was to improve terminal output only. Web-side log
storage, API shape, and UI rendering remain a later step after validation.

## Design Diff

- `internal/live/live.go`: add an optional per-source log callback and log each
  refresh attempt start and completion.
- `internal/live/live.go`: include refresh state, duration, next attempt time,
  non-complete coverage count, and errors in lifecycle logs.
- `internal/cli/serve.go`: connect Kubernetes and EKS collector progress
  callbacks to stderr with source prefixes.
- `internal/cli/serve.go`: log live server startup configuration and shutdown.
- `internal/live/live_test.go`: cover refresh lifecycle log emission.

## Decisions

- Keep logging human-readable on stderr for the first version, matching the
  existing scan progress behavior.
- Put attempt lifecycle logging in `internal/live.Store` so every live source
  gets consistent start and finish logs.
- Reuse the collectors' existing progress callbacks instead of duplicating scan
  step logging in the serve command.
- Leave web logs out of this change so the user can validate terminal behavior
  first.

## Rejected Alternatives

- Add web log APIs and UI in the same change; this would expand the surface
  before terminal logging has been validated.
- Introduce a structured logging dependency; the current CLI only needs simple
  stderr output and already has progress callbacks.
- Log only failures; the operability gap includes successful scans, durations,
  and next refresh timing.

## Constraints

- The live server must remain read-only.
- Logging must not expose credentials, kubeconfig contents, token values, or
  Kubernetes Secret data.
- Source refreshes remain independent and non-overlapping.
- Existing `scan` command output should not change.

## Evaluation Plan

- Run focused live and CLI tests.
- Run the full Go test suite.
- Manually validate terminal output with `teleskope serve k8s` or
  `teleskope serve eks`.

## Open Questions

- Whether the later web log view should use the existing `/api/snapshot` polling
  envelope or a separate streaming endpoint.

## Links

- Design files: none
- Implementation files: `internal/cli/serve.go`, `internal/live/live.go`,
  `internal/live/live_test.go`
- Related issues/tasks: user-requested live serve terminal operability logs
