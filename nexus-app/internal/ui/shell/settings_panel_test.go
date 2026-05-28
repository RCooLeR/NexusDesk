package shell

import (
	"errors"
	"strings"
	"testing"

	llmSvc "nexusdesk/internal/services/llm"
)

func TestSettingsFromFormParsesTokens(t *testing.T) {
	settings, err := settingsFromForm("ollama", "http://localhost:11434/v1", "qwen2.5-coder:14b", "api-key", "32768", "4096")
	if err != nil {
		t.Fatalf("settingsFromForm returned error: %v", err)
	}
	if settings.APIKey != "api-key" {
		t.Fatalf("expected API key to round-trip, got %#v", settings)
	}
	if settings.ContextTokens != 32768 || settings.ResponseReserveTokens != 4096 {
		t.Fatalf("unexpected settings: %#v", settings)
	}
}

func TestSettingsFromFormRejectsInvalidTokens(t *testing.T) {
	if _, err := settingsFromForm("ollama", "http://localhost:11434/v1", "qwen2.5-coder:14b", "api-key", "bad", "4096"); err == nil {
		t.Fatal("expected invalid context tokens to fail")
	}
}

func TestFormatSettingsProbeResultSummarizesProvider(t *testing.T) {
	message := formatSettingsProbeResult(llmSvc.ProbeResult{
		OK:           true,
		Message:      "Connected to provider.",
		Endpoint:     "http://localhost:11434/v1/models",
		ModelCount:   2,
		ModelSample:  []string{"llama3.2:3b", "qwen2.5-coder:14b"},
		Capabilities: []string{"model-list", "chat-completions"},
		Warnings:     []string{"Configured model was not returned by the provider."},
		Runtime:      &llmSvc.RuntimeStatus{Message: "Selected model is loaded on CPU."},
	}, nil)
	for _, part := range []string{"Connected to provider.", "Models: 2", "Capabilities:", "Runtime:", "Warnings:"} {
		if !strings.Contains(message, part) {
			t.Fatalf("expected probe summary to contain %q, got %q", part, message)
		}
	}
}

func TestFormatSettingsProbeResultHandlesErrors(t *testing.T) {
	message := formatSettingsProbeResult(llmSvc.ProbeResult{}, errors.New("connection refused"))
	if !strings.Contains(message, "connection refused") {
		t.Fatalf("expected probe error in message, got %q", message)
	}
}
