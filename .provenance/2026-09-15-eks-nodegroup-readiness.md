# EKS Nodegroup Readiness Provenance

## Trigger

The user asked to continue with the next non-AI issue after finishing the EKS managed add-on compatibility work. The next highest-priority non-AI issue is GitHub issue #13, "EKS AMI and nodegroup upgrade readiness."

## Scope

This record covers the report, live UI, terminal render, and deterministic Advisor behavior needed to compare EKS managed nodegroup configuration with actual Kubernetes node runtime evidence.

Primary implementation surfaces:

- `internal/report/eks_projection.go`
- `internal/report/report.go`
- `internal/report/report.html`
- `internal/render/human.go`
- `internal/advisor/advisor.go`
- Focused tests under `internal/report` and `internal/advisor`

## Conversation Summary

Issue #13 asks Teleskope to expose managed nodegroup releaseVersion, AMI type, launch template, and actual node runtime facts for upgrade readiness. The accepted direction is to keep AI work lower priority and continue landing non-AI EKS and Advisor capabilities first.

The implementation should preserve the distinction between AWS-side expected configuration and Kubernetes-side observed node state. It should also make insufficient evidence explicit for custom AMI, self-managed, and Karpenter cases rather than implying readiness.

## Design Diff

No separate design document is changed in this provenance commit. The planned design diff is:

- Add a projected EKS nodegroup readiness view that groups actual nodes by nodegroup, availability zone, AMI/OS image, kubelet version, and runtime facts where labels and node status provide evidence.
- Keep the existing managed nodegroups and Kubernetes nodes tables as raw evidence.
- Render the readiness projection in Markdown, embedded HTML, and human terminal output.
- Add Advisor findings for nodegroups that summarize observed runtime alignment and mark missing or non-managed evidence as unknown.

## Decisions

- Use Kubernetes node labels such as `eks.amazonaws.com/nodegroup` and `topology.kubernetes.io/zone` as the primary observed link between nodegroups and nodes.
- Treat EKS managed nodegroup fields as expected AWS configuration, not proof of actual runtime state.
- Treat nodes without a managed nodegroup label as unknown ownership, with explicit self-managed/Karpenter wording when available labels suggest those cases.
- Do not infer AMI IDs when only OS image or EKS AMI type is available; report the available evidence separately.
- Keep the first implementation block focused on projection and Advisor evidence, not deeper AWS EC2 launch template or AMI API collection.

## Rejected Alternatives

- Do not collapse expected nodegroup configuration and observed nodes into one wide raw table; that makes evidence provenance harder to read.
- Do not classify custom AMI, self-managed, or Karpenter readiness as pass/fail without stronger evidence.
- Do not add live cluster calls beyond the existing snapshot inputs in this issue block.

## Constraints

- Reports must be useful with partial snapshots.
- Existing report JSON and HTML consumers should remain backward compatible where possible.
- Advisor conclusions must stay conservative and evidence-backed.
- Verification must include focused Go tests, full `make check`, `make build`, and whitespace checks with the established temporary Go cache paths.

## Evaluation Plan

- Add focused projection tests for managed nodegroups with matching nodes, mixed AZ/kubelet facts, and unmanaged node evidence.
- Add Advisor tests for managed nodegroup runtime evidence and unknown ownership cases.
- Run `go test ./internal/report ./internal/advisor ./internal/render`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make build`.
- Run `git diff --check`.

## Open Questions

- Whether a later issue should add EC2 launch template version and AMI ID collection from Auto Scaling Groups or EC2 APIs.

## Links

- Related issues/tasks: #13
