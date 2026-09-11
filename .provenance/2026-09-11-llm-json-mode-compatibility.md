# LLM JSON Mode Compatibility Provenance

## Trigger

The user reported that using `gpt-oss-120b` with Teleskope LLM analysis failed with a provider-side HTTP 400 error, specifically `missing field name`.

## Scope

This record covers OpenAI-compatible LLM request compatibility for Teleskope analysis:

- `docs/llm-analysis.md`
- `README.md`
- `internal/analysis/config.go`
- `internal/analysis/openai.go`
- related tests in `internal/analysis/`

It does not cover adding provider-specific APIs, changing prompts per model family, or replacing the OpenAI-compatible chat completions adapter.

## Conversation Summary

The current OpenAI-compatible adapter always sends `response_format: {"type":"json_object"}` to request JSON mode. The user clarified that the observed error is returned as a provider HTTP 400, which means the request payload is rejected before Teleskope parses the assistant response. Some `gpt-oss-120b` deployments and compatible gateways do not fully support OpenAI JSON mode even when they accept chat completions requests.

The chosen design is to keep JSON mode enabled by default for providers that support it, while adding a config escape hatch for providers that reject it. When JSON mode is disabled, Teleskope still instructs the model to return strict JSON and uses tolerant extraction to parse a JSON object if the model wraps it in light prose.

## Design Diff

- `docs/llm-analysis.md` documents `llm.json_mode` in `$HOME/.teleskope/config.yaml`.
- The OpenAI-compatible adapter should omit `response_format` when `llm.json_mode: false`.
- Provider-side HTTP 400 errors that resemble JSON mode compatibility failures should suggest setting `llm.json_mode: false`.
- Response parsing should tolerate a single JSON object surrounded by small non-JSON text when JSON mode is disabled.

## Decisions

- Add `llm.json_mode` as an optional boolean config field.
- Default `json_mode` to true to preserve current behavior for providers with proper OpenAI JSON mode support.
- Set `json_mode: false` for providers such as some `gpt-oss-120b` deployments that reject JSON mode.
- Keep prompt-level strict JSON requirements even when provider-level JSON mode is disabled.
- Enhance error messages instead of hiding provider 400 details.

## Rejected Alternatives

- Removing `response_format` globally was rejected because it would weaken output reliability for providers that support JSON mode.
- Hard-coding model-name behavior for `gpt-oss-120b` was rejected because compatibility varies by serving gateway and deployment.
- Retrying automatically without `response_format` after a 400 was rejected for the first version because it could cause duplicate provider calls and obscure configuration behavior.

## Constraints

- Teleskope should remain OpenAI-compatible rather than provider-specific.
- LLM analysis should continue to produce structured `analysis.Result` output.
- Provider credentials and model configuration remain local in `$HOME/.teleskope/config.yaml`.
- Tests should verify default JSON mode, disabled JSON mode, and the 400 hint.

## Evaluation Plan

- Run `GOCACHE=/private/tmp/teleskope-go-cache go test ./internal/analysis`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache go test ./...`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache make build`.
- Run `git diff --check`.
- Manually verify that setting `llm.json_mode: false` omits `response_format` from the chat completions request.

## Open Questions

- Whether future versions should add a provider capability preset for common local serving gateways.
- Whether failed JSON mode requests should offer an explicit CLI retry flag.

## Links

- Design files: `docs/llm-analysis.md`, `.provenance/2026-09-11-llm-analysis-serve-ui.md`
- Related implementation: `internal/analysis/openai.go`, `internal/analysis/config.go`
