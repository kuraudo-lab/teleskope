# Architecture direction

## First-phase scope

Prioritize read-only, out-of-cluster, run-once collection of EKS and Kubernetes
information. Organize platform components, configuration evidence, and workload
relationships so people can assess cluster capabilities themselves.

Planned outputs are terminal summaries, TUI, JSON, YAML, Markdown, offline HTML,
and an interactive local web server. Polling-based continuous collection and an embedded web server are now
implemented through `serve k8s` and `serve eks`. Informer watches, packaged
in-cluster execution, migration analysis and active capability verification remain later stages.
Read-only capability summaries are implemented in `internal/advisor`; see
[Advisor](advisor.md) for evidence semantics and limitations.
Multi-cluster support should be modeled as single-cluster collectors remotely
writing cluster report envelopes to a separate hub process; see
[Multi-cluster hub design](multi-cluster-hub.md). Optional LLM-backed summaries
should be added as a post-collection narrative layer rather than replacing
advisor or compare rules; see [LLM analysis design](llm-analysis.md).

## Package boundaries

The repository uses Go's `cmd` and `internal` conventions. There is no single
mandatory standard Go project layout; directories are introduced when used.

- `cmd/teleskope`: process entry point only.
- `internal/cli`: argument handling and command orchestration.
- Future collection packages: AWS SDK for Go v2 and Kubernetes client-go adapters.
- Future model packages: snapshots, resources, component instances,
  configuration facts, relationships, evidence, and collection coverage.
- Future component adapters: identify implementations and organize their settings.
- Future relationship resolvers: connect explicit references and label matches,
  retaining their provenance and unresolved targets.
- Future rendering packages: consume the same snapshot model for every output.

Do not introduce a public `pkg` API until there is an actual external consumer.
Start with compiled-in modules rather than dynamic plugins.

## Data principles

- Discover unknown extensions even when a component-specific adapter is absent.
- Separate collected facts, inferred associations, and human conclusions.
- Preserve evidence and collection coverage, including denied or partial reads.
- Identify uncollected node-local runtime configuration explicitly.
- Retain workload templates even when there are no running Pods.
- Distinguish configured resource relationships from observed network traffic.
- Exclude Secret values and sanitize configuration evidence before export.
- Keep snapshots independently readable without cluster credentials.

## Provider boundaries

Use `provider` for an independently sourced capability or evidence stream, not
for every internal transformation. The intended provider families are:

- **Collection providers** acquire primary facts. The implemented sources are
  the Kubernetes API and AWS/EKS collectors; recorded snapshots and hub
  envelopes replay or transport the same fact model without recollecting it.
- **Evidence providers** add optional, time-bounded observations to existing
  object identities. Metrics API is the baseline metric source. Prometheus,
  OpenTelemetry, CNI observability, and AWS network telemetry may be integrated
  through explicit provider contracts. Every observation retains provider,
  time-window, sampling, freshness, and coverage metadata.
- **Analysis providers** derive outputs from collected evidence. Deterministic
  finding rule packs are authoritative. The OpenAI-compatible provider is an
  optional narrative layer and cannot create or override findings.

Component adapters interpret provider facts and relation resolvers construct
declared, resolved, ownership, or inferred relationships; neither should be
called a provider merely because it is modular. Observed communication is a
separate edge basis and must come from connection evidence.

Teleskope will not implement or ship a privileged node probe. It will not
request host PID/network access, runtime sockets, procfs, conntrack, debugfs, or
eBPF privileges to manufacture Scope-style connection visibility. When no
external connection provider is configured, declared and resolved topology
remains complete for the available inventory while runtime connection coverage
is explicitly `unavailable`.

## Live polling

`internal/live` owns source scheduling, publication, and the read-only HTTP
handler. Each source has one sequential worker with a bounded collection context.
Kubernetes and AWS collectors retain clients between attempts. HTTP readers
consume pre-encoded immutable responses and never access mutable collector state.

Publication keeps source data separate from the latest attempt's coverage and
errors. An initial partial result is visible. On a failed attempt or coverage
regression for an already observed resource, retain the previous whole source
and mark it stale. A successful empty list is authoritative. This deliberately
conservative policy preserves source-level derived relationships; resource-level
merging can be introduced with explicit dependency rules later.

`internal/report` shares its embedded page between offline and live rendering.
The live entry adds a same-origin polling script and collection status panel.
No frontend build or separate server is required.

Event-based Kubernetes updates should be added behind a separate watch runtime
module and then routed through the live publication path; see
[Event-based Kubernetes update design](event-based-kubernetes-updates.md).
