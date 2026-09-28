# Topology Circular Node Visual Contract Provenance

## Trigger

The user accepted the Scope-inspired topology behavior but rejected the production node rendering because it still used rectangular cards. The production graph must follow the accepted mock's circular icon treatment exactly.

## Scope

- `docs/topology-scope-reborn-design.md`
- `docs/mockups/topology-scope-reborn-prototype.html`
- Graph-mode topology node rendering and styling in `internal/report/report.html`
- Generated demo reports and focused report tests

## Conversation Summary

The mock's node silhouette is part of the accepted design, not an illustrative detail. Normal graph objects use circular icon containers with centered labels below; production rectangular cards are a visual regression. The existing evidence model, layout, inspector, filters, and interaction behavior remain unchanged.

## Design Diff

`docs/topology-scope-reborn-design.md` now makes the mock's 52px circular icon, centered bounded copy, type-specific outline, health marker, selected halo, and distinct pod/runtime and infrastructure silhouettes an explicit visual contract.

## Decisions

- Remove the rectangular card background from graph-mode nodes.
- Use the accepted mock's circular 52px icon container for external, gateway, service, and workload nodes.
- Keep the mock's non-circular pod/runtime and infrastructure silhouettes so shape also communicates type.
- Place the bounded label and secondary line below the icon, centered within a fixed 120px footprint.
- Preserve accessible names, tooltips, keyboard focus, selection state, evidence inspection, and graph behavior.

## Rejected Alternatives

- Merely increasing the existing card's border radius: this remains a card and does not match the accepted mock.
- Making every object identical circles: this drops the mock's pod/runtime and infrastructure shape distinction.
- Replacing the graph with the prototype file: the product must retain the real topology/evidence contracts and report-shell behavior.

## Constraints

- The accepted mock is the visual source of truth.
- No node probe or new evidence provider is introduced.
- Full identities must remain available outside the bounded visible labels.
- Graph/Table, filters, fit, zoom/pan, focus, live updates, theme, export, Events, and inspector behavior must not regress.

## Evaluation Plan

- Render the recorded EKS fixture through the real embedded product path.
- Compare default and selected graph nodes against the accepted mock.
- Verify circular silhouettes, centered bounded labels, health marker, selection halo, hover/focus states, and light/dark themes.
- Verify `aws-node` focus, Search, Services filtering, Graph/Table, Fit, and relationship inspection.
- Run focused report tests, `make check`, `make build`, and `git diff --check`.

## Open Questions

None.

## Links

- Design: `docs/topology-scope-reborn-design.md`
- Accepted mock: `docs/mockups/topology-scope-reborn-prototype.html`
- Related issue: #49
