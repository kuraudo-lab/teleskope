# EKS Collector Implementation Provenance

## Trigger

The user asked to start implementing EKS information collection after the initial
Go scaffold. The requested slice must read AWS CLI-related identity and region
configuration, require a cluster name parameter, use Cobra for the CLI framework,
and make the terminal output style direction informed by tw93/mole.

This record is created after the first implementation pass because the user
explicitly invoked the provenance-commit workflow together with the implementation
commit workflow. It is a structured summary, not a raw transcript.

## Scope

This record covers the first AWS-side EKS collector slice:

- Cobra command wiring for `teleskope scan eks`.
- AWS SDK for Go v2 configuration loading from the AWS default config chain.
- Effective AWS identity, region, and profile recording.
- EKS control-plane, managed add-on, managed node group, access entry, and Pod
  Identity association collection.
- A versioned snapshot model with coverage records.
- Compact terminal and JSON renderers.

It does not cover Kubernetes API discovery, workload topology, component-specific
CNI/CSI/CRI adapters, TUI interaction, HTML reports, or the mini web server.

## Conversation Summary

The broader accepted direction is to collect and organize complete EKS and
Kubernetes evidence first, especially platform add-ons and non-default capability
signals from CNI, CSI, CRI, and workload topology. Human reviewers will judge
capabilities from the evidence in the early phase.

For this implementation slice, the user narrowed the immediate task to EKS data
collection. The CLI must read AWS identity and region from AWS CLI-compatible
configuration, include a cluster-name argument, use Cobra, and take terminal
style inspiration from mole. The implementation chose AWS SDK for Go v2 rather
than shelling out to `aws` because the SDK natively uses the shared AWS config
and credentials chain and gives typed EKS responses.

## Design Diff

- `internal/cli`: replace the hand-written `flag` CLI with a Cobra command tree.
- `internal/awseks`: introduce the AWS-side EKS collector.
- `internal/inventory`: introduce the first versioned snapshot structs.
- `internal/render`: introduce human and JSON renderers sharing the same snapshot.
- `internal/buildinfo` and `Makefile`: move build version injection out of the
  CLI package so collectors can record source metadata without importing CLI.
- `README.md`: document the first implemented scan command and current scope.

The previous design record at `.provenance/2026-09-05-eks-inventory.md` remains
the broader product provenance. This record narrows it to the first executable
collector slice.

## Decisions

- Use AWS SDK for Go v2 and `config.LoadDefaultConfig` so AWS CLI shared config,
  credentials, SSO, environment variables, and explicit `--profile`/`--region`
  overrides follow the standard AWS resolution chain.
- Require `--cluster` for the EKS scan command. The collector does not scan all
  clusters in an account.
- Start with AWS-side EKS inventory: cluster, add-ons, managed node groups,
  access entries with associated policies, Pod Identity associations, and STS
  caller identity.
- Preserve partial collection using coverage records. Non-critical list failures
  are recorded as partial coverage after `DescribeCluster` succeeds.
- Keep terminal output compact and status-oriented. JSON remains the richer
  machine-readable output for future renderers and tests.
- Keep collector, data model, renderer, and CLI packages separate so Kubernetes
  discovery can be added without tying API traversal to Cobra.

## Rejected Alternatives

- Shelling out to `aws eks ...` for the first implementation. It is compatible
  with AWS CLI configuration, but creates brittle text/JSON process boundaries
  and weakens typed collection.
- Implementing Kubernetes API discovery in the same slice. That would expand the
  credential and topology surface before the EKS collector boundary is stable.
- Claiming CNI/CSI/CRI capability conclusions from EKS add-on presence. This
  slice records facts and leaves capability judgment to later component adapters
  and human review.
- Building a full Bubble Tea TUI immediately. The current need is usable
  run-once collection; the compact human renderer gives a terminal baseline
  while preserving the same snapshot for richer TUI work.

## Constraints

- Go version declarations remain `1.26.0`.
- The scan is read-only, out-of-cluster, and run-once.
- Secret values are not collected in this slice.
- AWS and later Kubernetes reads will not be atomic; snapshots must preserve
  collection time and coverage.
- EKS API access can be partial. Denied or unavailable reads must be visible
  instead of silently omitted.
- The implementation commit must reference this provenance record.

## Evaluation Plan

- Run `go fmt ./...`, `make build`, and `make check`.
- Smoke test `./bin/teleskope --version` and `./bin/teleskope scan eks --help`.
- On a real EKS account, run `teleskope scan eks --cluster <name>` with a profile
  whose permissions include `eks:DescribeCluster`, EKS list/describe permissions,
  and `sts:GetCallerIdentity`.
- Verify JSON output contains the effective AWS region, caller identity,
  `DescribeCluster` facts, add-ons, node groups, access entries, Pod Identity
  associations, and coverage entries.
- Verify partial permission failures are represented in coverage.

## Open Questions

- Which real cluster/profile should define the first live fixture for EKS output?
- Should `--output yaml` be added immediately or after the snapshot model has
  Kubernetes resources?
- Should partial collection continue independently per item after one add-on or
  node group describe call fails?

## Links

- Broader provenance: `.provenance/2026-09-05-eks-inventory.md`
- Design files: `docs/architecture.md`, `docs/eks-inventory-v1.md`
- External style reference: `https://github.com/tw93/mole`
- Related issues/tasks: none recorded.
