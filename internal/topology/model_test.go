package topology

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestStableIdentityEscapesSegments(t *testing.T) {
	id := NodeID("kubernetes", "arn:aws:eks:region:123:cluster/prod", "Deployment", "app/team", "web/api")
	for _, want := range []string{"kubernetes/", "deployment/", "app%2Fteam", "web%2Fapi"} {
		if !strings.Contains(id, want) {
			t.Fatalf("NodeID() = %q, want segment %q", id, want)
		}
	}
	if id != NodeID("kubernetes", "arn:aws:eks:region:123:cluster/prod", "Deployment", "app/team", "web/api") {
		t.Fatal("NodeID is not stable")
	}
}

func TestValidateRejectsObservedEdgeWithoutObservedEvidence(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 2, 3, 0, time.UTC)
	scope := Scope{Provider: "kubernetes", Cluster: "prod"}
	aID := NodeID("kubernetes", "prod", "pod", "", "a")
	bID := NodeID("kubernetes", "prod", "service", "", "b")
	edgeID := EdgeID(EdgeObserved, aID, bID, "otel")
	graph := Graph{
		SchemaVersion: SchemaVersion,
		ClusterID:     "prod",
		GeneratedAt:   now,
		Nodes: []Node{
			{ID: aID, Kind: "pod", Name: "a", Identity: "a", Scope: scope},
			{ID: bID, Kind: "service", Name: "b", Identity: "b", Scope: scope},
		},
		Edges: []Edge{{
			ID: edgeID, Source: aID, Target: bID, Kind: EdgeObserved, Provider: "otel", Directed: true,
			Evidence: []Evidence{{Provider: "kubernetes", Basis: BasisDeclared, Freshness: "current", Confidence: "high", Summary: "selector"}},
		}},
		Coverage: []Coverage{{Provider: "external", Capability: "runtime-connections", Status: "unavailable"}},
	}
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "requires observed evidence") {
		t.Fatalf("Validate() error = %v", err)
	}

	graph.Edges[0].Evidence = []Evidence{{Provider: "otel", Basis: BasisObserved, ObservedAt: &now, Freshness: "current", Confidence: "high", Summary: "flow"}}
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "sentinel conflicts") {
		t.Fatalf("Validate() sentinel conflict error = %v", err)
	}
	graph.Coverage = []Coverage{{Provider: "otel", Capability: "runtime-connections", Status: "unavailable", ObservedAt: &now}}
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "incompatible with runtime-connections coverage") {
		t.Fatalf("Validate() coverage mismatch error = %v", err)
	}
	graph.Coverage[0].Status = "complete"
	if err := graph.Validate(); err != nil {
		t.Fatalf("Validate() with observed evidence and coverage: %v", err)
	}
	old := now.Add(-time.Hour)
	graph.Coverage[0].ObservedAt = &old
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "incompatible with runtime-connections coverage") {
		t.Fatalf("Validate() point-time mismatch error = %v", err)
	}
}

func TestProjectSeparatesDeclaredResolvedAndUnavailableRuntimeEvidence(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 2, 3, 0, time.UTC)
	snapshot := &inventory.Snapshot{
		CollectedAt: now,
		EKS: inventory.EKSInventory{
			Cluster:    inventory.Cluster{Name: "prod", ARN: "arn:aws:eks:eu-west-1:123:cluster/prod"},
			Nodegroups: []inventory.Nodegroup{{Name: "system", ARN: "arn:aws:eks:eu-west-1:123:nodegroup/prod/system/1"}},
		},
		AWS: inventory.AWSIdentity{AccountID: "123", Region: "eu-west-1"},
		Kubernetes: inventory.Kubernetes{
			Context:        "prod",
			Nodes:          []inventory.Node{{ObjectRef: inventory.ObjectRef{Kind: "Node", Name: "node-a"}, Labels: map[string]string{"eks.amazonaws.com/nodegroup": "system"}}},
			Workloads:      []inventory.Workload{{ObjectRef: inventory.ObjectRef{Kind: "Deployment", Namespace: "app", Name: "web"}, Selector: map[string]string{"app": "web"}}},
			Pods:           []inventory.Pod{{ObjectRef: inventory.ObjectRef{Kind: "Pod", Namespace: "app", Name: "web-a"}, NodeName: "node-a", OwnerReferences: []inventory.ObjectRef{{Kind: "Deployment", Namespace: "app", Name: "web"}}}},
			Services:       []inventory.Service{{ObjectRef: inventory.ObjectRef{Kind: "Service", Namespace: "app", Name: "web"}, Selector: map[string]string{"app": "web"}}},
			EndpointSlices: []inventory.EndpointSlice{{ObjectRef: inventory.ObjectRef{Kind: "EndpointSlice", Namespace: "app", Name: "web-a"}, ServiceName: "web", Endpoints: []inventory.EndpointTarget{{TargetRef: inventory.ObjectRef{Kind: "Pod", Namespace: "app", Name: "web-a"}}}}},
		},
	}

	graph := Project(snapshot, 7)
	if err := graph.Validate(); err != nil {
		t.Fatalf("Validate(): %v", err)
	}
	if graph.Revision != 7 || graph.SchemaVersion != SchemaVersion {
		t.Fatalf("graph identity = %#v", graph)
	}
	kinds := map[EdgeKind]int{}
	for _, edge := range graph.Edges {
		kinds[edge.Kind]++
		if edge.Kind == EdgeObserved {
			t.Fatalf("projection fabricated observed edge: %+v", edge)
		}
	}
	if kinds[EdgeDeclared] == 0 || kinds[EdgeResolved] == 0 || kinds[EdgeOwnership] == 0 {
		t.Fatalf("edge kinds = %#v", kinds)
	}
	if !hasCoverage(graph.Coverage, "external", "runtime-connections", "unavailable") {
		t.Fatalf("coverage = %+v", graph.Coverage)
	}

	first, err := graph.MarshalDeterministic()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Project(snapshot, 7).MarshalDeterministic()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("graph JSON is not deterministic\n%s\n%s", first, second)
	}
	var roundTrip Graph
	if err := json.Unmarshal(first, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if err := roundTrip.Validate(); err != nil {
		t.Fatalf("round-trip validation: %v", err)
	}
}

func TestValidateRejectsDanglingAndDuplicateIdentities(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 2, 3, 0, time.UTC)
	scope := Scope{Provider: "kubernetes", Cluster: "prod"}
	aID := NodeID("kubernetes", "prod", "pod", "", "a")
	graph := Graph{SchemaVersion: SchemaVersion, ClusterID: "prod", GeneratedAt: now, Nodes: []Node{{ID: aID, Kind: "pod", Name: "a", Identity: "a", Scope: scope}, {ID: aID, Kind: "pod", Name: "a", Identity: "a", Scope: scope}}, Coverage: []Coverage{{Provider: "external", Capability: "runtime-connections", Status: "unavailable"}}}
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate node") {
		t.Fatalf("duplicate validation error = %v", err)
	}
	graph.Nodes = graph.Nodes[:1]
	graph.Edges = []Edge{{ID: EdgeID(EdgeDeclared, aID, "missing", "kubernetes"), Source: aID, Target: "missing", Kind: EdgeDeclared, Provider: "kubernetes", Evidence: []Evidence{{Provider: "kubernetes", Basis: BasisDeclared, Freshness: "current", Confidence: "high", Summary: "reference"}}}}
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "missing target") {
		t.Fatalf("dangling validation error = %v", err)
	}
}

func TestValidateRejectsNonCanonicalIdentityAndMissingRuntimeCoverage(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 2, 3, 0, time.UTC)
	scope := Scope{Provider: "kubernetes", Cluster: "prod"}
	graph := Graph{SchemaVersion: SchemaVersion, ClusterID: "prod", GeneratedAt: now, Nodes: []Node{{ID: "pod-a", Kind: "pod", Name: "a", Identity: "a", Scope: scope}}, Coverage: []Coverage{{Provider: "external", Capability: "runtime-connections", Status: "unavailable"}}}
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "not canonical") {
		t.Fatalf("non-canonical validation error = %v", err)
	}
	graph.Nodes[0].ID = NodeID("kubernetes", "prod", "pod", "", "a")
	graph.Coverage = nil
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "runtime-connections coverage is required") {
		t.Fatalf("missing runtime coverage error = %v", err)
	}
}

func TestProjectKeepsSameNameWorkloadKindsAndPseudoTargetsDistinct(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 2, 3, 0, time.UTC)
	snapshot := &inventory.Snapshot{CollectedAt: now, EKS: inventory.EKSInventory{Cluster: inventory.Cluster{Name: "prod", VPC: inventory.VPCConfig{EndpointPublicAccess: true, PublicAccessCIDRs: []string{"203.0.113.0/24"}}}}, Kubernetes: inventory.Kubernetes{
		Context: "prod",
		Workloads: []inventory.Workload{
			{ObjectRef: inventory.ObjectRef{Kind: "Deployment", Namespace: "app", Name: "web"}, Selector: map[string]string{"app": "web"}},
			{ObjectRef: inventory.ObjectRef{Kind: "StatefulSet", Namespace: "app", Name: "web"}, Selector: map[string]string{"app": "stateful-web"}},
		},
		Pods: []inventory.Pod{
			{ObjectRef: inventory.ObjectRef{Kind: "Pod", Namespace: "app", Name: "web-a"}, OwnerReferences: []inventory.ObjectRef{{Kind: "Deployment", Namespace: "app", Name: "web"}}},
			{ObjectRef: inventory.ObjectRef{Kind: "Pod", Namespace: "app", Name: "ghost-a"}, OwnerReferences: []inventory.ObjectRef{{Kind: "ReplicaSet", Namespace: "app", Name: "ghost"}}},
		},
		Services: []inventory.Service{{ObjectRef: inventory.ObjectRef{Kind: "Service", Namespace: "app", Name: "public"}, ExternalIPs: []string{"203.0.113.10"}}},
	}}
	graph := Project(snapshot, 1)
	if err := graph.Validate(); err != nil {
		t.Fatal(err)
	}
	var workloads, unresolved, external, internet int
	for _, node := range graph.Nodes {
		switch node.PseudoKind {
		case "unresolved":
			unresolved++
		case "external-endpoint":
			external++
		case "internet":
			internet++
		}
		if node.Kind == "workload" {
			workloads++
		}
	}
	if workloads != 2 || unresolved == 0 || external == 0 || internet != 1 {
		t.Fatalf("workloads=%d unresolved=%d external=%d internet=%d nodes=%+v", workloads, unresolved, external, internet, graph.Nodes)
	}
	deploymentID := NodeID("kubernetes", "prod", "workload", "app", "deployment:web")
	podID := NodeID("kubernetes", "prod", "pod", "app", "web-a")
	if !hasEdge(graph.Edges, EdgeOwnership, deploymentID, podID) {
		t.Fatalf("deployment ownership edge missing: %+v", graph.Edges)
	}
	internetID := PseudoNodeID("external", "prod", "internet", "", "internet")
	clusterID := NodeID("eks", "prod", "cluster", "", "prod")
	if !hasEdge(graph.Edges, EdgeDeclared, clusterID, internetID) {
		t.Fatalf("Internet projection edge missing: %+v", graph.Edges)
	}
	for _, edge := range graph.Edges {
		if edge.Source == clusterID && edge.Target == internetID && edge.Metadata["publicAccessCidrs"] != "" {
			return
		}
	}
	t.Fatal("Internet edge did not preserve public access CIDRs")
}

func TestProjectAttachesDeterministicFindingsToEvidenceNodes(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 2, 3, 0, time.UTC)
	snapshot := &inventory.Snapshot{CollectedAt: now, Kubernetes: inventory.Kubernetes{
		Context:        "prod",
		IngressClasses: []inventory.IngressClass{{ObjectRef: inventory.ObjectRef{APIVersion: "networking.k8s.io/v1", Kind: "IngressClass", Name: "nginx"}, Controller: "k8s.io/ingress-nginx"}},
	}}
	graph := Project(snapshot, 1)
	if err := graph.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, node := range graph.Nodes {
		if node.Name == "nginx" && contains(node.FindingRefs, "networking.ingress/v1") {
			return
		}
	}
	t.Fatalf("advisor finding was not attached: %+v", graph.Nodes)
}

func TestProjectAttachesEKSNodegroupFindingWithoutDuplicateNode(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 2, 3, 0, time.UTC)
	snapshot := &inventory.Snapshot{CollectedAt: now,
		EKS:        inventory.EKSInventory{Cluster: inventory.Cluster{Name: "prod"}, Nodegroups: []inventory.Nodegroup{{Name: "system", Status: "ACTIVE"}}},
		Kubernetes: inventory.Kubernetes{Context: "prod", Nodes: []inventory.Node{{ObjectRef: inventory.ObjectRef{Kind: "Node", Name: "node-a"}, Labels: map[string]string{"eks.amazonaws.com/nodegroup": "system"}}}},
	}
	graph := Project(snapshot, 1)
	if err := graph.Validate(); err != nil {
		t.Fatal(err)
	}
	var groups []Node
	var groupID, nodeID string
	for _, node := range graph.Nodes {
		if node.Kind == "nodegroup" && node.Name == "system" {
			groups = append(groups, node)
			groupID = node.ID
		}
		if node.Kind == "node" && node.Name == "node-a" {
			nodeID = node.ID
		}
	}
	if len(groups) != 1 || groups[0].Scope.Provider != "eks" || !contains(groups[0].FindingRefs, "eks.nodegroup/v1") {
		t.Fatalf("nodegroups = %+v", groups)
	}
	for _, edge := range graph.Edges {
		if edge.Source == groupID && edge.Target == nodeID && contains(edge.FindingRefs, "eks.nodegroup/v1") {
			return
		}
	}
	t.Fatalf("nodegroup finding edge missing: %+v", graph.Edges)
}

func TestNormalizeOrdersCompleteEvidenceAndFields(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 2, 3, 0, time.UTC)
	scope := Scope{Provider: "kubernetes", Cluster: "prod"}
	id := NodeID("kubernetes", "prod", "pod", "app", "web")
	graph := Graph{SchemaVersion: SchemaVersion, ClusterID: "prod", GeneratedAt: now, Nodes: []Node{{ID: id, Kind: "pod", Name: "web", Identity: "web", Scope: Scope{Provider: scope.Provider, Cluster: scope.Cluster, Namespace: "app"}, Evidence: []Evidence{
		{Provider: "z", Basis: BasisInferred, Freshness: "unknown", Confidence: "low", Summary: "same", Sampling: "1:100", Fields: []FieldReference{{Resource: "z"}, {Resource: "a"}}},
		{Provider: "a", Basis: BasisResolved, Freshness: "current", Confidence: "high", Summary: "same"},
	}}}, Coverage: []Coverage{{Provider: "external", Capability: "runtime-connections", Status: "unavailable"}}}
	graph.Normalize()
	if graph.Nodes[0].Evidence[0].Provider != "a" || graph.Nodes[0].Evidence[1].Fields[0].Resource != "a" {
		t.Fatalf("evidence normalization = %+v", graph.Nodes[0].Evidence)
	}
}

func TestValidateRejectsDuplicateCoverageAndWrongClusterScope(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 2, 3, 0, time.UTC)
	id := NodeID("kubernetes", "other", "pod", "", "web")
	graph := Graph{SchemaVersion: SchemaVersion, ClusterID: "prod", GeneratedAt: now, Nodes: []Node{{ID: id, Kind: "pod", Name: "web", Identity: "web", Scope: Scope{Provider: "kubernetes", Cluster: "other"}}}, Coverage: []Coverage{{Provider: "external", Capability: "runtime-connections", Status: "unavailable"}}}
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "does not match graph cluster") {
		t.Fatalf("cluster mismatch error = %v", err)
	}
	graph.ClusterID = "other"
	graph.Coverage = append(graph.Coverage, graph.Coverage[0])
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate coverage identity") {
		t.Fatalf("duplicate coverage error = %v", err)
	}
	graph.Coverage = graph.Coverage[:1]
	end, start := now.Add(-time.Minute), now
	graph.Coverage[0].WindowStart, graph.Coverage[0].WindowEnd = &start, &end
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "coverage window start is after") {
		t.Fatalf("coverage window error = %v", err)
	}
}

func TestProjectMergesDuplicateCoverageConservatively(t *testing.T) {
	old := time.Date(2026, 9, 27, 1, 0, 0, 0, time.UTC)
	newer := old.Add(time.Minute)
	snapshot := &inventory.Snapshot{CollectedAt: newer, Kubernetes: inventory.Kubernetes{Context: "prod"}, Coverage: []inventory.CoverageItem{
		{Area: "kubernetes", Resource: "Pods", Status: "complete", CollectedAt: newer},
		{Area: "kubernetes", Resource: "Pods", Status: "partial", Reason: "forbidden", CollectedAt: old},
	}}
	graph := Project(snapshot, 1)
	if err := graph.Validate(); err != nil {
		t.Fatal(err)
	}
	var matches []Coverage
	for _, coverage := range graph.Coverage {
		if coverage.Provider == "kubernetes" && coverage.Capability == "Pods" {
			matches = append(matches, coverage)
		}
	}
	if len(matches) != 1 || matches[0].Status != "partial" || matches[0].Reason != "forbidden" || matches[0].ObservedAt == nil || !matches[0].ObservedAt.Equal(old) {
		t.Fatalf("merged coverage = %+v", matches)
	}
}

func TestValidateRequiresPseudoKindInBothDirections(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 2, 3, 0, time.UTC)
	id := PseudoNodeID("external", "prod", "internet", "", "internet")
	graph := Graph{SchemaVersion: SchemaVersion, ClusterID: "prod", GeneratedAt: now, Nodes: []Node{{ID: id, Kind: "pseudo-internet", Name: "Internet", Identity: "internet", Scope: Scope{Provider: "external", Cluster: "prod"}}}, Coverage: []Coverage{{Provider: "external", Capability: "runtime-connections", Status: "unavailable"}}}
	if err := graph.Validate(); err == nil || !strings.Contains(err.Error(), "inconsistent") {
		t.Fatalf("pseudo direction validation error = %v", err)
	}
}

func hasEdge(edges []Edge, kind EdgeKind, source, target string) bool {
	for _, edge := range edges {
		if edge.Kind == kind && edge.Source == source && edge.Target == target {
			return true
		}
	}
	return false
}

func hasCoverage(items []Coverage, provider, capability, status string) bool {
	for _, item := range items {
		if item.Provider == provider && item.Capability == capability && item.Status == status {
			return true
		}
	}
	return false
}
