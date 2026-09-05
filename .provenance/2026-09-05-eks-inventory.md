# EKS Inventory and Resource Topology Provenance

## Trigger

The user requested an EKS discovery tool, refined the first phase to comprehensive
information collection and organization for human assessment, and explicitly
invoked provenance-commit to record the design context in Git.

## Scope

First-phase EKS/Kubernetes inventory, platform component configuration, workload
relationships, collection evidence, coverage, and offline presentation.
This commit includes this record, docs/architecture.md, and docs/eks-inventory-v1.md.

## Conversation Summary

The original request included EKS versions, AMIs, add-ons, IAM, Kubernetes
configuration, workloads, images, cluster capabilities, and migration gaps.
Run-once and out-of-cluster execution have priority; continuous operation and
in-cluster execution are longer-term goals. The user requested Go, modular
extension, terminal/TUI, YAML, JSON, Markdown, HTML, and a mini web server, with
relationship exploration inspired by Weave Scope and Headlamp.

The user then explicitly prioritized collecting and effectively organizing as
complete a picture as possible, especially add-ons, CNI, CSI, CRI, and workload
topology. Humans will initially judge capabilities from those facts. Automated
capability and migration assessment are deferred.

After moving into the repository, the user requested a conventional Go scaffold
with the module name derived from the Git remote. The remote is
git@github.com:kuraudo-lab/teleskope.git; the scaffold uses
github.com/kuraudo-lab/teleskope. That scaffold already exists as untracked files
and is deliberately excluded from this design commit. This is a structured
conversation summary, not a raw transcript or a claim that no code existed yet.

## Design Diff

- docs/eks-inventory-v1.md: brings the revised first-phase plan into the repository;
  defines broad discovery, deeper component adapters, collection fields,
  relationship evidence, runtime visibility limits, and acceptance criteria.
- docs/architecture.md: records intended package boundaries and the collection,
  evidence, snapshot, and presentation approach.
- Earlier migration-oriented research remains historical context outside this
  commit; the inventory-first plan governs first-phase work.

## Decisions

- Collect, organize, and link facts first; leave capability conclusions to humans.
- Include default and additional components: default components can have important
  custom settings. Unknown extensions remain visible without dedicated adapters.
- Use generic discovery for breadth and component adapters for configuration depth.
- Preserve provenance, collection time, depth, and missing-data reasons.
- Distinguish explicit references, selector matches, and inferred relationships.
- Use one versioned snapshot model across exports and interactive views.
- Use cmd/teleskope and internal packages, adding packages as functionality arrives.
- Proposed collection stack: AWS SDK for Go v2 and client-go; existing kubeconfig
  exec authentication remains compatible. Start with compiled-in Go modules.

## Rejected Alternatives

- An automated capability/gap engine as the first milestone: superseded by the
  user's inventory-first direction.
- Raw resource dumps as the entire product: insufficient organization for review.
- Hiding unknown CRDs or equating missing data with missing capability: loses
  coverage and produces misleading conclusions.
- Treating configuration topology as actual network traffic: lacks observation.
- The proposed design avoids a whole-product Scope fork and an early dynamic
  plugin system; these are engineering recommendations, not separately approved
  user mandates.

## Constraints

- Prioritize read-only, out-of-cluster, run-once collection.
- Collect non-sensitive component configuration while excluding Secret values and
  sanitizing evidence; completeness does not imply unrestricted secret export.
- Expose denied, partial, skipped, unavailable, and node-local information gaps.
- Keep templates for zero-replica workloads and unresolved resource references.
- AWS and Kubernetes reads do not form an atomic snapshot; preserve time windows.
- SDK objects should stay behind adapters rather than define the domain model.
- Do not claim runtime handler availability from RuntimeClass declarations alone.

## Evaluation Plan

- Compare inventory with source APIs across managed node groups, self-managed or
  Karpenter nodes, Auto Mode, and mixed compute where test access is available.
- Verify component versions, configuration sources, node distribution, and users
  are navigable from the same component record.
- Verify workload paths through routing, storage, configuration, identity, images,
  and compute, including unresolved references and zero-replica workloads.
- Exercise denied permissions, pagination failures, missing APIs, and redaction;
  verify coverage records distinguish them accurately.
- Verify exported and interactive views agree and work offline without credentials.
- Use sanitized fixtures for normalization and relation tests, plus real-cluster
  checks for collector behavior. No collectors have been implemented or tested yet.
- Existing scaffold build, go vet, and help/version checks passed; go test reported
  no test files. These checks establish only the scaffold baseline.

## Open Questions

- Which real clusters and component versions will define the first adapter matrix?
- Which custom resource fields may be exported under the collection profiles?
- Is node-local diagnostic import needed in the initial release?
- Final web/TUI libraries and measurable scale targets need implementation validation.
- Continuous mode, automatic assessment, and third-party plugins remain later work.

## Links

- Design: [Inventory v1](../docs/eks-inventory-v1.md)
- Architecture: [Package and data direction](../docs/architecture.md)
- Repository: git@github.com:kuraudo-lab/teleskope.git
- Related issues/tasks: none recorded.

## Implementation Linkage

Future implementation commits following this record should carry
Provenance-Commit with this commit's hash and
Provenance-File: .provenance/2026-09-05-eks-inventory.md.
