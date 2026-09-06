# Running Images Provenance

## Trigger

The user asked Teleskope to expose all Running container images, then clarified
that running images need their own deduplicated section and that the pod count
must be displayed after the image name.

## Scope

This record covers Kubernetes scan inventory and report presentation:

- `internal/inventory/snapshot.go`
- `internal/k8s/collector.go`
- `internal/k8s/collector_test.go`
- `internal/render/human.go`
- `internal/render/human_test.go`
- `internal/report/report.go`
- `internal/report/report_test.go`

## Conversation Summary

The initial implementation extracted actual Running container instances from Pod
status and preserved image, imageID, containerID, runtime, pod, node, service
account, and workload context. The user then clarified that running images
should be presented as a separate deduplicated section. The implementation
therefore keeps detailed running container records while adding a separate
image-level aggregate.

## Design Diff

The Kubernetes snapshot gains two related data surfaces:

- `kubernetes.runningContainers` records each observed container whose status is
  `State.Running`.
- `kubernetes.runningImages` records a deduplicated image list derived from those
  running containers.

The image aggregate groups by image name, counts distinct pods, counts running
container instances, records observed image IDs, runtimes, namespaces, and
deduplicated workloads. Human and Markdown outputs render an independent
`running-images` / `Running images` section where the image name is shown with
the pod count, for example `repo/web:v1 (pods=3)`.

## Decisions

- Use Pod container status rather than workload templates to identify Running
  images because status reflects what is actually running now.
- Keep `runningContainers` as detail and add `runningImages` as the primary
  deduplicated section, so consumers can use either the summary or raw instance
  evidence.
- Deduplicate images by the Kubernetes-reported image name from container status
  or spec, not by digest, because the user asked for image-name presentation.
- Count pods by unique `namespace/pod`, so multiple containers in the same pod
  using one image do not inflate pod count.
- Keep container count separately because it remains useful for capacity and
  blast-radius review.
- Include app, init, and ephemeral containers when they are in Running state.
- Resolve pod owners upward through known workload records when possible, such
  as ReplicaSet to Deployment.

## Rejected Alternatives

- Only listing per-container rows was rejected because the user asked for a
  deduplicated image section.
- Deduplicating by imageID/digest was rejected for the summary section because it
  would split a single image name across multiple rows and would not match the
  requested display.
- Counting containers instead of pods after the image name was rejected because
  the user explicitly requested pod count after the image name.

## Constraints

- Secret values remain excluded from collection.
- The collector remains read-only.
- Go version stays at `1.26.0`.
- Existing report and human outputs should keep the running container detail
  available after the new image summary.

## Evaluation Plan

- Run `go fmt ./...`.
- Run `make build`.
- Run `make check`.
- Add collector tests for Running-only filtering and image aggregation by pod
  count.
- Add renderer tests for the independent running image section and `(pods=N)`
  display.

## Open Questions

- Whether future output should support filtering running images by namespace,
  workload, runtime, or image registry.
- Whether image grouping should optionally normalize tags and digests.

## Links

- Design files: none
- Related issues/tasks: none
