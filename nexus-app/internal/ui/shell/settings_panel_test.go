package shell

import (
	"errors"
	"strings"
	"testing"

	llmSvc "nexusdesk/internal/services/llm"
	settingsSvc "nexusdesk/internal/services/settings"
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

func TestSettingsFromFormPreservesTaskModelRoutes(t *testing.T) {
	routes := []settingsSvc.ModelRoute{{ID: settingsSvc.RouteMainCoding, Label: "Main coding model", Model: "qwen3-coder:30b"}}
	settings, err := settingsFromFormWithRoutes("ollama", "ollama-openai-compatible", "http://localhost:11434/v1", "qwen3:8b", "", "32768", "4096", routes)
	if err != nil {
		t.Fatalf("settingsFromFormWithRoutes returned error: %v", err)
	}
	if len(settings.ModelRoutes) != 1 || settings.ModelRoutes[0].Model != "qwen3-coder:30b" {
		t.Fatalf("expected model routes to be preserved, got %#v", settings.ModelRoutes)
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
	if _, ok := settingsModelLabelForID("qwen3-coder:30b"); !ok {
		t.Fatal("expected production coding model in settings catalog")
	}
	label, ok := settingsModelLabelForID("mistral-small3.2")
	if !ok || !strings.Contains(label, "Mistral Small") {
		t.Fatalf("expected :latest-insensitive model label lookup, got %q ok=%v", label, ok)
	}
}

func TestSettingsRouteHelpersUpdateSelectedRoute(t *testing.T) {
	routes := settingsSvc.DefaultModelRoutes()
	labels := settingsRouteOptionLabels(routes)
	if len(labels) == 0 || !strings.Contains(labels[0], "Main coding") {
		t.Fatalf("expected route labels, got %#v", labels)
	}
	route, ok := settingsRouteByLabel(routes, "Research / summaries")
	if !ok || route.ID != settingsSvc.RouteResearchSummaries {
		t.Fatalf("expected research route lookup, got %#v ok=%v", route, ok)
	}
	routes = settingsModelRoutesWithModel(routes, settingsSvc.RouteResearchSummaries, "qwen3.6:27b")
	updated, ok := settingsRouteByLabel(routes, "Research / summaries")
	if !ok || updated.Model != "qwen3.6:27b" || updated.ResponseReserveTokens == 0 {
		t.Fatalf("expected route model update, got %#v ok=%v", updated, ok)
	}
	detail := settingsRouteDetail(routes, settingsSvc.RouteVisionScreenshot)
	if !strings.Contains(detail, "Alternative: qwen3.6:27b") {
		t.Fatalf("expected alternate model in route detail, got %q", detail)
	}
}

func TestSettingsSectionSearchMatchesTitlesSummariesAndKeywords(t *testing.T) {
	sections := []settingsPanelSection{
		{Title: "Provider & Runtime", Summary: "Configure endpoint", Keywords: []string{"ollama", "context"}},
		{Title: "Secrets & Credentials", Summary: "Protected API key storage", Keywords: []string{"dpapi", "keychain"}},
		{Title: "Task Model Routes", Summary: "Workflow defaults", Keywords: []string{"vision", "coding"}},
	}
	if got := settingsVisibleSectionTitles(sections, "api key"); len(got) != 1 || got[0] != "Secrets & Credentials" {
		t.Fatalf("expected API key search to find secrets section, got %#v", got)
	}
	if got := settingsVisibleSectionTitles(sections, "VISION"); len(got) != 1 || got[0] != "Task Model Routes" {
		t.Fatalf("expected case-insensitive keyword search, got %#v", got)
	}
	if got := settingsVisibleSectionTitles(sections, ""); len(got) != 3 {
		t.Fatalf("expected empty search to show all sections, got %#v", got)
	}
}

func TestSettingsValidationSummarizesWarningsAndErrors(t *testing.T) {
	issues := settingsValidationIssues(
		"ollama",
		"ollama-openai-compatible",
		"http://localhost:11434/v1",
		"",
		"4096",
		"4096",
		[]settingsSvc.ModelRoute{{ID: settingsSvc.RouteMainCoding, Label: "Main coding model"}},
	)
	text := settingsValidationText(issues)
	for _, expected := range []string{
		"Settings need attention:",
		"WARNING: Global chat model is not selected",
		"ERROR: Response reserve must be smaller than the context window.",
		"WARNING: 1 task model route(s) have no default model.",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected validation text to contain %q, got %q", expected, text)
		}
	}
	if err := settingsBlockingValidationError(issues); err == nil || !strings.Contains(err.Error(), "Response reserve must be smaller") {
		t.Fatalf("expected blocking validation error for token budget, got %v", err)
	}
}

func TestSettingsValidationReadyMessage(t *testing.T) {
	issues := settingsValidationIssues(
		"ollama",
		"ollama-openai-compatible",
		"http://localhost:11434/v1",
		"qwen3-coder:30b",
		"32768",
		"4096",
		settingsSvc.DefaultModelRoutes(),
	)
	if len(issues) != 0 {
		t.Fatalf("expected ready settings, got %#v", issues)
	}
	if text := settingsValidationText(issues); !strings.Contains(text, "Settings look ready") {
		t.Fatalf("expected ready validation text, got %q", text)
	}
	if err := settingsBlockingValidationError(issues); err != nil {
		t.Fatalf("expected ready settings to be non-blocking, got %v", err)
	}
}

func TestFormatSettingsProbeResultSummarizesProvider(t *testing.T) {
	message := formatSettingsProbeResultWithConfig(llmSvc.Config{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434/v1",
		Model:    "missing-model:latest",
	}, llmSvc.ProbeResult{
		OK:           true,
		Message:      "Connected to provider.",
		Endpoint:     "http://localhost:11434/v1/models",
		Protocol:     "ollama-openai-compatible",
		ModelCount:   2,
		ModelSample:  []string{"llama3.2:3b", "qwen2.5-coder:14b"},
		Capabilities: []string{"model-list", "chat-completions"},
		Warnings:     []string{"Configured model was not returned by the provider."},
		Runtime: &llmSvc.RuntimeStatus{
			Message:             "Selected model is loaded on CPU.",
			SelectedModel:       "qwen2.5-coder:14b",
			SelectedModelLoaded: true,
			LoadedModels:        []llmSvc.RuntimeModel{{Name: "qwen2.5-coder:14b", Model: "qwen2.5-coder:14b", ContextLength: 32768}},
		},
	}, nil)
	for _, part := range []string{"Connected to provider.", "Protocol: ollama-openai-compatible", "Models: 2", "Capabilities:", "Runtime:", "Runtime context: 32768 tokens", "Warnings:", "Guidance:", "ollama pull missing-model:latest"} {
		if !strings.Contains(message, part) {
			t.Fatalf("expected probe summary to contain %q, got %q", part, message)
		}
	}
}

func TestFormatSettingsProbeResultHandlesErrors(t *testing.T) {
	message := formatSettingsProbeResultWithConfig(llmSvc.Config{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434/v1",
	}, llmSvc.ProbeResult{}, errors.New("connection refused"))
	if !strings.Contains(message, "connection refused") {
		t.Fatalf("expected probe error in message, got %q", message)
	}
	if !strings.Contains(message, "ollama serve") {
		t.Fatalf("expected Ollama remediation guidance, got %q", message)
	}
}
