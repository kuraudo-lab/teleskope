# Hub Fleet UI Docs Provenance

## Trigger

After the hub receiver and collector remote-write path were implemented, the next reviewable part is making the hub understandable and usable from the browser and README without relying on the design document.

## Scope

This record covers hub fleet UI improvements, fleet export content, recent cross-cluster event display, and README documentation for starting the hub and connecting collectors.

## Conversation Summary

The design asks for a selector, fleet inventory and health, source status, recent events, single-cluster drilldown, and JSON/Markdown multi-cluster export. The first hub page already listed clusters; this slice makes the first browser and docs experience closer to that design by exposing source health and recent events at fleet level.

## Design Diff

No design file changes are included in this provenance commit. README gains first-version usage and limitation details for `serve-hub`, `--hub-url`, `TELESKOPE_HUB_URL`, `--hub-token`, `TELESKOPE_HUB_TOKEN`, and stable cluster IDs.

## Decisions

- Keep the hub UI as a quiet operational table rather than a marketing page.
- Add source-state badges per cluster so operators can quickly distinguish ready, partial, stale, and error sources.
- Add a bounded recent-events table to the fleet response and Markdown export for optional all-cluster event review.
- Document that the hub is memory-only and credential-free, and that collector writes are best-effort with newest-envelope retry.

## Rejected Alternatives

- Building a separate deep single-cluster UI inside the hub was rejected because the existing report UI already supports drilldown.
- Adding user accounts or browser authentication was rejected for this first version; token support covers write authorization only.
- Recording durable event history was rejected because the design's first hub version keeps state in memory.

## Constraints

- The hub page must remain self-contained and not require a frontend build step.
- Existing report UI and live UI should not change behavior.
- Documentation should be enough to run the hub locally or behind a private boundary.

## Evaluation Plan

- Run focused Go tests for `internal/hub`, `internal/live`, `internal/report`, and `internal/cli`.
- Verify fleet summaries include event counts and fleet events.
- Verify Markdown export includes the recent events section when events exist.
- Review README commands for consistency with implemented flags and environment variables.

## Open Questions

- Future UI work can add richer filters or comparisons across clusters once users exercise the first hub flow.

## Links

- Design files: `docs/multi-cluster-hub.md`
- Documentation: `README.md`
- Implementation files: `internal/hub/*`
