# Topology Namespace Filter Empty-State Provenance

## Trigger

The user reported that applying the namespace filter produced an empty topology.

## Scope

- Namespace options used by the Overview topology workspace.
- `renderHeader` and topology projection namespace derivation in `internal/report/report.html`.
- Report tests, generated demos, and recorded-fixture browser verification.

## Conversation Summary

The recorded fixture reproduced the problem for `kube-node-lease` and `kube-public`: these namespaces appear in the global cluster namespace inventory, but the Workloads projection has no external, entry, Service, or Workload object for either namespace. `default` and `kube-system` correctly render two and eight objects respectively. The selector therefore advertised scopes that were guaranteed to be empty in the active product surface.

## Design Diff

The formal topology design now requires namespace choices to be derived from displayable Workloads-projection graph objects when graph evidence is available. Namespace-only inventory remains accessible in the relevant Kubernetes inventory surfaces.

## Decisions

- Derive topology namespace choices from graph nodes allowed by the Workloads projection.
- Preserve `All namespaces`.
- Reset stale/deep-linked namespaces that are absent from the current projection.
- Preserve the namespace value when it is still valid across live revisions.
- Do not manufacture workload nodes for namespaces that only contain leases or unrelated resources.

## Rejected Alternatives

- Show an unexplained empty graph for guaranteed-empty options: this is the reported bug.
- Add namespace placeholder nodes to Workloads: this violates the accepted projection contract.
- Treat Lease or Namespace inventory as a workload relationship: this misrepresents evidence semantics.

## Constraints

- No node probe or new evidence provider.
- Existing `default` and `kube-system` filtering, URLs, Search, Graph/Table, Fit, and live updates must remain functional.
- Inventory pages may continue to expose namespaces outside the Workloads projection where relevant.

## Evaluation Plan

- Before the fix, verify the browser loop returns zero objects for offered `kube-node-lease` and `kube-public` options.
- After the fix, assert every offered namespace has at least one graph node.
- Verify `default` returns two objects and `kube-system` returns eight with the recorded fixture.
- Verify a stale `?namespace=kube-public` URL falls back to all namespaces.
- Run focused report tests, `make check`, `make build`, and `git diff --check`.

## Open Questions

None.

## Links

- Design: `docs/topology-scope-reborn-design.md`
- Related issue: #49
