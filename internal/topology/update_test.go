package topology

import (
	"strings"
	"testing"
	"time"
)

func TestDiffAndApplyTopologyUpdate(t *testing.T) {
	base := updateTestGraph(4)
	next := updateTestGraph(5)
	next.Nodes[0].Metadata = map[string]string{"phase": "Running"}
	service := updateTestNode("service", "app", "api", "service-api")
	next.Nodes = append(next.Nodes, service)
	edge := Edge{
		ID:       EdgeID(EdgeResolved, service.ID, next.Nodes[0].ID, "kubernetes"),
		Source:   service.ID,
		Target:   next.Nodes[0].ID,
		Kind:     EdgeResolved,
		Provider: "kubernetes",
		Relation: "selects",
		Directed: true,
		Evidence: []Evidence{{Provider: "kubernetes", Basis: BasisResolved, Freshness: "current", Confidence: "high", Summary: "selector resolved to pod"}},
	}
	next.Edges = append(next.Edges, edge)

	update, err := Diff(&base, next)
	if err != nil {
		t.Fatal(err)
	}
	if update.Kind != UpdateDelta || update.BaseRevision != 4 || update.Revision != 5 {
		t.Fatalf("unexpected update envelope: %+v", update)
	}
	if len(update.UpsertNodes) != 2 || len(update.UpsertEdges) != 1 {
		t.Fatalf("unexpected upserts: nodes=%d edges=%d", len(update.UpsertNodes), len(update.UpsertEdges))
	}
	applied, err := Apply(&base, update)
	if err != nil {
		t.Fatal(err)
	}
	applied.Normalize()
	next.Normalize()
	if string(mustGraphJSON(t, applied)) != string(mustGraphJSON(t, next)) {
		t.Fatalf("applied graph differs\n got: %s\nwant: %s", mustGraphJSON(t, applied), mustGraphJSON(t, next))
	}
	duplicate, err := Apply(&applied, update)
	if err != nil || duplicate.Revision != applied.Revision {
		t.Fatalf("duplicate apply = revision %d, err %v", duplicate.Revision, err)
	}
}

func TestDiffCapturesExplicitDeletes(t *testing.T) {
	base := updateTestGraph(7)
	service := updateTestNode("service", "app", "api", "service-api")
	base.Nodes = append(base.Nodes, service)
	base.Edges = append(base.Edges, Edge{
		ID: EdgeID(EdgeResolved, service.ID, base.Nodes[0].ID, "kubernetes"), Source: service.ID, Target: base.Nodes[0].ID,
		Kind: EdgeResolved, Provider: "kubernetes", Directed: true,
		Evidence: []Evidence{{Provider: "kubernetes", Basis: BasisResolved, Freshness: "current", Confidence: "high", Summary: "selector resolved to pod"}},
	})
	next := updateTestGraph(8)
	update, err := Diff(&base, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(update.DeleteNodeIDs) != 1 || update.DeleteNodeIDs[0] != service.ID || len(update.DeleteEdgeIDs) != 1 {
		t.Fatalf("delete update = %+v", update)
	}
	if _, err := Apply(&base, update); err != nil {
		t.Fatal(err)
	}
}

func TestTopologyUpdateForcesFullResync(t *testing.T) {
	current := updateTestGraph(9)
	wrongCluster := updateTestGraph(8)
	wrongCluster.ClusterID = "other"
	wrongCluster.Nodes[0].Scope.Cluster = "other"
	wrongCluster.Nodes[0].ID = NodeID("kubernetes", "other", "pod", "app", "pod-api")
	update, err := Diff(&wrongCluster, current)
	if err != nil {
		t.Fatal(err)
	}
	if update.Kind != UpdateFull || update.Graph == nil {
		t.Fatalf("incompatible base should receive full reset: %+v", update)
	}
	if _, err := Apply(nil, update); err != nil {
		t.Fatal(err)
	}

	delta, err := Diff(ptrGraph(updateTestGraph(8)), current)
	if err != nil {
		t.Fatal(err)
	}
	stale := updateTestGraph(7)
	if _, err := Apply(&stale, delta); err == nil || !strings.Contains(err.Error(), "full resync required") {
		t.Fatalf("out-of-order apply error = %v", err)
	}
}

func TestTopologyUpdateUnchangedAndRevisionCollision(t *testing.T) {
	graph := updateTestGraph(3)
	update, err := Diff(&graph, graph)
	if err != nil || update.Kind != UpdateUnchanged {
		t.Fatalf("unchanged update = %+v, err %v", update, err)
	}
	changed := updateTestGraph(3)
	changed.Nodes[0].Name = "changed"
	update, err = Diff(&graph, changed)
	if err == nil {
		if update.Kind != UpdateFull {
			t.Fatalf("revision collision should force full reset: %+v", update)
		}
	} else if !strings.Contains(err.Error(), "canonical") {
		t.Fatal(err)
	}
}

func updateTestGraph(revision uint64) Graph {
	return Graph{
		SchemaVersion: SchemaVersion,
		ClusterID:     "prod",
		Revision:      revision,
		GeneratedAt:   time.Date(2026, 9, 27, 8, int(revision), 0, 0, time.UTC),
		Nodes:         []Node{updateTestNode("pod", "app", "api-0", "pod-api")},
		Coverage:      []Coverage{{Provider: "external", Capability: "runtime-connections", Status: "unavailable", Reason: "no external connection evidence provider configured"}},
	}
}

func updateTestNode(kind, namespace, name, identity string) Node {
	return Node{
		ID: NodeID("kubernetes", "prod", kind, namespace, identity), Kind: kind, Name: name, Identity: identity,
		Scope: Scope{Provider: "kubernetes", Cluster: "prod", Namespace: namespace},
	}
}

func ptrGraph(graph Graph) *Graph { return &graph }

func mustGraphJSON(t *testing.T, graph Graph) []byte {
	t.Helper()
	data, err := graph.MarshalDeterministic()
	if err != nil {
		t.Fatal(err)
	}
	return data
}
