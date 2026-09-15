# EKS Networking Deep Inventory Provenance

## Trigger

After accepting issue #6, the user asked to continue to the next item. The next non-AI issue by priority is #14, "EKS networking deep inventory."

## Scope

This record covers the first implementation slice for EKS networking inventory using evidence already present in Teleskope snapshots:

- `internal/report/eks_projection.go`
- `internal/report/report.go`
- `internal/report/report.html`
- `internal/render/human.go`
- Focused report/render tests

## Conversation Summary

Issue #14 asks Teleskope to deepen networking inventory for VPC CNI, subnets, security groups, ENIConfig, SecurityGroupPolicy, prefix delegation, and custom networking. The accepted direction is to continue non-AI work first and keep landing independently verifiable slices.

The first slice should project visible evidence from existing collectors rather than adding new AWS EC2 API calls. It should make VPC CNI configuration traceable from add-ons, Kubernetes workload/config objects, custom resource instances, and node labels where available. It should also make missing EC2 subnet/security group detail explicit as a coverage gap.

## Design Diff

No separate design document is changed in this provenance commit. The planned implementation diff is:

- Add an EKS networking details projection that associates cluster VPC facts, nodegroup subnets, node AZ labels, and CNI-related Kubernetes evidence.
- Detect VPC CNI sources from managed add-on records, `aws-node` workloads, CNI ConfigMaps, and CRD-backed `ENIConfig` / `SecurityGroupPolicy` objects when present.
- Surface prefix delegation and custom networking hints from visible configuration strings, labels, selectors, and object names.
- Render the networking detail projection in Markdown, embedded HTML, and human terminal output.
- Add tests for traceability and explicit EC2 coverage gaps.

## Decisions

- Treat EKS cluster and managed nodegroup fields as AWS-side declared networking evidence.
- Treat Kubernetes objects and node labels as observed runtime/configuration evidence.
- Do not infer subnet CIDRs, route tables, NAT, or security group rules unless those details are actually collected.
- Preserve existing generic Kubernetes networking tables; add an EKS-specific networking projection for EKS/nodegroup/CNI context.
- Keep missing EC2 details as `unknown` coverage gaps rather than unsupported findings.

## Rejected Alternatives

- Do not add EC2 `DescribeSubnets`, `DescribeSecurityGroups`, or ENI collection in this first slice.
- Do not bury VPC CNI details only in raw ConfigMap or workload tables; users need a traceable EKS networking view.
- Do not claim custom networking or prefix delegation is disabled simply because CNI configuration objects are absent.

## Constraints

- Reports must remain useful for partial snapshots.
- The projection must be deterministic for tests.
- Existing report JSON/HTML consumers should remain compatible.
- Verification must include focused tests, full `make check`, build, and whitespace validation.

## Evaluation Plan

- Add report projection tests for VPC CNI add-on/workload/config visibility and ENIConfig/SecurityGroupPolicy traces.
- Add report artifact tests for Markdown and HTML rendering of networking details and EC2 coverage gaps.
- Run `go test ./internal/report ./internal/render`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make build`.
- Run `git diff --check`.

## Open Questions

- Whether a later issue should add direct EC2 subnet, route table, ENI, and security group rule collection.

## Links

- Related issues/tasks: #14
