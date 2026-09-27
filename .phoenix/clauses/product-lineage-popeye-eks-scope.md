# Teleskope Product Lineage Canonical Clauses

## Source Material

- Product-owner direction in the 2026-09-26 conversation.
- Product-owner rejection of a Teleskope node probe in the 2026-09-27 conversation.
- `.provenance/2026-09-05-eks-inventory.md`
- `.provenance/2026-09-11-kubernetes-offline-inventory-details.md`
- `docs/competitive-capabilities-popeye-headlamp-2026-09-26.md`
- `docs/weave-scope-product-ui-research-2026-09-26.md`
- `docs/live-probe-research.md`

## Requirements

### LINEAGE-REQ-001

Statement: Teleskope shall provide a maintained successor and practical replacement for Popeye's read-only live-cluster linting workflow.
Rationale: Popeye's product category is Teleskope's primary baseline rather than an optional adjacent feature set.
Source: Product-owner direction in the 2026-09-26 conversation.
Status: accepted

### LINEAGE-REQ-002

Statement: Teleskope shall extend the Popeye replacement baseline with EKS-specific evidence, relationships, and deterministic analysis.
Rationale: AWS-side infrastructure and EKS-managed capabilities are the product's second layer of specialization.
Source: Product-owner direction in the 2026-09-26 conversation and `.provenance/2026-09-05-eks-inventory.md`.
Status: accepted

### LINEAGE-REQ-003

Statement: Teleskope shall provide a Scope-inspired interactive topology experience for understanding live infrastructure and application relationships.
Rationale: Weave Scope, rather than Headlamp, is the intended lineage for the interactive platform experience.
Source: Product-owner direction in the 2026-09-26 conversation.
Status: accepted

### LINEAGE-REQ-004

Statement: Popeye replacement work shall be specified and prioritized before Scope-inspired expansion work.
Rationale: The linter replacement is the primary product promise and establishes findings and evidence consumed by later interactive views.
Source: Product-owner direction in the 2026-09-26 conversation.
Status: accepted

## Constraints

### LINEAGE-CON-001

Statement: Headlamp feature parity is not a product goal; overlap is limited to topology and resource-context navigation needed by Teleskope.
Rationale: General Kubernetes dashboard and administration workflows would dilute the intended product lineage.
Source: Product-owner direction in the 2026-09-26 conversation.
Status: accepted

### LINEAGE-CON-002

Statement: Scope-inspired work must preserve Teleskope's evidence, coverage, freshness, and unknown-versus-unsupported semantics.
Rationale: A live visual surface must not weaken the reliability boundary already established by snapshots and deterministic analysis.
Source: `docs/live-probe-research.md` and current Advisor semantics.
Status: accepted

### LINEAGE-CON-003

Statement: Secret values shall remain excluded and interactive topology shall not imply observed network traffic unless runtime telemetry actually supports that claim.
Rationale: Relationship evidence and observed communication are distinct data classes with different security and correctness boundaries.
Source: `.provenance/2026-09-05-eks-inventory.md`.
Status: accepted

### LINEAGE-CON-004

Statement: Teleskope shall not implement or ship a privileged node probe for process, container-runtime, or network-connection collection.
Rationale: Host PID, host network, runtime socket, procfs, conntrack, debugfs, and eBPF access would expand the product's privilege and operational boundary beyond the accepted design.
Source: Product-owner direction in the 2026-09-27 conversation.
Status: accepted

## Decisions

### LINEAGE-DEC-001

Statement: Product planning shall use a three-layer model: Popeye successor first, EKS specialization second, and Weave Scope reborn third.
Rationale: This model captures the intended lineage and gives sequencing rules for the issue backlog.
Alternatives: Treat Teleskope as a generic inventory report; treat Headlamp as the primary competitor; build a generic Kubernetes administration dashboard.
Status: accepted

### LINEAGE-DEC-002

Statement: The earlier decision not to implement Headlamp-style exec, log, edit, and delete workflows remains in force.
Rationale: Scope-inspired topology does not require reproducing Headlamp's general CRUD and terminal surface.
Alternatives: Full Headlamp feature parity.
Status: accepted

### LINEAGE-DEC-003

Statement: The earlier engineering recommendation to avoid a whole-product Scope fork no longer limits product direction; Scope behavior shall instead be re-evaluated feature by feature against current security and architecture constraints.
Rationale: The product owner has now explicitly selected Weave Scope as the interactive lineage.
Alternatives: Keep Scope only as a visual reference for one topology page.
Status: accepted

### LINEAGE-DEC-004

Statement: Runtime connection evidence shall enter Teleskope through external evidence providers and a normalized provider contract, while Kubernetes and AWS relationships remain available without runtime telemetry.
Rationale: This preserves a low-privilege core while allowing users to attach evidence from systems they already operate, such as OpenTelemetry, CNI observability, or AWS network telemetry.
Alternatives: A Teleskope-owned privileged node DaemonSet; presenting declared relationships as observed communication.
Status: accepted

## Invariants

### LINEAGE-INV-001

Statement: Deterministic collectors and analyzers remain authoritative; optional LLM output may explain but shall not create or override findings.
Verification: Tests and API review confirm every finding is reproducible without an LLM provider.
Status: accepted

### LINEAGE-INV-002

Statement: Every linter finding and topology relationship shall retain a machine-readable path to its supporting evidence and collection coverage.
Verification: Schema tests require evidence references, rule or relation identifiers, coverage, and freshness metadata.
Status: accepted

### LINEAGE-INV-003

Statement: Offline snapshots, live single-cluster views, and hub drilldowns shall consume compatible finding and topology semantics.
Verification: Shared fixtures produce equivalent findings and relationships in offline, live, and hub render paths.
Status: accepted

## Acceptance Criteria

### LINEAGE-ACC-001

Statement: The issue backlog contains an explicitly ordered Popeye-replacement series covering compatibility, finding rules, policy configuration, automation outputs, metrics, and recurring execution.
Verification: Each capability has a bounded issue with dependencies and observable acceptance criteria.
Status: proposed

### LINEAGE-ACC-002

Statement: A primary-source Weave Scope analysis identifies preserved interaction principles, rejected legacy behavior, telemetry boundaries, and UI acceptance criteria before implementation issues are finalized.
Verification: A source-cited research note and linked issue series exist.
Status: proposed

### LINEAGE-ACC-003

Statement: The Scope-inspired issue series is explicitly blocked behind the relevant Popeye finding and shared graph-model foundations where dependencies exist.
Verification: The issue map shows the ordering and dependency edges.
Status: proposed

## Risks And Open Questions

### LINEAGE-RISK-001

Statement: "Popeye replacement" may mean behavioral compatibility, workflow compatibility, output compatibility, or all three.
Impact: Without an explicit compatibility contract, the backlog may claim replacement while omitting workflows required by existing Popeye users.
Owner: unassigned
Status: open

### LINEAGE-RISK-002

Statement: External connection providers may offer different identity precision, sampling, time windows, and cluster coverage.
Impact: The normalized contract and UI must preserve provider-specific limitations instead of presenting heterogeneous evidence as equally complete.
Owner: unassigned
Status: open

### LINEAGE-RISK-003

Statement: Live graph scale and update stability targets are not yet quantified.
Impact: A visually faithful implementation may still become unusable on large clusters.
Owner: unassigned
Status: open
