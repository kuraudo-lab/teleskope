package analysis

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

type runnerFakeAnalyzer struct {
	calls int
	req   Request
	err   error
	check func(context.Context)
}

func (f *runnerFakeAnalyzer) Analyze(ctx context.Context, req Request) (Result, error) {
	f.calls++
	f.req = req
	if f.check != nil {
		f.check(ctx)
	}
	if f.err != nil {
		return Result{}, f.err
	}
	return Result{
		UseCase:       req.UseCase,
		Model:         "fake-model",
		PromptVersion: "test",
		Summary:       "analysis ready",
		Sections:      []Section{{Title: "Findings", Items: []Item{{Summary: "workload is reviewable", Severity: "info"}}}},
	}, nil
}

func TestRunnerUsesInjectedAnalyzerCachesAndReturnsMarkdown(t *testing.T) {
	fake := &runnerFakeAnalyzer{}
	var events []RunEvent
	runner := Runner{
		Analyzer: fake,
		Cache:    &RunCache{},
		EventSink: func(event RunEvent) {
			events = append(events, event)
		},
	}
	req := Request{UseCase: UseCaseScan, Snapshot: &inventory.Snapshot{CollectedAt: time.Date(2026, 9, 13, 1, 2, 3, 0, time.UTC)}}
	job := Job{Request: req, Key: CacheKey{Subject: "live", Revision: 7}, RequestID: "abc"}

	first, err := runner.Run(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 || fake.req.UseCase != UseCaseScan {
		t.Fatalf("calls=%d req=%+v", fake.calls, fake.req)
	}
	if !strings.Contains(first.Markdown, "# Teleskope LLM analysis") || !strings.Contains(first.Markdown, "analysis ready") {
		t.Fatalf("markdown = %q", first.Markdown)
	}

	second, err := runner.Run(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 {
		t.Fatalf("calls=%d, want cache hit to avoid second analyzer call", fake.calls)
	}
	if second.Analysis.Summary != first.Analysis.Summary {
		t.Fatalf("cached response = %+v, want %+v", second, first)
	}
	if !containsRunEvent(events, "context ready") || !containsRunEvent(events, "cache_hit") || !containsRunEvent(events, "using injected analyzer") {
		t.Fatalf("events = %#v", events)
	}
}

func TestRunnerClassifiesProviderErrors(t *testing.T) {
	wantErr := errors.New("provider down")
	runner := Runner{Analyzer: &runnerFakeAnalyzer{err: wantErr}, Cache: &RunCache{}}
	_, err := runner.Run(context.Background(), Job{
		Request:   Request{UseCase: UseCaseScan, Snapshot: &inventory.Snapshot{}},
		Key:       CacheKey{Subject: "live", Revision: 1},
		RequestID: "err",
	})
	var runErr *RunError
	if !errors.As(err, &runErr) || runErr.Phase != "request" || !errors.Is(runErr, wantErr) {
		t.Fatalf("err = %#v", err)
	}
}

func TestRunnerAppliesTimeoutOverrideToConfigAndContext(t *testing.T) {
	override := 25 * time.Millisecond
	var gotConfig LLMConfig
	fake := &runnerFakeAnalyzer{check: func(ctx context.Context) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected runner context deadline")
		}
		if remaining := time.Until(deadline); remaining <= 0 || remaining > time.Second {
			t.Fatalf("deadline remaining = %s", remaining)
		}
	}}
	runner := Runner{
		Timeout: override,
		LoadConfig: func(path string) (Config, error) {
			if path != "custom.yaml" {
				t.Fatalf("config path = %q", path)
			}
			return Config{LLM: LLMConfig{Provider: "openai-compatible", BaseURL: "https://example.invalid/v1", APIKey: "secret", Model: "model", Timeout: time.Minute}}, nil
		},
		NewAnalyzer: func(cfg LLMConfig) Analyzer {
			gotConfig = cfg
			return fake
		},
		ConfigPath: "custom.yaml",
	}

	_, err := runner.Run(context.Background(), Job{Request: Request{UseCase: UseCaseScan, Snapshot: &inventory.Snapshot{}}})
	if err != nil {
		t.Fatal(err)
	}
	if gotConfig.Timeout != override {
		t.Fatalf("timeout = %s, want %s", gotConfig.Timeout, override)
	}
	if fake.calls != 1 {
		t.Fatalf("calls = %d", fake.calls)
	}
}

func containsRunEvent(events []RunEvent, want string) bool {
	for _, event := range events {
		if strings.Contains(event.Message, want) {
			return true
		}
	}
	return false
}
