package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"NexusDesk/internal/storage"
)

func TestChatCallsOpenAICompatibleChatCompletionsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("missing auth header")
		}

		var body chatCompletionRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}
		if body.Model != "test-model" {
			t.Fatalf("unexpected model: %s", body.Model)
		}
		if len(body.Messages) != 2 {
			t.Fatalf("expected two messages, got %d", len(body.Messages))
		}
		if body.Messages[1].Content == "Explain it" {
			t.Fatal("expected workspace context in user prompt")
		}

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Here is the answer."}}]}`))
	}))
	defer server.Close()

	client := NewClient()
	result, err := client.Chat(context.Background(), storage.LLMSettings{
		BaseURL: server.URL + "/v1",
		Model:   "test-model",
		APIKey:  "secret",
	}, ChatRequest{
		Prompt:         "Explain it",
		ContextRelPath: "README.md",
		ContextContent: "hello",
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if result.Message != "Here is the answer." {
		t.Fatalf("unexpected response: %s", result.Message)
	}
	if result.ContextRelPath != "README.md" {
		t.Fatalf("unexpected context path: %s", result.ContextRelPath)
	}
}

func TestChatRequiresConfiguredModel(t *testing.T) {
	client := NewClient()

	_, err := client.Chat(context.Background(), storage.LLMSettings{
		BaseURL: "https://example.test/v1",
	}, ChatRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected missing model error")
	}
}
