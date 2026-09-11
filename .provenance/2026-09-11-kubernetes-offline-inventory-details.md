# Kubernetes Offline Inventory Details Provenance

## Trigger

The user compared teleskope with Headlamp and decided not to add interactive dashboard behavior. Instead, teleskope should add offline scan/report data for Kubernetes resources that are useful for migration planning.

## Scope

This record covers first-class offline inventory details for:

- Job, CronJob, ReplicaSet, DaemonSet, and StatefulSet type-specific workload fields
- RBAC Role, ClusterRole, RoleBinding, and ClusterRoleBinding details
- NetworkPolicy, ResourceQuota, LimitRange, and PodDisruptionBudget details
- MutatingWebhookConfiguration and ValidatingWebhookConfiguration details

Relevant implementation surfaces:

- `internal/inventory/snapshot.go`
- `internal/k8s/collector.go`
- `internal/k8s/collector_test.go`
- `internal/cli/serve.go`
- `internal/report/report.go`
- `internal/report/report.html`
- `internal/report/report_test.go`
- `internal/render/human.go`

## Conversation Summary

The user asked to compare teleskope with Headlamp and identify data Headlamp has that teleskope lacks. The conclusion was that teleskope should not try to match Headlamp's interactive operations such as exec, logs, editing, or deletion. Instead, teleskope should strengthen deterministic offline collection and reporting for migration planning.

The accepted next step was to collect more detailed data for selected workload, RBAC, policy, quota, limit, disruption budget, and admission webhook resources. Existing coarse objects should remain compatible while new detail fields provide migration-relevant facts.

## Design Diff

No standalone design document was changed. The design captured here is:

- Keep existing `workloads` aggregation and extend it with type-specific fields for DaemonSet, StatefulSet, ReplicaSet, Job, and CronJob.
- Keep existing RBAC and policy ObjectRef lists for compatibility, and add detail arrays alongside them.
- Record RBAC rules, binding roleRefs, and subjects.
- Record NetworkPolicy selector/type/rule/peer counts, ResourceQuota hard/used values, LimitRange items, and PodDisruptionBudget selector/status/budget values.
- Add `admissionWebhooks` as a first-class Kubernetes inventory field with webhook client, rules, operations, resources, failure policy, match policy, timeout, side effects, and admission review versions.
- Surface the new data in JSON, Markdown, terminal output, live serve partial-data detection, and static web report tables.

## Decisions

- Do not add Headlamp-style interactive features.
- Prefer append-only optional JSON fields to avoid breaking existing scan consumers.
- Keep Workload as the common aggregation surface for built-in workload kinds while enriching it with fields needed by those kinds.
- Use detailed RBAC and policy collections alongside existing ObjectRef collections rather than replacing the existing shape.
- Include admission webhooks because they often explain migration-time admission failures and sidecar/defaulting behavior.

## Rejected Alternatives

- Do not replace the existing `Workloads` model with separate top-level arrays for each workload kind in this change.
- Do not collect logs, exec shells, live edits, or other Headlamp interaction features.
- Do not remove existing ObjectRef-only RBAC and policy fields because downstream consumers may rely on them.

## Constraints

- Maintain backward-compatible JSON by only adding optional fields.
- Keep collection deterministic and usable in offline reports.
- Avoid storing secret values; this change only adds metadata, rules, references, selectors, and quantities.
- Preserve live serve partial-result behavior for newly added data types.
- Tests should verify new detail mappers and web render bindings.

## Evaluation Plan

- Add collector tests for workload-specific status/spec fields.
- Add collector tests for RBAC rule and subject details.
- Add collector tests for PDB, NetworkPolicy, ResourceQuota, and LimitRange details.
- Add collector tests for admission webhook details.
- Add report HTML tests for new Security and Policies table bindings.
- Run the full Go test suite with `go test ./...`.
- Run `git diff --check` before committing implementation.

## Open Questions

- Whether HPA/VPA details and Kubernetes Events should be added in a later pass.
- Whether workload kinds should eventually move from a shared `workloads` array to both shared and kind-specific top-level collections.

## Links

- Design files: `.provenance/2026-09-11-kubernetes-offline-inventory-details.md`
- Related issues/tasks: Headlamp comparison follow-up and user-approved offline data expansion in this Codex task.
