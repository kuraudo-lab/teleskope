# Topology Scale, Performance, And Accessibility Provenance

## Trigger

After issue #44 completed the contextual topology Inspector, the product owner
asked the agent to continue autonomously through the Scope-reborn sequence with
independent acceptance and two-stage commits.

## Scope

Issue #46: measurable topology fixture sizes, render and interaction budgets,
layout stability, browser/server resource expectations, accessibility checks,
and deterministic degradation when a semantic view is too complex.

## Conversation Summary

Teleskope should inherit Scope's explicit large-graph escape hatches without
copying its privileged Probe architecture. The embedded report must stay useful
from small Recorded demos through realistically large EKS inventories, and its
limits must be observable and testable rather than hidden heuristics.

## Decisions

- Define small, medium, and large deterministic synthetic graph fixtures and
  publish their exact node/edge/update sizes beside the gates they exercise.
- Budget initial render, incremental update, search, selection, layout drift,
  serialized payload size, and browser memory with reproducible harness output.
- Measure server graph projection/update paths with Go benchmarks and keep UI
  timing checks in a browser harness; do not claim laboratory precision from a
  unit test.
- Render a graph only below an explicit per-view complexity budget. Above it,
  preserve filters and selection but default to the relation table and explain
  how to narrow the view.
- Preserve stable ranks for unchanged identities. Incremental updates may add
  or remove ranks but must not reorder unchanged nodes within a semantic lane.
- Keep keyboard selection, visible focus, native table semantics, meaningful
  SVG labels, zoom controls, contrast, and reduced-motion behavior as release
  gates at every supported scale.
- Treat browser, live server, hub, collector, and optional evidence providers
  as separate budgets. External providers remain optional and their absence is
  a coverage state, not a reason to weaken core budgets.

## Rejected Alternatives

- Unlimited SVG rendering followed by browser-dependent slowdown or crashes.
- Hiding excess nodes without a count, explanation, or table fallback.
- Using wall-clock assertions in ordinary unit tests as a performance claim.
- Re-running a global layout after every revision and moving unchanged nodes.
- Requiring a privileged node Probe or DaemonSet to meet scale targets.
- Treating mouse-only pan/zoom or color-only relationship encoding as
  accessible interaction.

## Constraints

- Offline report, live server, and hub drilldown use the same limits and
  semantics.
- Degradation never drops evidence from JSON export or the relation table.
- Thresholds are defaults documented as product gates, not promises for every
  browser, device, or optional provider.
- Performance evidence records fixture identity, build revision, environment,
  and measured samples.
- Existing URL state, filters, metrics, findings, and incremental update
  contracts remain compatible.

## Evaluation Plan

- Add deterministic scale fixtures and contract tests for their shape.
- Add projection/update benchmarks and a repeatable browser measurement guide.
- Add complexity fallback, reduced-motion, focus, keyboard, and responsive UI
  tests; verify representative behavior in the embedded browser.
- Run focused tests, full checks, build, JavaScript parsing, and diff checks.
- Obtain independent sub-agent acceptance before the implementation commit.

## Open Questions

- Release packaging and the graduation relationship to the Popeye replacement
  remain owned by issue #47.
- A future optional runtime evidence provider must publish its own collection
  budgets before it can be included in a release gate.

## Links

- Research baseline: `docs/weave-scope-product-ui-research-2026-09-26.md`
- Topology contract: `docs/topology-contract.md`
- Related issue: https://github.com/kuraudo-lab/teleskope/issues/46
