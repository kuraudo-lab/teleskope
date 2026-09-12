package hub

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// Store keeps the latest accepted envelope for every cluster.
type Store struct {
	mu       sync.RWMutex
	revision uint64
	latest   map[string]Envelope
	body     []byte
	etag     string
}

// FleetResponse is the hub API response used by the fleet UI.
type FleetResponse struct {
	Revision uint64    `json:"revision"`
	Clusters []Summary `json:"clusters"`
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
	s.revision++
	s.encodeLocked()
	return true, s.latest[env.Cluster.ID], nil
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
	for _, env := range s.latest {
		summaries = append(summaries, summarize(env))
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Cluster.Name < summaries[j].Cluster.Name
	})
	return FleetResponse{Revision: s.revision, Clusters: summaries}
}

func (s *Store) encodedFleet() ([]byte, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]byte(nil), s.body...), s.etag
}

func (s *Store) ensure() {
	if s.latest == nil {
		s.latest = map[string]Envelope{}
		s.encodeLocked()
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
