# Architecture direction

## First-phase scope

Prioritize read-only, out-of-cluster, run-once collection of EKS and Kubernetes
information. Organize platform components, configuration evidence, and workload
relationships so people can assess cluster capabilities themselves.

Planned outputs are terminal summaries, TUI, JSON, YAML, Markdown, offline HTML,
and an interactive local web server. Polling-based continuous collection and an embedded web server are now
implemented through `serve k8s` and `serve eks`. Informer watches, packaged
in-cluster execution, automated capability assessment, and migration analysis
remain later stages.

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
