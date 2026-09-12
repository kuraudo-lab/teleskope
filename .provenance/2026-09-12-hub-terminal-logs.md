# Hub Terminal Logs Provenance

## Trigger

The user reported that the hub side has no useful terminal logs, creating an operability gap for `teleskope serve-hub`.

## Scope

This record covers adding hub-side operational logging for server lifecycle, remote-write ingestion, authentication failures, stale envelope handling, and cluster-scoped analysis/export requests.

## Conversation Summary

The existing `serve` command already writes collection, remote-write, and analysis events to stderr. The hub handler has a logging hook for analysis, but `serve-hub` does not pass a logger and the remote-write ingest path is silent. The fix should make hub terminal output useful without turning every browser poll into noisy logs.

## Design Diff

No design file changes are included in this provenance commit. The implementation follows `docs/multi-cluster-hub.md` by keeping hub as a receiver and adding observability around hub-owned HTTP operations.

## Decisions

- Use stderr for `serve-hub` operational logs, matching `teleskope serve`.
- Pass a synchronized logger into `hub.HandlerOptions` from the CLI.
- Log server start, shutdown, accepted remote-write envelopes, ignored stale envelopes, authentication failures, invalid payloads, and analysis lifecycle.
- Keep routine fleet polling quiet to avoid noisy terminal output while the dashboard is open.

## Rejected Alternatives

- Logging every HTTP request was rejected because the hub UI polls frequently and would bury meaningful ingestion and analysis events.
- Adding a structured logging dependency was rejected because the existing CLI uses simple stderr logging and the first hub version should stay lightweight.

## Constraints

- Existing hub API behavior and response bodies must remain compatible.
- Logs must not expose bearer tokens.
- Tests should validate log emission without depending on wall-clock exact output.

## Evaluation Plan

- Add focused tests for unauthorized remote writes, accepted envelopes, and stale envelope log lines.
- Run focused Go tests for `internal/hub` and `internal/cli`.
- Run the full project check after committing the implementation.

## Open Questions

- None.

## Links

- Design files: `docs/multi-cluster-hub.md`
- Implementation files: `internal/cli/cli.go`, `internal/hub/http.go`, `internal/hub/store_test.go`
