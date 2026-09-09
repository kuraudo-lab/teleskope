# Cluster capability advisor

`internal/advisor.Analyze` is a deterministic, read-only analysis of an inventory
snapshot. The first version summarizes Ingress classes, Gateway classes, RWX
bindings, StorageClass expansion configuration and custom API declarations.
It does not connect to Kubernetes, run probes or use an LLM.

Every capability has a stable key, scope, implementation, assessment, evidence
basis, constraints, resource/field evidence, coverage, freshness and versioned
rule identifier. `supported` with `declared` means configuration is present;
it is not an operational readiness guarantee. `observed` RWX requires a matching
Bound PVC/PV pair with RWX on both sides. Absence, unknown drivers and incomplete
reads do not establish lack of support. No provider-name heuristics are used.
Unrecorded expansion settings remain unknown. Explicit false disables expansion
for that StorageClass only.

The Advisor tab is cluster-wide and independent of inventory namespace/resource
filters. Expand a capability to inspect evidence and limitations. Offline scans
write `advisor.json`, embed the same analysis in HTML, and include a concise
Markdown summary. Raw snapshot schema is unchanged. Offline freshness is
`snapshot`, not a claim that the data is current.

Live `/api/snapshot` includes `advisor` alongside `snapshot`. Kubernetes source
state supplies freshness. Status-only failures update the Advisor even when data
revision is unchanged. Retained source evidence keeps its original collection
time; AWS updates do not refresh Kubernetes evidence.

The initial rules deliberately do not infer controller health, route feature
compatibility, dynamic provisioning success, CRD establishment, operator health
or multi-node storage behavior. Gateway conditions, provider/version-specific
rules and workload migration matching are future extensions. Rule changes must
include counterexample tests for missing evidence and bump their version.
