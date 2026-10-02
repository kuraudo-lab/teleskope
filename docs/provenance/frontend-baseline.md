# Frontend migration baseline provenance

## Trigger and approved scope
The user approved Vue 3 + TypeScript + Vite migration, complete implementation and verification of all GitHub tasks before final acceptance. `serve snapshot` is the standard interactive report entry point. Retain standalone HTML only when it shares components and requires no separate application implementation.

## Constraints
- Embed all runtime assets in the Go binary; no Node or CDN at runtime.
- Preserve data, evidence semantics, JSON/Markdown exports, themes, search, detail, topology and Hub.
- No redesign, invented evidence, new collection or cluster credentials for replay.
- Preserve zero-replica workload template relationships, unknown coverage, topology node/edge selection, graph mode and URL state.
- Live updates preserve namespace even when absent, selection and viewport; delta replay is idempotent and bad deltas resync.
- Deterministic advisor/projection remains Go-owned; AI is optional narrative.

## Evaluation
Establish fixture/browser behavior baseline, test new pure modules and Vue components, then verify actual embedded binary and replay paths. Run Go checks/build and source diff checks at commit boundaries. Check frontend build reproducibility and offline HTML feasibility early. Final validation includes Hub and topology scale fixtures.

## Links
https://github.com/kuraudo-lab/teleskope/issues/50
https://github.com/kuraudo-lab/teleskope/issues/51
