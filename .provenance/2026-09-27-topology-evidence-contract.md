# Topology Evidence Contract Provenance

## Trigger

The product owner asked Teleskope to begin the Scope-reborn work from the
shared contract, while explicitly rejecting any privileged node probe.

## Scope

The provider-neutral graph model, stable node and edge identity, relationship
semantics, evidence metadata, coverage, and transport compatibility across
recorded snapshot, live, and hub paths.

## Conversation Summary

Teleskope is a Popeye successor and EKS-specialized deterministic diagnostic
engine whose interactive experience should rebirth Weave Scope's contextual
relationship exploration. The first implementation slice must create one graph
and evidence vocabulary shared by offline and live modes. Runtime connections
are optional external evidence, not a prerequisite for useful topology.

## Design Diff

- `docs/topology-contract.md` defines graph documents, stable identity,
  relationship kinds, evidence tiers, coverage, transport compatibility, and
  the security boundary.
- The implementation will add a focused topology package and project graph
  revisions into recorded, live, and hub outputs without replacing the current
  renderer in this issue.

## Decisions

- Keep topology as a provider-neutral projection rather than extending each
  collector with UI-shaped fields.
- Distinguish ownership, declared, resolved, inferred, and observed edges.
- Preserve provider, observation window, sampling, freshness, confidence, and
  coverage on normalized evidence.
- Publish a full deterministic graph revision first; defer delta transport and
  layout stability to issue #43.
- Represent missing runtime connection telemetry as unavailable coverage.

## Rejected Alternatives

- Building graph edges only in browser JavaScript, because live, recorded, and
  hub consumers would continue to disagree on identity and semantics.
- Treating selectors and configuration references as observed traffic.
- Implementing a privileged node probe or coupling the graph contract to one
  telemetry vendor.
- Solving incremental deltas, renderer layout, and aggregation in the contract
  issue.

## Constraints

- No host PID/network access, runtime sockets, procfs, conntrack/debugfs, or
  eBPF privilege requirement.
- Existing snapshot JSON remains backward compatible.
- Equal evidence must yield deterministic graph output.
- Deterministic evidence remains authoritative; optional narrative providers
  do not create graph facts.

## Evaluation Plan

- Unit tests for identity construction, validation, observed-edge rules,
  coverage defaults, deterministic ordering, and JSON round trips.
- Integration tests proving recorded artifacts, live responses, and hub
  envelopes expose the same graph contract.
- Full repository checks and build.
- Independent sub-agent review against issue #41 and the no-node-probe
  constraint before the implementation commit.

## Open Questions

- Exact delta protocol and stale-retention rules are deferred to issue #43.
- View aggregation and navigation behavior are deferred to issue #42.

## Links

- Design files: `docs/topology-contract.md`, `docs/architecture.md`
- Related issue: https://github.com/kuraudo-lab/teleskope/issues/41
