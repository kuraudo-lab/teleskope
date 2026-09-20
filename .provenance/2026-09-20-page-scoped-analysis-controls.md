# Page-scoped Analysis Controls Provenance

## Trigger

The user asked to start GitHub issue #17, “Page-scoped Analyze with AI
buttons.” The issue requires EKS, Nodes, Workloads, Network, Storage, and
Security pages to analyze their current context, send the active page scope
and current revision, and distinguish facts, inference, and limitations in the
rendered result.

## Current State

The shared analysis runner and browser request contract already accept page,
namespace, resource type, and selected references. Live and hub pages expose a
single toolbar action, but the action redirects every result to the Advisory
page. The browser does not include the displayed revision in its request, and
the scan context only narrows workloads and running images. As a result, a
Nodes, Network, Storage, Security, or EKS scope does not carry the page's
resource evidence to the model.

## Scope

- Add an analysis surface to each of the six issue-listed subpages while
  retaining the toolbar action for cluster-wide and other views.
- Build one browser request path used by toolbar and page actions. It must send
  the current revision plus page id, namespace, page resource type, and any
  selected resource reference.
- Reject an explicitly stale browser revision in live and hub handlers while
  preserving compatibility for older callers that omit revision.
- Extend the trimmed scan context with page-relevant EKS, node, workload,
  network, storage, and security evidence. Apply namespace and selected-ref
  filters without accepting browser-owned snapshot data.
- Render scoped results in the originating page and group result items into
  observed facts and inference, followed by a separate limitations area.

## Decisions

- The server-owned snapshot and revision remain authoritative. The client
  revision is only an optimistic concurrency check against the displayed view.
- Page controls reuse the existing `/api/analyze` and
  `/api/cluster/analyze` endpoints and the shared analysis runner.
- Page resource types use stable plural identifiers: `eks`, `nodes`,
  `workloads`, `network`, `storage`, and `security`.
- An item is rendered as an observed fact only when its `basis` explicitly
  identifies observed, factual, or deterministic evidence. Other conclusions
  are rendered as inference so missing classification cannot be mistaken for a
  collected fact.
- LLM output remains optional narrative. The inventory snapshot and
  deterministic Advisor output remain the evidence sources of truth.

## Compatibility and Safety

- Empty analysis request bodies continue to work.
- Provider credentials remain server-side.
- Secret values remain excluded; security context contains metadata and policy
  configuration only.
- Static reports keep analysis actions hidden because no analysis endpoint is
  available.

## Evaluation Plan

- Add analysis context tests for all six page resource groups and filtering.
- Add live and hub HTTP tests for accepted and stale revisions.
- Add embedded UI contract tests for six page controls, revision-bearing
  requests, stable resource types, and facts/inference/limitations rendering.
- Run focused package tests, embedded JavaScript syntax validation, full
  `make check`, `make build`, and `git diff --check` with the established
  temporary Go caches.

## Links

- Issue: https://github.com/kuraudo-lab/teleskope/issues/17
- Prior contract: `.provenance/2026-09-14-scoped-analysis-request.md`
- Design: `docs/llm-analysis.md`
