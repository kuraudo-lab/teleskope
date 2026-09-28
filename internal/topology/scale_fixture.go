package topology

import (
	"fmt"
	"time"
)

// ScaleProfile describes a deterministic graph used for performance and
// browser acceptance. It is not a collector or runtime evidence provider.
type ScaleProfile struct {
	Name  string
	Nodes int
	Edges int
}

var scaleProfileCatalog = map[string]ScaleProfile{
	"small":  {Name: "small", Nodes: 100, Edges: 180},
	"medium": {Name: "medium", Nodes: 400, Edges: 800},
	"large":  {Name: "large", Nodes: 1500, Edges: 2800},
}

func ScaleProfiles() []ScaleProfile {
	return []ScaleProfile{scaleProfileCatalog["small"], scaleProfileCatalog["medium"], scaleProfileCatalog["large"]}
}

func LookupScaleProfile(name string) (ScaleProfile, bool) {
	profile, ok := scaleProfileCatalog[name]
	return profile, ok
}

func BuildScaleFixture(profile ScaleProfile, revision uint64) Graph {
	graph := Graph{
		SchemaVersion: SchemaVersion,
		ClusterID:     "scale-" + profile.Name,
		Revision:      revision,
		GeneratedAt:   time.Date(2026, 9, 27, 12, 0, int(revision), 0, time.UTC),
		Coverage:      []Coverage{{Provider: "external", Capability: "runtime-connections", Status: "unavailable", Reason: "scale fixture has no runtime provider"}},
	}
	for i := 0; i < profile.Nodes; i++ {
		name := fmt.Sprintf("workload-%04d", i)
		graph.Nodes = append(graph.Nodes, Node{
			ID: NodeID("kubernetes", graph.ClusterID, "workload", "scale", name), Kind: "workload", Name: name, Identity: name,
			Scope: Scope{Provider: "kubernetes", Cluster: graph.ClusterID, Namespace: "scale"},
		})
	}
	for offset := 1; len(graph.Edges) < profile.Edges; offset++ {
		for source := 0; source < profile.Nodes && len(graph.Edges) < profile.Edges; source++ {
			target := (source + offset) % profile.Nodes
			if source == target {
				continue
			}
			sourceID, targetID := graph.Nodes[source].ID, graph.Nodes[target].ID
			graph.Edges = append(graph.Edges, Edge{
				ID: EdgeID(EdgeDeclared, sourceID, targetID, "fixture"), Source: sourceID, Target: targetID,
				Kind: EdgeDeclared, Provider: "fixture", Relation: "depends-on", Directed: true,
				Evidence: []Evidence{{Provider: "fixture", Basis: BasisDeclared, Freshness: "current", Confidence: "high", Summary: "deterministic scale edge"}},
			})
		}
	}
	graph.Normalize()
	return graph
}
