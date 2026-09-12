# LLM Opportunistic JSON Fallback Provenance

## Trigger

The user reported that older OpenAI-compatible models such as gpt-oss-120b can reject provider-side JSON mode while still returning JSON-shaped content in ordinary text responses. When Teleskope treated those responses as plain text, the analysis rendered poorly in CLI and web outputs.

## Scope

This record covers Teleskope's LLM analysis response handling for scan, compare, and advisory analysis workflows, specifically the OpenAI-compatible provider path in `internal/analysis/openai.go`, tests in `internal/analysis/openai_test.go`, and configuration documentation in `README.md` and `docs/llm-analysis.md`.

## Conversation Summary

The user asked whether Teleskope should support older models by either parsing plain-text responses as JSON when possible before falling back to text, or by switching request and response behavior to Markdown-only text. The accepted direction was to keep the existing single `json_mode` switch but give it narrower semantics: when disabled, Teleskope should omit OpenAI `response_format` from the request, yet still opportunistically parse structured JSON from the response. Only responses that cannot be parsed as JSON should fall back to preserved plain text.

## Design Diff

No separate design file was prepared before implementation. The intended design change is to update the analyzer response parser so it can recognize structured JSON even when returned through a text-only provider response path, including JSON surrounded by prose, fenced JSON, and JSON encoded as a string. Documentation should clarify that `json_mode: false` disables provider-side JSON mode, not Teleskope's response-side JSON parsing.

## Decisions

- Keep `json_mode` as the single compatibility switch to avoid adding another configuration knob.
- Treat `json_mode: false` as a request-side compatibility mode for providers that reject `response_format`.
- Parse response content as structured JSON whenever possible so CLI, web, and export rendering can keep using the normal analysis model.
- Preserve the previous plain-text fallback for genuinely non-JSON responses from models that cannot or do not follow the structured prompt.

## Rejected Alternatives

- Markdown-only request and response mode was rejected because it would lose structured sections, severity, resources, limitations, citations, and consistent web rendering.
- Adding a second parse-mode configuration option was rejected for now because the existing behavior can be improved without making the configuration surface more complex.

## Constraints

- Providers that reject OpenAI JSON mode must not receive `response_format` when `json_mode: false` is configured.
- Existing strict behavior should remain when `json_mode` is enabled: invalid analysis JSON should still be reported as an error.
- Plain-text fallback must remain available for models that return useful analysis text but no parseable JSON.
- The parser should be conservative enough to avoid treating arbitrary prose as structured output.

## Evaluation Plan

- Add unit tests for JSON returned while `json_mode` is disabled.
- Add unit tests for JSON encoded as a string in message content.
- Add unit tests for fenced JSON in a text response.
- Keep the existing plain-text fallback and strict JSON-mode failure tests passing.
- Run `go test ./internal/analysis`, `go test ./...`, `make build`, and `git diff --check`.

## Open Questions

- None.

## Links

- Design files: `README.md`, `docs/llm-analysis.md`
- Implementation files: `internal/analysis/openai.go`, `internal/analysis/openai_test.go`
- Related issues/tasks: none
