# Web Advisory Provenance

## Trigger

The user requested the Web-side workflow for comparing two cluster scan results after the deterministic CLI comparison capability was implemented. The UI must let a user select two scan snapshots and review migration guidance without introducing an LLM.

## Scope

This record covers the local `teleskope advisory` command, its HTTP API and embedded UI, reuse of the shared comparison module, tests, and user documentation.

## Conversation Summary

The accepted workflow is to scan source and target clusters separately, then compare the resulting snapshots. The Web mode is an advisory view where the user manually chooses the two scan result files. The comparison should use the same report model and deterministic analysis as the CLI.

## Design Diff

The Web implementation adds `internal/advisory/` for the server, embedded HTML, and handler tests; adds `internal/cli/advisory.go` and command registration; extends `internal/compare/compare.go` with reader-based snapshot decoding; updates CLI tests and `README.md` with usage guidance.

## Decisions

- Use the command name `advisory` to distinguish the interactive Web workflow from the CLI `compare` command.
- Start a local server, defaulting to `127.0.0.1:8080`, with `--listen` for local address selection.
- Upload the two snapshot files through the browser because the browser cannot safely read arbitrary local paths.
- Perform decoding and analysis on the Go backend through the shared `compare.Analyze` module so CLI and Web results remain consistent.
- Return both structured JSON and rendered Markdown so the UI can present tabs and support downloads.

## Rejected Alternatives

- Browser-only comparison logic was rejected because it would duplicate the Go comparison rules and risk divergent results.
- Adding this flow to `serve` was rejected because `serve` remains focused on a single live cluster.
- A server-side path or configuration-file workflow was deferred; uploads keep the first version explicit and local.
- LLM-generated advice was deferred to preserve deterministic, inspectable output.

## Constraints

- Existing `serve` and CLI `compare` behavior must remain compatible.
- The advisory server must not require cluster credentials or connect to a cluster.
- Uploaded request bodies are bounded and malformed snapshots return useful HTTP errors.
- Output must preserve the scan snapshot and comparison report formats while fitting the Web presentation.

## Evaluation Plan

- Run `GOCACHE=/private/tmp/teleskope-go-cache go test ./...`.
- Run `git diff --check`.
- Exercise advisory handler tests for validation, decoding, comparison, and response rendering.
- Verify the staged implementation contains only the files in this scope.

## Open Questions

- How advisory results should later be stored or authenticated when integrated with the hub.
- Whether future Web flows should select snapshots from durable server-side storage instead of uploads.

## Links

- Design files: `README.md`, `internal/compare/compare.go`
- Related issues/tasks: None
