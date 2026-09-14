# EKS Report Projection Provenance

## Trigger

Issue #5 asked Teleskope to extract EKS report projection so Markdown, embedded HTML, live/hub JSON responses, and terminal output share the same row shaping and visibility semantics. The user accepted the implementation direction and asked to record provenance before committing the implementation. This record is a structured summary, not a raw transcript.

## Scope

This provenance covers EKS report projection for:

- Managed add-ons.
- Upgrade and rollback insights.
- Managed nodegroups.
- Kubernetes capacity rows shown in the EKS report.
- EKS network rows.
- EKS security and identity rows.
- Live and hub snapshot API payloads consumed by the embedded UI.
- Terminal EKS summary output.

Implementation files expected to follow this record:

- `internal/report/eks_projection.go`
- `internal/report/report.go`
- `internal/report/report.html`
- `internal/report/ui_runtime.go`
- `internal/report/live.js`
- `internal/live/live.go`
- `internal/hub/http.go`
- `internal/render/human.go`
- Focused tests under `internal/report`, `internal/live`, and `internal/render`.

## Conversation Summary

The grooming work identified EKS report rendering as a near-term architecture task: EKS-specific sorting, labeling, value display, and visibility rules had drift potential because each renderer shaped rows independently. The accepted direction was to introduce a shared projection object that preserves presentation flexibility while centralizing EKS data interpretation.

During implementation review, partial EKS snapshots were treated as first-class behavior. Insights-only, add-ons-only, and nodegroups-only snapshots should still render meaningful EKS views instead of depending on a fully populated cluster object.

## Design Diff

No standalone design document changed before implementation. The design diff is captured by this provenance record and by the planned implementation commit.

The intended code change is to move EKS row construction into a report projection layer and have renderers consume projected rows. Markdown remains responsible for Markdown syntax, HTML remains responsible for DOM table rendering, and terminal output remains responsible for compact CLI phrasing.

## Decisions

- Create a shared EKS projection in `internal/report` because the existing embedded UI and Markdown report surface already live there.
- Keep renderers presentation-specific: projection owns row shape, display values, ordering, and visibility; renderers own table syntax or compact text.
- Include `eksProjection` in live and hub snapshot responses so the embedded browser does not reimplement EKS row calculation.
- Use a pointer for live/hub `eksProjection` so non-EKS snapshots do not emit an empty object.
- Preserve terminal-specific compact wording where it carries distinct meaning, including nodegroup capacity type, while still sourcing values from the projection.
- Treat partial EKS snapshots as explicit, tested inputs: insights-only, add-ons-only, and nodegroups-only snapshots are visible EKS views.

## Rejected Alternatives

- Keep duplicated EKS shaping in Markdown, HTML, and terminal renderers. This was rejected because it leaves sorting, display labels, and partial snapshot behavior easy to drift.
- Move all rendering syntax into the projection. This was rejected because Markdown, browser DOM, and terminal output have different presentation constraints.
- Emit empty `eksProjection` in every live/hub response. This was rejected to keep Kubernetes-only responses clean.

## Constraints

- The UI remains embedded in the Go binary.
- Provider credentials must not be exposed to the browser.
- Existing user-facing report tables should remain stable unless the projection exposes a clearer field contract.
- Refactoring must be covered by automated tests.
- The implementation commit should include `fix #5` if the work is complete.

## Evaluation Plan

- Add table-driven projection tests for partial EKS snapshots.
- Update HTML/report tests to assert projection data is embedded and consumed.
- Add live response coverage for `eksProjection`.
- Add terminal output coverage for insights-only EKS snapshots.
- Run `make check`.
- Run `make build`.
- Run `git diff --check`.

## Open Questions

None.

## Links

- Related issue: #5
- Related modules: `internal/report`, `internal/live`, `internal/hub`, `internal/render`
