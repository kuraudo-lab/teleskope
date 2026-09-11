// Package advisory serves the local migration comparison web UI.
package advisory

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	_ "embed"

	"github.com/kuraudo-lab/teleskope/internal/analysis"
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

type analyzeResponse struct {
	Analysis analysis.Result `json:"analysis"`
	Markdown string          `json:"markdown"`
}

// Options configures the advisory HTTP handler.
type Options struct {
	Analyzer   analysis.Analyzer
	ConfigPath string
	Timeout    time.Duration
}

// Handler returns a local-only web UI for comparing two uploaded scan snapshots.
func Handler() http.Handler {
	return HandlerWithOptions(Options{})
}

// HandlerWithOptions returns an advisory handler with injectable analysis dependencies.
func HandlerWithOptions(opts Options) http.Handler {
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
		case "/api/analyze":
			if r.Method != http.MethodPost {
				w.Header().Set("Allow", "POST")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			handleAnalyze(w, r, opts)
		default:
			http.NotFound(w, r)
		}
	})
}

func handleAnalyze(w http.ResponseWriter, r *http.Request, opts Options) {
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
	req := analysis.Request{UseCase: analysis.UseCaseAdvisory, Source: source, Target: target, CompareReport: &report}
	analyzer := opts.Analyzer
	if analyzer == nil {
		cfg, err := analysis.LoadConfig(opts.ConfigPath)
		if err != nil {
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
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	out := analyzeResponse{Analysis: result, Markdown: analysis.Markdown(result)}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(out)
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
