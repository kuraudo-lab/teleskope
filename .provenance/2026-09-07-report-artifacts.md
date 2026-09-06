# Report Artifacts Provenance

## Trigger

The user refined the default scan behavior for Teleskope. Instead of only
printing a terminal summary or JSON to stdout, a normal scan should create a
timestamped output folder, place raw scan data JSON files in that folder, and
include a complete Markdown summary. The user also asked that cluster scans show
progress status while they run.

## Scope

This record covers the CLI scan output contract and the report artifact writer:

- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/report/`
- `README.md`

It applies to both `teleskope scan eks` and `teleskope scan k8s`.

## Conversation Summary

Earlier implementation supported human and JSON output to stdout. The user then
defined a more useful default for real cluster scans: create a timestamped
directory, save a set of raw JSON files, and generate a well-structured Markdown
summary in the same directory. Progress should be visible during the scan, so
long-running AWS and Kubernetes API collection phases do not look stalled.

## Design Diff

The implementation changes the scan default output format to a report directory.
The report writer creates a unique timestamped folder under the current working
directory or under `--output-dir`. It writes:

- `snapshot.json` as the full raw snapshot.
- `source.json` for source metadata.
- `aws.json` when AWS identity data exists.
- `eks.json` when EKS inventory exists.
- `kubernetes.json` when Kubernetes inventory or Kubernetes coverage exists.
- `coverage.json` for collection completeness.
- `summary.md` for a human-readable review document.

The CLI writes progress status to stderr and prints only the final report path to
stdout. Explicit `--output human` and `--output json` remain available for quick
terminal review and automation pipelines.

## Decisions

- Use `report` as the default output format for scan commands because it matches
  the product goal of preserving complete evidence for offline review.
- Keep `human` and `json` stdout modes as explicit options because they are
  useful during development, smoke testing, and shell pipelines.
- Write progress messages to stderr so stdout can remain machine-readable for
  report path capture or JSON output.
- Create timestamped directories with sanitized target names and collision
  suffixes so repeated scans in the same second do not overwrite previous data.
- Store raw data as JSON first. YAML remains a future output option rather than a
  default artifact format.
- Generate Markdown from the same snapshot model instead of introducing a second
  summary data model.

## Rejected Alternatives

- Writing only one `snapshot.json` file was rejected because the user explicitly
  asked for a series of raw JSON files.
- Making Markdown the only default output was rejected because raw structured
  evidence is required for later HTML, webserver, and migration analysis
  features.
- Printing progress to stdout was rejected because it would corrupt JSON stdout
  and make the final report path harder to consume from scripts.

## Constraints

- Secret values must remain excluded; Secret artifacts are metadata-only through
  the existing Kubernetes collector behavior.
- The scan remains read-only.
- Go version stays at `1.26.0`.
- Report generation should not require a real cluster for basic tests; missing
  kubeconfig coverage should still produce a report directory.
- Existing explicit stdout modes must continue to work.

## Evaluation Plan

- Run `go fmt ./...`.
- Run `make build`.
- Run `make check`.
- Smoke test `scan k8s` with a missing kubeconfig and `--output-dir` to verify
  progress output, report path output, JSON files, and `summary.md`.
- Review `git status --short` to ensure generated smoke-test report directories
  are not staged.

## Open Questions

- Whether YAML should be added as a first-class report artifact beside JSON.
- Whether report directories should use a dedicated default parent such as
  `./teleskope-runs` instead of the current working directory.
- Whether later interactive webserver mode should read `snapshot.json` directly
  or use the split raw JSON files as its primary source.

## Links

- Design files: `README.md`
- Related issues/tasks: none
