# Hub UI Drilldown Analysis Provenance

## Trigger

The user reported two acceptance gaps after the initial hub implementation: the hub cluster-list page looked visually different from the existing `serve` UI, and hub cluster drilldown pages were missing AI Analysis.

## Scope

This record covers aligning the hub fleet page with the existing report/live visual system and enabling cluster-scoped AI Analysis from hub drilldown pages without making the hub collect cluster data.

## Conversation Summary

The existing hub drilldown reused the offline report HTML, which explains why the AI Analysis button was absent. The existing live UI already supports analysis through `/api/analyze`, but its API paths were hardcoded for single-cluster `serve`. The fix is to make the report live HTML accept API path overrides, then serve hub drilldown as a read-only live-style page backed by the hub's stored envelope for that cluster.

## Design Diff

No design file changes are included in this provenance commit. The implementation remains consistent with `docs/multi-cluster-hub.md`: hub reads stored envelopes only, browser requests never trigger scans, and selected-cluster pages preserve the existing single-cluster navigation.

## Decisions

- Restyle `HubHTML` with the same sidebar, header, cards, panels, background tokens, and responsive layout patterns used by the report/live UI.
- Add configurable paths to `report.LiveHTMLWithOptions` so the same live UI can work for both normal `serve` and hub drilldown.
- Serve `/cluster?id=<cluster-id>` as a live-style report page whose snapshot, export, and analysis calls are scoped to that cluster ID.
- Cache hub AI Analysis per cluster revision, matching the live-server behavior that avoids repeated model calls for unchanged data.

## Rejected Alternatives

- Keeping hub drilldown on the offline HTML path was rejected because it cannot expose AI Analysis.
- Duplicating the live JavaScript for hub drilldown was rejected because path overrides keep one report UI implementation.
- Adding hub-side collection was rejected because hub remains a credential-free receiver.

## Constraints

- Existing `report.LiveHTML()` behavior for `teleskope serve` must remain unchanged.
- Hub drilldown analysis must use the stored snapshot and must not trigger collection.
- The hub fleet page remains self-contained with no frontend build step.
- Tests should not call a real LLM provider.

## Evaluation Plan

- Run focused Go tests for `internal/hub`, `internal/report`, `internal/live`, and `internal/cli`.
- Verify hub drilldown HTML contains the Analyze with AI control and cluster-scoped analysis path.
- Verify `/api/cluster/snapshot` returns a live-compatible response for a selected cluster.
- Verify `/api/cluster/analyze` uses the injected analyzer and caches by revision.

## Open Questions

- None.

## Links

- Design files: `docs/multi-cluster-hub.md`
- Implementation files: `internal/hub/html.go`, `internal/hub/http.go`, `internal/hub/store.go`, `internal/report/html.go`
