# LLM Analysis Design Provenance

## Trigger

The user asked to evaluate the feasibility of adding LLM-backed analysis and
summary capabilities to Teleskope for different workflows, including scan,
compare, and advisory. After the feasibility discussion, the user asked to write
the design into `docs/` for later implementation reference.

## Scope

This record covers the design documentation for optional LLM analysis in
Teleskope. The intended future implementation scope includes scan analysis,
compare analysis, advisory web analysis, prompt selection, model context
building, optional web search, result rendering, and provider configuration.

Files included in this provenance surface:

- `docs/llm-analysis.md`
- `docs/architecture.md`

## Conversation Summary

The design discussion concluded that LLM support is technically feasible and fits
best as a post-collection narrative analysis layer. Existing collectors should
continue to gather factual inventory, and existing deterministic modules such as
`internal/advisor` and `internal/compare` should remain the source of stable,
testable conclusions. LLM output should consume snapshots and deterministic
reports, then produce explanations, summaries, workload purpose inference, and
migration guidance.

The user specifically called out scan analysis as a use case where container
image information can help explain what a workload is likely doing. The design
records that image-based interpretation is useful but must be marked as
inference rather than fact, and that web search may help identify public
controllers, operators, and images when explicitly enabled.

## Design Diff

`docs/llm-analysis.md` was added as a new design note. It defines the direction,
scan/compare/advisory use cases, a provider-neutral analysis module shape,
context-builder requirements, prompt registry requirements, optional web search,
configuration options, output artifacts, privacy constraints, testing strategy,
implementation order, and explicit non-goals.

`docs/architecture.md` was updated to link the new LLM analysis design from the
first-phase architecture direction. The architecture note now states that
optional LLM-backed summaries should be a post-collection narrative layer rather
than a replacement for advisor or compare rules.

## Decisions

- Add LLM analysis as an optional layer after collection and deterministic
  analysis, so core scan, compare, advisory, serve, and export behavior remains
  usable without provider credentials.
- Keep `internal/advisor` and `internal/compare` deterministic and testable;
  LLM output should explain, group, prioritize, and summarize their results.
- Use different prompts and context builders for scan, compare, and advisory
  because each workflow has a different user goal.
- Introduce a small provider-neutral analysis interface in a future
  `internal/analysis` module, with provider adapters hidden behind that seam.
- Build trimmed, deterministic model contexts instead of sending entire
  snapshots by default.
- Treat container image interpretation and workload purpose detection as
  inferred conclusions with confidence, evidence, and resource references.
- Keep web search optional and explicit, with citations required when search
  contributes to a conclusion.
- Keep provider credentials server-side or CLI-side; browser JavaScript must not
  call LLM providers directly.

## Rejected Alternatives

- Replacing deterministic advisor or compare logic with LLM conclusions was
  rejected because it would make results harder to test and less reproducible.
- Sending whole snapshots to a provider by default was rejected because snapshots
  can be large and can contain sensitive operational metadata.
- Making LLM analysis mandatory during scan, compare, advisory, serve, or export
  was rejected because Teleskope should remain useful offline and without model
  credentials.
- Calling an LLM provider directly from the web UI was rejected because it would
  expose credentials and provider details to browser code.

## Constraints

- Cluster inventory may contain sensitive metadata such as object names,
  namespaces, image repositories, hostnames, annotations, labels, and IAM ARNs.
- Secret values remain out of scope and must not be sent to an LLM provider.
- LLM failures, disabled configuration, unavailable credentials, slow responses,
  or web search failures must not break deterministic outputs.
- Prompt versions and provider metadata should be recorded so future reports can
  explain how an analysis was produced.
- Provider integration tests should not run in the default test suite.

## Evaluation Plan

- Add tests for scan and compare/advisory context builders with representative
  snapshots.
- Add prompt selection tests for each use case.
- Add fake analyzer tests for CLI and web handlers.
- Add JSON schema or golden tests for analysis result rendering.
- Add error-path tests proving deterministic commands and report rendering still
  work when LLM analysis is disabled or fails.
- Keep real provider calls behind explicit manual or integration-test flags.

## Open Questions

- Exact CLI command and flag names for the first implementation.
- Exact environment variable names for provider, model, API key, and web search.
- Whether redaction should default to conservative masking or be user-configured
  per field category.
- Which LLM provider adapter should be implemented first.

## Links

- Design files: `docs/llm-analysis.md`, `docs/architecture.md`
- Related issues/tasks: None
