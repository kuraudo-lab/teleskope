# Scoped Analysis Request Provenance

## Trigger

The user asked to start work on GitHub issue #3 after accepting the shared
analysis runner refactor from issue #2. The issue covers preparing LLM analysis
for page-level analysis buttons, user-provided prompts, and future conversation
flows without reintroducing duplicated scope or cache logic across live and hub
surfaces.

After reviewing and accepting the implementation, the user asked to record
provenance and then create the implementation commit, with the implementation
commit explicitly closing issue #3 if the work is complete.

## Scope

This record covers the analysis request contract and cache semantics used by:

- `internal/analysis`
- `internal/live`
- `internal/hub`
- `internal/report`

The behavior covered is browser-provided analysis scope, custom prompt and
conversation input, request sanitization, analysis context shaping, cache key
fingerprinting, and hub cluster analysis concurrency.

## Conversation Summary

Issue #3 was created during backlog grooming as the next step after centralizing
analysis execution. The product direction was to make LLM analysis more
contextual: each UI subpage should eventually be able to trigger an analysis
for the current view, users should be able to supply their own prompt, and
later conversation should not require another duplicated execution path.

The accepted implementation direction was to extend the shared
`analysis.Request` shape first, then let live and hub adapters decode a common
client request body. The runner cache key should stop treating a cluster
revision as the whole identity; prompt, scope, conversation, web search, model,
and subject all affect what result is valid. Hub analysis should remain scoped
per cluster so one long-running cluster analysis does not block unrelated
clusters.

## Design Diff

- Extend `analysis.Request` with `Scope`, `CustomPrompt`, and `Conversation`.
- Add a browser client request decoder that applies only safe client-controlled
  fields onto server-owned request facts.
- Include scope in the model context and initially use it to narrow workload and
  running-image context by namespace, selected references, and resource type.
- Build analysis cache keys from subject, revision, model, custom prompt
  fingerprint, scope fingerprint, conversation fingerprint, and web-search flag.
- Teach live and hub HTTP analysis handlers to parse optional JSON request
  bodies and route them through the shared runner.
- Teach the embedded live UI to submit the current view scope with the existing
  Analyze button and to consider request scope when deciding whether a snapshot
  has already been analyzed.

## Decisions

- Keep snapshot ownership server-side. Browser input can narrow analysis, but it
  cannot supply or replace the trusted inventory snapshot.
- Sanitize selected object references before using them in scope or cache
  fingerprints so UID and runtime-only details do not become accidental cache
  identity.
- Put custom prompt and conversation before the final structured model context
  message so the model always receives the authoritative inventory payload last.
- Keep cache logs to subject and revision only; prompt and conversation content
  are represented by fingerprints and should not be emitted into operational
  logs.
- Preserve the existing single Analyze button as the first UI integration point,
  while shaping the request contract so per-page buttons can reuse it later.
- Treat different hub cluster IDs as different cache subjects, allowing
  concurrent analyses for unrelated clusters.

## Rejected Alternatives

- Do not add one-off prompt or scope parsing separately in live and hub; that
  would recreate the duplication removed by issue #2.
- Do not use revision alone as the cache identity; it would incorrectly reuse
  results when users change prompt, scope, conversation, model, or web-search
  behavior.
- Do not allow browser-provided snapshots in analysis requests; that would make
  the analysis path harder to trust and harder to cache.
- Do not block all hub analyses behind one global running flag; users should be
  able to analyze different clusters independently.
- Do not build the full conversation UI in this issue; this issue establishes
  the backend and request contract needed for that later surface.

## Constraints

- LLM provider credentials must remain server-side and never be embedded in the
  browser.
- Existing `POST /api/analyze` and `POST /api/cluster/analyze` callers with an
  empty body must remain compatible.
- The embedded UI remains inline CSS and vanilla JavaScript inside the binary.
- Tests must cover cache behavior and HTTP request propagation without requiring
  real LLM credentials.
- Operational logs should remain useful without exposing prompt or conversation
  text.

## Evaluation Plan

- Add `internal/analysis` tests proving cache keys miss for the same revision
  when prompt, scope, conversation, or web-search settings differ.
- Add context tests proving scoped requests are included in model context and
  narrow workload/running-image payloads.
- Add OpenAI-compatible request tests proving custom prompt and conversation are
  included in provider messages.
- Add live HTTP tests proving posted scope, prompt, and conversation reach the
  analysis module and that different prompts miss cache.
- Add hub HTTP tests proving posted scope reaches the analysis module and that
  analyses for different clusters can run concurrently.
- Run full repository validation through `make check`, `make build`, and
  `git diff --check`.

Validation was performed after implementation using:

- `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`
- `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make build`
- `git diff --check`

## Open Questions

- Future UI work still needs dedicated per-subpage analysis controls and a
  custom prompt or chat input surface.
- Future analysis context shaping can grow beyond workloads and running images
  as more pages expose resource-specific scopes.
- Future config plumbing may pass the resolved model into cache-key creation
  before provider construction instead of relying on injected tests for model
  coverage.

## Links

- Design files: `docs/llm-analysis.md`, `docs/architecture.md`
- Related issues/tasks: https://github.com/kuraudo-lab/teleskope/issues/3
