package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAICompatible struct {
	BaseURL    string
	APIKey     string
	Model      string
	Timeout    time.Duration
	JSONMode   bool
	HTTPClient *http.Client
}

func NewOpenAICompatible(cfg LLMConfig) *OpenAICompatible {
	client := http.DefaultClient
	if cfg.Timeout > 0 {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &OpenAICompatible{BaseURL: strings.TrimRight(cfg.BaseURL, "/"), APIKey: cfg.APIKey, Model: cfg.Model, Timeout: cfg.Timeout, JSONMode: cfg.UseJSONMode(), HTTPClient: client}
}

func (a *OpenAICompatible) Analyze(ctx context.Context, req Request) (Result, error) {
	prompt, err := promptFor(req.UseCase)
	if err != nil {
		return Result{}, err
	}
	modelInput, err := BuildContext(req)
	if err != nil {
		return Result{}, err
	}
	payload := chatCompletionRequest{
		Model: a.Model,
		Messages: []chatMessage{
			{Role: "system", Content: prompt},
			{Role: "user", Content: string(modelInput)},
		},
		Temperature: ptrFloat64(0.2),
	}
	if a.JSONMode {
		payload.ResponseFormat = &responseFormat{Type: "json_object"}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return Result{}, err
	}
	url := strings.TrimRight(a.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return Result{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if a.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)
	}
	client := a.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return Result{}, fmt.Errorf("call llm provider: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		if ctx.Err() != nil {
			return Result{}, fmt.Errorf("read llm response timed out after %s: %w", a.Timeout, ctx.Err())
		}
		return Result{}, fmt.Errorf("read llm response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		if resp.StatusCode == http.StatusBadRequest && a.JSONMode && looksLikeJSONModeError(message) {
			return Result{}, fmt.Errorf("llm provider returned %s: %s (try setting llm.json_mode: false for providers that do not support OpenAI JSON mode)", resp.Status, message)
		}
		return Result{}, fmt.Errorf("llm provider returned %s: %s", resp.Status, message)
	}
	var chatResp chatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return Result{}, fmt.Errorf("decode llm response: %w", err)
	}
	if len(chatResp.Choices) == 0 || strings.TrimSpace(chatResp.Choices[0].Message.Content) == "" {
		return Result{}, fmt.Errorf("llm response did not include message content")
	}
	result, err := parseAnalysisResult(chatResp.Choices[0].Message.Content)
	if err != nil {
		if !a.JSONMode {
			return normalizeResult(textResult(chatResp.Choices[0].Message.Content), req, "openai-compatible", a.Model, PromptVersion), nil
		}
		return Result{}, fmt.Errorf("decode llm analysis JSON: %w", err)
	}
	return normalizeResult(result, req, "openai-compatible", a.Model, PromptVersion), nil
}

type chatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    *float64        `json:"temperature,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func ptrFloat64(v float64) *float64 { return &v }

func textResult(content string) Result {
	content = strings.TrimSpace(stripJSONFence(content))
	if content == "" {
		content = "LLM returned an empty text response."
	}
	return Result{
		Summary: content,
		Sections: []Section{{
			Title: "Text response",
			Items: []Item{{
				Severity:   "info",
				Summary:    content,
				Basis:      "provider text response",
				Confidence: "unknown",
			}},
		}},
		Limitations: []string{"Provider response was not valid JSON; json_mode is disabled, so Teleskope preserved it as plain text."},
	}
}

func looksLikeJSONModeError(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "response_format") || strings.Contains(message, "json") || strings.Contains(message, "missing field name")
}

func parseAnalysisResult(value string) (Result, error) {
	candidates := jsonContentCandidates(value)
	var lastErr error
	for _, candidate := range candidates {
		result, err := parseAnalysisResultCandidate(candidate, 0)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("empty response")
	}
	return Result{}, lastErr
}

func parseAnalysisResultCandidate(value string, depth int) (Result, error) {
	value = strings.TrimSpace(stripJSONFence(value))
	if value == "" {
		return Result{}, fmt.Errorf("empty response")
	}
	var result Result
	if err := json.Unmarshal([]byte(value), &result); err == nil {
		return result, nil
	} else if depth >= 3 {
		return Result{}, err
	}
	var nested string
	if err := json.Unmarshal([]byte(value), &nested); err == nil {
		return parseAnalysisResultCandidate(nested, depth+1)
	}
	return Result{}, fmt.Errorf("invalid JSON analysis response")
}

func jsonContentCandidates(value string) []string {
	var candidates []string
	seen := map[string]struct{}{}
	add := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}
	add(value)
	add(stripJSONFence(value))
	for _, candidate := range fencedJSONContents(value) {
		add(candidate)
	}
	for _, candidate := range jsonObjectContents(value) {
		add(candidate)
	}
	return candidates
}

func fencedJSONContents(value string) []string {
	var candidates []string
	rest := value
	for {
		start := strings.Index(rest, "```")
		if start < 0 {
			break
		}
		rest = rest[start+3:]
		end := strings.Index(rest, "```")
		if end < 0 {
			break
		}
		candidate := rest[:end]
		candidate = strings.TrimSpace(candidate)
		if strings.HasPrefix(strings.ToLower(candidate), "json") {
			candidate = strings.TrimSpace(candidate[len("json"):])
		}
		candidates = append(candidates, candidate)
		rest = rest[end+3:]
	}
	return candidates
}

func jsonObjectContents(value string) []string {
	var candidates []string
	start := -1
	depth := 0
	inString := false
	escaped := false
	for i, r := range value {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch r {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		switch r {
		case '"':
			inString = true
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				candidate := strings.TrimSpace(value[start : i+1])
				if json.Valid([]byte(candidate)) {
					candidates = append(candidates, candidate)
				}
				start = -1
			}
		}
	}
	return candidates
}

func stripJSONFence(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "```") {
		value = strings.TrimPrefix(value, "```")
		value = strings.TrimSpace(value)
		value = strings.TrimPrefix(value, "json")
		value = strings.TrimSpace(value)
		value = strings.TrimSuffix(value, "```")
		value = strings.TrimSpace(value)
	}
	return value
}
