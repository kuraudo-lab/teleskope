# Workload Detail Graph Provenance

## Trigger

Issue #16 requested workload detail pages or drawers that expose compute, storage, network, identity, config, and image dependencies. The user accepted the implementation and asked to record provenance before creating the implementation commit.

## Scope

- `internal/report/report.html`
- `internal/report/report_test.go`
- GitHub issue #16

## Conversation Summary

The feature was scoped to the existing embedded HTML report UI rather than a new frontend bundle. The current tables and topology already opened a generic JSON drawer for every resource, so the chosen direction was to keep the drawer model and upgrade workload entries into a structured detail view. A lightweight workload lookup was added inside the Workloads section to satisfy search-entry navigation without preempting the broader fleet search work planned separately.

## Design Diff

- Add a `Workload lookup` panel with a search input and workload result buttons.
- Reuse table rows, topology nodes, and search results to open the same workload detail renderer.
- Replace raw JSON for workload drawer content with grouped dependency rows.
- Keep raw JSON rendering for non-workload resources.
- Expand report fixture coverage with a zero-replica workload that still has template-declared image, service account, PVC, config, secret, and image pull secret dependencies.

## Decisions

- Use a drawer, not a new page, to preserve the current single-file embedded UI and avoid introducing routing or a frontend build step.
- Keep dependency resolution in small browser-side helper functions so the report can work for static offline HTML and live UI snapshots.
- Treat zero-replica workload dependencies as template facts from the existing workload model instead of requiring observed pods or running containers.
- Add workload lookup as a local Workloads-section entry point, leaving global search for a later dedicated issue.

## Rejected Alternatives

- Add a separate SPA route for workload details; rejected because it would add navigation complexity to the embedded report before the UI shell is ready for that split.
- Add a new graph library; rejected because the existing report is dependency-free and the relationship view can be expressed as dense grouped rows.
- Infer dependencies only from running pods; rejected because issue #16 explicitly requires zero-replica workloads to show template dependencies.

## Constraints

- The UI remains embedded in the Go binary through `internal/report/report.html`.
- No Kubernetes credentials or provider secrets are exposed to browser-side code.
- Static offline reports and live reports must share the same rendering surface.
- The change should not alter collector behavior unless the existing inventory model is insufficient.

## Evaluation Plan

- Run `go test ./internal/report`.
- Run `go test ./internal/k8s ./internal/report`.
- Run a JavaScript syntax check over the embedded report scripts.
- Run `git diff --check`.
- Run `make check`.
- Run `make build`.
- Manually verify that workload rows, topology nodes, and workload lookup results open the structured drawer.

## Open Questions

- Whether workload details should eventually graduate from drawer-only to a routable page once the embedded UI is reorganized.

## Links

- Design files: `internal/report/report.html`, `internal/report/report_test.go`
- Related issues/tasks: #16
