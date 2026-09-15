# EKS Add-on Compatibility Provenance

## Trigger

The user asked to start GitHub issue 12 after prioritizing non-AI work:
EKS managed add-on compatibility view.

## Scope

- EKS report projection and rendered artifacts.
- Static/live HTML report tables that consume the EKS projection.
- Deterministic Advisor capabilities for EKS managed add-on health and upgrade
  readiness evidence.
- Existing AWS-side EKS inventory fields: managed add-ons and EKS insights.

## Conversation Summary

Issue 12 asks Teleskope to show each managed add-on's current version, upgrade
path, target Kubernetes compatibility, and health issues. The current code
already collects add-ons and EKS insight `AddonCompatibility` details, and it
already renders a managed add-ons table. This block should connect those pieces
without introducing AI or external lookups.

## Design Diff

- Extend `EKSAddonRow` with target Kubernetes, compatible versions, and
  compatibility recommendation fields derived from EKS insights.
- Feed the same row fields into JSON projection, Markdown summary, terminal
  rendering, and live/static HTML.
- Add deterministic Advisor capabilities for add-on health and compatibility
  evidence, using observed add-on status/issues and AWS-reported insight data.

## Decisions

- Treat AWS EKS insight records as provider-reported evidence, not inferred
  runtime truth.
- Keep missing compatibility data explicit as unknown rather than unsupported.
- Count add-on health issues and non-active status as action-worthy findings.
- Do not call AWS or Kubernetes APIs from report/advisor code; consume the
  snapshot only.

## Rejected Alternatives

- Hard-code EKS add-on version matrices: rejected because it would drift and
  duplicate AWS-owned compatibility knowledge.
- Put compatibility only in the raw Insights table: rejected because the issue
  asks for each add-on's current version and upgrade path in one view.
- Implement the whole EKS upgrade dossier from issue 6 first: rejected because
  issue 12 can be delivered as a smaller verified block.

## Constraints

- Preserve existing EKS projection consumers.
- Keep conclusions evidence-backed and conservative.
- Do not add AI behavior.
- Use focused tests plus full repository verification.

## Evaluation Plan

- Add projection tests for add-on compatibility details merged from insights.
- Add report/HTML tests proving JSON, Markdown, and HTML expose the same fields.
- Add Advisor tests for add-on health/compatibility capabilities.
- Run `go test ./internal/report ./internal/advisor`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`.
- Run the same-cache `make build`.
- Run `git diff --check`.

## Open Questions

- None for this first issue 12 implementation block.

## Links

- Related issue: https://github.com/kuraudo-lab/teleskope/issues/12
