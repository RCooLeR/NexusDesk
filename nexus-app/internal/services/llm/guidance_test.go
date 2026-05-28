package llm

import (
	"errors"
	"strings"
	"testing"
)

func TestProviderGuidanceExplainsOllamaConnectionRefused(t *testing.T) {
	guidance := ProviderGuidance(Config{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434/v1",
		Model:    "qwen3-coder:30b",
	}, ProbeResult{}, errors.New("dial tcp 127.0.0.1:11434: connect: connection refused"))

	joined := strings.Join(guidance, "\n")
	for _, expected := range []string{
		"network reachability",
		"ollama serve",
		"ollama list",
		"runtime or remote endpoint is running",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in guidance:\n%s", expected, joined)
		}
	}
}

func TestProviderGuidanceExplainsAuthAndBaseURLFailures(t *testing.T) {
	auth := strings.Join(ProviderGuidance(Config{
		Provider: "custom-openai-compatible",
		BaseURL:  "https://example.test/v1",
	}, ProbeResult{
		OK:      false,
		Message: "Provider returned HTTP 401",
	}, nil), "\n")
	if !strings.Contains(auth, "API key") || !strings.Contains(auth, "bearer") {
		t.Fatalf("expected credential guidance, got:\n%s", auth)
	}

	notFound := strings.Join(ProviderGuidance(Config{
		Provider: "openai-compatible",
		BaseURL:  "http://localhost:1234",
	}, ProbeResult{
		OK:      false,
		Message: "Provider returned HTTP 404",
	}, nil), "\n")
	if !strings.Contains(notFound, "/v1") {
		t.Fatalf("expected base URL guidance, got:\n%s", notFound)
	}
}

func TestProviderGuidanceExplainsMissingAndUnloadedModels(t *testing.T) {
	guidance := ProviderGuidance(Config{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434/v1",
		Model:    "qwen3-coder:30b",
	}, ProbeResult{
		OK:         true,
		ModelCount: 2,
		Warnings:   []string{"Configured model was not returned by the provider."},
		Runtime: &RuntimeStatus{
			SelectedModel:       "qwen3-coder:30b",
			SelectedModelLoaded: false,
			Message:             "Selected model is not loaded in Ollama runtime yet.",
		},
	}, nil)

	joined := strings.Join(guidance, "\n")
	for _, expected := range []string{
		"ollama pull qwen3-coder:30b",
		"ollama run qwen3-coder:30b",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in guidance:\n%s", expected, joined)
		}
	}
}
