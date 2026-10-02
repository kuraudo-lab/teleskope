# Embedded frontend migration

Tracking: https://github.com/kuraudo-lab/teleskope/issues/50

## Acceptance contract

The standard interactive entry is `teleskope serve snapshot <snapshot.json-or-report-directory>`. Replay must require no AWS/Kubernetes credentials or background collectors. JSON and Markdown remain independently readable archives. A standalone interactive HTML may survive only as another package of the same UI, never as a second application.

Preserve current navigation, light/dark theme, exports, resource filters, workload template relationships (including zero replicas), EKS evidence, Advisor findings, optional scoped AI, events, Hub drilldown and topology graph/table interactions. Preserve missing/unknown evidence and actual collection time. Namespace deletion must not silently widen selection. Topology deltas must be revision-consistent and replay-safe. Graph mode must never silently become table mode.

## Baseline

Current production UI is `internal/report/report.html` + `live.js`; `RenderUI` owns boot configuration and Go-derived payloads. Hub has a separate Go HTML string. Existing Go tests cover inventory/projections and many source-string contracts; browser behavior tests are the replacement for implementation-string assertions during migration.

Browser baseline: `cd web && npm ci && npx playwright install chromium && npm run test:e2e` after `make build`. Fixture: `testdata/recorded/eks-demo-snapshot.json`. Topology profiles and budgets: `docs/topology-scale-performance-accessibility.md`.

## Execution order

F01 baseline → F02 build/embed → F03 data → F03a replay → F04 shell → F05 workloads → F06 resources → F07 EKS/analysis → F08 topology → F09 Hub → F10 cleanup/release. All tasks are implemented and verified before final user acceptance.

## Packaging decision (F02)

Vite library mode produces one IIFE and one CSS file with Vue included. Both are embedded by Go and inlined by the same RenderUI entry. This adds only packaging, so standalone HTML is retained provisionally; no second application, route or component set is introduced. Checked-in `internal/report/assets` allow ordinary Go-only builds. `make ui-check` rebuilds and checks for drift. TypeScript is pinned to 5.9.3 because the initially resolved newer major was incompatible with vue-tsc's compiler entry point.

## Archive and replay contract (F03a)

`snapshot.json` is the inventory evidence archive. Current supported schema is `teleskope.io/snapshot/v1alpha1`; schema-less older snapshots are accepted. Unknown fields within that schema remain accepted for forward-compatible additions. Unsupported schema versions, malformed JSON and trailing documents fail explicitly. Pass either the file or its report directory to `teleskope serve snapshot`.

Replay does not contact a cluster or use cluster credentials. The installed Teleskope version recomputes deterministic Advisor, EKS presentation and topology from archived inventory; its publication revision is not historical cluster time. Read `snapshot.collectedAt` for original collection time. AI analysis artifacts, live event history and external observations not stored in the snapshot are not reconstructed. Optional online analysis is not enabled by recorded serve. Keep separate generated analysis artifacts when archiving those outputs.
