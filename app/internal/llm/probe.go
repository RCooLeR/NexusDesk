package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"NexusDesk/internal/storage"
)

const probeTimeout = 8 * time.Second

type ProbeResult struct {
	OK           bool     `json:"ok"`
	Message      string   `json:"message"`
	Endpoint     string   `json:"endpoint"`
	ModelCount   int      `json:"modelCount"`
	ModelSample  []string `json:"modelSample"`
	Capabilities []string `json:"capabilities"`
	Warnings     []string `json:"warnings"`
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: probeTimeout},
	}
}

func NewClientWithHTTPClient(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

func (c *Client) Probe(ctx context.Context, settings storage.LLMSettings) (ProbeResult, error) {
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: probeTimeout}
	}

	endpoint, err := modelsEndpoint(settings.BaseURL)
	if err != nil {
		return ProbeResult{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ProbeResult{}, err
	}

	request.Header.Set("Accept", "application/json")
	if settings.APIKey != "" {
		request.Header.Set("Authorization", "Bearer "+settings.APIKey)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return ProbeResult{}, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return ProbeResult{
			OK:           false,
			Message:      fmt.Sprintf("Provider returned HTTP %d", response.StatusCode),
			Endpoint:     endpoint,
			Capabilities: []string{},
			Warnings:     []string{},
		}, nil
	}

	var payload modelsResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return ProbeResult{}, err
	}

	modelIDs := make([]string, 0, len(payload.Data))
	for _, model := range payload.Data {
		if model.ID != "" {
			modelIDs = append(modelIDs, model.ID)
		}
	}

	message := "Connected to provider."
	if len(modelIDs) == 0 {
		message = "Connected, but no models were returned."
	}

	return ProbeResult{
		OK:           true,
		Message:      message,
		Endpoint:     endpoint,
		ModelCount:   len(modelIDs),
		ModelSample:  sampleModels(modelIDs),
		Capabilities: inferCapabilities(modelIDs),
		Warnings:     inferWarnings(settings.Model, modelIDs),
	}, nil
}

func modelsEndpoint(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("LLM base URL must be a valid HTTP URL")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("LLM base URL must use http or https")
	}

	if strings.HasSuffix(parsed.Path, "/models") {
		return parsed.String(), nil
	}

	parsed.Path = path.Join(parsed.Path, "models")
	return parsed.String(), nil
}

func sampleModels(modelIDs []string) []string {
	if len(modelIDs) <= 5 {
		return modelIDs
	}
	return modelIDs[:5]
}

func inferCapabilities(modelIDs []string) []string {
	if len(modelIDs) == 0 {
		return []string{}
	}

	capabilities := []string{"model-list"}
	hasChat := false
	hasEmbeddings := false
	hasVision := false
	hasReranking := false

	for _, modelID := range modelIDs {
		normalized := strings.ToLower(modelID)
		if containsAny(normalized, "gpt", "chat", "instruct", "llama", "mistral", "gemma", "qwen", "deepseek", "claude") {
			hasChat = true
		}
		if containsAny(normalized, "embed", "embedding", "bge-m3", "nomic-embed") {
			hasEmbeddings = true
		}
		if containsAny(normalized, "vision", "vl", "llava", "gpt-4o", "omni") {
			hasVision = true
		}
		if containsAny(normalized, "rerank", "reranker") {
			hasReranking = true
		}
	}

	if hasChat {
		capabilities = append(capabilities, "chat-completions")
	}
	if hasEmbeddings {
		capabilities = append(capabilities, "embeddings")
	}
	if hasVision {
		capabilities = append(capabilities, "vision")
	}
	if hasReranking {
		capabilities = append(capabilities, "reranking")
	}

	return capabilities
}

func inferWarnings(configuredModel string, modelIDs []string) []string {
	if strings.TrimSpace(configuredModel) == "" || len(modelIDs) == 0 {
		return []string{}
	}

	for _, modelID := range modelIDs {
		if modelID == configuredModel {
			return []string{}
		}
	}

	return []string{"Configured model was not returned by the provider."}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

type modelsResponse struct {
	Data []modelInfo `json:"data"`
}

type modelInfo struct {
	ID string `json:"id"`
}
