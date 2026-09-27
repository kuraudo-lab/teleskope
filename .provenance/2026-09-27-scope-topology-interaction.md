# Scope Topology Interaction Provenance

## Trigger

After accepting the unified topology/evidence contract in issue #41, the
product owner asked the agent to proceed autonomously through the Scope-reborn
issue sequence with independent acceptance and two-stage commits.

## Scope

Issue #42: semantic topology views, graph/table presentation, filters, compound
search, selection focus, URL-restorable state, keyboard behavior, and
coexistence with the current swimlane topology.

## Conversation Summary

Teleskope should rebirth Scope's relationship-first investigation experience,
not merely copy its force graph. The interaction must remain evidence-first,
read-only, EKS-aware, and usable without runtime connection telemetry. A
throwaway prototype will compare three structurally different models against
the Recorded fixture; the accepted structure will then be rewritten into the
embedded production UI.

## Design Diff

- `docs/topology-interaction-prototype.md` states the design question, three
  variants, shared state contract, evaluation tasks, and acceptance boundary.
- The prototype will be captured on a throwaway branch; the main branch keeps
  only this intent record and the selected production implementation.

## Decisions

- Prototype in the existing topology page context with real Recorded data.
- Compare semantic swimlanes, a Scope-style whole canvas, and a bounded
  relation explorer rather than cosmetic variants.
- Use one URL-restorable state model across graph and table modes.
- Retain the current swimlane option during this issue.
- Treat the table as an accessible and scale-safe peer, not an export-only
  afterthought.

## Rejected Alternatives

- Selecting a final interaction from static prose alone.
- Keeping a permanent prototype switcher in production.
- Using a force graph whose motion destroys selection or evidence context.
- Adding terminal, CRUD, lifecycle, or node-probe functionality.

## Constraints

- Embedded single-file HTML/CSS/JavaScript delivery remains intact.
- Existing theme, export, events, analysis, and detail interactions remain
  functional.
- Keyboard focus is visible and not obscured; all compact controls use native
  semantics and announced pressed/selected state.
- No search/filter result may alter evidence classification.

## Evaluation Plan

- Run three variants with the Recorded EKS fixture and capture a verdict.
- Rewrite the winner into production code without retaining the variant
  switcher.
- Add focused HTML contract tests and browser interaction tests.
- Run full repository checks and build.
- Obtain independent sub-agent acceptance before the implementation commit.

## Open Questions

- Exact large-graph fallback thresholds remain owned by issue #46.
- Incremental update and anti-dance protocol remains owned by issue #43.

## Links

- Design file: `docs/topology-interaction-prototype.md`
- Contract: `docs/topology-contract.md`
- Related issue: https://github.com/kuraudo-lab/teleskope/issues/42
