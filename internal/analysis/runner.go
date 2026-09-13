package analysis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RunResponse is the shared response shape returned by CLI and web adapters.
type RunResponse struct {
	Analysis Result `json:"analysis"`
	Markdown string `json:"markdown"`
}

// RunEvent records one observable step in an analysis run.
type RunEvent struct {
	Level   string
	Message string
}

// CacheKey identifies an analysis result for the current simple revision cache.
type CacheKey struct {
	Subject  string
	Revision uint64
}

func (k CacheKey) valid() bool {
	return k.Subject != "" || k.Revision != 0
}

// RunCache stores completed results and rejects duplicate concurrent runs.
type RunCache struct {
	mu      sync.Mutex
	values  map[CacheKey]RunResponse
	running map[CacheKey]struct{}
}

func (c *RunCache) get(key CacheKey) (RunResponse, bool) {
	if c == nil || !key.valid() {
		return RunResponse{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	value, ok := c.values[key]
	return value, ok
}

func (c *RunCache) begin(key CacheKey) bool {
	if c == nil || !key.valid() {
		return true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running == nil {
		c.running = map[CacheKey]struct{}{}
	}
	if _, ok := c.running[key]; ok {
		return false
	}
	c.running[key] = struct{}{}
	return true
}

func (c *RunCache) finish(key CacheKey, response RunResponse, store bool) {
	if c == nil || !key.valid() {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.running, key)
	if !store {
		return
	}
	if c.values == nil {
		c.values = map[CacheKey]RunResponse{}
	}
	c.values[key] = response
}

// Runner owns analysis execution details shared by CLI, live, hub, and advisory adapters.
type Runner struct {
	Analyzer    Analyzer
	ConfigPath  string
	Timeout     time.Duration
	LoadConfig  func(string) (Config, error)
	NewAnalyzer func(LLMConfig) Analyzer
	Cache       *RunCache
	EventSink   func(RunEvent)
}

// Job describes one analysis run.
type Job struct {
	Request   Request
	Key       CacheKey
	RequestID string
	Operation string
}

// RunError classifies analysis failures for HTTP adapters.
type RunError struct {
	Phase string
	Err   error
}

func (e *RunError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *RunError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

var ErrAlreadyRunning = errors.New("analysis already running")

// Run executes a configured analysis request and returns structured analysis plus Markdown.
func (r Runner) Run(ctx context.Context, job Job) (RunResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	op := job.Operation
	if op == "" {
		op = "LLM analysis"
	}
	requestID := job.RequestID
	if requestID == "" {
		requestID = fmt.Sprintf("%x", time.Now().UnixNano())
	}
	if cached, ok := r.Cache.get(job.Key); ok {
		r.emit("info", "%s cache_hit id=%s%s model=%s", op, requestID, keyLog(job.Key), cached.Analysis.Model)
		return cached, nil
	}
	if !r.Cache.begin(job.Key) {
		r.emit("warn", "%s rejected id=%s%s reason=already_running", op, requestID, keyLog(job.Key))
		return RunResponse{}, &RunError{Phase: "running", Err: ErrAlreadyRunning}
	}
	if r.Cache != nil && job.Key.valid() {
		defer r.Cache.finish(job.Key, RunResponse{}, false)
	}

	started := time.Now()
	r.emit("info", "%s started id=%s%s", op, requestID, keyLog(job.Key))
	if contextBytes, err := BuildContext(job.Request); err != nil {
		r.emit("warn", "%s context sizing failed id=%s error=%s", op, requestID, err)
	} else {
		r.emit("debug", "%s context ready id=%s%s bytes=%d%s", op, requestID, keyLog(job.Key), len(contextBytes), collectedAtLog(job.Request))
	}

	analyzer := r.Analyzer
	if analyzer == nil {
		loadConfig := r.LoadConfig
		if loadConfig == nil {
			loadConfig = LoadConfig
		}
		cfg, err := loadConfig(r.ConfigPath)
		if err != nil {
			r.emit("error", "%s failed id=%s phase=config error=%s", op, requestID, err)
			return RunResponse{}, &RunError{Phase: "config", Err: err}
		}
		if r.Timeout > 0 {
			cfg.LLM.Timeout = r.Timeout
		}
		r.emit("info", "%s config loaded id=%s provider=%s model=%s timeout=%s", op, requestID, cfg.LLM.Provider, cfg.LLM.Model, cfg.LLM.Timeout)
		newAnalyzer := r.NewAnalyzer
		if newAnalyzer == nil {
			newAnalyzer = func(cfg LLMConfig) Analyzer { return NewOpenAICompatible(cfg) }
		}
		analyzer = newAnalyzer(cfg.LLM)
	} else {
		r.emit("debug", "%s using injected analyzer id=%s", op, requestID)
	}

	runCtx := ctx
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}
	result, err := analyzer.Analyze(runCtx, job.Request)
	if err != nil {
		r.emit("error", "%s failed id=%s phase=request duration=%s error=%s", op, requestID, time.Since(started).Round(time.Millisecond), err)
		return RunResponse{}, &RunError{Phase: "request", Err: err}
	}
	out := RunResponse{Analysis: result, Markdown: Markdown(result)}
	r.Cache.finish(job.Key, out, true)
	r.emit("info", "%s completed id=%s%s model=%s duration=%s", op, requestID, keyLog(job.Key), result.Model, time.Since(started).Round(time.Millisecond))
	return out, nil
}

func (r Runner) emit(level, format string, args ...any) {
	if r.EventSink != nil {
		r.EventSink(RunEvent{Level: level, Message: fmt.Sprintf(format, args...)})
	}
}

func keyLog(key CacheKey) string {
	if !key.valid() {
		return ""
	}
	var subject string
	if key.Subject != "" {
		subject = " subject=" + key.Subject
	}
	if key.Revision != 0 {
		return subject + fmt.Sprintf(" revision=%d", key.Revision)
	}
	return subject
}

func collectedAtLog(req Request) string {
	if req.Snapshot != nil && !req.Snapshot.CollectedAt.IsZero() {
		return " collectedAt=" + req.Snapshot.CollectedAt.Format(time.RFC3339)
	}
	return ""
}
