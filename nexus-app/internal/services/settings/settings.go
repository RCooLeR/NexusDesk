package settings

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

const (
	ProtocolOpenAICompatible       = "openai-compatible"
	ProtocolOllamaOpenAICompatible = "ollama-openai-compatible"
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
