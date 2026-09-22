# EKS Infrastructure Backend Collection Provenance

## Trigger

After accepting the Recorded fixture and project-scoped EKS UI, the user asked to start implementing backend collection. The existing snapshot contract already models EC2 instances, Auto Scaling groups, VPCs, subnets, and security groups, but the live AWS collector does not populate those fields.

## Scope

This implementation slice connects the existing `EKSInfrastructure` contract to AWS SDK for Go v2 APIs. It covers managed-nodegroup Auto Scaling groups and their EC2 instances, the cluster and nodegroup VPC/subnet/security-group footprint, VPC DNS attributes, subnet route-table associations, and NAT gateway references.

It does not add ENI, load balancer, VPC endpoint, Network ACL, flow log, or security-group rule-body models. Those remain follow-up depth rather than being inferred from incomplete evidence.

## Design Diff

- Add EC2 and Auto Scaling SDK clients to the reusable EKS collector.
- Resolve managed-nodegroup ASG names from existing EKS evidence, then describe those groups and their instances.
- Resolve VPC, subnet, and security-group IDs from the EKS cluster, nodegroups, remote-access groups, and collected instances.
- Describe route tables for the cluster VPC and map explicit or main route-table associations to collected subnets; retain NAT gateway IDs referenced by routes.
- Populate the existing normalized `EKSInfrastructure` records and deterministic relationship keys.
- Record each infrastructure API family independently in snapshot coverage so a denied or failed auxiliary read does not discard the EKS control-plane snapshot.
- Document the additional read-only IAM actions required for deep infrastructure collection.

## Decisions

- Keep infrastructure records normalized and preserve the data contract accepted through the Recorded fixture.
- Use the EC2 private DNS name as the Kubernetes node-name association when AWS supplies it; do not fabricate a node name from the private IP.
- Treat ASG names declared by managed nodegroups as the authoritative starting set; do not scan unrelated account-wide groups.
- Restrict EC2 instance collection to instance IDs returned by those ASGs.
- Restrict network collection to the cluster VPC and the subnet/security-group IDs reachable from the cluster, nodegroups, and collected instances.
- Prefer explicit subnet route-table associations, falling back to the VPC main route table when needed.
- Keep auxiliary AWS failures non-fatal and visible through `partial` coverage entries.

## Constraints

- `DescribeCluster` remains the only mandatory EKS read after AWS configuration loads.
- Existing snapshots and callers remain compatible when infrastructure fields are absent.
- Collection remains read-only and does not enumerate unrelated EC2 or Auto Scaling resources.
- Output ordering must be deterministic for stable snapshots and tests.
- No credential material, user data, or secret values are collected.

## Evaluation Plan

- Add focused mapper and collection tests for nodegroup-to-ASG-to-instance relationships and VPC/subnet/route/security-group facts.
- Verify auxiliary API failures retain already collected infrastructure and append partial coverage.
- Run `go test ./internal/awseks ./internal/inventory ./internal/report`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`.
- Run `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make build`.
- Run `git diff --check`.
- On an authorized AWS account, verify `teleskope scan eks --cluster <name>` reports the new coverage entries and relationships.

## Links

- Contract and UI provenance: `.provenance/2026-09-22-eks-project-navigation-and-infrastructure-fixture.md`
- Initial collector provenance: `.provenance/2026-09-05-eks-collector.md`
- Networking evidence provenance: `.provenance/2026-09-15-eks-networking-deep-inventory.md`
