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
	HTTPClient *http.Client
}

func NewOpenAICompatible(cfg LLMConfig) *OpenAICompatible {
	client := http.DefaultClient
	if cfg.Timeout > 0 {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &OpenAICompatible{BaseURL: strings.TrimRight(cfg.BaseURL, "/"), APIKey: cfg.APIKey, Model: cfg.Model, Timeout: cfg.Timeout, HTTPClient: client}
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
		Temperature:    ptrFloat64(0.2),
		ResponseFormat: &responseFormat{Type: "json_object"},
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
		return Result{}, fmt.Errorf("llm provider returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var chatResp chatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return Result{}, fmt.Errorf("decode llm response: %w", err)
	}
	if len(chatResp.Choices) == 0 || strings.TrimSpace(chatResp.Choices[0].Message.Content) == "" {
		return Result{}, fmt.Errorf("llm response did not include message content")
	}
	var result Result
	content := stripJSONFence(chatResp.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(content), &result); err != nil {
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
