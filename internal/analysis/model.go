// Package analysis turns Teleskope facts into optional LLM-backed narrative analysis.
package analysis

import (
	"context"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/compare"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

const SchemaVersion = "teleskope.io/llm-analysis/v1alpha1"

type UseCase string

const (
	UseCaseScan     UseCase = "scan"
	UseCaseCompare  UseCase = "compare"
	UseCaseAdvisory UseCase = "advisory"
)

type Analyzer interface {
	Analyze(ctx context.Context, req Request) (Result, error)
}

type Request struct {
	UseCase       UseCase             `json:"useCase"`
	Snapshot      *inventory.Snapshot `json:"snapshot,omitempty"`
	Source        *inventory.Snapshot `json:"source,omitempty"`
	Target        *inventory.Snapshot `json:"target,omitempty"`
	CompareReport *compare.Report     `json:"compareReport,omitempty"`
	WebSearch     bool                `json:"webSearch,omitempty"`
}

type Result struct {
	SchemaVersion string     `json:"schemaVersion"`
	GeneratedAt   time.Time  `json:"generatedAt"`
	UseCase       UseCase    `json:"useCase"`
	Provider      string     `json:"provider,omitempty"`
	Model         string     `json:"model,omitempty"`
	PromptVersion string     `json:"promptVersion"`
	WebSearch     bool       `json:"webSearch,omitempty"`
	Summary       string     `json:"summary"`
	Sections      []Section  `json:"sections,omitempty"`
	Limitations   []string   `json:"limitations,omitempty"`
	Citations     []Citation `json:"citations,omitempty"`
}

type Section struct {
	Title string `json:"title"`
	Items []Item `json:"items,omitempty"`
}

type Item struct {
	Severity       string                `json:"severity,omitempty"`
	Summary        string                `json:"summary"`
	Detail         string                `json:"detail,omitempty"`
	Recommendation string                `json:"recommendation,omitempty"`
	Resources      []inventory.ObjectRef `json:"resources,omitempty"`
	Basis          string                `json:"basis,omitempty"`
	Confidence     string                `json:"confidence,omitempty"`
	Evidence       []string              `json:"evidence,omitempty"`
	Citations      []Citation            `json:"citations,omitempty"`
}

type Citation struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

func normalizeResult(r Result, req Request, provider, model, promptVersion string) Result {
	if r.SchemaVersion == "" {
		r.SchemaVersion = SchemaVersion
	}
	if r.GeneratedAt.IsZero() {
		r.GeneratedAt = time.Now().UTC()
	}
	if r.UseCase == "" {
		r.UseCase = req.UseCase
	}
	if r.Provider == "" {
		r.Provider = provider
	}
	if r.Model == "" {
		r.Model = model
	}
	if r.PromptVersion == "" {
		r.PromptVersion = promptVersion
	}
	r.WebSearch = req.WebSearch
	return r
}
