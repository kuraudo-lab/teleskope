# LLM analysis design

Date: 2026-09-11. Scope: define how Teleskope can add optional LLM-backed
analysis for scan, compare, and advisory workflows. This is a design note, not
an implemented feature.

## Direction

Teleskope should add LLM analysis as an optional narrative layer after existing
collection and deterministic analysis. Collectors should continue to collect
facts. `internal/advisor` and `internal/compare` should continue to produce
stable, testable conclusions. The LLM should consume those facts and conclusions
and generate explanations, summaries, and migration guidance.

The first design target is:

```text
snapshot/advisor/compare report -> context builder -> use-case prompt -> LLM -> analysis result
```

This keeps the LLM seam outside the inventory model and deterministic rules. A
failed, disabled, unavailable, or slow LLM request must not prevent scan,
compare, advisory, or report export from working.

## Use cases

The same analysis module should support different use cases with different
prompts and input builders.

### Scan analysis

Scan analysis consumes one inventory snapshot and the deterministic advisor
report. It should help the user understand what is running in the cluster and
what deserves attention.

Useful outputs include:

- A concise cluster summary.
- Workload purpose explanations inferred from namespace, workload name, labels,
  annotations, owner kind, container images, ports, service accounts, services,
  ingresses, and gateway routes.
- High-signal operational or migration risks, tied back to specific resources.
- Questions or follow-up checks when the evidence is weak.

Container image interpretation is valuable but must be labelled as inference.
An image such as `redis`, `nginx`, `istio/proxyv2`, `external-dns`,
`cert-manager`, or `aws-load-balancer-controller` can suggest purpose, but it
cannot prove the business role of a workload. The output should distinguish
observed facts from inferred purpose.

### Compare analysis

Compare analysis consumes source and target snapshots plus the deterministic
`compare.Report`. It should turn structured differences into a migration plan.

Useful outputs include:

- Executive summary of migration readiness.
- Blockers, risks, and non-blocking differences.
- Workload and platform dependencies that need manual review.
- Suggested migration order and validation checklist.
- Areas where incomplete collection weakens confidence.

The deterministic comparison remains the source of truth for exact differences.
The LLM explains, groups, and prioritizes them.

### Advisory analysis

Advisory analysis uses the same underlying comparison data as CLI compare, but
presents it for the local web UI. Its output should be easy to render as cards,
sections, and exportable Markdown.

Useful outputs include:

- Migration overview.
- Prioritized action items.
- Per-category recommendations.
- Resource-linked supporting evidence.
- Confidence and limitations for each major conclusion.

Advisory should call the same analysis module as CLI compare so web and CLI
results stay consistent.

## Module shape

Introduce a small analysis module, for example `internal/analysis`, with a deep
interface. Callers should not need to know provider request details, prompt
assembly, token budgeting, retry policy, or result parsing.

A first interface can be:

```go
type UseCase string

const (
    UseCaseScan     UseCase = "scan"
    UseCaseCompare  UseCase = "compare"
    UseCaseAdvisory UseCase = "advisory"
)

type Analyzer interface {
    Analyze(ctx context.Context, req Request) (Result, error)
}
```

`Request` should carry the use case, the relevant snapshot or comparison data,
model options, and tool permissions. `Result` should be structured enough for
CLI, Markdown, JSON, and web rendering.

A possible result shape is:

```json
{
  "schemaVersion": "teleskope.io/llm-analysis/v1alpha1",
  "generatedAt": "2026-09-11T00:00:00Z",
  "useCase": "scan",
  "summary": "...",
  "sections": [
    {
      "title": "Workload purpose",
      "items": [
        {
          "severity": "info",
          "summary": "...",
          "detail": "...",
          "resources": [
            {"kind": "Deployment", "namespace": "kube-system", "name": "aws-load-balancer-controller"}
          ],
          "basis": "inferred",
          "confidence": "medium",
          "evidence": ["container image", "labels", "service account"]
        }
      ]
    }
  ],
  "limitations": ["..."],
  "citations": []
}
```

The exact schema can evolve, but it should preserve resource references, basis,
confidence, and limitations from the first version.

## Context builder

Do not send a whole snapshot to the model by default. Snapshots can become large
and may contain sensitive operational names. Add a context builder that selects,
compresses, and orders only the fields relevant to the use case.

For scan, the context should prioritize:

- Cluster identity, Kubernetes version, source mode, and coverage.
- Advisor capabilities.
- Workloads, pods, running images, running containers, services, ingresses,
  gateways, gateway routes, service accounts, ConfigMaps and selected policy
  references.
- RBAC, network policies, resource quotas, PDBs, and admission webhooks when
  they affect workload interpretation or migration risk.

For compare and advisory, the context should prioritize:

- The deterministic `compare.Report`.
- Source and target cluster summaries.
- Changed or migration-relevant inventory categories.
- Workloads and dependencies that appear only on one side.
- Coverage warnings that affect confidence.

The context builder should apply stable limits and deterministic ordering so
analysis is reproducible enough to debug. If data is omitted due to limits, the
analysis result should say so.

## Prompt registry

Prompts should be versioned and selected by use case. Keep prompt text close to
the analysis module rather than scattering it through CLI and web handlers.

Each prompt should instruct the model to:

- Treat snapshots, advisor reports, and compare reports as the only authoritative
  cluster evidence.
- Separate observed facts from inference.
- Avoid claiming runtime health unless observed by collected status fields.
- Preserve resource references in findings.
- Report uncertainty and incomplete collection.
- Produce the requested structured JSON result.

Prompt versions should be recorded in the analysis result. Changing a prompt in a
way that changes semantics should bump the prompt version.

## Web search

Web search can improve scan analysis, especially for container images,
controllers, operators, and uncommon components. It should be optional and
explicitly enabled.

Initial behavior:

- Default web search off.
- Enable through a CLI flag or environment variable, for example
  `--llm-web-search`.
- Use search mainly to identify public software from image names, chart names,
  or controller names.
- Require citations when web search contributes to a conclusion.
- Never send Secret values. Continue excluding Secret data from snapshots.

The web UI must not call an LLM provider directly from the browser. Provider API
keys and tool access belong in the local server or CLI process.

## Configuration

LLM analysis should be opt-in. A first CLI shape can be:

```bash
teleskope analyze scan ./scan-report --output markdown
teleskope analyze compare --source ./source-scan --target ./target-scan --output markdown
```

Later, scan and compare can grow convenience flags:

```bash
teleskope scan k8s --llm-analysis --llm-model <model>
teleskope compare --source ./source --target ./target --llm-analysis
```

Possible environment variables:

- `TELESKOPE_LLM_PROVIDER`
- `TELESKOPE_LLM_MODEL`
- `TELESKOPE_LLM_API_KEY`
- `TELESKOPE_LLM_WEB_SEARCH`

Exact names can be decided during implementation. Avoid adding a config file in
the first version.

## Output artifacts

Offline reports can add optional files when LLM analysis runs:

- `llm-analysis.json`
- `llm-analysis.md`

The existing `snapshot.json`, `advisor.json`, `summary.md`, and `index.html`
should remain valid without LLM output. The HTML report can render the analysis
when the artifact is present and hide the section otherwise.

For advisory, `/api/compare` should continue returning the deterministic report.
A separate endpoint such as `/api/analyze` can generate LLM analysis from the
uploaded snapshots and comparison report. This avoids making every comparison
slow or dependent on provider credentials.

## Privacy and safety

Cluster inventory can contain sensitive operational metadata even without Secret
values. Namespaces, object names, image repositories, hostnames, annotations,
IAM ARNs, and labels may reveal internal systems.

The first implementation should include:

- A clear opt-in flag before any data is sent to an LLM provider.
- Redaction hooks for known sensitive fields.
- A dry-run or debug mode that writes the model input context locally for review.
- Output metadata that records provider, model, prompt version, tool usage, and
  whether web search was enabled.

## Testing strategy

The analysis module should be testable without network calls.

Recommended tests:

- Context builder tests with representative snapshots.
- Prompt selection tests for scan, compare, and advisory.
- Fake analyzer tests for CLI and web handlers.
- JSON schema or golden tests for result rendering.
- Error-path tests proving scan, compare, advisory, and report rendering still
  work when LLM analysis fails or is disabled.

Do not make provider integration tests part of the default test suite. Keep real
provider calls behind explicit manual or integration-test flags.

## Implementation order

1. Define the `internal/analysis` request/result schema, prompt registry, and
   provider-neutral analyzer interface.
2. Build deterministic context builders for scan and compare/advisory.
3. Add a fake analyzer and rendering tests.
4. Add CLI `teleskope analyze scan` and `teleskope analyze compare` as offline
   post-processing commands.
5. Add a provider adapter and environment-variable configuration.
6. Add optional web search support behind an explicit flag.
7. Extend offline report artifacts and HTML rendering to include optional
   analysis results.
8. Add advisory web support through a separate analysis endpoint.

## Explicit non-goals for the first version

- Do not replace `internal/advisor` or `internal/compare` with LLM conclusions.
- Do not require LLM access for scan, compare, advisory, serve, or export.
- Do not send whole snapshots to a provider without context trimming.
- Do not expose provider credentials to browser JavaScript.
- Do not infer runtime behavior that the collected inventory does not observe.
