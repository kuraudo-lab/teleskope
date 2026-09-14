# Live Publication Pipeline Provenance

## Trigger

Issue #10 asks Teleskope to rework live publication so Kubernetes watch updates
and EKS polling updates can flow through one shared publication path. The user
accepted the #7 event-based Kubernetes update design and asked to complete #10
with provenance and an implementation commit. This record is a structured
summary, not a raw transcript.

## Scope

This provenance record covers:

- `internal/live` source publication semantics.
- Mixed polling and watch source status representation.
- Revision, retained evidence, and publish hook behavior.
- Compatibility with existing polling `serve` commands and hub remote-write.

Implementation files:

- `internal/live/live.go`
- `internal/live/live_test.go`

## Conversation Summary

The accepted event-based update direction keeps Kubernetes informer mechanics
under `internal/k8s` while `internal/live` remains responsible for publication,
revision, source freshness, ETags, Advisor refresh, and hub publish hooks. Issue
#10 is the bridging refactor before #8 wires real watch events into the runtime.

The implementation therefore should expose a publication API that accepts an
already-built source snapshot, records whether the source was updated by polling
or watch, and merges the retained per-source evidence into the shared live
response. The existing polling loop should keep its current user-visible
behavior by routing its completed collection attempts through the same API.

## Design Diff

- Add a source publication shape for source name, update mode, snapshot,
  collection/watch error, and next attempt metadata.
- Move the old polling-only finish logic behind the shared publication path.
- Preserve source-level stale behavior when a watch or polling update fails.
- Publish hub/live hooks from the shared publication path instead of from the
  polling loop.
- Add tests that simulate watch publications directly, without starting
  Kubernetes collectors.

## Decisions

- Use `internal/live.PublishSource` as the narrow boundary that #8 can call
  after a watch runtime rebuilds a Kubernetes snapshot.
- Track per-source mode as `poll` or `watch` in `live.Status`.
- Derive merged snapshot mode as `live/poll`, `live/watch`, or `live/mixed`
  from the currently retained source snapshots.
- Keep publish hooks best-effort and non-blocking.
- Treat watch failures like polling failures: retain prior evidence and mark the
  source stale instead of clearing the source snapshot.

## Rejected Alternatives

- Let the watch runtime write directly to hub remote-write. This was rejected
  because hub writes must remain downstream of the live publication response.
- Trigger source collectors from the publication API. This was rejected because
  publication must be a pure handoff for already-collected or already-rebuilt
  source state.
- Replace polling behavior as part of #10. This was rejected because #10 is the
  publication refactor; real watch runtime integration belongs to #8.

## Constraints

- Existing `teleskope serve k8s` and `teleskope serve eks` polling behavior must
  remain compatible.
- Kubernetes watch updates must not trigger EKS collection.
- EKS polling updates must not overwrite newer retained Kubernetes watch state.
- Hub envelope revisions must remain monotonic across mixed source updates.
- Stale sources must retain prior evidence and coverage.

## Evaluation Plan

- Run focused `internal/live` tests for direct watch publication, mixed
  watch/poll publication, stale evidence retention, and publish hook revision
  delivery.
- Run full repository checks and build before committing the implementation.
- Manually verify live serve still reports `live/poll` for existing polling
  commands and that browser refreshes do not initiate collection.

## Open Questions

None for #10. Actual informer lifecycle, relist behavior, and CLI exposure are
left to #8 and later follow-up issues.

## Links

- Related issue: #10
- Preceding design issue: #7
- Follow-up issue: #8
- Design file: `docs/event-based-kubernetes-updates.md`
