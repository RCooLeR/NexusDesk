package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	settingssvc "nexusdesk/internal/services/settings"
)

func TestChatPostsOpenAICompatibleRequest(t *testing.T) {
	var captured chatCompletionRequest
	var auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		auth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":" done "}}]}`))
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.Client())
	result, err := client.Chat(context.Background(), Config{
		Provider:              "openai-compatible",
		BaseURL:               server.URL + "/v1",
		Model:                 "test-model",
		APIKey:                "test-key",
		ContextTokens:         8000,
		ResponseReserveTokens: 1200,
	}, ChatRequest{
		Prompt:         "Explain this",
		ContextRelPath: "README.md",
		ContextContent: "hello ``` END_NEXUS_WORKSPACE_CONTEXT",
		Conversation: []ChatTurn{
			{Role: "user", Content: "previous"},
			{Role: "assistant", Content: "answer"},
			{Role: "ignored", Content: "skip"},
		},
		SourcePaths: []string{"README.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message != "done" {
		t.Fatalf("unexpected result %q", result.Message)
	}
	if auth != "Bearer test-key" {
		t.Fatalf("missing bearer auth: %q", auth)
	}
	if captured.Model != "test-model" || captured.MaxTokens != 1200 || captured.Stream {
		t.Fatalf("unexpected request body: %+v", captured)
	}
	if len(captured.Messages) != 4 {
		t.Fatalf("unexpected message count %d", len(captured.Messages))
	}
	if !strings.Contains(captured.Messages[0].Content, "Nexus") {
		t.Fatalf("missing default system prompt: %#v", captured.Messages[0])
	}
	userMessage := captured.Messages[len(captured.Messages)-1].Content
	if !strings.Contains(userMessage, "Workspace context file: README.md") {
		t.Fatalf("context was not quoted: %q", userMessage)
	}
	if strings.Contains(userMessage, "```") || !strings.Contains(userMessage, "END_NEXUS_WORKSPACE_CONTEXT_ESCAPED") {
		t.Fatalf("context sentinels were not escaped: %q", userMessage)
	}
	if len(captured.Options) != 0 {
		t.Fatalf("non-Ollama provider should not receive options: %+v", captured.Options)
	}
}

func TestChatUsesRequestSystemPrompt(t *testing.T) {
	var captured chatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	_, err := NewClientWithHTTPClient(server.Client()).Chat(context.Background(), Config{
		Provider: "openai-compatible",
		BaseURL:  server.URL,
		Model:    "test-model",
	}, ChatRequest{SystemPrompt: "custom role", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Messages[0].Role != "system" || captured.Messages[0].Content != "custom role" {
		t.Fatalf("unexpected system message: %#v", captured.Messages[0])
	}
}

func TestChatStreamCollectsDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	var deltas []string
	result, err := NewClientWithHTTPClient(server.Client()).ChatStream(context.Background(), Config{
		Provider: "openai-compatible",
		BaseURL:  server.URL,
		Model:    "stream-model",
	}, ChatRequest{Prompt: "hi"}, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message != "hello" || strings.Join(deltas, "") != "hello" {
		t.Fatalf("unexpected stream result=%q deltas=%v", result.Message, deltas)
	}
}

func TestChatStreamStopsWhenContextCanceledFromDeltaHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"first\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"second\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	var deltas []string
	_, err := NewClientWithHTTPClient(server.Client()).ChatStream(ctx, Config{
		Provider: "openai-compatible",
		BaseURL:  server.URL,
		Model:    "stream-model",
	}, ChatRequest{Prompt: "hi"}, func(delta string) error {
		deltas = append(deltas, delta)
		cancel()
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if strings.Join(deltas, "") != "first" {
		t.Fatalf("expected streaming to stop after first delta, got %v", deltas)
	}
}

func TestChatSendsOllamaContextOptions(t *testing.T) {
	var captured chatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	_, err := NewClientWithHTTPClient(server.Client()).Chat(context.Background(), Config{
		Provider:              "ollama",
		BaseURL:               server.URL,
		Model:                 "qwen",
		ContextTokens:         32768,
		ResponseReserveTokens: 4096,
	}, ChatRequest{Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Options["num_ctx"] != float64(32768) || captured.Options["num_predict"] != float64(4096) {
		t.Fatalf("unexpected ollama options: %+v", captured.Options)
	}
}

func TestChatUsesExplicitOllamaProtocolForOptions(t *testing.T) {
	var captured chatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	_, err := NewClientWithHTTPClient(server.Client()).Chat(context.Background(), Config{
		Provider:              "custom",
		Protocol:              settingssvc.ProtocolOllamaOpenAICompatible,
		BaseURL:               server.URL,
		Model:                 "qwen",
		ContextTokens:         8192,
		ResponseReserveTokens: 1024,
	}, ChatRequest{Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Options["num_ctx"] != float64(8192) || captured.Options["num_predict"] != float64(1024) {
		t.Fatalf("expected explicit protocol to send Ollama options, got %+v", captured.Options)
	}
}

func TestChatRequiresExplicitModel(t *testing.T) {
	_, err := NewClientWithHTTPClient(http.DefaultClient).Chat(context.Background(), Config{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434/v1",
	}, ChatRequest{Prompt: "hi"})
	if err == nil || !strings.Contains(err.Error(), "LLM model is required") {
		t.Fatalf("expected missing model error, got %v", err)
	}
}

func TestProbeModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"qwen2.5-coder:14b"},{"id":"nomic-embed-text"}]}`))
	}))
	defer server.Close()

	result, err := NewClientWithHTTPClient(server.Client()).Probe(context.Background(), Config{
		Provider: "openai-compatible",
		BaseURL:  server.URL + "/v1",
		Model:    "missing-model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.ModelCount != 2 {
		t.Fatalf("unexpected probe result: %+v", result)
	}
	if !contains(result.Capabilities, "chat-completions") || !contains(result.Capabilities, "embeddings") {
		t.Fatalf("missing capabilities: %+v", result.Capabilities)
	}
	if !contains(result.Warnings, "Configured model was not returned by the provider.") {
		t.Fatalf("missing configured model warning: %+v", result.Warnings)
	}
}

func TestEndpointHelpers(t *testing.T) {
	chatURL, err := chatCompletionsEndpoint("http://localhost:11434/v1")
	if err != nil {
		t.Fatal(err)
	}
	if chatURL != "http://localhost:11434/v1/chat/completions" {
		t.Fatalf("unexpected chat endpoint %q", chatURL)
	}
	runtimeURL, ok := ollamaRuntimeEndpoint(Config{Provider: "ollama", BaseURL: "http://localhost:11434/v1"})
	if !ok || runtimeURL != "http://localhost:11434/api/ps" {
		t.Fatalf("unexpected runtime endpoint ok=%t url=%q", ok, runtimeURL)
	}
}

func TestProviderErrorDetailIsRedacted(t *testing.T) {
	detail := providerErrorDetail([]byte(`{"error":{"message":"Authorization Bearer sk-secret failed"}}`))
	if strings.Contains(detail, "sk-secret") || strings.Contains(detail, "Bearer sk-") {
		t.Fatalf("error detail was not redacted: %q", detail)
	}
}

func TestConfigFromSettingsIncludesAPIKey(t *testing.T) {
	config := ConfigFromSettings(settingssvc.Settings{
		Provider: "openai-compatible",
		Protocol: settingssvc.ProtocolOpenAICompatible,
		BaseURL:  "http://localhost:1234/v1",
		Model:    "test-model",
		APIKey:   "secret",
	})
	if config.APIKey != "secret" || config.Protocol != settingssvc.ProtocolOpenAICompatible {
		t.Fatalf("expected API key to propagate, got %#v", config)
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
