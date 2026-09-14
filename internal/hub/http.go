package hub

import (
	"encoding/json"
	"errors"
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
			s.writeClusterSnapshot(w, r, r.URL.Query().Get("id"), opts)
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
			s.writeClusterHTML(w, r, id, opts)
		case r.URL.Path == "/cluster":
			if !allow(w, r, http.MethodGet, http.MethodHead) {
				return
			}
			s.writeClusterHTML(w, r, r.URL.Query().Get("id"), opts)
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
				s.logf(opts, "[hub] cluster snapshot export failed cluster=%s error=not_found", r.URL.Query().Get("id"))
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
				s.logf(opts, "[hub] cluster summary export failed cluster=%s error=not_found", r.URL.Query().Get("id"))
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

func (s *Store) writeClusterHTML(w http.ResponseWriter, r *http.Request, id string, opts HandlerOptions) {
	env, ok := s.Get(id)
	if !ok || env.Snapshot == nil {
		s.logf(opts, "[hub] cluster page failed cluster=%s path=%s error=not_found", id, r.URL.Path)
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

func (s *Store) writeClusterSnapshot(w http.ResponseWriter, r *http.Request, id string, opts HandlerOptions) {
	env, ok := s.Get(id)
	if !ok || env.Snapshot == nil {
		s.logf(opts, "[hub] cluster snapshot failed cluster=%s error=not_found", id)
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
	eksProjection := report.BuildEKSProjection(env.Snapshot)
	if eksProjection.Visible {
		out.EKSProjection = &eksProjection
	}
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (s *Store) handleAnalyze(w http.ResponseWriter, r *http.Request, id string, opts HandlerOptions) {
	requestID := fmt.Sprintf("%x", time.Now().UnixNano())
	env, ok := s.Get(id)
	if !ok || env.Snapshot == nil {
		s.logf(opts, "[analysis] hub cluster analysis failed id=%s cluster=%s phase=snapshot error=not_found", requestID, id)
		http.NotFound(w, r)
		return
	}
	clientReq, err := analysis.DecodeClientRequest(r.Body)
	if err != nil {
		s.logf(opts, "[analysis] hub cluster analysis failed id=%s cluster=%s phase=request_body error=%s", requestID, id, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req := analysis.ApplyClientRequest(analysis.Request{UseCase: analysis.UseCaseScan, Snapshot: env.Snapshot}, clientReq)
	s.mu.Lock()
	s.ensure()
	cache := s.analysis
	s.mu.Unlock()
	out, err := analysis.Runner{
		Analyzer:   opts.Analyzer,
		ConfigPath: opts.ConfigPath,
		Timeout:    opts.Timeout,
		Cache:      cache,
		EventSink: func(event analysis.RunEvent) {
			s.logf(opts, "[analysis] %s", event.Message)
		},
	}.Run(r.Context(), analysis.Job{
		Request:   req,
		Key:       analysis.NewCacheKey(id, env.Revision, "", req),
		RequestID: requestID,
		Operation: "hub cluster analysis",
	})
	if err != nil {
		http.Error(w, err.Error(), analysisHTTPStatus(err))
		return
	}
	s.writeAnalyzeResponse(w, out)
}

func (s *Store) writeAnalyzeResponse(w http.ResponseWriter, out analysis.RunResponse) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(out)
}

func analysisHTTPStatus(err error) int {
	var runErr *analysis.RunError
	if !errors.As(err, &runErr) {
		return http.StatusBadGateway
	}
	switch runErr.Phase {
	case "running":
		return http.StatusConflict
	case "config":
		return http.StatusInternalServerError
	default:
		return http.StatusBadGateway
	}
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
		s.logf(opts, "[hub] remote write unauthorized remote=%s", r.RemoteAddr)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	defer r.Body.Close()
	var env Envelope
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<20))
	if err := decoder.Decode(&env); err != nil {
		s.logf(opts, "[hub] remote write rejected remote=%s phase=decode error=%s", r.RemoteAddr, err)
		http.Error(w, fmt.Sprintf("decode envelope: %v", err), http.StatusBadRequest)
		return
	}
	incomingRevision := env.Revision
	accepted, stored, err := s.Put(env)
	if err != nil {
		s.logf(opts, "[hub] remote write rejected cluster=%s revision=%d phase=validate error=%s", env.Cluster.ID, incomingRevision, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if accepted {
		s.logf(opts, "[hub] remote write accepted cluster=%s name=%s provider=%s revision=%d collected_at=%s sources=%d events=%d",
			stored.Cluster.ID, stored.Cluster.Name, stored.Cluster.Provider, stored.Revision, stored.CollectedAt.Format(time.RFC3339), len(stored.Sources), len(stored.Events))
	} else {
		s.logf(opts, "[hub] remote write ignored cluster=%s revision=%d stored_revision=%d reason=stale_or_duplicate",
			stored.Cluster.ID, incomingRevision, stored.Revision)
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
