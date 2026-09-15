# EKS Upgrade Dossier Advisor Provenance

## Trigger

After accepting issue #13, the user asked to continue to the next item. The next highest-priority non-AI open issue is #6, "Add EKS upgrade dossier in Advisor."

## Scope

This record covers the Advisor-facing upgrade dossier for EKS evidence:

- `internal/advisor/advisor.go`
- `internal/advisor/advisor_test.go`
- Existing consumers of `advisor.Analyze`, including report artifacts, compare capability diffs, and the advisory UI

## Conversation Summary

Recent work added Advisor findings for EKS managed add-ons and nodegroup runtime readiness. Issue #6 asks that add-on, AMI, nodegroup, and EKS insight evidence become deterministic upgrade readiness findings and remain conservative when evidence is missing.

The remaining slice should add EKS insight and deprecation-detail findings to the same Advisor report model, so the report layer, comparison layer, and advisory layer can consume them without a parallel data path.

## Design Diff

No separate design document is changed in this provenance commit. The planned implementation diff is:

- Add Advisor capabilities for EKS upgrade/readiness insights.
- Convert EKS insight status into conservative assessments:
  - passing/ok style statuses become supported.
  - warning/error/failing statuses become unsupported when AWS reports an issue.
  - missing or ambiguous status remains unknown.
- Include deprecation details, affected resources, recommendations, and additional info as evidence.
- Keep add-on and nodegroup findings from the prior issues intact.
- Add focused synthetic snapshot tests for healthy, warning/error, and unknown insight cases.

## Decisions

- Use `eks.insight` as the Advisor capability key for EKS upgrade/readiness insight findings.
- Use AWS-reported insight data as `basis=aws-reported`.
- Treat missing insight collection coverage as incomplete, not as proof of no upgrade risk.
- Do not duplicate report projection tables; Advisor findings remain the shared dossier consumed by report, compare, and advisory flows.
- Keep assessments conservative: missing evidence is unknown, not unsupported.

## Rejected Alternatives

- Do not add a separate upgrade dossier JSON artifact outside Advisor for this slice; that would fragment consumers.
- Do not infer deprecation safety from absence of resources unless EKS insight coverage is complete and the insight status is passing.
- Do not revisit add-on compatibility or nodegroup runtime readiness in this issue except to preserve existing behavior.

## Constraints

- Advisor output must be deterministic for synthetic snapshots.
- Compare, analysis, and report consumers should continue to use the same Advisor findings.
- Existing capability keys should remain stable for already-added add-on and nodegroup findings.
- Verification must include focused Advisor tests, full repository checks, build, and whitespace validation.

## Evaluation Plan

- Add Advisor tests for passing, warning/error, and unknown EKS insight cases.
- Verify EKS insight capabilities appear in the report artifact's `advisor.json`.
- Verify existing compare capability behavior can consume the new `eks.insight` key through `advisor.Analyze`.
- Run `go test ./internal/advisor ./internal/report ./internal/compare`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make build`.
- Run `git diff --check`.

## Open Questions

- Whether later issues should add richer UI grouping for EKS upgrade dossier findings beyond the generic Advisor capability rendering.

## Links

- Related issues/tasks: #6
