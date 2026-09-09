// Package live maintains independently refreshed sources and serves a shared view.
package live

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/buildinfo"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"github.com/kuraudo-lab/teleskope/internal/report"
)

// Source is a sequential polling task. Collect transfers ownership of its result.
type Source struct {
	Name              string
	Interval, Timeout time.Duration
	Collect           func(context.Context) (*inventory.Snapshot, error)
}

// Status describes the latest attempt independently of the retained data.
type Status struct {
	State       string                   `json:"state"`
	Refreshing  bool                     `json:"refreshing"`
	LastAttempt *time.Time               `json:"lastAttempt,omitempty"`
	LastSuccess *time.Time               `json:"lastSuccess,omitempty"`
	NextAttempt *time.Time               `json:"nextAttempt,omitempty"`
	Error       string                   `json:"error,omitempty"`
	Coverage    []inventory.CoverageItem `json:"coverage,omitempty"`
}

type entry struct {
	status   Status
	snapshot *inventory.Snapshot
}

// Store owns all mutable state; handlers receive pre-encoded immutable responses.
type Store struct {
	mu       sync.RWMutex
	order    []string
	sources  []Source
	entries  map[string]*entry
	revision uint64
	body     []byte
	etag     string
}

type response struct {
	Revision uint64              `json:"revision"`
	Snapshot *inventory.Snapshot `json:"snapshot"`
	Sources  map[string]Status   `json:"sources"`
}

// New validates the source configuration before any collection starts.
func New(sources []Source) (*Store, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("at least one source is required")
	}
	s := &Store{entries: make(map[string]*entry), sources: append([]Source(nil), sources...)}
	for _, source := range sources {
		if source.Name != "kubernetes" && source.Name != "eks" {
			return nil, fmt.Errorf("unknown source %q", source.Name)
		}
		if _, ok := s.entries[source.Name]; ok {
			return nil, fmt.Errorf("duplicate source %q", source.Name)
		}
		if source.Interval <= 0 || source.Timeout <= 0 || source.Collect == nil {
			return nil, fmt.Errorf("source %s requires positive interval, timeout and collector", source.Name)
		}
		s.order = append(s.order, source.Name)
		s.entries[source.Name] = &entry{status: Status{State: "loading"}}
	}
	s.encode()
	return s, nil
}

// Run polls immediately, then waits after each completed attempt. Slow scans never
// overlap or accumulate a backlog. Run returns after cancellation and worker exit.
func (s *Store) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, source := range s.sources {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				s.begin(source.Name)
				attemptCtx, cancel := context.WithTimeout(ctx, source.Timeout)
				snapshot, err := source.Collect(attemptCtx)
				if attemptCtx.Err() != nil {
					err = attemptCtx.Err()
				}
				cancel()
				if ctx.Err() != nil {
					return
				}
				// Small jitter prevents synchronized full lists from multiple installations.
				delay := source.Interval + time.Duration(float64(source.Interval)*rand.Float64()*0.1)
				s.finish(source.Name, snapshot, err, time.Now().UTC().Add(delay))
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
		}()
	}
	wg.Wait()
}

func (s *Store) begin(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entries[name]
	now := time.Now().UTC()
	e.status.Refreshing, e.status.LastAttempt, e.status.NextAttempt = true, &now, nil
	s.encode()
}

// coverageRegression conservatively keeps the whole previous source if an
// already observed resource can no longer be read. This also keeps its derived
// relationships consistent. A successful empty list is a valid replacement.
func coverageRegression(old, next []inventory.CoverageItem) error {
	byKey := map[string]inventory.CoverageItem{}
	for _, c := range next {
		byKey[c.Area+"/"+c.Resource] = c
	}
	for _, c := range old {
		if c.Status != "complete" && c.ObjectCount == 0 {
			continue
		}
		n, ok := byKey[c.Area+"/"+c.Resource]
		if !ok || n.Status != "complete" {
			return fmt.Errorf("%s/%s could not be refreshed; retaining previous source data", c.Area, c.Resource)
		}
	}
	return nil
}

func (s *Store) finish(name string, snapshot *inventory.Snapshot, err error, next time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entries[name]
	e.status.Refreshing, e.status.NextAttempt = false, &next
	e.status.Coverage = nil
	if snapshot != nil {
		e.status.Coverage = append([]inventory.CoverageItem(nil), snapshot.Coverage...)
	}
	if err == nil && snapshot == nil {
		err = fmt.Errorf("collector returned no snapshot")
	}
	if err == nil && e.snapshot != nil {
		err = coverageRegression(e.snapshot.Coverage, snapshot.Coverage)
	}
	var owned inventory.Snapshot
	if err == nil {
		// Clone once at publication; no caller-owned maps enter the shared store.
		var data []byte
		data, err = json.Marshal(snapshot)
		if err == nil {
			err = json.Unmarshal(data, &owned)
		}
	}
	if err != nil {
		e.status.State = "error"
		if e.snapshot != nil {
			e.status.State = "stale"
		}
		e.status.Error = err.Error()
	} else {
		e.snapshot = &owned
		now := time.Now().UTC()
		e.status.LastSuccess, e.status.Error, e.status.State = &now, "", "ready"
		for _, c := range snapshot.Coverage {
			if c.Status != "complete" {
				e.status.State = "partial"
				break
			}
		}
		s.revision++
	}
	s.encode()
}

// encode runs under the writer lock, or during construction.
func (s *Store) encode() {
	out := response{Revision: s.revision, Sources: map[string]Status{}}
	for _, name := range s.order {
		e := s.entries[name]
		out.Sources[name] = e.status
		if e.snapshot == nil {
			continue
		}
		if out.Snapshot == nil {
			out.Snapshot = &inventory.Snapshot{
				SchemaVersion: "teleskope.io/snapshot/v1alpha1",
				Source:        inventory.Source{Tool: "teleskope", Version: buildinfo.Version, Mode: "live/poll"},
			}
		}
		if name == "kubernetes" {
			out.Snapshot.Kubernetes = e.snapshot.Kubernetes
		} else {
			out.Snapshot.AWS, out.Snapshot.EKS = e.snapshot.AWS, e.snapshot.EKS
		}
		out.Snapshot.Coverage = append(out.Snapshot.Coverage, e.snapshot.Coverage...)
		if e.snapshot.CollectedAt.After(out.Snapshot.CollectedAt) {
			out.Snapshot.CollectedAt = e.snapshot.CollectedAt
		}
	}
	s.body, _ = json.Marshal(out)
	s.etag = fmt.Sprintf("\"%x\"", sha256.Sum256(s.body))
}

// Handler only reads the store. HTTP traffic cannot initiate collection.
func (s *Store) Handler() http.Handler {
	page := report.LiveHTML()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "read-only endpoint", http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if r.Method == http.MethodGet {
				_, _ = strings.NewReader(page).WriteTo(w)
			}
		case "/api/snapshot":
			s.mu.RLock()
			body, etag := s.body, s.etag
			s.mu.RUnlock()
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("ETag", etag)
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			if r.Method == http.MethodGet {
				_, _ = w.Write(body)
			}
		default:
			http.NotFound(w, r)
		}
	})
}
