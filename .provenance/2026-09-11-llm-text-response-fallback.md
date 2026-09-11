# LLM Text Response Fallback Provenance

## Trigger

The user reported that `gpt-oss-120b` has a second compatibility issue: after disabling OpenAI JSON mode, the provider may return ordinary text instead of JSON. The user requested handling that response as plain text, controlled by the same `json_mode` switch.

## Scope

This record covers how Teleskope handles non-JSON assistant content from OpenAI-compatible LLM providers when JSON mode is disabled:

- `docs/llm-analysis.md`
- `README.md`
- `internal/analysis/openai.go`
- `internal/analysis/openai_test.go`

It does not cover adding a second config switch, model-specific adapters, or automatic retries.

## Conversation Summary

Teleskope previously added `llm.json_mode: false` to omit `response_format` for providers that reject OpenAI JSON mode. The user then clarified that some `gpt-oss-120b` deployments also return non-JSON text. The selected behavior is to make `json_mode` the single compatibility switch: when enabled, Teleskope expects structured JSON; when disabled, Teleskope accepts either JSON or plain text.

Plain text responses should still fit the existing UI and CLI rendering paths. Therefore, Teleskope should wrap non-JSON content into a normal `analysis.Result` with a summary, a `Text response` section, and a limitation explaining that the provider did not return valid JSON.

## Design Diff

- `docs/llm-analysis.md` documents that `json_mode: false` also accepts plain text provider responses.
- `README.md` documents the same operator-facing configuration behavior.
- The OpenAI-compatible adapter should first try to parse JSON, including a JSON object embedded in light surrounding prose.
- If parsing fails and JSON mode is disabled, preserve the provider response as plain text analysis instead of failing the request.
- If JSON mode is enabled, non-JSON responses should remain an error because JSON mode is expected to produce structured output.

## Decisions

- Use the existing `llm.json_mode` switch for both request JSON mode and response strictness.
- Keep `json_mode` defaulting to true.
- Preserve text in a standard `analysis.Result` so CLI, serve UI, advisory UI, and Markdown rendering can reuse existing paths.
- Add a limitation to plain text results so users know the provider did not return structured JSON.
- Keep strict JSON errors for `json_mode: true` to catch provider or prompt regressions.

## Rejected Alternatives

- Adding a separate `plain_text_response` switch was rejected because the user requested a single `json_mode` control.
- Silently discarding non-JSON text was rejected because it would hide useful provider output.
- Always accepting non-JSON text was rejected because it would weaken validation for providers that support JSON mode.

## Constraints

- The fallback must not change provider request behavior except through `json_mode`.
- The fallback must preserve the existing `analysis.Result` contract.
- Plain text results should render in all existing CLI and web analysis views.
- Tests must prove both disabled-mode fallback and enabled-mode strict failure.

## Evaluation Plan

- Run `GOCACHE=/private/tmp/teleskope-go-cache go test ./internal/analysis`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache go test ./...`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache make build`.
- Run `git diff --check`.

## Open Questions

- Whether future providers need richer response-shape presets beyond the single JSON mode switch.

## Links

- Design files: `docs/llm-analysis.md`, `.provenance/2026-09-11-llm-json-mode-compatibility.md`
- Related implementation: `internal/analysis/openai.go`, `internal/analysis/openai_test.go`
