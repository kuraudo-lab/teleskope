# Release Smoke Kubeconfig Provenance

## Trigger

The user reported a release pipeline failure in the native smoke test. The
GitHub Actions job ran `python scripts/release.py smoke packages "$VERSION"
"$TARGET_OS" "$TARGET_ARCH"` on a macOS runner and failed because
`teleskope scan k8s --kubeconfig <missing path>` returned exit status 1.

## Scope

This record covers the release archive smoke-test behavior:

- `scripts/release.py`

It does not change the CLI scan contract, GoReleaser archive layout, workflow
triggers, or release publishing policy.

## Conversation Summary

The release pipeline had already built platform archives and reached the native
binary smoke step. The smoke script extracted the expected binary and validated
`--version` and `--help`, then attempted an offline `scan k8s` run using a
missing kubeconfig path. The script expected that command to succeed, print a
`report <path>` line, and write report artifacts.

Current CLI behavior and unit tests say the opposite: a missing kubeconfig is a
Kubernetes connection error, exits with status 1, writes no stdout, and does not
create a report directory. The release smoke should therefore validate that
documented failure behavior rather than expect a successful scan without a real
cluster.

## Design Diff

The smoke helper should return the full `subprocess.CompletedProcess` so tests
can inspect exit status, stdout, and stderr. The version and help checks should
continue to require successful commands. The missing-kubeconfig scan should run
with `check=False` and assert the expected error contract:

- exit code 1
- empty stdout
- stderr includes Kubernetes connection and kubeconfig context
- no report directory is written

The unused JSON import should be removed because the smoke test no longer reads
report artifacts from a failed scan.

## Decisions

- Keep the release smoke fully offline; do not require a live Kubernetes cluster
  or kubeconfig secret for release verification.
- Align the smoke test with `internal/cli` tests instead of changing the CLI to
  produce a report for failed Kubernetes connection setup.
- Keep checksum and archive-set verification unchanged, because the packaging
  contract is separate from the CLI smoke command.
- Validate the expected failure explicitly so a future accidental change to the
  missing-kubeconfig contract fails in CI with a useful message.

## Rejected Alternatives

- Do not add a real kubeconfig or cluster dependency to release jobs. That would
  turn binary packaging into an environment-sensitive integration test.
- Do not make `scan k8s` silently succeed when it cannot load kubeconfig. That
  would weaken the CLI contract and hide configuration errors.
- Do not skip the `scan k8s` smoke entirely. It still proves the release binary
  can execute a real CLI path and report a meaningful runtime error.

## Constraints

- Release smoke must run on all six native targets without cloud credentials.
- Stable release publishing must remain blocked unless every native smoke job
  passes.
- The command must avoid writing generated report artifacts when kubeconfig
  loading fails.
- The implementation commit should reference this provenance record.

## Evaluation Plan

- Run `go test ./...`.
- Generate snapshot release archives with GoReleaser.
- Run `python3 scripts/release.py smoke dist <snapshot-version> darwin arm64`
  on the local native platform.
- Run `python3 scripts/release.py verify dist <snapshot-version>`.
- Run `actionlint` and `git diff --check`.
- Push the implementation commit and create a new stable tag so GitHub Actions
  tests the same fix on all six native runner targets.

## Open Questions

- None.

## Links

- Design files: none
- Related issues/tasks: current Codex task
