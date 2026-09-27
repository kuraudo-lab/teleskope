# Scope Reborn Release Slices And Graduation Gate

This document is the normative release plan for Teleskope's Scope-inspired
interaction work. The machine-readable companion is
`docs/scope-reborn-release-gates.json`; run `make scope-release-check` to
validate and print it.

## Product order and naming

Teleskope is first a deterministic Popeye replacement, specialized for EKS.
The Scope lineage supplies the relationship-first interactive shell around that
judgment engine. Until the Popeye acceptance corpus and graduation gate in
issue #40 pass, release notes must call this work the **Scope-inspired topology
preview**. Only a release that passes #40 plus every required gate below may use
the **Scope reborn** label.

Issues #33–#40 define the Popeye replacement foundation. Scope work does not
replace or shortcut those issues; it consumes the same identities, findings,
evidence and coverage.

## Cumulative release slices

| Slice | Value delivered | Required issues | Product dependency |
| --- | --- | --- | --- |
| API graph baseline | Evidence graph, semantic views, graph/table exploration, search, filters, drill-down and restorable state | #41, #42 | Preview after its own gates pass |
| Live continuity | Revisioned bounded updates while selection, layout, freshness and coverage remain stable | #43 | Graph baseline |
| Contextual investigation | Findings, relatives, metrics, metadata and raw evidence in one selected-object flow | #44 | Graph baseline and live continuity |
| Optional external evidence | Explicitly authorized runtime connections or metrics with provider provenance and coverage | #45 security boundary plus a future adapter issue | Contextual investigation; never required for core graduation |

The scale, performance, accessibility and migration expectations in #46 and
this release contract are cross-cutting gates, not another feature slice.

### Slice 1 — API graph baseline

The self-contained report embeds the whole deterministic graph. Live serve
projects its latest published snapshot through the same contract. Hub scopes
that contract to one cluster and must not merge identities. Browser acceptance
starts with a finding or search result, reaches the matching object, drills from
a Service or workload to Pods, Node and EKS infrastructure, then restores the
same state from a copied URL.

### Slice 2 — Live continuity

Offline stays honestly static. Live consumes the existing watch/poll
publication boundary and applies revisioned deltas; it must not poll Kubernetes
more often merely to animate topology. Hub accepts cluster-scoped remote
revisions, retains bounded history and resets on gaps. Unchanged nodes do not
move, selection survives when the object still exists, and unavailable
evidence remains unavailable.

### Slice 3 — Contextual investigation

Offline renders recorded deterministic findings and provider coverage. Live
updates fixed Inspector sections from the accepted revision. Hub keeps
findings, metrics and evidence inside the selected cluster and revision. The
same object identity connects relatives, scoped findings, provider-aware
metrics, metadata and raw evidence; navigation may switch semantic view without
losing the target.

### Slice 4 — Optional external evidence

This slice is an enhancement, not a hidden core dependency. Offline accepts
only sanitized imported evidence with provenance and limitations. Live runs
only an explicitly configured server-side adapter. Hub accepts attested
provider evidence through the remote-writer boundary. Every contribution names
its provider, observation window, sampling, freshness, confidence and coverage;
failure is isolated and shown as unavailable.

No external adapter is implemented by this release plan. A real adapter needs
its own issue, authorization model, resource/backpressure budget, test fixture
and security review.

## What is inherited from Scope

- **Core:** semantic views, relationship-first navigation, search/filter,
  selection, contextual detail, stable layout and a dense table fallback.
- **Adapted:** Kubernetes ownership and declared relationships replace host
  probe discovery; watches and revisions replace probe streams; evidence tier,
  freshness, confidence and coverage become explicit product concepts.
- **Deferred:** external observed-network and short-window metric adapters,
  provider-contributed read-only links, and history/replay.
- **Rejected:** a privileged node probe or DaemonSet, host/runtime access,
  terminal and lifecycle controls, observed-looking inferred traffic, and a
  general browser plugin store.

## Graduation checklist

A release can use the Scope reborn label only when all of these are evidenced:

1. Issues #33–#40 are complete and the corpus-based gate in #40 passes.
2. Required slices pass deterministic contract/unit tests and the recorded
   browser walkthrough in `docs/scope-reborn-walkthrough.md`.
3. Small and medium graphs satisfy the SVG budgets; Large uses the accessible
   relation table. The performance and accessibility evidence in
   `docs/topology-scale-performance-accessibility.md` is current.
4. Offline, live and hub preserve identity, evidence tier, freshness, coverage
   and finding references. Optional-provider failure is isolated and disclosed.
5. Security review confirms there is no privileged node workload, browser
   credential, implicit external transfer, write control or cross-cluster leak.
6. Existing topology entry points, static exports and restorable query state
   remain compatible, or a migration note and regression test are supplied.

Passing only the topology gates allows a preview; it does not imply Popeye
replacement graduation. `make scope-release-check` validates the contract's
shape and invariants but deliberately does not query GitHub or declare open
issues complete.

## Extension policy

The typed topology graph, metric, coverage and raw-evidence records are the
current provider seam. Fixed UI sections render those records. Providers run
server-side after explicit configuration and cannot inject browser code or
credentials.

A general reporter or UI plugin system is premature. Reconsider a narrow
registration API only after two independent external evidence implementations
need the same lifecycle. This avoids freezing an imagined abstraction and
keeps optional evidence outside the core security boundary.
