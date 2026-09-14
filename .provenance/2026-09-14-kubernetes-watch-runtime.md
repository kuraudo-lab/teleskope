# Kubernetes Watch Runtime Provenance

## Trigger

Issue #8 asks Teleskope to implement event-based Kubernetes data updates for
core live resources. The user accepted the #10 publication refactor and asked to
complete #8 with provenance and an implementation commit. This record is a
structured summary, not a raw transcript.

## Scope

This provenance record covers:

- Kubernetes list/watch runtime under `internal/k8s`.
- CLI opt-in wiring for `teleskope serve k8s --watch` and
  `teleskope serve eks --watch`.
- Translation from watch updates into the shared `internal/live` publication
  path.
- Tests for initial list, add/update/delete events, stale watch health, and
  publication-only live sources.
- README usage updates for the new live watch mode.

Implementation files:

- `internal/k8s/watch.go`
- `internal/k8s/watch_test.go`
- `internal/cli/serve.go`
- `internal/cli/cli_test.go`
- `internal/live/live.go`
- `internal/live/live_test.go`
- `README.md`
- `go.mod`
- `go.sum`

## Conversation Summary

The event-based update design from #7 keeps Kubernetes informer mechanics inside
`internal/k8s`, while #10 established `internal/live.PublishSource` as the shared
publication boundary for polling and watch updates. Issue #8 is the first
vertical feature slice that connects those decisions to actual live serve
behavior.

The implementation should use client-go informer/list-watch semantics for stable
typed resources, rebuild immutable Teleskope snapshots from informer stores, and
publish them through the live store with source mode `watch`. Existing polling
must remain the default, and EKS AWS-side inventory must continue polling
independently when `serve eks --watch` is used.

## Design Diff

- Add a watch runtime interface with options, health, and immutable update
  objects.
- Start typed shared informers for core resources and wait for initial cache
  sync before the first publication.
- Rebuild Kubernetes inventory snapshots from informer stores after initial list
  and debounced add/update/delete events.
- Reuse existing mapper and derived index logic for pods, workloads, services,
  storage, policy objects, running containers, and running images.
- Add `--watch` and `--watch-debounce` to live serve commands.
- Register Kubernetes as a publication-only live source when watch mode is
  enabled.

## Decisions

- Keep polling as the default `serve` behavior; watch mode is explicitly
  opt-in.
- Use client-go `SharedInformerFactory` instead of hand-written raw watch HTTP
  loops.
- Cover stable typed core resources first: namespaces, nodes, service accounts,
  pods, workloads, services, endpoint slices, ingresses, persistent volumes,
  persistent volume claims, storage classes, pod disruption budgets, network
  policies, resource quotas, limit ranges, and HPAs.
- Keep CRDs, APIService, Gateway dynamic resources, RBAC, admission webhooks,
  runtime classes, CSI extended storage resources, ConfigMaps, and Secrets on
  the polling/dynamic follow-up path.
- Translate watch failures into live stale/error updates without clearing the
  retained Kubernetes snapshot.
- Keep hub remote-write downstream of live publication.

## Rejected Alternatives

- Replace polling serve behavior by default. This was rejected because #8 should
  be an opt-in first slice with low operational surprise.
- Watch every discovered GVR. This was rejected because dynamic informer
  lifecycle belongs to #9.
- Duplicate inventory mapping logic for watch resources. This was rejected
  because it would fork semantics from the existing collector and reports.
- Let Kubernetes watch updates publish directly to the hub. This was rejected
  because #10 made live publication the revision and envelope boundary.

## Constraints

- Watch add, update, and delete events must advance live snapshot revision after
  a usable debounced rebuild.
- Watch events must not trigger EKS AWS collection.
- `serve eks --watch` must keep AWS-side EKS inventory polling-based.
- Browser requests must remain read-only and must not initiate watch or polling
  collection.
- Watch failures must mark Kubernetes stale or error without clearing retained
  data.
- Existing tests for polling, hub publication, analysis, and exports must remain
  green.

## Evaluation Plan

- Run fake-client watch runtime tests for initial list and pod add/update/delete
  publications.
- Run fake-client error tests for watch health degradation.
- Run live store tests for publication-only Kubernetes sources.
- Run CLI validation tests for `--watch`, `--watch-debounce`, and incompatible
  `--skip-kubernetes --watch`.
- Run full repository `make check`, `make build`, and `git diff --check`.
- Manually verify `./bin/teleskope serve k8s --help` and
  `./bin/teleskope serve eks --help` expose the new flags.

## Open Questions

- Dynamic resource watch support remains intentionally deferred to #9.
- UI representation of watch freshness details remains intentionally deferred to
  #11.

## Links

- Related issue: #8
- Preceding design issues: #7, #10
- Follow-up issues: #9, #11
- Design file: `docs/event-based-kubernetes-updates.md`
