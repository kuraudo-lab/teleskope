# Hub Fleet Search Provenance

## Trigger

The user accepted issue 21 and asked to start GitHub issue 23:
fleet-wide search and resource index for the multi-cluster hub.

## Scope

- `internal/hub` in-memory store and HTTP API.
- Embedded hub UI rendered by `internal/hub/html.go`.
- Stored cluster envelopes only; no Kubernetes or AWS API access from the hub.

## Conversation Summary

The work follows issue 23's goal: search workloads, images, namespaces,
services, gateways, PVCs, and IAM roles across clusters already reporting to the
hub. The search should use the hub's latest accepted envelopes and link results
back to cluster drilldown pages.

## Design Diff

- Add a hub resource-search response shape and an index builder over stored
  envelopes.
- Add a same-origin HTTP endpoint such as `/api/search?q=...`.
- Add a compact search panel to the existing hub page.
- Keep the search index derived on read from memory-resident envelopes, avoiding
  durable storage or external credentials in this block.

## Decisions

- Build search results from stored envelopes, not live cluster clients, so hub
  remains credential-free.
- Treat the search query as a case-insensitive substring match over structured
  fields: kind, name, namespace, image, IAM role, and cluster metadata.
- Link every result to the cluster drilldown URL rather than inventing deep
  resource anchors the drilldown page does not currently expose.
- Keep API output small and predictable with a bounded result count.

## Rejected Alternatives

- Full-text search library or persistent index: rejected as unnecessary for the
  memory-only first hub implementation.
- Browser-only indexing from `/api/clusters`: rejected because fleet summaries do
  not contain resource-level evidence.
- Hub-side cluster API discovery: rejected because it would violate the
  credential-free hub boundary.

## Constraints

- Do not log bearer tokens or add provider credentials to hub behavior.
- Search must not trigger collection.
- Preserve existing hub API and drilldown behavior.
- Avoid broad UI redesign while adding the search workflow.

## Evaluation Plan

- Add focused tests for resource index contents and HTTP search behavior.
- Add embedded UI hook tests for the search controls and endpoint usage.
- Run `go test ./internal/hub`.
- Run repository checks with temporary Go caches:
  `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`
- Run the same-cache `make build`.
- Run `git diff --check`.

## Open Questions

- None for this first search slice.

## Links

- Design files: `.provenance/2026-09-15-hub-fleet-search.md`
- Related issue: https://github.com/kuraudo-lab/teleskope/issues/23
