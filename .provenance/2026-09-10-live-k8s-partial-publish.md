# Live Kubernetes Partial Publish Provenance

## Trigger

The user reported a bug in Kubernetes collection: when any collection step hit an error, the web page could not display any Kubernetes content, even if earlier steps had already collected useful data. The user also suggested increasing the collection context timeout from 2 minutes to 5 minutes and lengthening the Kubernetes polling interval.

## Scope

This record covers live Kubernetes publication behavior, the `serve` command's Kubernetes source wrapper, default collection timeouts, default Kubernetes polling interval, tests, and documentation updates.

## Conversation Summary

The diagnosis found that the live store already retained stale data after failed refreshes, but did not publish a new snapshot when a collector returned both partial data and an error. The `serve k8s` wrapper also converted any Kubernetes collector error into a nil snapshot, which meant partial collection results could not reach the live store. The accepted behavior is to keep the UI usable with partial Kubernetes data, mark the source as partial, preserve the error for visibility, and avoid publishing snapshots that contain only connection metadata and no collected data. Defaults should also give larger clusters more time and reduce polling pressure.

## Design Diff

`internal/live/live.go` changes publication semantics so a non-nil snapshot with an attempt error can be cloned and published as partial when it is not a coverage regression. `internal/cli/serve.go` builds a live Kubernetes snapshot before checking errors and only drops it when no real Kubernetes data or coverage was collected. `internal/cli/cli.go` centralizes the default collection timeout at 5 minutes. `README.md` and `docs/live-probe-research.md` are updated to describe the 5-minute Kubernetes serve interval and timeout. Tests cover partial publication and the new defaults.

## Decisions

- Treat `snapshot + err` as publishable partial data when the snapshot includes actual Kubernetes collection output or coverage.
- Keep the existing stale-data protection for coverage regressions, so a later partial refresh cannot replace previously complete source data for an observed resource.
- Preserve the collection error in source status after publishing partial data.
- Do not publish snapshots that only contain kubeconfig context/server metadata without collected resources or coverage.
- Change default collection timeout to 5 minutes and default `serve k8s` polling interval to 5 minutes.

## Rejected Alternatives

- Returning nil on every Kubernetes collector error was rejected because it hides usable partial data from the web UI.
- Publishing any non-nil Kubernetes struct was rejected because a failed connection can still include context metadata but no collected inventory.
- Shortening polling by default was rejected because full-cluster list collection can be expensive and slow on larger clusters.

## Constraints

- The web UI should continue using the same `/api/snapshot` contract.
- Existing successful empty lists must still be valid replacements.
- Existing stale retention behavior must remain for failed refreshes and coverage regressions.
- Tests should exercise the actual live store publication seam.

## Evaluation Plan

- Run the focused live regression test for partial snapshot publication.
- Run the focused CLI defaults and Kubernetes data detection tests.
- Run `GOCACHE=/private/tmp/teleskope-go-cache go test ./...`.
- Run `git diff --check`.
- Inspect staged changes before the implementation commit.

## Open Questions

- Whether the Kubernetes collector should return aggregate non-fatal errors more explicitly in a future API instead of relying on `snapshot + err` semantics.

## Links

- Design files: `internal/live/live.go`, `internal/cli/serve.go`, `internal/cli/cli.go`, `README.md`, `docs/live-probe-research.md`
- Related issues/tasks: None
