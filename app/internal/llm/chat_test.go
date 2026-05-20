package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestChatUsesLongerTimeoutThanProbe(t *testing.T) {
	client := NewClient()
	chatClient := client.chatHTTPClient()
	if chatClient.Timeout != chatTimeout {
		t.Fatalf("expected chat timeout, got %s", chatClient.Timeout)
	}
	if client.httpClient.Timeout != probeTimeout {
		t.Fatalf("expected probe client timeout to stay unchanged, got %s", client.httpClient.Timeout)
	}
	if chatTimeout < time.Minute {
		t.Fatalf("chat timeout is too short: %s", chatTimeout)
	}
}

func TestChatStreamReadsDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var body chatCompletionRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}
		if !body.Stream {
			t.Fatal("expected streaming request")
		}

		response.Header().Set("Content-Type", "text/event-stream")
		_, _ = response.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		_, _ = response.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n"))
		_, _ = response.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	var deltas []string
	client := NewClient()
	result, err := client.ChatStream(context.Background(), storage.LLMSettings{
		BaseURL: server.URL + "/v1",
		Model:   "test-model",
	}, ChatRequest{Prompt: "Say hello"}, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}

	if result.Message != "Hello world" {
		t.Fatalf("unexpected streamed response: %s", result.Message)
	}
	if len(deltas) != 2 || deltas[0] != "Hello" || deltas[1] != " world" {
		t.Fatalf("unexpected deltas: %#v", deltas)
	}
}

func TestChatReturnsProviderErrorBodyOnHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusBadRequest)
		_, _ = response.Write([]byte(`{"error":{"message":"invalid api_key=secret-token in request"}}`))
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.Chat(context.Background(), storage.LLMSettings{
		BaseURL: server.URL,
		Model:   "test-model",
	}, ChatRequest{
		Prompt: "Explain it",
	})
	if err == nil {
		t.Fatal("expected provider error")
	}

	if got, want := err.Error(), "provider returned HTTP 400"; !strings.Contains(got, want) {
		t.Fatalf("expected %q in error, got %q", want, got)
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("expected provider token to be redacted, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "[redacted]") {
		t.Fatalf("expected redacted token marker, got %q", err.Error())
	}
}

func TestChatAuditsProviderErrorRedaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusUnauthorized)
		_, _ = response.Write([]byte(`{"error":{"message":"invalid api_key=secret-token"}}`))
	}))
	defer server.Close()

	client := NewClient()
	var logOutput bytes.Buffer
	originalAuditf := providerFailureAuditf
	providerFailureAuditf = func(endpoint string, status int, redacted bool, truncated bool, snippet string) {
		logOutput.WriteString(strconv.Itoa(status))
		logOutput.WriteString(" redacted=")
		logOutput.WriteString(strconv.FormatBool(redacted))
		logOutput.WriteString(" truncated=")
		logOutput.WriteString(strconv.FormatBool(truncated))
		logOutput.WriteString(" snippet=")
		logOutput.WriteString(snippet)
	}
	defer func() {
		providerFailureAuditf = originalAuditf
	}()

	_, err := client.Chat(context.Background(), storage.LLMSettings{
		BaseURL: server.URL,
		Model:   "test-model",
	}, ChatRequest{
		Prompt: "Explain it",
	})
	if err == nil {
		t.Fatal("expected provider error")
	}

	line := logOutput.String()
	if !strings.Contains(line, "redacted=true") {
		t.Fatalf("expected redaction flag in audit log, got %q", line)
	}
	if !strings.Contains(line, "401") {
		t.Fatalf("expected status code in audit log, got %q", line)
	}
	if strings.Contains(line, "secret-token") {
		t.Fatalf("expected audit log to avoid raw secret, got %q", line)
	}
}

func TestChatAuditsProviderErrorTruncation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusBadGateway)
		body := `{"error":{"message":"` + strings.Repeat("x", 2048) + `"}}`
		_, _ = response.Write([]byte(body))
	}))
	defer server.Close()

	client := NewClient()
	var logOutput bytes.Buffer
	originalAuditf := providerFailureAuditf
	providerFailureAuditf = func(endpoint string, status int, redacted bool, truncated bool, snippet string) {
		logOutput.WriteString(strconv.Itoa(status))
		logOutput.WriteString(" redacted=")
		logOutput.WriteString(strconv.FormatBool(redacted))
		logOutput.WriteString(" truncated=")
		logOutput.WriteString(strconv.FormatBool(truncated))
		logOutput.WriteString(" snippet=")
		logOutput.WriteString(snippet)
	}
	defer func() {
		providerFailureAuditf = originalAuditf
	}()

	_, err := client.Chat(context.Background(), storage.LLMSettings{
		BaseURL: server.URL,
		Model:   "test-model",
	}, ChatRequest{
		Prompt: "Summarize",
	})
	if err == nil {
		t.Fatal("expected provider error")
	}

	line := logOutput.String()
	if !strings.Contains(line, "truncated=true") {
		t.Fatalf("expected truncation flag in audit log, got %q", line)
	}
	if !strings.Contains(line, "502") {
		t.Fatalf("expected status code in audit log, got %q", line)
	}
}

func TestChatSanitizesLongProviderErrorsToSafeLength(t *testing.T) {
	body := `{"error":{"message":"` + strings.Repeat("x", 3000) + `"}}`
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusBadGateway)
		_, _ = response.Write([]byte(body))
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.Chat(context.Background(), storage.LLMSettings{
		BaseURL: server.URL,
		Model:   "test-model",
	}, ChatRequest{
		Prompt: "Summarize",
	})
	if err == nil {
		t.Fatal("expected provider error")
	}

	if !strings.Contains(err.Error(), "provider returned HTTP 502") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
	if len(err.Error()) > 1200 {
		t.Fatalf("expected sanitization/truncation bound, got %q", err.Error())
	}
}
