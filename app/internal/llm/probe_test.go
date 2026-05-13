package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"NexusDesk/internal/storage"
)

func TestProbeConnectsToOpenAICompatibleModelsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("missing auth header")
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"data":[{"id":"alpha"},{"id":"beta"}]}`))
	}))
	defer server.Close()

	client := NewClient()
	result, err := client.Probe(context.Background(), storage.LLMSettings{
		BaseURL: server.URL + "/v1",
		APIKey:  "secret",
	})
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}

	if !result.OK {
		t.Fatalf("expected OK result")
	}
	if result.ModelCount != 2 {
		t.Fatalf("expected 2 models, got %d", result.ModelCount)
	}
	if len(result.ModelSample) != 2 || result.ModelSample[0] != "alpha" {
		t.Fatalf("unexpected model sample: %#v", result.ModelSample)
	}
}

func TestProbeReturnsNonOKForProviderStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient()
	result, err := client.Probe(context.Background(), storage.LLMSettings{
		BaseURL: server.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}

	if result.OK {
		t.Fatal("expected non-OK result")
	}
	if result.Message == "" {
		t.Fatal("expected message")
	}
}

func TestModelsEndpointHandlesExistingModelsPath(t *testing.T) {
	endpoint, err := modelsEndpoint("http://localhost:11434/v1/models")
	if err != nil {
		t.Fatalf("modelsEndpoint returned error: %v", err)
	}

	if endpoint != "http://localhost:11434/v1/models" {
		t.Fatalf("unexpected endpoint: %s", endpoint)
	}
}
