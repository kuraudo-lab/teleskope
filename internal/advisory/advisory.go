// Package advisory serves the local migration comparison web UI.
package advisory

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	_ "embed"

	"github.com/kuraudo-lab/teleskope/internal/compare"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

const maxSnapshotUploadBytes = 128 << 20

//go:embed advisory.html
var page string

type compareResponse struct {
	Report   compare.Report `json:"report"`
	Markdown string         `json:"markdown"`
}

// Handler returns a local-only web UI for comparing two uploaded scan snapshots.
func Handler() http.Handler {
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
		case "/api/compare":
			if r.Method != http.MethodPost {
				w.Header().Set("Allow", "POST")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			handleCompare(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func handleCompare(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSnapshotUploadBytes*2)
	if err := r.ParseMultipartForm(maxSnapshotUploadBytes); err != nil {
		http.Error(w, "parse upload: "+err.Error(), http.StatusBadRequest)
		return
	}
	source, err := decodeUploadedSnapshot(r.MultipartForm, "source")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	target, err := decodeUploadedSnapshot(r.MultipartForm, "target")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	report := compare.Analyze(source, target)
	out := compareResponse{Report: report, Markdown: compare.Markdown(report)}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(out)
}

func decodeUploadedSnapshot(form *multipart.Form, field string) (*inventory.Snapshot, error) {
	if form == nil || form.File == nil || len(form.File[field]) == 0 {
		return nil, fmt.Errorf("%s snapshot is required", field)
	}
	header := form.File[field][0]
	file, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s snapshot: %w", field, err)
	}
	defer file.Close()
	return compare.DecodeSnapshot(header.Filename, file)
}
