package settings

type Settings struct {
	Provider              string
	BaseURL               string
	Model                 string
	APIKey                string
	ContextTokens         int
	ResponseReserveTokens int
}

func Defaults() Settings {
	return Settings{
		Provider:              "ollama",
		BaseURL:               "http://localhost:11434/v1",
		Model:                 "qwen2.5-coder:14b",
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
	return []string{"ollama", "openai-compatible"}
}
