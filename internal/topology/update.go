package topology

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

const UpdateSchemaVersion = "teleskope.io/topology-update/v1alpha1"

type UpdateKind string

const (
	UpdateFull      UpdateKind = "full"
	UpdateDelta     UpdateKind = "delta"
	UpdateUnchanged UpdateKind = "unchanged"
)

// Update is a deterministic, idempotent graph revision handoff. Full updates
// reset the client baseline; deltas are valid only for BaseRevision.
type Update struct {
	SchemaVersion string     `json:"schemaVersion"`
	Kind          UpdateKind `json:"kind"`
	ClusterID     string     `json:"clusterId"`
	BaseRevision  uint64     `json:"baseRevision,omitempty"`
	Revision      uint64     `json:"revision"`
	GeneratedAt   time.Time  `json:"generatedAt"`
	Graph         *Graph     `json:"graph,omitempty"`
	UpsertNodes   []Node     `json:"upsertNodes,omitempty"`
	DeleteNodeIDs []string   `json:"deleteNodeIds,omitempty"`
	UpsertEdges   []Edge     `json:"upsertEdges,omitempty"`
	DeleteEdgeIDs []string   `json:"deleteEdgeIds,omitempty"`
	Coverage      []Coverage `json:"coverage,omitempty"`
}

// Diff returns a full reset when the base is absent or incompatible and a
// canonical delta otherwise. The input graphs are never mutated.
func Diff(base *Graph, current Graph) (Update, error) {
	next, err := cloneGraph(current)
	if err != nil {
		return Update{}, err
	}
	next.Normalize()
	if err := next.Validate(); err != nil {
		return Update{}, fmt.Errorf("current graph: %w", err)
	}
	full := func() Update {
		return Update{SchemaVersion: UpdateSchemaVersion, Kind: UpdateFull, ClusterID: next.ClusterID, Revision: next.Revision, GeneratedAt: next.GeneratedAt, Graph: &next}
	}
	if base == nil {
		return full(), nil
	}
	previous, err := cloneGraph(*base)
	if err != nil || previous.Validate() != nil || previous.SchemaVersion != next.SchemaVersion || previous.ClusterID != next.ClusterID || previous.Revision > next.Revision {
		return full(), nil
	}
	previous.Normalize()
	if previous.Revision == next.Revision {
		if !reflect.DeepEqual(previous, next) {
			return full(), nil
		}
		return Update{SchemaVersion: UpdateSchemaVersion, Kind: UpdateUnchanged, ClusterID: next.ClusterID, BaseRevision: next.Revision, Revision: next.Revision, GeneratedAt: next.GeneratedAt}, nil
	}

	update := Update{
		SchemaVersion: UpdateSchemaVersion,
		Kind:          UpdateDelta,
		ClusterID:     next.ClusterID,
		BaseRevision:  previous.Revision,
		Revision:      next.Revision,
		GeneratedAt:   next.GeneratedAt,
		Coverage:      append([]Coverage(nil), next.Coverage...),
	}
	previousNodes := make(map[string]Node, len(previous.Nodes))
	for _, node := range previous.Nodes {
		previousNodes[node.ID] = node
	}
	nextNodes := make(map[string]Node, len(next.Nodes))
	for _, node := range next.Nodes {
		nextNodes[node.ID] = node
		if old, ok := previousNodes[node.ID]; !ok || !reflect.DeepEqual(old, node) {
			update.UpsertNodes = append(update.UpsertNodes, node)
		}
	}
	for id := range previousNodes {
		if _, ok := nextNodes[id]; !ok {
			update.DeleteNodeIDs = append(update.DeleteNodeIDs, id)
		}
	}
	previousEdges := make(map[string]Edge, len(previous.Edges))
	for _, edge := range previous.Edges {
		previousEdges[edge.ID] = edge
	}
	nextEdges := make(map[string]Edge, len(next.Edges))
	for _, edge := range next.Edges {
		nextEdges[edge.ID] = edge
		if old, ok := previousEdges[edge.ID]; !ok || !reflect.DeepEqual(old, edge) {
			update.UpsertEdges = append(update.UpsertEdges, edge)
		}
	}
	for id := range previousEdges {
		if _, ok := nextEdges[id]; !ok {
			update.DeleteEdgeIDs = append(update.DeleteEdgeIDs, id)
		}
	}
	sort.Strings(update.DeleteNodeIDs)
	sort.Strings(update.DeleteEdgeIDs)
	return update, nil
}

// Apply validates and applies one update. Reapplying an already accepted
// revision is a no-op; any other base mismatch requires a full resync.
func Apply(base *Graph, update Update) (Graph, error) {
	if err := update.validateEnvelope(); err != nil {
		return Graph{}, err
	}
	if update.Kind == UpdateFull {
		if update.Graph == nil {
			return Graph{}, fmt.Errorf("full update requires graph")
		}
		graph, err := cloneGraph(*update.Graph)
		if err != nil {
			return Graph{}, err
		}
		if graph.ClusterID != update.ClusterID || graph.Revision != update.Revision || !graph.GeneratedAt.Equal(update.GeneratedAt) {
			return Graph{}, fmt.Errorf("full graph envelope does not match update")
		}
		graph.Normalize()
		if err := graph.Validate(); err != nil {
			return Graph{}, err
		}
		return graph, nil
	}
	if base == nil {
		return Graph{}, fmt.Errorf("%s update requires base graph", update.Kind)
	}
	current, err := cloneGraph(*base)
	if err != nil {
		return Graph{}, err
	}
	if current.SchemaVersion != SchemaVersion || current.ClusterID != update.ClusterID {
		return Graph{}, fmt.Errorf("update cluster or graph schema does not match base")
	}
	if current.Revision == update.Revision {
		return current, nil
	}
	if current.Revision != update.BaseRevision {
		return Graph{}, fmt.Errorf("topology base revision %d does not match update base %d; full resync required", current.Revision, update.BaseRevision)
	}
	if update.Kind == UpdateUnchanged {
		return Graph{}, fmt.Errorf("unchanged update revision %d does not match base revision %d", update.Revision, current.Revision)
	}

	nodes := make(map[string]Node, len(current.Nodes)+len(update.UpsertNodes))
	for _, node := range current.Nodes {
		nodes[node.ID] = node
	}
	for _, id := range update.DeleteNodeIDs {
		delete(nodes, id)
	}
	for _, node := range update.UpsertNodes {
		nodes[node.ID] = node
	}
	edges := make(map[string]Edge, len(current.Edges)+len(update.UpsertEdges))
	for _, edge := range current.Edges {
		edges[edge.ID] = edge
	}
	for _, id := range update.DeleteEdgeIDs {
		delete(edges, id)
	}
	for _, edge := range update.UpsertEdges {
		edges[edge.ID] = edge
	}
	current.Revision = update.Revision
	current.GeneratedAt = update.GeneratedAt
	current.Coverage = append([]Coverage(nil), update.Coverage...)
	current.Nodes = current.Nodes[:0]
	for _, node := range nodes {
		current.Nodes = append(current.Nodes, node)
	}
	current.Edges = current.Edges[:0]
	for _, edge := range edges {
		current.Edges = append(current.Edges, edge)
	}
	current.Normalize()
	if err := current.Validate(); err != nil {
		return Graph{}, fmt.Errorf("apply topology update: %w", err)
	}
	return current, nil
}

func (u Update) validateEnvelope() error {
	if u.SchemaVersion != UpdateSchemaVersion {
		return fmt.Errorf("unsupported topology update schema %q", u.SchemaVersion)
	}
	if strings.TrimSpace(u.ClusterID) == "" || u.Revision == 0 || u.GeneratedAt.IsZero() {
		return fmt.Errorf("clusterId, revision, and generatedAt are required")
	}
	switch u.Kind {
	case UpdateFull:
		if u.BaseRevision != 0 {
			return fmt.Errorf("full update must not declare baseRevision")
		}
	case UpdateDelta:
		if u.BaseRevision == 0 || u.BaseRevision >= u.Revision {
			return fmt.Errorf("delta baseRevision must be positive and older than revision")
		}
	case UpdateUnchanged:
		if u.BaseRevision != u.Revision {
			return fmt.Errorf("unchanged update must use the current revision as its base")
		}
	default:
		return fmt.Errorf("unsupported topology update kind %q", u.Kind)
	}
	return nil
}

func cloneGraph(graph Graph) (Graph, error) {
	data, err := json.Marshal(graph)
	if err != nil {
		return Graph{}, fmt.Errorf("clone topology graph: %w", err)
	}
	var clone Graph
	if err := json.Unmarshal(data, &clone); err != nil {
		return Graph{}, fmt.Errorf("clone topology graph: %w", err)
	}
	return clone, nil
}
