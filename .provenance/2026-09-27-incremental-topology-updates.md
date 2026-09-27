# Incremental Topology Updates Provenance

## Trigger

After issue #42 established the production Scope-style topology interaction,
the product owner asked the agent to continue autonomously through the
Scope-reborn sequence with independent acceptance and two-stage commits.

## Scope

Issue #43: revisioned full/delta graph delivery, resynchronization, bounded
retention, layout stability, browser-state preservation, and reconnect
behavior for live and hub topology consumers.

## Conversation Summary

Teleskope should feel live without inheriting Scope's privileged per-node
probe. Kubernetes and AWS collectors, the live store, hub envelopes, and
optional external evidence providers remain the evidence sources. The browser
must never infer missing observations, and an unavailable external provider
must remain explicit coverage rather than silently deleting relationships.

## Decisions

- Keep conditional HTTP fetch as the baseline transport. It already supports
  embedded live and hub deployments without a second connection protocol.
- Define a deterministic full/delta topology update contract keyed by canonical
  node and edge identities.
- Retain a small bounded graph-revision history in live stores. A client whose
  base revision is absent, ahead, or incompatible receives a full reset.
- Treat deletions as explicit node/edge identifiers; applying the same update
  more than once must be safe.
- Coalesce collector changes at publication boundaries. Browser polling never
  triggers collection and never receives mutable store state.
- Anchor layout by stable node ID and preserve selection, filters, viewport,
  paused state, and drawer context across accepted revisions.
- Keep source freshness and coverage on the full graph/update. Do not convert a
  stale source into invented object churn.

## Rejected Alternatives

- WebSocket as a mandatory transport for a read-only periodic publication
  model.
- Unbounded server-side revision history or replay logs.
- Index-based node identity or force-layout motion that moves unchanged nodes.
- Clearing the graph on reconnect, out-of-order delivery, or collector failure.
- A privileged node probe, host PID/network access, or a DaemonSet.

## Constraints

- Existing `/api/snapshot`, ETag, offline report, hub envelope, and remote-write
  behavior remain compatible.
- Revisions are monotonic per live store or hub cluster; stale and duplicate
  updates do not regress state.
- A delta is only valid for its declared base revision and cluster/schema.
- History, payload, retry, and debounce behavior are bounded and testable.
- The embedded single-file UI remains dependency-free and read-only.

## Evaluation Plan

- Add topology update/diff/apply tests for add, update, delete, duplicate,
  out-of-order, wrong-cluster, and resync cases.
- Add a bounded live graph history and conditional topology endpoint tests.
- Apply browser deltas while preserving graph state and stable layout anchors.
- Exercise Recorded/live browser reconnect and refresh behavior.
- Run full repository checks and build.
- Obtain independent sub-agent acceptance before the implementation commit.

## Open Questions

- Large-graph thresholds and measurable performance budgets remain owned by
  issue #46.
- Metric sampling and contextual overlays remain owned by issue #44.

## Links

- Contract: `docs/topology-contract.md`
- Prior interaction provenance:
  `.provenance/2026-09-27-scope-topology-interaction.md`
- Related issue: https://github.com/kuraudo-lab/teleskope/issues/43
