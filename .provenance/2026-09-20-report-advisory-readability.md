# Report Advisory Readability Redesign Provenance

## Trigger

The user reported that the current Advisory subpage was difficult to read,
approved an interactive readability mock, and asked for that accepted direction
to replace the existing Advisory page in the current embedded UI.

## Scope

This record covers the Advisory section markup, styling, client-side rendering,
local filtering/search interactions, focused tests, and browser verification in
`internal/report`. Advisor rules, schemas, exports, and live API contracts remain
unchanged.

## Conversation Summary

The accepted mock replaces the current summary plus dense expandable text with a
review-oriented hierarchy: cluster posture first, then assessment and domain
filters, then priority-ordered capability cards. Each card keeps the concise
deterministic conclusion visible and progressively reveals interpretation
boundaries, collection gaps, trace metadata, and evidence. The existing report
shell, theme, export, Events status, live refresh, and cluster-wide scope remain
intact. Sample-only mock content is not copied into production; production copy
is derived from the existing advisor payload.

## Design Diff

- Replace the single generic Advisory panel with an Advisory workspace header,
  posture metrics, local filters/search, and capability cards.
- Derive display labels, counts, priority, and domains from existing capability
  fields without changing deterministic conclusions.
- Keep assessment, coverage, basis, freshness, rule ID, constraints, collection
  status, and evidence visibly distinct.
- Preserve expanded-card state across live advisor rerenders.
- Keep the optional AI analysis panel below the deterministic Advisory surface
  with a clear boundary between the two.

## Decisions

- Treat the first viewport as a review surface, not a report introduction.
- Use `unsupported` as “Needs attention”; keep `unknown` explicit and surface
  incomplete coverage as an independent “Coverage gap” count and filter.
- Let assessment and coverage counts overlap because those source dimensions are
  independent.
- Order cards by review priority: unsupported, incomplete coverage, unknown,
  then supported; preserve deterministic key/scope ordering within a tier.
- Use capability constraints as “Interpretation boundary”; do not synthesize
  remediation that is absent from the advisor schema.
- Use inline SVG only for small functional controls already consistent with the
  embedded single-file report.
- Provide search and filters with contextual result-count updates and semantic
  expanded state; do not rely on color alone.

## Rejected Alternatives

- A cosmetic recolor of the existing `<details>` list was rejected because it
  would not repair information hierarchy or scanability.
- Frontend-generated recommendations were rejected because the advisor payload
  has no recommendation field and the UI must not create new conclusions.
- Changing the Go advisor model was rejected because this task replaces the page,
  not its deterministic analysis contract.
- Removing the AI analysis panel was rejected because it is an existing report
  capability; it remains visually secondary to deterministic findings.

## Constraints

- Preserve `advisorReport` as the sole Advisory data source in both live and
  offline modes.
- Preserve status-only live updates when the snapshot revision is unchanged.
- Preserve cluster-wide semantics and disabled global namespace/resource filters.
- Preserve existing theme, export, Events, navigation, AI-analysis, and embedded
  single-file behavior.
- Keep all user-controlled or collected strings escaped before HTML insertion.
- Maintain readable light and dark themes, keyboard-visible focus, responsive
  reflow, and a contextual result count.
- Do not change advisor JSON, Markdown, or API contracts.

## Evaluation Plan

- Add focused report tests for the new Advisory workspace contracts, filter and
  search controls, status labels, evidence/constraint rendering, and removal of
  the old dense list renderer.
- Run `go test ./internal/report`.
- Validate embedded JavaScript syntax.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`.
- Run the same-cache `make build` and `git diff --check`.
- Serve the recorded EKS fixture, inspect `/api/snapshot`, and verify Advisory in
  the browser in both themes, including search, filters, expansion, and evidence.

## Open Questions

- A future advisor schema may add first-class remediation. Until then, the page
  should continue to show only collected conclusions and interpretation bounds.

## Links

- Accepted mock: local Codex deliverable `teleskope-advisory-mock.html`
- Design and implementation target: `internal/report/report.html`
- Related design: `docs/advisor.md`, `.provenance/2026-09-09-cluster-advisor.md`
- Implementation memory: `.phoenix/implementation-memory/report-advisory-page.md`
