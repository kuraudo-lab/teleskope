# Analysis Runner Refactor Provenance

## Trigger

The user asked to start work on GitHub issue #2, "Refactor shared Analysis
Action module", and explicitly required automated tests to protect the
refactoring result. After reviewing and accepting the implementation, the user
asked to record provenance and then create the implementation commit, with the
implementation commit explicitly closing issue #2 if the work is complete.

## Scope

This record covers the shared LLM analysis execution path used by:

- `internal/analysis`
- `internal/live`
- `internal/hub`
- `internal/advisory`
- `internal/cli`

The behavior covered is configuration loading, timeout handling, context sizing
metadata, provider invocation, Markdown rendering, request error phase
classification, event/log emission, and simple revision-based cache/running
guard behavior.

## Conversation Summary

Issue #2 was created during backlog grooming as an architecture-enabling
refactor before adding page-scoped analysis, custom prompts, conversation, and
broader hub analysis features. The codebase already had optional LLM analysis
working in CLI, live serve, hub cluster drilldown, and advisory uploads, but the
execution sequence was duplicated in multiple modules.

The accepted implementation direction was to introduce a deeper analysis runner
module. Callers should continue to own their transport-specific behavior:
snapshot lookup, HTTP status mapping, response serialization, and local event
sinks. The shared runner should own the repeated execution behavior so future
analysis features do not spread prompt/provider/cache/timeout semantics across
live, hub, CLI, and advisory code.

## Design Diff

- Add an `internal/analysis` runner with a small execution interface for an
  analysis job.
- Add an `internal/analysis` cache/running guard for the current revision-keyed
  live and hub use cases.
- Keep existing HTTP and CLI surfaces stable while routing execution through the
  shared runner.
- Keep future richer cache-key work scoped to the follow-up issue #3.

No user-facing command shape or web endpoint shape changes are intended by this
refactor.

## Decisions

- Keep provider configuration and analyzer construction inside the runner when
  no analyzer is injected, so live, hub, CLI, and advisory do not duplicate that
  sequence.
- Keep snapshot lookup and HTTP status mapping in adapters, because those are
  transport concerns and differ between live, hub, and advisory.
- Keep the runner cache key intentionally small for this issue: subject plus
  revision. Broader prompt, scope, model, web-search, and conversation
  invalidation belongs to issue #3.
- Preserve fake analyzer injection so automated tests and local development do
  not require provider credentials.
- Include advisory analysis in the refactor because it used the same duplicated
  execution pattern as live, hub, and CLI.

## Rejected Alternatives

- Do not leave live, hub, and CLI with independent config/timeout/provider
  execution paths; that would make page-scoped analysis and conversation
  duplicate more logic.
- Do not move snapshot lookup into the runner in this slice; the lookup source
  is adapter-specific and would make the runner interface wider.
- Do not introduce the full future cache-key design in this issue; issue #3
  covers page scope, prompt fingerprint, model, web search, and conversation.
- Do not make real provider calls part of the default test suite.

## Constraints

- LLM analysis remains optional and must continue to work with injected fake
  analyzers.
- Browser JavaScript must not receive provider credentials.
- Existing CLI commands and HTTP endpoints must remain compatible.
- Deterministic scan, compare, advisor, report, live, and hub behavior must not
  depend on provider availability.
- Tests should verify the shared runner directly instead of asserting duplicated
  internals in each adapter.

## Evaluation Plan

- Add `internal/analysis` runner tests covering cache miss/cache hit, fake
  analyzer execution, provider error phase classification, timeout override,
  and Markdown output.
- Run focused package tests for `internal/analysis`, `internal/live`,
  `internal/hub`, `internal/advisory`, and `internal/cli`.
- Run full repository checks through `make check`.
- Run `make build`.
- Run `git diff --check`.

Validation was performed after implementation using:

- `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`
- `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make build`
- `git diff --check`

## Open Questions

- The richer cache key for scoped analysis, custom prompts, web search, model
  changes, and conversation remains for issue #3.
- Future page-scoped analysis should decide how much browser state is sent to
  the runner versus reconstructed from snapshot evidence server-side.

## Links

- Design files: `docs/llm-analysis.md`, `docs/architecture.md`
- Related issues/tasks: https://github.com/kuraudo-lab/teleskope/issues/2
