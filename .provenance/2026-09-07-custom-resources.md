# Custom Resources Provenance

## Trigger

The user asked Teleskope to add a CRD section that shows two related views in
one place: a `CRD instance -> CRD type` list and a `CRD type -> CRD instance
count` list.

## Scope

This record covers Kubernetes CRD instance collection and presentation:

- `internal/inventory/snapshot.go`
- `internal/k8s/collector.go`
- `internal/k8s/collector_test.go`
- `internal/render/human.go`
- `internal/render/human_test.go`
- `internal/report/report.go`
- `internal/report/report_test.go`

## Conversation Summary

The scanner already collected CRD registration metadata. The user now needs
runtime CRD evidence that is easier to review: which custom resource instances
exist, which CRD type each instance belongs to, and how many instances each CRD
type has. The presentation should keep both views in the same section rather
than scattering the information across unrelated tables.

## Design Diff

The Kubernetes snapshot gains:

- `kubernetes.customResourceInstances` for CRD-backed object instances and their
  owning CRD type metadata.
- `kubernetes.customResourceCounts` for per-CRD type instance and namespace
  counts.

The collector lists CRDs first, chooses the storage/served version for each CRD,
then uses the dynamic client to list instances according to CRD scope. Namespaced
CRDs are listed across all namespaces, while cluster-scoped CRDs are listed at
cluster scope. Failures for individual CRDs are captured as coverage rather than
aborting the scan.

Human and Markdown renderers add one `custom-resources` section containing a CRD
type count table and a CRD instance mapping table.

## Decisions

- Drive CRD instance collection from CRD registration metadata because it gives
  the scanner group, plural, scope, and version information without hardcoding
  custom APIs.
- Prefer a storage version that is also served; fall back to storage-only, then
  served-only, so the collector has a deterministic version choice.
- Keep zero-instance CRD types in the count list because absence is meaningful
  inventory evidence.
- Record instance labels and owner references while avoiding full custom object
  specs, keeping the report concise and less likely to include sensitive
  arbitrary custom fields.
- Treat per-CRD list failures as partial coverage so one denied CRD does not hide
  the rest of the custom resource inventory.

## Rejected Alternatives

- Listing only CRD definitions was rejected because the requested output needs
  actual CRD instances.
- Listing full custom resources was rejected because CRD schemas are arbitrary
  and full specs could be noisy or sensitive.
- Splitting type counts and instance mappings into separate report sections was
  rejected because the user explicitly asked for a single section.

## Constraints

- The collector remains read-only.
- Secret values remain excluded from collection.
- Go version stays at `1.26.0`.
- CRD instance collection must tolerate RBAC gaps and partial failures.

## Evaluation Plan

- Run `go fmt ./...`.
- Run `make build`.
- Run `make check`.
- Add dynamic fake client tests for CRD instance collection and type counts.
- Add human and Markdown renderer assertions for the combined custom resources
  section.

## Open Questions

- Whether future reports should include selected custom resource spec/status
  fields through allowlisted extractors.
- Whether large clusters need limits or sampling for very high-cardinality CRDs.

## Links

- Design files: none
- Related issues/tasks: none
