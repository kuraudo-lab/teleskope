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
	"sync/atomic"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/advisor"
	"github.com/kuraudo-lab/teleskope/internal/analysis"
	"github.com/kuraudo-lab/teleskope/internal/buildinfo"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"github.com/kuraudo-lab/teleskope/internal/report"
)

// Source is a sequential polling task. Collect transfers ownership of its result.
type Source struct {
	Name              string
	Interval, Timeout time.Duration
	Collect           func(context.Context) (*inventory.Snapshot, error)
	Log               func(format string, args ...any)
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

// Event records a recent live-server operation for display in the web UI.
type Event struct {
	Sequence uint64    `json:"sequence"`
	At       time.Time `json:"at"`
	Source   string    `json:"source"`
	Level    string    `json:"level"`
	Message  string    `json:"message"`
}

type entry struct {
	status   Status
	snapshot *inventory.Snapshot
}

const maxEvents = 200

// Store owns all mutable state; handlers receive pre-encoded immutable responses.
type Store struct {
	mu              sync.RWMutex
	order           []string
	sources         []Source
	entries         map[string]*entry
	events          []Event
	eventSeq        uint64
	revision        uint64
	body            []byte
	etag            string
	analysisCache   *analysisCache
	analysisRunning atomic.Bool
	publishHook     func(Response)
}

// Response is the immutable live view served to browsers and remote hub writers.
type Response struct {
	Advisor  *advisor.Report     `json:"advisor,omitempty"`
	Revision uint64              `json:"revision"`
	Snapshot *inventory.Snapshot `json:"snapshot"`
	Sources  map[string]Status   `json:"sources"`
	Events   []Event             `json:"events,omitempty"`
}

type AnalyzeResponse struct {
	Analysis analysis.Result `json:"analysis"`
	Markdown string          `json:"markdown"`
}

type analysisCache struct {
	revision uint64
	response AnalyzeResponse
}

// HandlerOptions configures optional live HTTP actions.
type HandlerOptions struct {
	Analyzer   analysis.Analyzer
	ConfigPath string
	Timeout    time.Duration
	Log        func(format string, args ...any)
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
				started := time.Now().UTC()
				s.AddEvent(source.Name, "info", "refresh starting timeout=%s", source.Timeout)
				logf(source.Log, "[%s] refresh starting timeout=%s", source.Name, source.Timeout)
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
				status := s.finish(source.Name, snapshot, err, time.Now().UTC().Add(delay))
				level, message := refreshResultEvent(status, time.Since(started).Round(time.Millisecond))
				s.AddEvent(source.Name, level, "%s", message)
				logRefreshResult(source.Log, source.Name, status, time.Since(started).Round(time.Millisecond))
				if status.State == "ready" || status.State == "partial" {
					s.publishLatest()
				}
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

func logf(fn func(format string, args ...any), format string, args ...any) {
	if fn != nil {
		fn(format, args...)
	}
}

// SetPublishHook installs a non-blocking callback for newly published ready or partial data.
func (s *Store) SetPublishHook(fn func(Response)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publishHook = fn
}

// View returns a copy of the current live response.
func (s *Store) View() (Response, error) {
	s.mu.RLock()
	body := append([]byte(nil), s.body...)
	s.mu.RUnlock()
	var out Response
	if err := json.Unmarshal(body, &out); err != nil {
		return Response{}, err
	}
	return out, nil
}

func (s *Store) publishLatest() {
	s.mu.RLock()
	hook := s.publishHook
	s.mu.RUnlock()
	if hook == nil {
		return
	}
	view, err := s.View()
	if err != nil || view.Snapshot == nil {
		return
	}
	go hook(view)
}

// AddEvent appends a bounded operational event and republishes the response.
func (s *Store) AddEvent(source, level, format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventSeq++
	if level == "" {
		level = "info"
	}
	s.events = append(s.events, Event{
		Sequence: s.eventSeq,
		At:       time.Now().UTC(),
		Source:   source,
		Level:    level,
		Message:  fmt.Sprintf(format, args...),
	})
	if len(s.events) > maxEvents {
		copy(s.events, s.events[len(s.events)-maxEvents:])
		s.events = s.events[:maxEvents]
	}
	s.encode()
}

func refreshResultEvent(status Status, duration time.Duration) (string, string) {
	next := "-"
	if status.NextAttempt != nil {
		next = status.NextAttempt.Format(time.RFC3339)
	}
	switch status.State {
	case "ready":
		return "info", fmt.Sprintf("refresh published state=ready duration=%s next=%s", duration, next)
	case "partial":
		return "warn", fmt.Sprintf("refresh published state=partial duration=%s non_complete=%d next=%s", duration, nonCompleteCoverage(status.Coverage), next)
	case "stale":
		return "warn", fmt.Sprintf("refresh retained previous data state=stale duration=%s error=%s next=%s", duration, status.Error, next)
	default:
		return "error", fmt.Sprintf("refresh failed state=%s duration=%s error=%s next=%s", status.State, duration, status.Error, next)
	}
}

func logRefreshResult(fn func(format string, args ...any), name string, status Status, duration time.Duration) {
	if fn == nil {
		return
	}
	next := "-"
	if status.NextAttempt != nil {
		next = status.NextAttempt.Format(time.RFC3339)
	}
	switch status.State {
	case "ready":
		fn("[%s] refresh published state=ready duration=%s next=%s", name, duration, next)
	case "partial":
		fn("[%s] refresh published state=partial duration=%s non_complete=%d next=%s", name, duration, nonCompleteCoverage(status.Coverage), next)
	case "stale":
		fn("[%s] refresh retained previous data state=stale duration=%s error=%s next=%s", name, duration, status.Error, next)
	default:
		fn("[%s] refresh failed state=%s duration=%s error=%s next=%s", name, status.State, duration, status.Error, next)
	}
}

func nonCompleteCoverage(items []inventory.CoverageItem) int {
	count := 0
	for _, item := range items {
		if item.Status != "complete" {
			count++
		}
	}
	return count
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

func (s *Store) finish(name string, snapshot *inventory.Snapshot, err error, next time.Time) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entries[name]
	attemptErr := err
	e.status.Refreshing, e.status.NextAttempt = false, &next
	e.status.Coverage = nil
	if snapshot != nil {
		e.status.Coverage = append([]inventory.CoverageItem(nil), snapshot.Coverage...)
	}
	if err == nil && snapshot == nil {
		err = fmt.Errorf("collector returned no snapshot")
	}
	var regressionErr error
	if snapshot != nil && e.snapshot != nil {
		regressionErr = coverageRegression(e.snapshot.Coverage, snapshot.Coverage)
		if regressionErr != nil {
			err = regressionErr
		}
	}
	var owned inventory.Snapshot
	publish := snapshot != nil && regressionErr == nil
	if publish {
		// Clone once at publication; no caller-owned maps enter the shared store.
		var data []byte
		data, err = json.Marshal(snapshot)
		if err == nil {
			err = json.Unmarshal(data, &owned)
		}
		publish = err == nil
	}
	if !publish {
		e.status.State = "error"
		if e.snapshot != nil {
			e.status.State = "stale"
		}
		e.status.Error = err.Error()
	} else {
		e.snapshot = &owned
		now := time.Now().UTC()
		e.status.LastSuccess, e.status.Error, e.status.State = &now, "", "ready"
		if attemptErr != nil {
			e.status.Error = attemptErr.Error()
			e.status.State = "partial"
		}
		for _, c := range snapshot.Coverage {
			if c.Status != "complete" {
				e.status.State = "partial"
				break
			}
		}
		s.revision++
	}
	s.encode()
	return e.status
}

// encode runs under the writer lock, or during construction.
func (s *Store) encode() {
	out := Response{Revision: s.revision, Sources: map[string]Status{}}
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
	if out.Snapshot != nil {
		analysis := advisor.Analyze(out.Snapshot)
		state := "unavailable"
		if source, ok := out.Sources["kubernetes"]; ok {
			state = source.State
		}
		analysis.SetFreshness(state)
		out.Advisor = &analysis
	}
	out.Events = append([]Event(nil), s.events...)
	s.body, _ = json.Marshal(out)
	s.etag = fmt.Sprintf("\"%x\"", sha256.Sum256(s.body))
}

// Handler serves the live UI. HTTP traffic cannot initiate collection; LLM analysis is only triggered by explicit POST.
func (s *Store) Handler() http.Handler {
	return s.HandlerWithOptions(HandlerOptions{})
}

// HandlerWithOptions serves the live UI with injectable LLM analysis dependencies.
func (s *Store) HandlerWithOptions(opts HandlerOptions) http.Handler {
	page := report.LiveHTML()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		switch r.URL.Path {
		case "/":
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.Header().Set("Allow", "GET, HEAD")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if r.Method == http.MethodGet {
				_, _ = strings.NewReader(page).WriteTo(w)
			}
		case "/api/snapshot":
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.Header().Set("Allow", "GET, HEAD")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
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
		case "/api/export/snapshot.json":
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.Header().Set("Allow", "GET, HEAD")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			snapshot, err := s.exportSnapshot()
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			body, err := report.SnapshotJSON(snapshot)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", `attachment; filename="snapshot.json"`)
			if r.Method == http.MethodGet {
				_, _ = strings.NewReader(body).WriteTo(w)
			}
		case "/api/export/summary.md":
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.Header().Set("Allow", "GET, HEAD")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			snapshot, err := s.exportSnapshot()
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="summary.md"`)
			if r.Method == http.MethodGet {
				_, _ = strings.NewReader(report.Markdown(snapshot)).WriteTo(w)
			}
		case "/api/analyze":
			if r.Method != http.MethodPost {
				w.Header().Set("Allow", "POST")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			s.handleAnalyze(w, r, opts)
		default:
			http.NotFound(w, r)
		}
	})
}

func (s *Store) handleAnalyze(w http.ResponseWriter, r *http.Request, opts HandlerOptions) {
	started := time.Now()
	requestID := fmt.Sprintf("%x", started.UnixNano())
	snapshot, revision, err := s.exportSnapshotView()
	if err != nil {
		s.addAnalysisEvent(opts, "error", "LLM analysis failed id=%s phase=snapshot error=%s", requestID, err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if cached, ok := s.cachedAnalysis(revision); ok {
		s.addAnalysisEvent(opts, "info", "LLM analysis cache_hit id=%s revision=%d model=%s", requestID, revision, cached.Analysis.Model)
		s.writeAnalyzeResponse(w, cached)
		return
	}
	if !s.analysisRunning.CompareAndSwap(false, true) {
		s.addAnalysisEvent(opts, "warn", "LLM analysis rejected id=%s revision=%d reason=already_running", requestID, revision)
		http.Error(w, "analysis already running", http.StatusConflict)
		return
	}
	defer s.analysisRunning.Store(false)
	s.addAnalysisEvent(opts, "info", "LLM analysis started id=%s revision=%d", requestID, revision)
	req := analysis.Request{UseCase: analysis.UseCaseScan, Snapshot: snapshot}
	if contextBytes, err := analysis.BuildContext(req); err != nil {
		s.addAnalysisEvent(opts, "warn", "LLM analysis context sizing failed id=%s error=%s", requestID, err)
	} else {
		s.addAnalysisEvent(opts, "debug", "LLM analysis context ready id=%s revision=%d bytes=%d collectedAt=%s", requestID, revision, len(contextBytes), snapshot.CollectedAt.Format(time.RFC3339))
	}
	analyzer := opts.Analyzer
	if analyzer == nil {
		cfg, err := analysis.LoadConfig(opts.ConfigPath)
		if err != nil {
			s.addAnalysisEvent(opts, "error", "LLM analysis failed id=%s phase=config error=%s", requestID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if opts.Timeout > 0 {
			cfg.LLM.Timeout = opts.Timeout
		}
		s.addAnalysisEvent(opts, "info", "LLM analysis config loaded id=%s provider=%s model=%s timeout=%s", requestID, cfg.LLM.Provider, cfg.LLM.Model, cfg.LLM.Timeout)
		analyzer = analysis.NewOpenAICompatible(cfg.LLM)
	} else {
		s.addAnalysisEvent(opts, "debug", "LLM analysis using injected analyzer id=%s", requestID)
	}
	ctx := r.Context()
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	result, err := analyzer.Analyze(ctx, req)
	if err != nil {
		s.addAnalysisEvent(opts, "error", "LLM analysis failed id=%s phase=request duration=%s error=%s", requestID, time.Since(started).Round(time.Millisecond), err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	s.addAnalysisEvent(opts, "info", "LLM analysis completed id=%s revision=%d model=%s duration=%s", requestID, revision, result.Model, time.Since(started).Round(time.Millisecond))
	out := AnalyzeResponse{Analysis: result, Markdown: analysis.Markdown(result)}
	s.storeAnalysis(revision, out)
	s.writeAnalyzeResponse(w, out)
}

func (s *Store) addAnalysisEvent(opts HandlerOptions, level, format string, args ...any) {
	s.AddEvent("analysis", level, format, args...)
	if opts.Log != nil {
		opts.Log("[analysis] "+format, args...)
	}
}

func (s *Store) cachedAnalysis(revision uint64) (AnalyzeResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.analysisCache == nil || s.analysisCache.revision != revision {
		return AnalyzeResponse{}, false
	}
	return s.analysisCache.response, true
}

func (s *Store) storeAnalysis(revision uint64, out AnalyzeResponse) {
	s.mu.Lock()
	s.analysisCache = &analysisCache{revision: revision, response: out}
	s.mu.Unlock()
}

func (s *Store) writeAnalyzeResponse(w http.ResponseWriter, out AnalyzeResponse) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(out)
}

func (s *Store) exportSnapshot() (*inventory.Snapshot, error) {
	snapshot, _, err := s.exportSnapshotView()
	return snapshot, err
}

func (s *Store) exportSnapshotView() (*inventory.Snapshot, uint64, error) {
	s.mu.RLock()
	body := append([]byte(nil), s.body...)
	s.mu.RUnlock()

	var out Response
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, 0, err
	}
	if out.Snapshot == nil {
		return nil, out.Revision, fmt.Errorf("snapshot unavailable")
	}
	return out.Snapshot, out.Revision, nil
}
