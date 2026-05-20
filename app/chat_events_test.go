package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"NexusDesk/internal/llm"
	"NexusDesk/internal/storage"
)

func TestAskLLMStreamEmitsRedactedErrorEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusUnauthorized)
		_, _ = response.Write([]byte(`{"error":{"message":"invalid api_key=secret-token"}}`))
	}))
	defer server.Close()

	app := NewApp()
	app.ctx = context.Background()
	app.llmClient = llm.NewClientWithHTTPClient(http.DefaultClient)
	app.llmStore = storage.NewLLMSettingsStore(filepath.Join(t.TempDir(), "llm-settings.json"))
	if _, err := app.llmStore.Save(storage.LLMSettings{
		BaseURL: server.URL,
		Model:   "test-model",
	}); err != nil {
		t.Fatalf("save settings failed: %v", err)
	}

	captured := []ChatStreamEvent{}
	originalEmitter := emitChatStreamEventFn
	emitChatStreamEventFn = func(_ context.Context, name string, payload any) {
		if name != chatStreamEventName {
			return
		}
		event, ok := payload.(ChatStreamEvent)
		if !ok {
			t.Fatalf("unexpected event type %T", payload)
		}
		captured = append(captured, event)
	}
	defer func() {
		emitChatStreamEventFn = originalEmitter
	}()

	_, err := app.AskLLMStream("say hello", "", "stream-err-1")
	if err == nil {
		t.Fatal("expected provider error")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("expected redacted provider message, got %q", err.Error())
	}

	found := false
	for _, event := range captured {
		if event.RequestID == "stream-err-1" && event.Type == "error" {
			found = true
			if !strings.Contains(event.Message, "[redacted]") {
				t.Fatalf("expected redacted event message, got %q", event.Message)
			}
			if strings.Contains(event.Message, "secret-token") {
				t.Fatalf("expected event message to redact provider secret, got %q", event.Message)
			}
		}
	}
	if !found {
		t.Fatalf("expected stream error event for request, got %#v", captured)
	}
}
