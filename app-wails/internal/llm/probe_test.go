package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"NexusAugenticStudio/internal/storage"
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
	assertCapability(t, result.Capabilities, "model-list")
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
	if result.Capabilities == nil || result.Warnings == nil {
		t.Fatal("expected non-nil capability and warning slices")
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

func TestProbeInfersCapabilitiesFromModelIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"data":[{"id":"gpt-4o-mini"},{"id":"text-embedding-3-small"},{"id":"bge-reranker"}]}`))
	}))
	defer server.Close()

	client := NewClient()
	result, err := client.Probe(context.Background(), storage.LLMSettings{
		BaseURL: server.URL + "/v1",
		Model:   "missing-model",
	})
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}

	assertCapability(t, result.Capabilities, "chat-completions")
	assertCapability(t, result.Capabilities, "embeddings")
	assertCapability(t, result.Capabilities, "vision")
	assertCapability(t, result.Capabilities, "reranking")

	if len(result.Warnings) != 1 {
		t.Fatalf("expected configured model warning, got %#v", result.Warnings)
	}
}

func TestProbeReportsOllamaRuntimeGPUOffload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/models":
			_, _ = response.Write([]byte(`{"data":[{"id":"qwen3:4b-instruct"}]}`))
		case "/api/ps":
			_, _ = response.Write([]byte(`{"models":[{"name":"qwen3:4b-instruct","model":"qwen3:4b-instruct","size":3571659904,"size_vram":3571659904,"context_length":4096}]}`))
		default:
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient()
	result, err := client.Probe(context.Background(), storage.LLMSettings{
		ProviderName: "Ollama",
		BaseURL:      server.URL + "/v1",
		Model:        "qwen3:4b-instruct",
	})
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}

	if result.Runtime == nil {
		t.Fatal("expected Ollama runtime status")
	}
	if !result.Runtime.SelectedModelLoaded {
		t.Fatal("expected selected model to be loaded")
	}
	if result.Runtime.SelectedModelVRAM == 0 {
		t.Fatal("expected selected model VRAM")
	}
	assertCapability(t, result.Capabilities, "gpu-offload")
}

func TestProbeWarnsWhenOllamaSelectedModelIsOnCPU(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/models":
			_, _ = response.Write([]byte(`{"data":[{"id":"qwen3:4b-instruct"}]}`))
		case "/api/ps":
			_, _ = response.Write([]byte(`{"models":[{"name":"qwen3:4b-instruct","model":"qwen3:4b-instruct","size":3571659904,"size_vram":0,"context_length":4096}]}`))
		default:
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient()
	result, err := client.Probe(context.Background(), storage.LLMSettings{
		ProviderName: "Ollama",
		BaseURL:      server.URL + "/v1",
		Model:        "qwen3:4b-instruct",
	})
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}

	if result.Runtime == nil || result.Runtime.SelectedModelVRAM != 0 {
		t.Fatalf("unexpected runtime status: %#v", result.Runtime)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected CPU offload warning")
	}
}

func assertCapability(t *testing.T, capabilities []string, expected string) {
	t.Helper()

	for _, capability := range capabilities {
		if capability == expected {
			return
		}
	}

	t.Fatalf("expected capability %q in %#v", expected, capabilities)
}
