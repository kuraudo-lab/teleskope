# Scan Progress Output Provenance

## Trigger

The user asked for more detailed terminal progress output while scanning clusters and suggested borrowing visual elements from the Mole project.

## Scope

This record covers terminal progress reporting for `scan eks` and `scan k8s`, including CLI progress formatting and collector-level progress callbacks.

Covered implementation files:

- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/awseks/collector.go`
- `internal/k8s/collector.go`

## Conversation Summary

The scanner already printed a few coarse progress messages such as collecting EKS inventory, collecting Kubernetes inventory, and writing report artifacts. The user wanted more useful progress during cluster scans, especially because AWS and Kubernetes API calls can take noticeable time and the previous terminal output did not show what the tool was doing inside each phase.

The chosen direction is a lightweight terminal-compatible progress stream rather than a full TUI or spinner. Collectors emit detailed progress events, and the CLI renders them with a compact visual hierarchy inspired by Mole-style terminal presentation: numbered primary steps, indented detail lines, and a final success marker.

## Design Diff

The implementation should add optional progress callbacks to AWS EKS and Kubernetes collector options. The collectors should call those callbacks before and after meaningful sub-steps, including AWS config load, caller identity, EKS cluster description, add-ons, nodegroups, insights, access entries, Pod Identity, kubeconfig load, Kubernetes client creation, server version, API resource discovery, core resources, CRDs/extensions, workloads, networking, storage, runtime classes, RBAC, policy objects, and running image indexing.

The CLI progress renderer should format major phases as numbered lines, detail events as indented branch lines, and completion as a check-mark line. The progress output should remain plain text so it works in normal terminals, CI logs, and redirected stderr.

## Decisions

- Use plain stderr progress output instead of a live TUI dependency, because scans are blocking API calls and the current CLI is run-once oriented.
- Add optional `Progress func(format string, args ...any)` callbacks to collector options so collectors can report internal phases without depending on CLI packages.
- Keep major CLI phases separate from collector detail lines to make the output scannable.
- Include object counts after each collection group so users can see useful intermediate results before the final report is written.

## Rejected Alternatives

- Do not add a spinner-only indicator. It would show liveness but not explain which API group is being collected.
- Do not introduce a full-screen TUI for scan progress in this step. It would complicate CI/log output and is unnecessary for run-once scanning.
- Do not write progress to stdout, because stdout is reserved for machine-readable output or the final report path.

## Constraints

- Progress must not change the JSON, human, or report output written to stdout.
- Progress callbacks must be optional and safe for tests or library callers that do not need terminal output.
- The scanner remains read-only.
- The output should stay readable on terminals that do not support advanced cursor control.

## Evaluation Plan

- Add unit coverage for the progress renderer format.
- Run `go test ./...` and `go vet ./...` through `make check`.
- Run `make build`.
- Keep the existing HTML report JavaScript syntax check green because this work is based on a working tree that also contains report-generation code.

## Open Questions

- Whether a future interactive mode should add an actual TUI progress dashboard with elapsed time, retry state, and per-resource latency.

## Links

- Design files: none
- Related issues/tasks: current Codex task
