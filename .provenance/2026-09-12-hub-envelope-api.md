# Hub Envelope API Provenance

## Trigger

The user asked to begin implementing the multi-cluster hub capability described in `docs/multi-cluster-hub.md`. They clarified that each reviewable part should be closed with a provenance commit and then an implementation commit before continuing to the next part.

## Scope

This record covers the first reviewable hub slice: a hub-owned cluster envelope model, stable identity derivation, in-memory latest-per-cluster storage, HTTP endpoints for collector publication and fleet reads, a basic fleet page with single-cluster drilldown, and a `teleskope serve-hub` CLI entry point.

## Conversation Summary

The implementation starts from the existing single-cluster `serve` model. The hub should not connect to Kubernetes or AWS APIs; collectors continue to own credentials, collection, retries, and per-source state. The first slice creates the hub receiving side so later collector remote-write support has a concrete target.

## Design Diff

No design file changes are included in this provenance commit. The implementation follows `docs/multi-cluster-hub.md`, especially the envelope shape, stable cluster identity requirement, in-memory hub state, selector/drilldown UI, and JSON/Markdown fleet export goals.

## Decisions

- Represent hub input as an envelope containing cluster metadata, monotonic revision, collected time, latest snapshot, advisor report, source statuses, and events.
- Derive the cluster ID from EKS cluster ARN first, then EKS cluster name, then a deterministic Kubernetes server/context hash when no explicit ID is provided.
- Keep hub state memory-only and accept only newer revisions for each cluster ID, making duplicate or older writes idempotent.
- Add optional bearer-token enforcement on writes through `--hub-token` or `TELESKOPE_HUB_TOKEN`; reads remain unauthenticated in this first local/private-network-oriented version.
- Reuse the existing single-cluster static report for drilldown pages instead of cloning that UI into the hub.

## Rejected Alternatives

- Letting the hub connect directly to cluster APIs was rejected because the design keeps credentials and network reachability in collectors.
- Using kubeconfig context alone as the stable identity was rejected because it is machine-local and may differ between operators.
- Adding durable storage was rejected for this slice because the design only requires latest in-memory state.

## Constraints

- Existing `scan` and single-cluster `serve` behavior must remain usable.
- Hub POST writes must not block or trigger collection work.
- Implementation commits must reference this provenance file.
- Tests should not require AWS credentials, kubeconfig access, or a real browser.

## Evaluation Plan

- Run focused Go tests for `internal/hub`, `internal/live`, `internal/report`, and `internal/cli`.
- Verify duplicate and older revisions are ignored.
- Verify hub POST, fleet JSON, Markdown/JSON export, and single-cluster drilldown routes.
- Verify `serve-hub` rejects an invalid listen address before serving.

## Open Questions

- Collector remote-write retry behavior and exact publish trigger will be implemented in a later slice.
- The final retained event-history policy across clusters remains bounded in memory for this first implementation.

## Links

- Design files: `docs/multi-cluster-hub.md`
- Implementation files: `internal/hub/*`, `internal/cli/cli.go`, `internal/cli/cli_test.go`
