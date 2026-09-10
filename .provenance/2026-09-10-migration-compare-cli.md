# Migration Compare CLI Provenance

## Trigger

The user wants to compare two Kubernetes/EKS clusters before migration planning.
The desired first implementation is to scan the source cluster, scan the target
cluster, then load the two scan results and produce a deterministic comparison
without using an LLM.

The user explicitly accepted the design direction that the core should be a
shared analysis module and that the first delivery should implement the core
module plus CLI entry point only. The web advisory flow will come later.

## Scope

This record covers the first migration comparison implementation:

- `internal/compare`: deterministic comparison report model, snapshot loading,
  analysis rules, and JSON/Markdown/human renderers.
- `internal/cli`: `teleskope compare` command wiring.
- `README.md`: user-facing command documentation.
- Tests for the core comparison module and CLI validation/smoke behavior.

The scope does not include the future web advisory UI.

## Conversation Summary

The user described a practical migration workflow: scan the source cluster, scan
the target cluster, run a CLI comparison over the two scan outputs, and later run
a web advisory mode where users manually choose two scan results in the UI.

The accepted design places the seam at a deterministic comparison module:

```go
compare.Analyze(source *inventory.Snapshot, target *inventory.Snapshot) Report
```

CLI and future web advisory should call the same module so the comparison rules
do not split across Go and browser JavaScript. The CLI should accept either scan
report directories or direct `snapshot.json` paths.

## Design Diff

- `internal/compare/compare.go`: add `Report`, `Finding`, `CapabilityDiff`,
  `InventoryDiff`, and `CoverageWarning` models; implement snapshot loading,
  comparison rules, and human/Markdown/JSON output.
- `internal/compare/compare_test.go`: cover migration-relevant differences,
  report directory loading, and Markdown rendering.
- `internal/cli/cli.go`: register `teleskope compare`, support `--source`,
  `--target`, positional paths, and `--output human|markdown|json`.
- `internal/cli/cli_test.go`: cover CLI loading from report directories and
  validation errors.
- `README.md`: document migration comparison usage and deterministic scope.

## Decisions

- Name the CLI command `compare` because the first shipped behavior compares two
  scan results directly.
- Keep web advisory out of this implementation so the shared comparison module
  can settle before UI work.
- Accept both report directories and `snapshot.json` files to match existing scan
  artifacts and reduce CLI friction.
- Support `human`, `markdown`, and `json` outputs from the start because CLI
  review, migration documentation, and future UI integration need different
  consumers.
- Reuse `internal/advisor.Analyze` inside comparison for capability-level diffs
  instead of duplicating advisor rules.
- Treat incomplete coverage as a first-class warning because missing data should
  not be interpreted as missing capability.

## Rejected Alternatives

- Use an LLM for migration advice in the first version. This was rejected because
  the user wants deterministic analysis for now.
- Put comparison logic in the future web UI. This was rejected because CLI and
  web should share one rule implementation.
- Add comparison to `serve`. This was rejected because `serve` is single-cluster
  live inventory, while this feature compares offline scan results.
- Require users to pass only `snapshot.json` files. This was rejected because
  existing scans produce report directories, and directory input is easier.

## Constraints

- The comparison must not connect to source or target clusters.
- The implementation must preserve existing scan, report, advisor, and serve
  behavior.
- The first comparison rules should be deterministic and evidence-backed.
- Secret values are not collected; secret comparison can only compare metadata,
  type, and keys already present in snapshots.
- Coverage gaps must remain visible in the output.

## Evaluation Plan

- Run `go test ./...` with a writable Go cache.
- Run `git diff --check`.
- Verify CLI tests cover directory input, output rendering, and validation.
- Review JSON/Markdown/human report surfaces for future web advisory reuse.

## Open Questions

- How much of the first comparison report should be displayed in the future web
  advisory UI versus rendered as a downloadable artifact?
- Should future comparison support report bundles from `serve-hub` in addition
  to two explicit scan outputs?

## Links

- Design files: `internal/compare/compare.go`, `internal/cli/cli.go`, `README.md`
- Related design note: `docs/multi-cluster-hub.md`
- Related tasks: local conversation request
