package hub

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/kuraudo-lab/teleskope/internal/analysis"
	"github.com/kuraudo-lab/teleskope/internal/topology"
)

const maxGraphRevisions = 16

// Store keeps the latest accepted envelope for every cluster.
type Store struct {
	mu       sync.RWMutex
	revision uint64
	latest   map[string]Envelope
	graphs   map[string][]topology.Graph
	body     []byte
	etag     string
	analysis *analysis.RunCache
}

// AnalyzeResponse is returned by cluster-scoped hub LLM analysis.
type AnalyzeResponse = analysis.RunResponse

// FleetResponse is the hub API response used by the fleet UI.
type FleetResponse struct {
	Revision uint64       `json:"revision"`
	Clusters []Summary    `json:"clusters"`
	Events   []FleetEvent `json:"events,omitempty"`
}

// Put validates and stores an envelope. Duplicate and older revisions are ignored.
func (s *Store) Put(env Envelope) (accepted bool, stored Envelope, err error) {
	env, err = CompleteEnvelope(env)
	if err != nil {
		return false, Envelope{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensure()
	current, ok := s.latest[env.Cluster.ID]
	if ok && current.Revision >= env.Revision {
		return false, current, nil
	}
	s.latest[env.Cluster.ID] = cloneEnvelope(env)
	if storedGraph := s.latest[env.Cluster.ID].Graph; storedGraph != nil {
		history := append(s.graphs[env.Cluster.ID], *storedGraph)
		if len(history) > maxGraphRevisions {
			history = append([]topology.Graph(nil), history[len(history)-maxGraphRevisions:]...)
		}
		s.graphs[env.Cluster.ID] = history
	}
	s.revision++
	s.encodeLocked()
	return true, s.latest[env.Cluster.ID], nil
}

// TopologyUpdate returns a cluster-scoped delta or a full reset when the
// requested revision is no longer retained.
func (s *Store) TopologyUpdate(id string, since uint64) (topology.Update, bool, error) {
	s.mu.RLock()
	history := s.graphs[id]
	if len(history) == 0 {
		s.mu.RUnlock()
		return topology.Update{}, false, nil
	}
	current := history[len(history)-1]
	var base *topology.Graph
	for i := range history {
		if history[i].Revision == since {
			candidate := history[i]
			base = &candidate
			break
		}
	}
	s.mu.RUnlock()
	update, err := topology.Diff(base, current)
	return update, true, err
}

// Get returns a copy of one cluster's latest envelope.
func (s *Store) Get(id string) (Envelope, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	env, ok := s.latest[id]
	if !ok {
		return Envelope{}, false
	}
	return cloneEnvelope(env), true
}

// Fleet returns sorted cluster summaries.
func (s *Store) Fleet() FleetResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fleetLocked()
}

func (s *Store) fleetLocked() FleetResponse {
	summaries := make([]Summary, 0, len(s.latest))
	events := make([]FleetEvent, 0)
	for _, env := range s.latest {
		summaries = append(summaries, summarize(env))
		for _, event := range env.Events {
			events = append(events, FleetEvent{Cluster: env.Cluster, Event: event})
		}
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Cluster.Name < summaries[j].Cluster.Name
	})
	sort.Slice(events, func(i, j int) bool {
		if events[i].Event.At.Equal(events[j].Event.At) {
			return events[i].Event.Sequence > events[j].Event.Sequence
		}
		return events[i].Event.At.After(events[j].Event.At)
	})
	if len(events) > 200 {
		events = events[:200]
	}
	return FleetResponse{Revision: s.revision, Clusters: summaries, Events: events}
}

func (s *Store) encodedFleet() ([]byte, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]byte(nil), s.body...), s.etag
}

func (s *Store) ensure() {
	if s.latest == nil {
		s.latest = map[string]Envelope{}
		s.graphs = map[string][]topology.Graph{}
		s.encodeLocked()
	}
	if s.graphs == nil {
		s.graphs = map[string][]topology.Graph{}
	}
	if s.analysis == nil {
		s.analysis = &analysis.RunCache{}
	}
}

func (s *Store) encodeLocked() {
	body, _ := json.Marshal(s.fleetLocked())
	s.body = body
	s.etag = fmt.Sprintf("\"%x\"", sha256.Sum256(body))
}

func cloneEnvelope(env Envelope) Envelope {
	data, err := json.Marshal(env)
	if err != nil {
		return env
	}
	var clone Envelope
	if err := json.Unmarshal(data, &clone); err != nil {
		return env
	}
	return clone
}
