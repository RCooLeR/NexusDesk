package settings

import "strings"

type Settings struct {
	Provider              string
	Protocol              string
	BaseURL               string
	Model                 string
	APIKey                string
	ContextTokens         int
	ResponseReserveTokens int
}

type ProviderProfile struct {
	ID             string
	Label          string
	Protocol       string
	DefaultBaseURL string
	RequiresAPIKey bool
	RuntimeProbe   bool
}

type ModelOption struct {
	ID               string
	Label            string
	ChatLabel        string
	MaxContextTokens int
}

const (
	ProtocolOpenAICompatible       = "openai-compatible"
	ProtocolOllamaOpenAICompatible = "ollama-openai-compatible"
	FallbackModelContextTokens     = 32768
)

func Defaults() Settings {
	return Settings{
		Provider:              "ollama",
		Protocol:              ProtocolOllamaOpenAICompatible,
		BaseURL:               "http://localhost:11434/v1",
		Model:                 "",
		ContextTokens:         32768,
		ResponseReserveTokens: 4096,
	}
}

func ModelOptions() []string {
	recommended := RecommendedModelOptions()
	options := make([]string, 0, len(recommended))
	for _, option := range recommended {
		options = append(options, option.ID)
	}
	return options
}

func RecommendedModelOptions() []ModelOption {
	return []ModelOption{
		{ID: "qwen3:4b-instruct", Label: "Qwen3 4B Instruct - fast local", ChatLabel: "Qwen3 4B", MaxContextTokens: 32768},
		{ID: "qwen3:8b", Label: "Qwen3 8B - balanced", ChatLabel: "Qwen3 8B", MaxContextTokens: 40960},
		{ID: "qwen3.5:9b", Label: "Qwen3.5 9B - workspace chat", ChatLabel: "Qwen3.5 9B", MaxContextTokens: 131072},
		{ID: "phi4:14b", Label: "Phi-4 14B - reasoning", ChatLabel: "Phi-4 14B", MaxContextTokens: 16384},
		{ID: "phi4-reasoning:14b", Label: "Phi-4 Reasoning 14B - deep reasoning", ChatLabel: "Phi-4 Reasoning", MaxContextTokens: 32768},
		{ID: "gpt-oss:20b", Label: "GPT-OSS 20B - strong general", ChatLabel: "GPT-OSS 20B", MaxContextTokens: 131072},
		{ID: "mistral-small3.2:latest", Label: "Mistral Small 3.2 - long context", ChatLabel: "Mistral Small", MaxContextTokens: 131072},
		{ID: "gemma4:26b", Label: "Gemma 4 26B - max local", ChatLabel: "Gemma 4 26B", MaxContextTokens: 131072},
	}
}

func LegacyModelOptions() []string {
	return []string{
		"qwen2.5-coder:7b",
		"qwen2.5-coder:14b",
		"deepseek-coder-v2:16b",
		"mistral-small:24b",
		"phi4:14b",
		"llama3.1:8b",
		"gemma3:12b",
	}
}

func ModelContextWindow(model string) int {
	for _, option := range RecommendedModelOptions() {
		if modelMatches(option.ID, model) {
			return option.MaxContextTokens
		}
	}
	return FallbackModelContextTokens
}

func ResponseReserveForContext(maxContextTokens int) int {
	if maxContextTokens <= 0 {
		return 4096
	}
	reserve := maxContextTokens / 8
	if reserve < 2048 {
		return 2048
	}
	if reserve > 32768 {
		return 32768
	}
	return reserve
}

func SettingsForSelectedModel(settings Settings, model string) Settings {
	settings.Model = model
	settings.ContextTokens = ModelContextWindow(model)
	settings.ResponseReserveTokens = ResponseReserveForContext(settings.ContextTokens)
	return settings
}

func ProviderOptions() []string {
	profiles := ProviderProfiles()
	options := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		options = append(options, profile.ID)
	}
	return options
}

func ProtocolOptions() []string {
	return []string{ProtocolOllamaOpenAICompatible, ProtocolOpenAICompatible}
}

func ProviderProfiles() []ProviderProfile {
	return []ProviderProfile{
		{
			ID:             "ollama",
			Label:          "Ollama",
			Protocol:       ProtocolOllamaOpenAICompatible,
			DefaultBaseURL: "http://localhost:11434/v1",
			RuntimeProbe:   true,
		},
		{
			ID:             "openai-compatible",
			Label:          "OpenAI-compatible",
			Protocol:       ProtocolOpenAICompatible,
			DefaultBaseURL: "http://localhost:1234/v1",
		},
		{
			ID:             "custom-openai-compatible",
			Label:          "Custom OpenAI-compatible",
			Protocol:       ProtocolOpenAICompatible,
			DefaultBaseURL: "https://api.openai.com/v1",
			RequiresAPIKey: true,
		},
	}
}

func ProviderProfileByID(id string) (ProviderProfile, bool) {
	for _, profile := range ProviderProfiles() {
		if profile.ID == id {
			return profile, true
		}
	}
	return ProviderProfile{}, false
}

func modelMatches(left string, right string) bool {
	return normalizeModelName(left) == normalizeModelName(right)
}

func normalizeModelName(value string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ":latest")
}
