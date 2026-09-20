# Report Advisory Page Implementation Memory

## Target

The embedded Advisory page in `internal/report/report.html`, its live refresh
path in `internal/report/live.js`, and focused report rendering tests.

## Summary

The current page is visually minimal, but it preserves several important
semantics: the advisor remains deterministic and cluster-wide, live status-only
updates can refresh conclusions without a snapshot revision change, uncertainty
must not be collapsed into a boolean, and every conclusion retains its basis,
coverage, freshness, constraints, collection status, and raw evidence.

## Findings

### MEM-001: Advisor rendering is independently live-updated

Category: operational
Location: `internal/report/live.js:203-204`
Observation: Live polling replaces `advisorReport` and calls `renderAdvisor()`
before applying a changed snapshot revision.
Likely Reason: Source status changes can make retained evidence stale even when
the inventory revision is unchanged.
Risk If Removed: The page could display a current-looking conclusion after its
source became partial, paused, stale, or unavailable.
Preserve As: requirement
Suggested Clause: Advisory rendering must remain independently refreshable from
`data.advisor`, without requiring a changed snapshot revision.
Suggested Evaluation: Verify a changed advisor payload rerenders while the
snapshot revision remains unchanged.

### MEM-002: Namespace and resource filters do not scope Advisory

Category: product
Location: `internal/report/report.html:1418`, `syncResourceFilterState`
Observation: The page explicitly states that the assessment is cluster-wide and
disables unrelated namespace and resource filters.
Likely Reason: Capabilities and their evidence span cluster-scoped and
namespaced resources and must not imply a narrower analysis than the backend
performed.
Risk If Removed: Users may mistake a cluster-wide conclusion for a filtered
namespace or resource assessment.
Preserve As: requirement
Suggested Evaluation: Select Advisory and assert both global selectors are
disabled while Advisory-local filters remain usable.

### MEM-003: Assessment dimensions are intentionally independent

Category: product
Location: `internal/advisor/model.go`, `docs/advisor.md`
Observation: Assessment, basis, coverage, freshness, constraints, collection
status, and evidence are separate fields.
Likely Reason: A declared capability is not a runtime guarantee; missing or
partial evidence cannot establish lack of support.
Risk If Removed: The UI could overstate confidence, hide coverage gaps, or
misrepresent an unknown as unsupported.
Preserve As: requirement
Suggested Evaluation: Render supported, unsupported, unknown, and partial
coverage examples and verify every dimension remains inspectable.

### MEM-004: Open capability state survives rerenders

Category: product
Location: `internal/report/report.html:1951-1955`
Observation: `renderAdvisor()` records open capability keys and restores them
after replacing the rendered list.
Likely Reason: Frequent live refreshes should not collapse the item a user is
actively reading.
Risk If Removed: Live polling repeatedly interrupts evidence review.
Preserve As: requirement
Suggested Evaluation: Open a capability, update `advisorReport`, rerender, and
assert the same capability remains expanded when it still exists.

### MEM-005: Advisor JSON stays the source of truth

Category: data-safety
Location: `internal/report/report.go`, `internal/report/html.go`,
`internal/report/report_test.go:530-555`
Observation: Offline and live pages receive the same encoded advisor schema; the
HTML is a renderer, not an analyzer.
Likely Reason: Deterministic Go rules must remain consistent across JSON,
Markdown, offline HTML, and live HTML.
Risk If Removed: Frontend-derived conclusions could diverge from exported
artifacts and API responses.
Preserve As: constraint
Suggested Evaluation: Keep report placeholder/encoding tests and avoid deriving
new capability conclusions in JavaScript.

## Unknowns

- Capability records currently have constraints but no dedicated remediation or
  recommendation field. The UI should present constraints as verification
  boundaries rather than inventing actions.
- Assessment and coverage can overlap. Counts and filters must not imply they
  are mutually exclusive.
