# Static HTML Report Provenance

## Trigger

The user reviewed the interactive static HTML mock for Teleskope cluster inventory and asked to implement the production feature based on that mock.

## Scope

This record covers the default report artifact workflow and the static HTML report surface generated beside `summary.md`.

In scope:

- `internal/report/report.go`
- `internal/report/html.go`
- `internal/report/report.html`
- `internal/report/report_test.go`
- `internal/cli/cli_test.go`
- `README.md`

## Conversation Summary

The report workflow already created a timestamped directory containing raw JSON artifacts and `summary.md`. A mock `index.html` was generated and iterated after UI feedback: fix the loading state, correct namespace filtering for resources such as `kube-system`, remove the Coverage panel, make Topology fill the available main display area, and apply namespace filtering to topology elements. The user then approved the mock direction and requested formal implementation.

## Design Diff

The intended implementation adds a production HTML renderer under `internal/report`, embeds the current snapshot JSON into a self-contained static `index.html`, and updates the default report directory writer to emit that file beside `summary.md`. Tests should verify that default report generation includes the HTML artifact and that the HTML contains the core interactive sections and filtering logic.

## Decisions

- Generate a single self-contained `index.html` with inline CSS, inline JavaScript, and embedded snapshot JSON so the report can be copied, archived, and opened offline.
- Keep `summary.md` as the textual review artifact and add `index.html` as the visual/interactive artifact in the same timestamped directory.
- Preserve the approved mock layout: sidebar navigation, cluster heading, cards, a full main Topology area, and table-based detail sections.
- Exclude a dedicated Coverage panel from the HTML UI based on user feedback, while keeping raw `coverage.json` in the report directory.
- Support namespace and search filtering across tables and topology by deriving namespaces from object fields and nested references.

## Rejected Alternatives

- Serving assets from a CDN was rejected implicitly by the offline report requirement; the artifact should work as a portable local file.
- Keeping HTML as a mock-only artifact was rejected after the user approved the mock and asked for formal implementation.
- Reintroducing Coverage as a visual panel was rejected because the user explicitly asked not to display it.

## Constraints

- Do not collect or reveal Kubernetes Secret values; existing collection remains metadata-only for secrets.
- Do not stage generated scan artifact directories as source changes.
- Keep the implementation in Go and integrate with the existing report package.
- Preserve the existing default output contract: progress on stderr and final report path on stdout.

## Evaluation Plan

- Run `go fmt ./...`.
- Run `make check`.
- Run `make build`.
- Generate a smoke report and verify that `index.html`, `summary.md`, and raw JSON files are produced.
- Check the embedded HTML script syntax with `node --check` when Node is available.

## Open Questions

- None.

## Links

- Design files: `.provenance/2026-09-07-static-html-report.md`
- Related implementation files: `internal/report/report.go`, `internal/report/html.go`, `internal/report/report.html`, `internal/report/report_test.go`, `internal/cli/cli_test.go`, `README.md`
- Related issues/tasks: current Codex task
