package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/analysis"
	"github.com/kuraudo-lab/teleskope/internal/live"
	"github.com/kuraudo-lab/teleskope/internal/report"
)

// HandlerOptions controls hub HTTP behavior.
type HandlerOptions struct {
	Token      string
	Analyzer   analysis.Analyzer
	ConfigPath string
	Timeout    time.Duration
	Log        func(format string, args ...any)
}

// Handler serves the fleet API and embedded UI.
func (s *Store) Handler(opts HandlerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		switch {
		case r.URL.Path == "/":
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if r.Method == http.MethodGet {
				_, _ = strings.NewReader(HubHTML()).WriteTo(w)
			}
		case r.URL.Path == "/api/clusters":
			if r.Method == http.MethodPost {
				s.handlePostEnvelope(w, r, opts)
				return
			}
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			body, etag := s.encodedFleet()
			if body == nil {
				body, _ = json.Marshal(FleetResponse{})
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("ETag", etag)
			if r.Header.Get("If-None-Match") == etag && etag != "" {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			if r.Method == http.MethodGet {
				_, _ = w.Write(body)
			}
		case strings.HasPrefix(r.URL.Path, "/api/clusters/"):
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			id := strings.TrimPrefix(r.URL.Path, "/api/clusters/")
			s.writeEnvelope(w, r, id)
		case r.URL.Path == "/api/cluster":
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			s.writeEnvelope(w, r, r.URL.Query().Get("id"))
		case r.URL.Path == "/api/cluster/snapshot":
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			s.writeClusterSnapshot(w, r, r.URL.Query().Get("id"))
		case r.URL.Path == "/api/cluster/analyze":
			if !allow(w, r, http.MethodPost) {
				return
			}
			s.handleAnalyze(w, r, r.URL.Query().Get("id"), opts)
		case strings.HasPrefix(r.URL.Path, "/clusters/"):
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/clusters/"), "/")
			s.writeClusterHTML(w, r, id)
		case r.URL.Path == "/cluster":
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			s.writeClusterHTML(w, r, r.URL.Query().Get("id"))
		case r.URL.Path == "/api/export/fleet.json":
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", `attachment; filename="teleskope-fleet.json"`)
			if r.Method == http.MethodGet {
				encoder := json.NewEncoder(w)
				encoder.SetIndent("", "  ")
				_ = encoder.Encode(s.Fleet())
			}
		case r.URL.Path == "/api/export/summary.md":
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="teleskope-fleet-summary.md"`)
			if r.Method == http.MethodGet {
				_, _ = strings.NewReader(Markdown(s.Fleet())).WriteTo(w)
			}
		case r.URL.Path == "/api/cluster/export/snapshot.json":
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			env, ok := s.Get(r.URL.Query().Get("id"))
			if !ok || env.Snapshot == nil {
				http.NotFound(w, r)
				return
			}
			body, err := report.SnapshotJSON(env.Snapshot)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", `attachment; filename="snapshot.json"`)
			if r.Method == http.MethodGet {
				_, _ = strings.NewReader(body).WriteTo(w)
			}
		case r.URL.Path == "/api/cluster/export/summary.md":
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			env, ok := s.Get(r.URL.Query().Get("id"))
			if !ok || env.Snapshot == nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="summary.md"`)
			if r.Method == http.MethodGet {
				_, _ = strings.NewReader(report.Markdown(env.Snapshot)).WriteTo(w)
			}
		default:
			http.NotFound(w, r)
		}
	})
}

func (s *Store) writeEnvelope(w http.ResponseWriter, r *http.Request, id string) {
	env, ok := s.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(env)
	}
}

func (s *Store) writeClusterHTML(w http.ResponseWriter, r *http.Request, id string) {
	env, ok := s.Get(id)
	if !ok || env.Snapshot == nil {
		http.NotFound(w, r)
		return
	}
	query := "?id=" + urlQueryEscape(id)
	html := report.LiveHTMLWithOptions(report.LiveHTMLOptions{
		SnapshotPath:       "/api/cluster/snapshot" + query,
		AnalyzePath:        "/api/cluster/analyze" + query,
		ExportSnapshotPath: "/api/cluster/export/snapshot.json" + query,
		ExportSummaryPath:  "/api/cluster/export/summary.md" + query,
	})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method == http.MethodGet {
		_, _ = strings.NewReader(html).WriteTo(w)
	}
}

func (s *Store) writeClusterSnapshot(w http.ResponseWriter, r *http.Request, id string) {
	env, ok := s.Get(id)
	if !ok || env.Snapshot == nil {
		http.NotFound(w, r)
		return
	}
	out := live.Response{
		Advisor:  env.Advisor,
		Revision: env.Revision,
		Snapshot: env.Snapshot,
		Sources:  env.Sources,
		Events:   env.Events,
	}
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (s *Store) handleAnalyze(w http.ResponseWriter, r *http.Request, id string, opts HandlerOptions) {
	started := time.Now()
	requestID := fmt.Sprintf("%x", started.UnixNano())
	env, ok := s.Get(id)
	if !ok || env.Snapshot == nil {
		http.NotFound(w, r)
		return
	}
	if cached, ok := s.cachedAnalysis(id, env.Revision); ok {
		s.logf(opts, "[analysis] hub cluster analysis cache_hit id=%s cluster=%s revision=%d model=%s", requestID, id, env.Revision, cached.Analysis.Model)
		s.writeAnalyzeResponse(w, cached)
		return
	}
	if !s.running.CompareAndSwap(false, true) {
		s.logf(opts, "[analysis] hub cluster analysis rejected id=%s cluster=%s revision=%d reason=already_running", requestID, id, env.Revision)
		http.Error(w, "analysis already running", http.StatusConflict)
		return
	}
	defer s.running.Store(false)
	req := analysis.Request{UseCase: analysis.UseCaseScan, Snapshot: env.Snapshot}
	if contextBytes, err := analysis.BuildContext(req); err == nil {
		s.logf(opts, "[analysis] hub cluster analysis context ready id=%s cluster=%s revision=%d bytes=%d", requestID, id, env.Revision, len(contextBytes))
	}
	analyzer := opts.Analyzer
	if analyzer == nil {
		cfg, err := analysis.LoadConfig(opts.ConfigPath)
		if err != nil {
			s.logf(opts, "[analysis] hub cluster analysis failed id=%s cluster=%s phase=config error=%s", requestID, id, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if opts.Timeout > 0 {
			cfg.LLM.Timeout = opts.Timeout
		}
		analyzer = analysis.NewOpenAICompatible(cfg.LLM)
	}
	ctx := r.Context()
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	result, err := analyzer.Analyze(ctx, req)
	if err != nil {
		s.logf(opts, "[analysis] hub cluster analysis failed id=%s cluster=%s phase=request duration=%s error=%s", requestID, id, time.Since(started).Round(time.Millisecond), err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	out := AnalyzeResponse{Analysis: result, Markdown: analysis.Markdown(result)}
	s.storeAnalysis(id, env.Revision, out)
	s.logf(opts, "[analysis] hub cluster analysis completed id=%s cluster=%s revision=%d model=%s duration=%s", requestID, id, env.Revision, result.Model, time.Since(started).Round(time.Millisecond))
	s.writeAnalyzeResponse(w, out)
}

func (s *Store) cachedAnalysis(id string, revision uint64) (AnalyzeResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cached, ok := s.analysis[id]
	if !ok || cached.revision != revision {
		return AnalyzeResponse{}, false
	}
	return cached.response, true
}

func (s *Store) storeAnalysis(id string, revision uint64, out AnalyzeResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.analysis == nil {
		s.analysis = map[string]analysisEntry{}
	}
	s.analysis[id] = analysisEntry{revision: revision, response: out}
}

func (s *Store) writeAnalyzeResponse(w http.ResponseWriter, out AnalyzeResponse) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(out)
}

func (s *Store) logf(opts HandlerOptions, format string, args ...any) {
	if opts.Log != nil {
		opts.Log(format, args...)
	}
}

func urlQueryEscape(value string) string {
	return url.QueryEscape(value)
}

func (s *Store) handlePostEnvelope(w http.ResponseWriter, r *http.Request, opts HandlerOptions) {
	if opts.Token != "" && r.Header.Get("Authorization") != "Bearer "+opts.Token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	defer r.Body.Close()
	var env Envelope
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<20))
	if err := decoder.Decode(&env); err != nil {
		http.Error(w, fmt.Sprintf("decode envelope: %v", err), http.StatusBadRequest)
		return
	}
	accepted, stored, err := s.Put(env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if accepted {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(struct {
		Accepted bool   `json:"accepted"`
		Cluster  string `json:"cluster"`
		Revision uint64 `json:"revision"`
	}{Accepted: accepted, Cluster: stored.Cluster.ID, Revision: stored.Revision})
}

func allow(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

// Markdown renders a compact fleet report compatible with existing markdown exports.
func Markdown(fleet FleetResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Teleskope fleet summary\n\n")
	fmt.Fprintf(&b, "- Hub revision: `%d`\n", fleet.Revision)
	fmt.Fprintf(&b, "- Clusters: `%d`\n", len(fleet.Clusters))
	fmt.Fprintf(&b, "- Generated at: `%s`\n\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "| Cluster | Provider | Region | State | Revision | Collected | Nodes | Workloads | Pods | Images |\n")
	fmt.Fprintf(&b, "| --- | --- | --- | --- | ---: | --- | ---: | ---: | ---: | ---: |\n")
	for _, cluster := range fleet.Clusters {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %d | %s | %d | %d | %d | %d |\n",
			mdCell(cluster.Cluster.Name),
			mdCell(cluster.Cluster.Provider),
			mdCell(cluster.Cluster.Region),
			mdCell(cluster.State),
			cluster.Revision,
			mdCell(cluster.CollectedAt.Format(time.RFC3339)),
			cluster.Nodes,
			cluster.Workloads,
			cluster.Pods,
			cluster.Images,
		)
	}
	if len(fleet.Events) > 0 {
		fmt.Fprintf(&b, "\n## Recent events\n\n")
		fmt.Fprintf(&b, "| Time | Cluster | Source | Level | Message |\n")
		fmt.Fprintf(&b, "| --- | --- | --- | --- | --- |\n")
		for _, event := range fleet.Events {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				mdCell(event.Event.At.Format(time.RFC3339)),
				mdCell(event.Cluster.Name),
				mdCell(event.Event.Source),
				mdCell(event.Event.Level),
				mdCell(event.Event.Message),
			)
		}
	}
	return b.String()
}

func mdCell(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	if value == "" {
		return "-"
	}
	return value
}
