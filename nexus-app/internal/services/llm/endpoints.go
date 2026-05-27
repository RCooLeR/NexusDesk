package llm

import (
	"errors"
	"net/url"
	"path"
	"strings"
)

func chatCompletionsEndpoint(baseURL string) (string, error) {
	return endpointWithSuffix(baseURL, "/chat/completions", "chat/completions")
}

func modelsEndpoint(baseURL string) (string, error) {
	return endpointWithSuffix(baseURL, "/models", "models")
}

func endpointWithSuffix(baseURL string, suffix string, joinPart string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("LLM base URL must be a valid HTTP URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("LLM base URL must use http or https")
	}
	if strings.HasSuffix(parsed.Path, suffix) {
		return parsed.String(), nil
	}
	parsed.Path = path.Join(parsed.Path, joinPart)
	return parsed.String(), nil
}

func ollamaRuntimeEndpoint(config Config) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(config.BaseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	provider := strings.ToLower(config.Provider)
	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	isLocalHost := host == "localhost" || host == "127.0.0.1" || host == "::1"
	isOllama := strings.Contains(provider, "ollama") || (isLocalHost && port == "11434")
	if !isOllama {
		return "", false
	}
	parsed.Path = "/api/ps"
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), true
}
