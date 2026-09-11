# LLM Analysis Serve UI Provenance

## Trigger

The user asked to bring the existing `analyze` capability into the `serve` web UI so that a user can trigger analysis by clicking in the live page. Subsequent UI feedback refined the behavior: the Analyze action should be unavailable until the first scan completes, should not repeatedly call the provider after a single click, should expose better backend logs, and should be visually integrated into the Advisory page.

## Scope

This record covers the live web analysis flow for Teleskope:

- `docs/llm-analysis.md`
- `internal/analysis/`
- `internal/cli/analyze.go`
- `internal/cli/serve.go`
- `internal/live/`
- `internal/report/report.html`
- `internal/report/live.js`
- related tests and README usage notes

It does not cover changing the deterministic Kubernetes collection model, replacing `internal/advisor`, or adding provider-specific web search tools.

## Conversation Summary

The implementation direction is to keep LLM analysis optional and explicitly user-triggered. The CLI supports offline `analyze scan` and `analyze compare`; live `serve` should expose the scan analysis through the local server using the same `$HOME/.teleskope/config.yaml` OpenAI-compatible configuration. The browser must not hold provider credentials and live snapshot polling must not trigger LLM calls.

The user observed that clicking Analyze once appeared to produce repeated requests and no visible result. The diagnosed causes were that live refresh could re-enable the button while an analysis was in progress, duplicate POSTs needed backend protection, completed analysis needed caching for the same live snapshot revision, and the live HTTP server write timeout was shorter than real LLM request duration. The UI should also merge deterministic Advisor and AI Analysis into a single Advisory navigation page, and the Analyze button should read “Analyze with AI” with a stronger AI-style visual treatment.

## Design Diff

- `docs/llm-analysis.md` now states that LLM analysis is configured through `$HOME/.teleskope/config.yaml`, uses an OpenAI-compatible `/v1/chat/completions` provider seam, and treats serve UI analysis as a manual `/api/analyze` action.
- Scan analysis context should be workload-centric: infer workload purpose from names, namespaces, labels, service accounts, images, ports, storage, and network links; include core Kubernetes mechanisms only as migration context rather than standalone capabilities.
- Serve UI analysis should be explicit, idempotent for a live snapshot revision, logged in both Events and terminal output, and isolated from live polling.
- Advisory UI should combine deterministic cluster advisory and optional AI analysis into one page.

## Decisions

- Use the existing `internal/analysis` package as the shared seam for CLI and web analysis.
- Add a live `POST /api/analyze` endpoint instead of invoking providers from browser JavaScript.
- Read LLM provider configuration from `$HOME/.teleskope/config.yaml` by default, with test injection through live handler options.
- Cache successful live analysis by snapshot revision so repeated POSTs for the same published snapshot do not call the provider again.
- Reject concurrent analysis requests with HTTP 409 and a clear event message.
- Extend serve write timeout to cover long LLM calls.
- Keep Analyze disabled before the first valid live snapshot is available and after the current revision has already been analyzed.
- Merge Advisor and AI Analysis into one Advisory page and use a highlighted “Analyze with AI” button.

## Rejected Alternatives

- Automatically running analysis during live polling was rejected because it would make serve nondeterministic, slow, and provider-dependent.
- Calling the provider directly from browser JavaScript was rejected because it would expose credentials and provider details to the UI.
- Caching by HTTP ETag was rejected because Events are part of the live response ETag; analysis events change the ETag even when the underlying snapshot revision has not changed.
- Keeping separate Advisor and AI Analysis menu items was rejected because the concepts overlap in the UI.

## Constraints

- LLM analysis must remain optional; scan, compare, advisory, serve, and export must continue to work without provider credentials.
- The browser must never receive the provider API key.
- Provider calls must be manually triggered and not caused by background refresh.
- Analysis must preserve resource references and distinguish collected facts from inferred workload purpose.
- Kubernetes core resources such as RBAC, admission webhooks, resource quotas, limit ranges, PDBs, and NetworkPolicies should not be explained as standalone workload purpose.
- Existing untracked scan output directories are user data and must not be staged.

## Evaluation Plan

- Run `GOCACHE=/private/tmp/teleskope-go-cache go test ./...`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache make build`.
- Run `git diff --check`.
- Verify live handler tests cover successful analysis, detailed Events, cache hits for the same revision, and concurrent request rejection.
- Verify report tests cover the Advisory merge, Analyze with AI button, disabled-before-first-scan behavior, and AI visual styling strings.

## Open Questions

- Whether future versions should expose a manual web-search toggle per analysis request.
- Whether completed live analysis should be downloadable as `llm-analysis.json` and `llm-analysis.md` from serve.

## Links

- Design files: `docs/llm-analysis.md`, `.provenance/2026-09-11-llm-analysis-design.md`
- Related workflows: `teleskope analyze scan`, `teleskope analyze compare`, `teleskope serve`
