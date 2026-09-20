package analysis

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ClientRequest is the browser-provided part of an analysis request.
type ClientRequest struct {
	Revision     uint64    `json:"revision,omitempty"`
	WebSearch    bool      `json:"webSearch,omitempty"`
	Scope        Scope     `json:"scope,omitempty"`
	CustomPrompt string    `json:"customPrompt,omitempty"`
	Conversation []Message `json:"conversation,omitempty"`
}

// ValidateRevision rejects analysis of a browser view that is older than the
// server-owned snapshot. A zero revision preserves compatibility with callers
// that predate revision-aware scoped analysis.
func (r ClientRequest) ValidateRevision(current uint64) error {
	if r.Revision != 0 && r.Revision != current {
		return fmt.Errorf("analysis revision %d is stale; current revision is %d", r.Revision, current)
	}
	return nil
}

// DecodeClientRequest reads an optional JSON analysis request body.
func DecodeClientRequest(r io.Reader) (ClientRequest, error) {
	if r == nil {
		return ClientRequest{}, nil
	}
	data, err := io.ReadAll(io.LimitReader(r, 1<<20))
	if err != nil {
		return ClientRequest{}, fmt.Errorf("read analysis request: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return ClientRequest{}, nil
	}
	var req ClientRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return ClientRequest{}, fmt.Errorf("decode analysis request: %w", err)
	}
	return req, nil
}

// ApplyClientRequest combines trusted server-side facts with browser analysis options.
func ApplyClientRequest(req Request, client ClientRequest) Request {
	req.WebSearch = client.WebSearch
	req.Scope = cleanScope(client.Scope)
	req.CustomPrompt = strings.TrimSpace(client.CustomPrompt)
	req.Conversation = cleanMessages(client.Conversation)
	return req
}

func cleanMessages(messages []Message) []Message {
	out := make([]Message, 0, len(messages))
	for _, message := range messages {
		role := strings.TrimSpace(message.Role)
		if role != "assistant" && role != "user" {
			role = "user"
		}
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		out = append(out, Message{Role: role, Content: content})
	}
	return out
}
