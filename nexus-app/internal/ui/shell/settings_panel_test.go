package shell

import (
	"errors"
	"strings"
	"testing"

	llmSvc "nexusdesk/internal/services/llm"
)

func TestSettingsFromFormParsesTokens(t *testing.T) {
	settings, err := settingsFromForm("ollama", "ollama-openai-compatible", "http://localhost:11434/v1", "qwen2.5-coder:14b", "api-key", "32768", "4096")
	if err != nil {
		t.Fatalf("settingsFromForm returned error: %v", err)
	}
	if settings.Protocol != "ollama-openai-compatible" {
		t.Fatalf("expected protocol to round-trip, got %#v", settings)
	}
	if settings.APIKey != "api-key" {
		t.Fatalf("expected API key to round-trip, got %#v", settings)
	}
	if settings.ContextTokens != 32768 || settings.ResponseReserveTokens != 4096 {
		t.Fatalf("unexpected settings: %#v", settings)
	}
}

func TestSettingsFromFormRejectsInvalidTokens(t *testing.T) {
	if _, err := settingsFromForm("ollama", "ollama-openai-compatible", "http://localhost:11434/v1", "qwen2.5-coder:14b", "api-key", "bad", "4096"); err == nil {
		t.Fatal("expected invalid context tokens to fail")
	}
}

func TestSettingsModelOptionHelpersUseRecommendedCatalog(t *testing.T) {
	labels := settingsModelOptionLabels()
	if len(labels) == 0 || !strings.Contains(labels[0], "Qwen3 4B") {
		t.Fatalf("expected Wails recommended model labels, got %#v", labels)
	}
	option, ok := settingsModelOptionByLabel(labels[0])
	if !ok || option.ID != "qwen3:4b-instruct" {
		t.Fatalf("expected first recommended model option, got %#v ok=%v", option, ok)
	}
	label, ok := settingsModelLabelForID("mistral-small3.2")
	if !ok || !strings.Contains(label, "Mistral Small") {
		t.Fatalf("expected :latest-insensitive model label lookup, got %q ok=%v", label, ok)
	}
}

func TestFormatSettingsProbeResultSummarizesProvider(t *testing.T) {
	message := formatSettingsProbeResult(llmSvc.ProbeResult{
		OK:           true,
		Message:      "Connected to provider.",
		Endpoint:     "http://localhost:11434/v1/models",
		Protocol:     "ollama-openai-compatible",
		ModelCount:   2,
		ModelSample:  []string{"llama3.2:3b", "qwen2.5-coder:14b"},
		Capabilities: []string{"model-list", "chat-completions"},
		Warnings:     []string{"Configured model was not returned by the provider."},
		Runtime: &llmSvc.RuntimeStatus{
			Message:       "Selected model is loaded on CPU.",
			SelectedModel: "qwen2.5-coder:14b",
			LoadedModels:  []llmSvc.RuntimeModel{{Name: "qwen2.5-coder:14b", Model: "qwen2.5-coder:14b", ContextLength: 32768}},
		},
	}, nil)
	for _, part := range []string{"Connected to provider.", "Protocol: ollama-openai-compatible", "Models: 2", "Capabilities:", "Runtime:", "Runtime context: 32768 tokens", "Warnings:"} {
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
