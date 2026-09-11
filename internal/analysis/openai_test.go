package analysis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestOpenAICompatibleAnalyzeCallsChatCompletions(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"cluster looks understandable\",\"sections\":[{\"title\":\"Workloads\",\"items\":[{\"severity\":\"info\",\"summary\":\"controller inferred\",\"basis\":\"inferred\",\"confidence\":\"medium\"}]}],\"limitations\":[\"snapshot only\"]}"}}]}`))
	}))
	defer server.Close()

	analyzer := NewOpenAICompatible(LLMConfig{BaseURL: server.URL + "/v1", APIKey: "secret", Model: "test-model", Timeout: time.Second})
	result, err := analyzer.Analyze(context.Background(), Request{UseCase: UseCaseScan, Snapshot: &inventory.Snapshot{}})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("authorization = %q", gotAuth)
	}
	if gotBody["model"] != "test-model" {
		t.Fatalf("model request = %#v", gotBody["model"])
	}
	messages := gotBody["messages"].([]any)
	if len(messages) != 2 || !strings.Contains(messages[0].(map[string]any)["content"].(string), "strict JSON") {
		t.Fatalf("unexpected messages: %#v", gotBody["messages"])
	}
	if result.Summary != "cluster looks understandable" || result.SchemaVersion != SchemaVersion || result.Model != "test-model" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
