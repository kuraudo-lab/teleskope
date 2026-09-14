# Live Watch Freshness UI Provenance

## Trigger

The user accepted the implementation for issue #11 and asked to close the work with a provenance commit followed by an implementation commit containing `fixed #11`.

## Scope

This record covers the UI and live data contract work needed to make event-based Kubernetes watch freshness visible in the embedded report UI and multi-cluster hub UI.

Covered areas:

- `internal/live`: source freshness fields in live status responses.
- `internal/cli`: Kubernetes watch health publication into live status.
- `internal/report`: embedded live report freshness and event-kind rendering.
- `internal/hub`: fleet and drilldown freshness visibility through cluster source status.

## Conversation Summary

After implementing event-based Kubernetes watch updates, the next useful issue was selected as #11, "Add UI support for event-based freshness". The goal was to make watch-mode behavior understandable during live runs and hub drilldowns, especially when a watch is reconnecting or stale while previously published inventory remains valid.

The implementation direction was to keep the freshness contract in `live.Status` so the same data can be consumed by the local live report, hub remote-write envelopes, fleet summaries, and hub cluster drilldowns. The UI should distinguish watch reconnecting/stale state from full scan failure and classify operational events as collection, watch, publication, or analysis.

## Design Diff

The implementation adds a small source-freshness contract rather than a separate UI-specific endpoint:

- `live.Status` carries `lastEventAt`, `lastFullSyncAt`, and `reconnects`.
- Watch publication copies `k8s.WatchHealth` into `live.SourcePublication`.
- The live report renders source freshness near the page header and annotates recent events with a kind.
- The hub fleet page renders source badges with watch labels and the same freshness fields.

## Decisions

- Store freshness in `live.Status` so hub envelopes and drilldown responses reuse the same schema.
- Name the resync timestamp `lastFullSyncAt` in the public JSON contract to match user-facing UI language rather than the internal watch runtime's `LastRelistAt` name.
- Present stale watch state as `watch reconnecting` so users do not mistake a disconnected watch stream for a failed full scan.
- Keep pause behavior browser-local; pausing page updates does not pause background collection, watch processing, or hub publication.
- Use embedded vanilla JavaScript and CSS only, preserving the current binary-embedded UI architecture.

## Rejected Alternatives

- A separate `/api/freshness` endpoint was rejected because it would duplicate data already served in `/api/snapshot` and complicate hub drilldowns.
- Treating watch disconnects as ordinary collection errors was rejected because it obscures the distinction between stale-but-retained evidence and a failed initial scan.
- A larger UI layout refactor was deferred because #11 is specifically about event-based freshness visibility.

## Constraints

- The UI remains embedded in the Go binary.
- Provider credentials must not be exposed to browser JavaScript.
- Hub cluster drilldowns must remain cluster-scoped.
- Existing polling mode should continue to work without requiring watch-specific fields.
- Automated tests should cover the response contract and embedded UI markers.

## Evaluation Plan

- Run focused tests for `internal/live`, `internal/cli`, `internal/report`, and `internal/hub`.
- Run full `make check`.
- Run `make build` to verify the embedded UI still compiles into the binary.
- Run `git diff --check`.
- Manually verify with `bin/teleskope serve k8s --watch` that the live page displays source freshness and event kind labels.

## Open Questions

- None.

## Links

- Related issue: #11
- Design files: `internal/live/live.go`, `internal/cli/serve.go`, `internal/report/report.html`, `internal/report/live.js`, `internal/hub/html.go`
