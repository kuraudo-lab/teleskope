# Contextual Topology Details Provenance

## Trigger

After issue #43 established bounded incremental graph delivery, the product
owner asked the agent to continue autonomously through the Scope-reborn issue
sequence with independent acceptance and two-stage commits.

## Scope

Issue #44: selected node/edge context, parent/child/upstream/downstream
navigation, deterministic findings and evidence, metric overlays, raw evidence
access, and explicit read-only non-goals.

## Conversation Summary

Teleskope should rebirth Scope's investigation flow: selecting an object must
answer what it is, what it relates to, what deterministic findings apply, what
evidence supports those conclusions, and which metrics are actually known.
This remains an evidence browser, not a Kubernetes management console.

## Decisions

- Use the persistent right-hand Inspector as the primary context surface. Keep
  the drawer only for raw/full object evidence and existing non-topology detail
  flows.
- Classify relatives as parent, child, sibling, upstream, downstream, and
  cross-layer from canonical parent IDs and typed directed edges.
- Use native sortable tables and buttons for relation navigation; graph and
  relation-table selection remain the same state.
- Resolve finding references against the deterministic Advisor report and show
  assessment, evidence basis, coverage, summary, and constraints.
- Add a provider-neutral metric-series contract to topology nodes. Every series
  declares provider, semantic (`capacity`, `allocatable`, `declared`, or
  `usage`), unit, sampling, time/freshness, and numeric samples.
- Project Kubernetes API capacity/allocatable and declared request/limit facts
  as single-snapshot series. Never label them as live usage.
- Allow one metric to be pinned to topology cards through URL-restorable state.
  Draw a sparkline only when two or more ordered samples exist; otherwise show
  the point value and explicitly state that history is unavailable.
- Missing runtime usage is an explicit unavailable state. Future external
  metrics providers may add bounded series without changing UI semantics.

## Rejected Alternatives

- Showing CPU or memory requests as observed utilization.
- Fabricating sparklines from a single snapshot or interpolating missing data.
- A modal-only detail flow that hides the topology and breaks cross-layer
  navigation.
- Exec, logs, edit, delete, scale, restart, terminal, or lifecycle controls.
- A privileged node probe or implicit metrics scraper.

## Constraints

- Existing topology graph/table filters, selection, URL restoration, themes,
  offline/live/hub delivery, and incremental updates remain compatible.
- Metric series and samples are deterministically ordered and validated.
- Evidence source, window, freshness, and coverage remain visible beside any
  metric value.
- Inspector controls are keyboard accessible and usable at narrow widths.
- Raw evidence is available on demand without dominating the primary path.

## Evaluation Plan

- Add graph contract and projection tests for capacity, allocatable, declared,
  missing, stale, and multi-sample series semantics.
- Add contextual relation/finding/metric UI contract tests.
- Verify graph/table cross-navigation, sorting, metric pinning, URL reload,
  missing history, raw evidence, narrow layout, and live refresh in a browser.
- Regenerate checked-in demo reports and run full checks/build.
- Obtain independent sub-agent acceptance before the implementation commit.

## Open Questions

- Large-graph metric overlay thresholds and formal render budgets remain owned
  by issue #46.
- General provider/plugin registration remains deferred; issue #47 decides
  whether a narrower reporter seam is justified.

## Links

- Contract: `docs/topology-contract.md`
- Interaction design: `docs/topology-interaction-prototype.md`
- Related issue: https://github.com/kuraudo-lab/teleskope/issues/44
