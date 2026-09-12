package hub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/report"
)

// HandlerOptions controls hub HTTP behavior.
type HandlerOptions struct {
	Token string
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
	html, err := report.HTML(env.Snapshot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method == http.MethodGet {
		_, _ = strings.NewReader(html).WriteTo(w)
	}
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
