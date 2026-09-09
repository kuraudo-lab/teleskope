# Live Polling Inventory Provenance

## Trigger

The user requested research on evolving Teleskope into an out-of-cluster and
in-cluster live probe, with a simple web UI embedded in the Go binary and support
for periodic refresh or resource watches. After discussing shared snapshots, the
user authorized the first delivery step: embedded serving and polling. The user
then requested lower AWS request frequency and confirmed default configuration
behavior, and finally requested provenance and implementation commits.

This record is a retrospective structured summary of that conversation. The
implementation already existed in the working tree when this record was written;
the provenance commit precedes the implementation commit, not the coding work.
It is not a verbatim transcript or evidence of a pre-implementation review.

## Scope

- `serve k8s` and `serve eks` command orchestration in `internal/cli`.
- Shared snapshot publication, source health, polling, and HTTP in `internal/live`.
- Reusable clients in `internal/k8s` and `internal/awseks`.
- Shared offline/live embedded UI in `internal/report`.
- Research, architecture, usage documentation, and focused regression tests.

## Conversation Summary

A shared snapshot means one process-local published inventory used by every
browser, with replacement only after a new result is ready. Browser reads must
not trigger collection. Kubernetes and AWS refresh independently so an AWS error
does not block Kubernetes updates. The existing embedded HTML and vanilla
JavaScript are reused rather than adding a separate frontend service or build.

The first implementation uses polling. Watch/informer collection and packaged
in-cluster deployment remain later steps. The initially implemented five-minute
AWS interval was increased to fifteen minutes at the user's request to reduce
AWS API traffic. No real cluster was used during implementation verification.

## Design Diff

- `docs/live-probe-research.md` records the original research and staged direction,
  including watch semantics and later in-cluster considerations.
- `docs/architecture.md` describes the implemented source workers and publication
  boundary; it accompanies the implementation commit because it reports behavior.
- `README.md` documents serving, defaults, stale data, and the HTTP envelope; it
  also accompanies the implementation commit.

## Decisions

- Add `serve` while preserving run-once scans and offline reports.
- Start each source immediately, then wait after each completed attempt. Default
  Kubernetes delay is one minute; AWS delay is fifteen minutes, both with up to
  ten percent positive jitter and a configurable per-attempt timeout.
- Reuse clients and their credential mechanisms between sequential scans.
- Publish owned immutable data, with independent latest-attempt status and coverage.
- Accept usable partial initial inventory. Retain the entire previous source on
  failed refresh or coverage regression for an already observed resource, marked
  stale. Successful empty lists remove previous objects.
- Serve read-only HTTP from loopback by default. Browsers poll every five seconds
  with conditional requests; status changes also invalidate the response ETag.
- Preserve filters, active section, topology view, scrolling, and selected details.
  Page-update pause affects only that browser. Fix topology pointer capture so
  node clicks open details correctly.
- Preserve conventional AWS SDK configuration and credential resolution, and
  client-go kubeconfig/current-context defaults. Service flags use CLI or built-in
  defaults, without introducing custom environment variable bindings.

## Rejected Alternatives

- A separately deployed frontend or Node build chain: unnecessary for this UI.
- A collector per browser or HTTP-triggered scans: multiplies API load.
- Watch implementation in this first step: requires cache and recovery lifecycle.
- Kubernetes Event objects as inventory truth: best-effort diagnostic records.
- Mixing newly collected resources with failed old fragments in this first step:
  requires explicit relationship dependency rules.
- AWS polling at Kubernetes frequency: extra requests for comparatively slow
  changing metadata; the final default is fifteen minutes.

## Constraints

- Collection remains read-only and Secret retrieval stays metadata-only.
- Source failures must not terminate the web server or another source's worker.
- Collection attempts must not overlap, and shutdown must cancel pending work,
  including Kubernetes discovery requests.
- HTTP has no built-in authentication. ConfigMap contents and infrastructure
  details can be present, so nonlocal exposure requires an access boundary.
- Snapshot publication does not imply a transactional cluster-wide observation.
- Changing the external current-context after startup does not retarget a running
  process. AWS region is not inferred from kubeconfig.

## Evaluation Plan

- Run Go tests, vet, and race checks for live, CLI, and Kubernetes packages.
- Verify stale retention, coverage regression, successful deletion, source
  independence, HTTP conditional requests, concurrent readers, and cancellation.
- Verify metadata-only Secret requests and cancellable API discovery using a
  local mock API.
- Exercise the embedded UI against a local simulated Kubernetes API: initial
  failure, updated Pod details with filters retained, outage/stale display,
  pause/resume, and successful resource deletion.
- Build the executable and verify CLI defaults and graceful SIGTERM shutdown.

These checks passed during this task before the commit request. This records
local/mock validation; real EKS permissions, request volume, and large-cluster
performance have not been validated.

## Open Questions

- Automatically verify that `--cluster` and kubeconfig identify the same cluster.
- Refine EKS failure retention to subcategories if whole-source retention becomes
  too conservative in practice.
- Implement informer watches, packaged in-cluster deployment, and broader access
  controls in subsequent work.

## Links

- Research: `docs/live-probe-research.md`
- Architecture: `docs/architecture.md`
- Usage: `README.md`
- Related issues/tasks: current Codex task; no external issue provided.
