# EKS Project Navigation and Infrastructure Fixture Provenance

## Trigger

The user asked for the EKS menu to be organized into projects like the Kubernetes menu, with deeper expansion of existing EKS areas. Examples include EC2 instances and Auto Scaling groups below managed nodegroups, plus richer VPC and subnet networking detail. The requested delivery order is: update the sanitized Recorded fixture, adapt the UI, and defer backend collection changes to a later phase.

## Scope

This record covers the Recorded snapshot contract, the long-lived sanitized EKS fixture, EKS report projections, and embedded UI navigation/rendering. It does not add AWS EC2 or Auto Scaling API calls and does not change collector permissions.

## Conversation Summary

The current EKS UI is one large page while Kubernetes resources are grouped behind a project menu. Existing snapshots contain EKS control-plane and managed-nodegroup summaries, Kubernetes node evidence, VPC/subnet identifiers, and Auto Scaling group names, but not the deeper AWS infrastructure objects needed to inspect their relationships.

The first phase will model and populate realistic de-identified infrastructure evidence in the Recorded fixture, then expose it through project-scoped EKS pages. This creates a concrete UI and data contract that can be accepted before backend collection is implemented.

## Design Diff

- Extend the EKS inventory contract with optional infrastructure details for EC2 instances, Auto Scaling groups, VPCs, subnets, and security groups.
- Populate `testdata/recorded/eks-demo-snapshot.json` with deterministic sanitized objects whose IDs and relationships agree with the existing cluster, nodegroup, and Kubernetes node data.
- Split the EKS menu into Overview, Upgrades, Compute, Network, Security, and Add-ons pages.
- Render nodegroup-to-ASG-to-instance relationships and detailed VPC/subnet/security-group tables.
- Keep absent infrastructure evidence visibly empty or unknown for snapshots produced by the current backend.

## Decisions

- Treat the new fields as an optional snapshot contract preview; current collectors continue to emit no values until the backend phase.
- Keep AWS resources as normalized records rather than embedding duplicate EC2 and ASG objects under every nodegroup.
- Preserve relationship keys such as nodegroup name, ASG name, instance ID, subnet ID, VPC ID, security group ID, and Kubernetes node name so the UI can trace associations deterministically.
- Use reserved documentation-style identifiers, account `000000000000`, `.invalid` endpoints, and non-routable example addresses in the Recorded fixture.
- Retain existing deterministic EKS projections and add new rows without claiming evidence that is not present.

## Rejected Alternatives

- Do not implement EC2, Auto Scaling, or VPC collection in this phase; the user explicitly asked to adapt Recorded data and UI first.
- Do not keep EKS as one long scrolling page; that would not satisfy parity with the Kubernetes project menu.
- Do not infer detailed VPC/subnet/instance facts from Kubernetes node labels alone.
- Do not duplicate full instance objects inside nodegroups; references are sufficient and reduce drift.

## Constraints

- Existing snapshots without the optional infrastructure fields must remain compatible.
- The checked-in fixture must remain de-identified and internally consistent.
- The embedded static report and live `/api/snapshot` path must use the same UI and projection contract.
- Existing theme, export, event, search, and scoped-analysis behavior must remain functional.
- Verification must include focused projection/report tests, fixture relationship checks, full repository checks, build, and browser interaction against `serve snapshot`.

## Evaluation Plan

- Add focused tests for optional infrastructure projection and EKS project navigation.
- Validate every instance references an existing nodegroup, ASG, subnet, and Kubernetes node; validate every subnet and security group references the fixture VPC.
- Run `go test ./internal/inventory ./internal/report ./internal/live`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make build`.
- Run `git diff --check`.
- Start `teleskope serve snapshot testdata/recorded/eks-demo-snapshot.json` and verify EKS submenu navigation plus Compute and Network detail rendering in a browser.

## Open Questions

- Which EC2, Auto Scaling, and networking API calls and IAM permissions should be added in the later backend phase.
- Whether route tables, NAT gateways, VPC endpoints, ENIs, and load balancers should be collected in the first backend slice or split into follow-up slices.

## Links

- Related provenance: `.provenance/2026-09-15-recorded-serve-fixture.md`
- Related provenance: `.provenance/2026-09-15-eks-networking-deep-inventory.md`
- Planned implementation: `internal/inventory/snapshot.go`, `internal/report/eks_projection.go`, `internal/report/report.html`, `testdata/recorded/eks-demo-snapshot.json`
