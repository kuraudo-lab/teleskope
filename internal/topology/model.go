// Package topology defines the provider-neutral graph and evidence contract.
package topology

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const SchemaVersion = "teleskope.io/topology/v1alpha1"

type EdgeKind string

const (
	EdgeOwnership EdgeKind = "ownership"
	EdgeDeclared  EdgeKind = "declared"
	EdgeResolved  EdgeKind = "resolved"
	EdgeInferred  EdgeKind = "inferred"
	EdgeObserved  EdgeKind = "observed"
)

type Basis string

const (
	BasisDeclared Basis = "declared"
	BasisResolved Basis = "resolved"
	BasisInferred Basis = "inferred"
	BasisObserved Basis = "observed"
)

type Graph struct {
	SchemaVersion string     `json:"schemaVersion"`
	ClusterID     string     `json:"clusterId"`
	Revision      uint64     `json:"revision"`
	GeneratedAt   time.Time  `json:"generatedAt"`
	Nodes         []Node     `json:"nodes"`
	Edges         []Edge     `json:"edges"`
	Coverage      []Coverage `json:"coverage"`
}

type Node struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Name        string            `json:"name"`
	Identity    string            `json:"identity"`
	APIVersion  string            `json:"apiVersion,omitempty"`
	UID         string            `json:"uid,omitempty"`
	Scope       Scope             `json:"scope"`
	ParentID    string            `json:"parentId,omitempty"`
	PseudoKind  string            `json:"pseudoKind,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Evidence    []Evidence        `json:"evidence,omitempty"`
	FindingRefs []string          `json:"findingRefs,omitempty"`
}

type Scope struct {
	Provider  string `json:"provider"`
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace,omitempty"`
	Account   string `json:"account,omitempty"`
	Region    string `json:"region,omitempty"`
}

type Edge struct {
	ID          string            `json:"id"`
	Source      string            `json:"source"`
	Target      string            `json:"target"`
	Kind        EdgeKind          `json:"kind"`
	Provider    string            `json:"provider"`
	Relation    string            `json:"relation,omitempty"`
	Directed    bool              `json:"directed"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Evidence    []Evidence        `json:"evidence"`
	FindingRefs []string          `json:"findingRefs,omitempty"`
}

type Evidence struct {
	Provider    string           `json:"provider"`
	Basis       Basis            `json:"basis"`
	ObservedAt  *time.Time       `json:"observedAt,omitempty"`
	WindowStart *time.Time       `json:"windowStart,omitempty"`
	WindowEnd   *time.Time       `json:"windowEnd,omitempty"`
	Sampling    string           `json:"sampling,omitempty"`
	Freshness   string           `json:"freshness"`
	Confidence  string           `json:"confidence"`
	Summary     string           `json:"summary"`
	Fields      []FieldReference `json:"fields,omitempty"`
}

type FieldReference struct {
	Resource string `json:"resource"`
	Field    string `json:"field,omitempty"`
	Value    string `json:"value,omitempty"`
}

type Coverage struct {
	Provider    string     `json:"provider"`
	Capability  string     `json:"capability"`
	Namespace   string     `json:"namespace,omitempty"`
	Status      string     `json:"status"`
	Reason      string     `json:"reason,omitempty"`
	ObservedAt  *time.Time `json:"observedAt,omitempty"`
	WindowStart *time.Time `json:"windowStart,omitempty"`
	WindowEnd   *time.Time `json:"windowEnd,omitempty"`
}

// NodeID returns a stable escaped identity independent of display layout.
func NodeID(provider, cluster, kind, namespace, nameOrUID string) string {
	if namespace == "" {
		namespace = "_"
	}
	return strings.Join([]string{segment(provider), segment(cluster), segment(strings.ToLower(kind)), segment(namespace), segment(nameOrUID)}, "/")
}

// PseudoNodeID returns a canonical identity for unresolved, external, or
// Internet targets that are evidence-backed but not collected API objects.
func PseudoNodeID(provider, cluster, pseudoKind, namespace, identity string) string {
	return NodeID(provider, cluster, "pseudo-"+pseudoKind, namespace, identity)
}

// EdgeID returns a stable identity for one provider relationship.
func EdgeID(kind EdgeKind, source, target, provider string) string {
	return strings.Join([]string{string(kind), segment(source), segment(target), segment(provider)}, "/")
}

func segment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "_"
	}
	return url.PathEscape(value)
}

// Normalize makes graph encoding deterministic without changing semantics.
func (g *Graph) Normalize() {
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sort.Slice(g.Edges, func(i, j int) bool { return g.Edges[i].ID < g.Edges[j].ID })
	sort.Slice(g.Coverage, func(i, j int) bool {
		a, b := g.Coverage[i], g.Coverage[j]
		return coverageKey(a) < coverageKey(b)
	})
	for i := range g.Nodes {
		for j := range g.Nodes[i].Evidence {
			sortFieldReferences(g.Nodes[i].Evidence[j].Fields)
		}
		sort.Slice(g.Nodes[i].Evidence, func(a, b int) bool { return evidenceKey(g.Nodes[i].Evidence[a]) < evidenceKey(g.Nodes[i].Evidence[b]) })
		sort.Strings(g.Nodes[i].FindingRefs)
	}
	for i := range g.Edges {
		for j := range g.Edges[i].Evidence {
			sortFieldReferences(g.Edges[i].Evidence[j].Fields)
		}
		sort.Slice(g.Edges[i].Evidence, func(a, b int) bool { return evidenceKey(g.Edges[i].Evidence[a]) < evidenceKey(g.Edges[i].Evidence[b]) })
		sort.Strings(g.Edges[i].FindingRefs)
	}
}

func coverageKey(coverage Coverage) string {
	encoded, _ := json.Marshal(coverage)
	return coverage.Provider + "\x00" + coverage.Capability + "\x00" + coverage.Namespace + "\x00" + string(encoded)
}

func evidenceKey(e Evidence) string {
	encoded, _ := json.Marshal(e)
	return string(encoded)
}

func sortFieldReferences(fields []FieldReference) {
	sort.Slice(fields, func(i, j int) bool {
		a, b := fields[i], fields[j]
		return a.Resource+"\x00"+a.Field+"\x00"+a.Value < b.Resource+"\x00"+b.Field+"\x00"+b.Value
	})
}

// MarshalDeterministic validates, normalizes, and emits stable compact JSON.
func (g Graph) MarshalDeterministic() ([]byte, error) {
	g.Normalize()
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(g)
}

func (g Graph) Validate() error {
	if g.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported topology schema %q", g.SchemaVersion)
	}
	if strings.TrimSpace(g.ClusterID) == "" {
		return fmt.Errorf("clusterId is required")
	}
	if g.GeneratedAt.IsZero() {
		return fmt.Errorf("generatedAt is required")
	}
	nodes := make(map[string]struct{}, len(g.Nodes))
	for _, node := range g.Nodes {
		if node.ID == "" || node.Kind == "" || node.Name == "" || node.Scope.Provider == "" || node.Scope.Cluster == "" {
			return fmt.Errorf("node identity, kind, name, provider, and cluster are required: %+v", node)
		}
		if _, exists := nodes[node.ID]; exists {
			return fmt.Errorf("duplicate node id %q", node.ID)
		}
		if node.Scope.Cluster != g.ClusterID {
			return fmt.Errorf("node %q cluster %q does not match graph cluster %q", node.ID, node.Scope.Cluster, g.ClusterID)
		}
		identity := node.Identity
		if identity == "" {
			return fmt.Errorf("node %q identity is required", node.ID)
		}
		isPseudoKind := strings.HasPrefix(node.Kind, "pseudo-")
		if isPseudoKind != (node.PseudoKind != "") {
			return fmt.Errorf("node %q kind %q and pseudo kind %q are inconsistent", node.ID, node.Kind, node.PseudoKind)
		}
		if node.PseudoKind != "" {
			if !validPseudoKind(node.PseudoKind) || node.Kind != "pseudo-"+node.PseudoKind {
				return fmt.Errorf("node %q has invalid pseudo kind %q", node.ID, node.PseudoKind)
			}
		}
		if expected := NodeID(node.Scope.Provider, node.Scope.Cluster, node.Kind, node.Scope.Namespace, identity); node.ID != expected {
			return fmt.Errorf("node id %q is not canonical, want %q", node.ID, expected)
		}
		for _, evidence := range node.Evidence {
			if err := validateEvidence(evidence); err != nil {
				return fmt.Errorf("node %q: %w", node.ID, err)
			}
		}
		nodes[node.ID] = struct{}{}
	}
	for _, node := range g.Nodes {
		if node.ParentID != "" {
			if _, exists := nodes[node.ParentID]; !exists {
				return fmt.Errorf("node %q has missing parent %q", node.ID, node.ParentID)
			}
		}
	}
	edges := make(map[string]struct{}, len(g.Edges))
	observedProviders := map[string][]Evidence{}
	for _, edge := range g.Edges {
		if !validEdgeKind(edge.Kind) {
			return fmt.Errorf("edge %q has unsupported kind %q", edge.ID, edge.Kind)
		}
		if edge.Provider == "" {
			return fmt.Errorf("edge %q provider is required", edge.ID)
		}
		if expected := EdgeID(edge.Kind, edge.Source, edge.Target, edge.Provider); edge.ID != expected {
			return fmt.Errorf("edge id %q is not canonical, want %q", edge.ID, expected)
		}
		if _, exists := edges[edge.ID]; exists {
			return fmt.Errorf("duplicate edge id %q", edge.ID)
		}
		edges[edge.ID] = struct{}{}
		if _, exists := nodes[edge.Source]; !exists {
			return fmt.Errorf("edge %q has missing source %q", edge.ID, edge.Source)
		}
		if _, exists := nodes[edge.Target]; !exists {
			return fmt.Errorf("edge %q has missing target %q", edge.ID, edge.Target)
		}
		if len(edge.Evidence) == 0 {
			return fmt.Errorf("edge %q requires evidence", edge.ID)
		}
		for _, evidence := range edge.Evidence {
			if err := validateEvidence(evidence); err != nil {
				return fmt.Errorf("edge %q: %w", edge.ID, err)
			}
		}
		if edge.Kind == EdgeObserved && !hasObservedEvidence(edge.Evidence, edge.Provider) {
			return fmt.Errorf("observed edge %q requires observed evidence", edge.ID)
		}
		if edge.Kind == EdgeObserved {
			for _, evidence := range edge.Evidence {
				if evidence.Basis == BasisObserved {
					observedProviders[evidence.Provider] = append(observedProviders[evidence.Provider], evidence)
				}
			}
		}
	}
	runtimeCoverage := map[string][]Coverage{}
	coverageIdentities := map[string]struct{}{}
	for _, coverage := range g.Coverage {
		if coverage.Provider == "" || coverage.Capability == "" || !validCoverageStatus(coverage.Status) {
			return fmt.Errorf("invalid coverage: %+v", coverage)
		}
		identity := coverage.Provider + "\x00" + coverage.Capability + "\x00" + coverage.Namespace
		if _, exists := coverageIdentities[identity]; exists {
			return fmt.Errorf("duplicate coverage identity provider=%q capability=%q namespace=%q", coverage.Provider, coverage.Capability, coverage.Namespace)
		}
		coverageIdentities[identity] = struct{}{}
		if coverage.WindowStart != nil && coverage.WindowEnd != nil && coverage.WindowStart.After(*coverage.WindowEnd) {
			return fmt.Errorf("coverage window start is after window end for provider=%q capability=%q", coverage.Provider, coverage.Capability)
		}
		if coverage.Capability == "runtime-connections" {
			runtimeCoverage[coverage.Provider] = append(runtimeCoverage[coverage.Provider], coverage)
		}
	}
	if len(runtimeCoverage) == 0 {
		return fmt.Errorf("runtime-connections coverage is required")
	}
	if len(observedProviders) > 0 {
		for _, sentinel := range runtimeCoverage["external"] {
			if sentinel.Status == "unavailable" {
				return fmt.Errorf("external unavailable runtime-connections sentinel conflicts with observed provider coverage")
			}
		}
	}
	for provider, evidenceItems := range observedProviders {
		for _, evidence := range evidenceItems {
			if !coveredObservation(evidence, runtimeCoverage[provider]) {
				return fmt.Errorf("observed evidence from %q is incompatible with runtime-connections coverage", provider)
			}
		}
	}
	return nil
}

func validEdgeKind(kind EdgeKind) bool {
	return kind == EdgeOwnership || kind == EdgeDeclared || kind == EdgeResolved || kind == EdgeInferred || kind == EdgeObserved
}

func hasObservedEvidence(evidence []Evidence, provider string) bool {
	for _, item := range evidence {
		if item.Basis == BasisObserved && item.Provider == provider && (item.ObservedAt != nil || item.WindowEnd != nil) {
			return true
		}
	}
	return false
}

func validCoverageStatus(status string) bool {
	return status == "complete" || status == "partial" || status == "unavailable" || status == "unknown"
}

func validPseudoKind(kind string) bool {
	return kind == "unresolved" || kind == "external-endpoint" || kind == "internet"
}

func coveredObservation(evidence Evidence, coverages []Coverage) bool {
	for _, coverage := range coverages {
		if coverage.Status != "complete" && coverage.Status != "partial" {
			continue
		}
		if evidence.WindowStart != nil || evidence.WindowEnd != nil {
			if coverage.WindowStart == nil || coverage.WindowEnd == nil {
				continue
			}
			if evidence.WindowStart != nil && coverage.WindowStart.After(*evidence.WindowStart) {
				continue
			}
			if evidence.WindowEnd != nil && coverage.WindowEnd.Before(*evidence.WindowEnd) {
				continue
			}
		}
		if evidence.ObservedAt != nil {
			if coverage.WindowStart != nil && evidence.ObservedAt.Before(*coverage.WindowStart) {
				continue
			}
			if coverage.WindowEnd != nil && evidence.ObservedAt.After(*coverage.WindowEnd) {
				continue
			}
			if coverage.WindowStart == nil && coverage.WindowEnd == nil {
				if coverage.ObservedAt == nil || evidence.ObservedAt.After(*coverage.ObservedAt) {
					continue
				}
			}
		}
		return true
	}
	return false
}

func validateEvidence(evidence Evidence) error {
	if evidence.Provider == "" || evidence.Summary == "" {
		return fmt.Errorf("evidence provider and summary are required")
	}
	if evidence.Basis != BasisDeclared && evidence.Basis != BasisResolved && evidence.Basis != BasisInferred && evidence.Basis != BasisObserved {
		return fmt.Errorf("unsupported evidence basis %q", evidence.Basis)
	}
	if evidence.Freshness != "current" && evidence.Freshness != "stale" && evidence.Freshness != "unknown" {
		return fmt.Errorf("unsupported evidence freshness %q", evidence.Freshness)
	}
	if evidence.Confidence != "high" && evidence.Confidence != "medium" && evidence.Confidence != "low" && evidence.Confidence != "unknown" {
		return fmt.Errorf("unsupported evidence confidence %q", evidence.Confidence)
	}
	if evidence.WindowStart != nil && evidence.WindowEnd != nil && evidence.WindowStart.After(*evidence.WindowEnd) {
		return fmt.Errorf("evidence window start is after window end")
	}
	return nil
}
