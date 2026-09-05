# Kubernetes Collector Provenance

## Trigger

The user asked to start implementing Kubernetes scanning after the first
AWS-side EKS collector slice. During implementation, the user clarified that
Secret-like values must be redacted while other visible configuration does not
need redaction. The user then rejected the first human-readable output as not
useful enough and asked for table-like output, while noting that output quality
can be revisited later.

This record is created after the implementation pass because the user explicitly
invoked the provenance-commit workflow together with the implementation commit
workflow. It is a structured summary, not a raw transcript.

## Scope

This record covers the first Kubernetes API collector slice:

- `teleskope scan k8s` as a standalone command.
- Kubernetes API collection from kubeconfig and optional context override.
- Integration of Kubernetes collection into `teleskope scan eks`.
- Snapshot fields for API discovery, CRD/APIService registration, core objects,
  workloads, routing, storage, runtime, RBAC, and policy objects.
- Secret handling that avoids reading Secret data values.
- Human output improvements for routing, storage, runtime, platform component
  candidates, workload relations, and coverage.

It does not cover a full TUI, HTML report, mini web server, complete component
adapters, automatic capability decisions, migration gap scoring, or true runtime
node-local diagnostics.

## Conversation Summary

The accepted first-phase product direction is to collect and organize EKS and
Kubernetes information for human assessment. The user cares especially about
non-default capability evidence provided by add-ons, CNI, CSI, CRI, and workload
topology. The first implementation commit collected AWS-side EKS facts only.

For this slice, Kubernetes collection was added using client-go. The collector
loads kubeconfig through client-go's default loading rules, supports
`--kubeconfig` and `--kube-context`, records unavailable and denied reads in
coverage, and keeps collection read-only. The user explicitly clarified that
Secret values must be redacted, while other data such as ConfigMap data,
annotations, selector labels, and storage parameters can be collected normally.

The first human renderer only showed counts and was judged insufficient. It was
then changed to expose concrete facts such as Ingress routes, PVC access modes,
StorageClass provisioners, CSI drivers, node CRI runtime distribution, and
workload references to PVC, ConfigMap, Secret, and CSI volumes. A later change
converted those sections to tabular output with terminal column alignment.

## Design Diff

- `internal/k8s`: adds Kubernetes API collection through client-go typed,
  discovery, dynamic, and metadata clients.
- `internal/inventory`: adds Kubernetes snapshot structures for API resources,
  CRDs, APIServices, namespaces, nodes, service accounts, workloads, pods,
  services, EndpointSlices, Ingresses, storage, runtime, RBAC, policies,
  ConfigMaps, and Secrets.
- `internal/cli`: adds `scan k8s`; extends `scan eks` with Kubernetes collection
  flags and `--skip-kubernetes`.
- `internal/render`: expands the human renderer from count summaries into
  tabular fact views and adds renderer tests.
- `go.mod`/`go.sum`: adds Kubernetes client-go/API dependencies.
- `README.md`: documents Kubernetes collection and Secret metadata-only handling.

## Decisions

- Use Kubernetes client-go instead of shelling out to `kubectl`.
- Keep Kubernetes collection read-only and out-of-cluster.
- Use standard kubeconfig resolution, with explicit `--kubeconfig` and
  `--kube-context` overrides.
- `scan eks` attempts Kubernetes collection by default so EKS and in-cluster facts
  land in one snapshot; `--skip-kubernetes` keeps the AWS-only path.
- Use typed clients for common resources, discovery for API surface breadth,
  dynamic client for CRD/APIService registration, and metadata client for Secrets.
- Do not read Secret data values from the Kubernetes API. Secrets are currently
  metadata-only in the snapshot.
- Preserve ConfigMap data and other visible non-secret configuration because the
  user explicitly said only Secret-like data needs redaction.
- Human output can remain an interim renderer, but it must show useful facts and
  relationships rather than only counts.

## Rejected Alternatives

- Keeping human output as summary counts: rejected by the user because it was not
  useful for reviewing capabilities.
- Reading typed Secret objects and omitting values only at render time: rejected
  because the process would still fetch Secret data.
- Treating add-on or component presence as a capability conclusion: deferred.
- Building the complete interactive TUI in this slice: deferred until data
  coverage is more useful.

## Constraints

- Go version declarations remain `1.26.0`.
- Secret values must not be collected or emitted.
- Other visible configuration does not need broad redaction.
- Kubernetes API permissions may be partial; coverage must show denied,
  unavailable, partial, and complete reads.
- RuntimeClass declarations and node runtime versions are API-visible facts only;
  they do not prove node-local runtime handler installation.
- Workload topology represents configured references and selector evidence, not
  live application traffic.
- Output formatting remains subject to later review.

## Evaluation Plan

- Run `go fmt ./...`, `go mod tidy`, `make build`, and `make check`.
- Smoke test `./bin/teleskope scan k8s --help` and
  `./bin/teleskope scan eks --help`.
- Smoke test missing kubeconfig handling and verify it returns coverage instead
  of crashing.
- Renderer tests must assert that human output exposes Ingress routes, PVC access
  modes, CSI driver relationships, CRI runtime distribution, CNI candidates, and
  workload-to-PVC relationships.
- On a real cluster, verify ConfigMap data appears while Secret values do not.
- On a real EKS cluster, verify `scan eks` merges AWS-side and Kubernetes-side
  facts into the same snapshot.

## Open Questions

- Which real EKS/Kubernetes cluster should define the first live fixture?
- Should Secret key names and Secret type be collected through a separate
  explicit mode, or should Secrets remain metadata-only?
- Should the next output iteration move to a dedicated TUI/table library rather
  than plain `tabwriter`?
- How much CRD object data should be collected by default without component
  adapters?

## Links

- Broader provenance: `.provenance/2026-09-05-eks-inventory.md`
- Previous EKS collector provenance:
  `.provenance/2026-09-05-eks-collector.md`
- Design files: `docs/architecture.md`, `docs/eks-inventory-v1.md`
- Related issues/tasks: none recorded.
