# Cluster Advisor Provenance

## Trigger

The user asked whether Teleskope could explain cluster capabilities, including
Ingress versus Gateway and RWX storage, to support eventual migration gap
analysis. The accepted first step was an Advisor summarizing the monitored
cluster. The user then requested implementation, a padding correction for the
Cluster capabilities panel, and separate provenance and implementation commits.

## Scope

`internal/advisor`, live snapshot publication, offline JSON/Markdown/HTML reports,
the Advisor panel and documentation. Migration comparison is outside this version.

## Conversation Summary

This is a structured summary, not a raw transcript. It is recorded retrospectively:
the implementation and initial validation already exist in the working tree.
The provenance commit precedes the implementation commit, not implementation work.
The initial discussion proposed a broader capability model and eventual provider
rules and probes; the implemented first slice uses existing collected facts only.

## Design Diff

- `docs/advisor.md` documents capability semantics, integration and limitations.
- `docs/architecture.md` identifies read-only capability analysis as implemented.
- Separate derived analysis from the raw snapshot schema.

## Decisions

- Use a deterministic Go analyzer with no cluster access or LLM dependency.
- Scope capability records to classes or custom APIs, with stable keys and rules.
- Keep assessment, evidence basis, coverage and freshness distinct.
- Class presence establishes declared configuration, not operational readiness.
- Require mutually matching Bound PVC/PV objects declaring RWX for observed
  binding evidence; do not claim actual multi-node read/write verification.
- Unknown drivers, absent resources and incomplete coverage remain unknown.
- Expose evidence, constraints and collection errors in expandable details.
- Publish advisor data with live snapshots; status-only failures mark conclusions
  stale even without a data revision change.
- Keep the Advisor cluster-wide and disable irrelevant namespace filtering.
- Use the existing panel body padding: 18px horizontally and 16px vertically,
  with long-text wrapping.

## Rejected Alternatives

- Boolean-only capability flags would hide uncertainty and evidence strength.
- Treating absent objects as unsupported would misclassify partial scans.
- Provider-name heuristics without configuration/version rules are not included.
- Frontend-only analysis would diverge from offline and API results.
- Active probes and migration matching are deferred beyond this read-only slice.

## Constraints

- Preserve raw inventory and existing unrelated behavior.
- No external service calls or cluster mutations in the analyzer.
- Retained live evidence keeps its collection timestamps.
- Configuration, binding and API declarations are not health guarantees.

## Evaluation Plan

- Test unknown/denied/empty inventory and matching versus mismatched RWX bindings.
- Test coexisting unused route classes, determinism and input immutability.
- Test live freshness changes without a snapshot revision change.
- Verify advisor artifacts, embedded HTML and live placeholder handling.
- Run Go tests, vet, build, JavaScript syntax checks and diff whitespace checks.
- Review the panel visually when browser access is available.

Validation already performed: full Go suite, vet, build and JavaScript syntax
checks passed; report tests also passed after the padding correction. Browser
visual verification was blocked by an automatic approval timeout for localhost.
No real cluster or active runtime capability verification was performed.

## Open Questions

- Which controller/provisioner implementations should receive versioned rules next?
- How should Gateway readiness and actual route feature support be incorporated?
- When should workload requirement extraction and migration matching be added?

## Links

- Design files: `docs/advisor.md`, `docs/architecture.md`
- Implementation: `internal/advisor`, `internal/live`, `internal/report`
- Related issues/tasks: current user conversation; no external issue supplied.
