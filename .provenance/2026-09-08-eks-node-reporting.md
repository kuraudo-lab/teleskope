# EKS and Node Reporting Provenance

## Trigger

The user asked to add richer data presentation to both Markdown and HTML reports, specifically node detail and an EKS-specific HTML section covering upgrade and rollback insights, total compute capacity, declared usage, networking, and security. The user also clarified that Kubernetes cluster connection failure should be a command error rather than being silently captured only as skipped, denied, or unavailable coverage in the generated report.

## Scope

This record covers the scan inventory model, EKS collector, Kubernetes collector error boundary, CLI scan behavior, Markdown report generation, static HTML report generation, and tests for those behaviors.

Covered files:

- `internal/inventory/snapshot.go`
- `internal/awseks/collector.go`
- `internal/k8s/collector.go`
- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/report/report.go`
- `internal/report/report.html`
- `internal/report/report_test.go`

## Conversation Summary

The report already produced raw JSON files, `summary.md`, and a static interactive `index.html`. The user wanted the report to expose more EKS-specific context and node details so a human can judge cluster capabilities from organized evidence. The desired EKS view includes upgrade and rollback readiness, compute capacity, declared resource usage, VPC and subnet information, security groups, IAM roles, access entries, and Pod Identity associations.

After the first implementation pass, the user noted that a Kubernetes cluster connection failure should fail the scan instead of producing a report that only records skipped or denied coverage. The error policy was therefore refined: baseline connectivity failures are fatal, while resource-level collection gaps remain coverage entries.

## Design Diff

The implementation should add an EKS insight model and collector path using the AWS SDK EKS insight APIs. It should keep the collected insight data in raw snapshot output so `eks.json` and `snapshot.json` remain the source of truth.

The Markdown report should add EKS subsections for upgrade/rollback insights, compute capacity and declared requests/limits, networking, security and identity, managed add-ons, and managed nodegroups. The Kubernetes report should include a dedicated node details table with provider ID, readiness, schedulability, kubelet, runtime, OS, kernel, architecture, capacity, allocatable resources, taints, key EKS/topology labels, and node-local blind spots.

The static HTML report should add two navigation sections: `EKS` and `Nodes`. The EKS section should render overview, upgrade/rollback insights, compute capacity and declared usage, network, security, access entries, Pod Identity associations, add-ons, and nodegroups. The Nodes section should render the detailed node table.

The Kubernetes collector should return an error for kubeconfig load failure, client construction failure, or a failed baseline `ServerVersion` call. The CLI should propagate that error and avoid writing report artifacts in that case. Individual API list failures after a successful connection should continue to be represented as coverage records.

## Decisions

- Add `EKSInsight` and related nested types to the inventory model so insights are first-class raw data rather than report-only derived text.
- Filter EKS insights to `UPGRADE_READINESS` and `ROLLBACK_READINESS`, matching the user's requested upgrade and rollback focus.
- Treat declared requests and limits as the currently available usage signal because live metrics are not collected yet. Label this clearly as spec-derived data.
- Keep resource-level Kubernetes authorization and API failures as coverage because they represent partial collection, not total inability to connect.
- Treat kubeconfig/client/discovery baseline failures as fatal scan errors because a report generated from that state is misleading.
- Add HTML sections rather than only expanding Overview, so EKS and node detail are easier to scan and do not crowd topology.

## Rejected Alternatives

- Do not show Kubernetes connection failure only as coverage. That hides a failed scan behind a successful command exit.
- Do not call metrics-server or cloudwatch for live resource usage in this step. That would require additional API dependencies and a distinct interpretation of usage.
- Do not infer subnet details or security group rules through EC2 APIs yet. The current step uses EKS-visible VPC, subnet, and security group identifiers.
- Do not fold node details into workload or runtime tables only. A dedicated node section makes capacity and provider linkage easier to review.

## Constraints

- The scanner remains read-only.
- Secret contents stay redacted; only metadata is collected.
- Static HTML must remain self-contained and generated side-by-side with `summary.md`.
- Default `scan eks` still attempts Kubernetes inventory unless `--skip-kubernetes` is provided.
- Coverage remains useful for partial resource-level collection failures after baseline connection succeeds.

## Evaluation Plan

- Run JavaScript syntax validation for the embedded HTML report script.
- Run `go test ./...` and `go vet ./...` through `make check`.
- Run `make build` to ensure the CLI still builds.
- Update report tests to assert EKS insights, capacity, security, node details, and HTML section/table wiring.
- Update CLI tests to assert missing kubeconfig returns a non-zero exit and does not create report artifacts.

## Open Questions

- Whether future versions should collect live usage from metrics-server, CloudWatch Container Insights, or both.
- Whether subnet attributes and security group rules should be expanded through EC2 APIs in a later EKS network/security pass.

## Links

- Design files: none
- Related issues/tasks: current Codex task
