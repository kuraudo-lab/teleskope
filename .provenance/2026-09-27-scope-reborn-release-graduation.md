# Scope Reborn Release Graduation Provenance

Date: 2026-09-27

## Request

The product owner asked the agent to finish the Scope-reborn issue sequence
autonomously, preserving independent subagent acceptance and two-stage
provenance/implementation commits. Issue #47 asks how the completed topology
foundation becomes coherent, shippable releases after the Popeye replacement
foundation, without introducing a privileged node probe.

## Scope

This record covers the release slices, their dependency on the Popeye
replacement graduation gate, offline/live/hub behavior, browser and security
acceptance, compatibility with the existing topology UI, and the decision on a
reporter-style extension seam. It also covers a machine-readable release
contract and a local checker that keep the written roadmap testable.

It does not claim that open Popeye replacement issues are complete, implement
an external telemetry provider, add a general plugin system, or add node-local
collection.

## Product decision

Teleskope may describe the topology work as a Scope-inspired preview while the
Popeye replacement contract is still open. It may graduate the product as
"Scope reborn" only after the Popeye replacement corpus and graduation gate in
issue #40 pass. This preserves the product order: deterministic health and EKS
evidence are the judgment engine; relationship-first exploration is its
interactive shell.

The release is divided into four cumulative slices:

1. **Graph baseline** — one evidence contract, stable object identities,
   semantic views, search/filter, selection, drill-down, URL restoration, and
   a bounded table fallback.
2. **Live continuity** — revisioned graph deltas, reset semantics, freshness,
   stable selection and layout, and bounded polling/watch behavior. Offline
   reports remain static and complete without pretending to be live.
3. **Contextual investigation** — findings, relatives, metrics, metadata, raw
   evidence, and cross-view navigation share one object identity and coverage
   model.
4. **Optional external evidence** — separately authorized metrics or observed
   connection providers may contribute bounded evidence with source, window,
   sampling, freshness, confidence, and coverage. Absence is rendered as
   unavailable and does not block the core release.

Scale, accessibility, migration compatibility, and security are cross-cutting
graduation gates rather than a fifth feature slice.

## Scope inheritance

- **Core:** semantic graph views, relationship-first navigation, filtering,
  selection, contextual details, stable layout, and a dense table fallback.
- **Adapted:** Kubernetes ownership and declared relationships replace Scope's
  host-probe graph; Kubernetes watches and revisioned snapshots replace probe
  streams; evidence tier, freshness, confidence, and coverage are explicit.
- **Deferred:** real external observed-network adapters, richer short-window
  metric adapters, provider-contributed read-only links, and history/replay.
- **Rejected:** a privileged per-node probe or DaemonSet, host PID/network or
  runtime socket access, terminal/container lifecycle/scale/delete controls,
  inferred traffic presented as observation, and a general UI plugin store.

## Mode contract

- Offline reports embed a complete deterministic graph and contextual evidence;
  live-only capabilities are visibly unavailable rather than simulated.
- Live serve uses the existing server-side collector/watch/poll publication
  boundary. The browser receives revisions and evidence, never credentials.
- Hub preserves cluster scope and remote-writer provenance. It must not merge
  object identities, freshness, or provider coverage across clusters.
- External evidence is opt-in in every mode and cannot weaken the useful
  Kubernetes-only baseline.

## Graduation gates

- The Popeye replacement compatibility, finding, policy, metric, automation,
  operating-model, and corpus work in issues #33–#40 is complete; issue #40 is
  the blocking product graduation gate.
- Every Scope slice has deterministic unit/contract tests and a browser
  acceptance path using de-identified recorded or generated fixtures.
- Small and medium graphs satisfy the published SVG budgets; large graphs use
  the accessible table fallback. Contrast, keyboard, focus, reduced-motion,
  narrow-layout, and screen-reader semantics pass the published gates.
- Offline, live, and hub preserve identity, evidence tier, freshness, coverage,
  and finding references. Optional provider failure is isolated and disclosed.
- Security review confirms no privileged node workload, browser credentials,
  implicit external data transfer, write controls, or cross-cluster scope leak.
- Existing topology entry points, query parameters, and static exports remain
  compatible or have an explicit migration note and test.

## Extension decision

A general reporter or UI plugin interface is not justified before a second real
external evidence implementation proves the seam. The current typed topology
graph, metric, coverage, and evidence contracts are the extension boundary.
Future providers may append bounded data to fixed UI sections after explicit
server-side configuration and authorization; they may not inject arbitrary
browser code or credentials. Revisit a narrow provider registration API only
when two independent implementations need it.

## Deliverables

- `docs/scope-reborn-release-slices.md` is the normative release and graduation
  guide.
- `docs/scope-reborn-release-gates.json` is the machine-readable contract.
- `scripts/scope-release-check` validates the contract and prints a slice or
  full release checklist for maintainers and demos.
- `docs/scope-reborn-walkthrough.md` gives an end-to-end acceptance walkthrough.
- Make targets and tests make contract checking repeatable without network or
  provider credentials.

## Rejected alternatives

- Calling the graph baseline "Scope reborn" before the Popeye graduation gate
  was rejected because it would invert the agreed product lineage.
- Requiring observed network connections for the core release was rejected
  because it would make an optional high-privilege data source a hidden product
  dependency.
- Treating every historical issue as a release slice was rejected because it
  would ship implementation chronology instead of coherent user value.
- Adding a generic plugin SDK now was rejected because no second implementation
  yet proves the abstraction and arbitrary UI extension would expand the
  security surface.

## Verification expectation

The implementation commit must link to this record, close issue #47, pass the
full repository checks and release-contract tests, and receive independent
subagent acceptance. The final walkthrough must demonstrate the finding/search
to topology to contextual evidence loop, live continuity, large-graph fallback,
and explicit unavailable external-provider coverage without a node probe.
