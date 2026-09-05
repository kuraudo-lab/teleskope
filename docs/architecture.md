# Architecture direction

## First-phase scope

Prioritize read-only, out-of-cluster, run-once collection of EKS and Kubernetes
information. Organize platform components, configuration evidence, and workload
relationships so people can assess cluster capabilities themselves.

Planned outputs are terminal summaries, TUI, JSON, YAML, Markdown, offline HTML,
and an interactive local web server. Continuous collection, in-cluster execution,
automated capability assessment, and migration analysis are later stages.

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
