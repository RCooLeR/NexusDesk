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

	"NexusAugenticStudio/internal/storage"
)

const probeTimeout = 8 * time.Second

type ProbeResult struct {
	OK           bool           `json:"ok"`
	Message      string         `json:"message"`
	Endpoint     string         `json:"endpoint"`
	ModelCount   int            `json:"modelCount"`
	ModelSample  []string       `json:"modelSample"`
	Capabilities []string       `json:"capabilities"`
	Warnings     []string       `json:"warnings"`
	Runtime      *RuntimeStatus `json:"runtime,omitempty"`
}

type RuntimeStatus struct {
	Provider            string         `json:"provider"`
	Endpoint            string         `json:"endpoint"`
	Message             string         `json:"message"`
	SelectedModel       string         `json:"selectedModel"`
	SelectedModelLoaded bool           `json:"selectedModelLoaded"`
	SelectedModelVRAM   int64          `json:"selectedModelVram"`
	LoadedModels        []RuntimeModel `json:"loadedModels"`
}

type RuntimeModel struct {
	Name          string `json:"name"`
	Model         string `json:"model"`
	Size          int64  `json:"size"`
	SizeVRAM      int64  `json:"sizeVram"`
	ContextLength int    `json:"contextLength"`
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

	capabilities := inferCapabilities(modelIDs)
	warnings := inferWarnings(settings.Model, modelIDs)
	runtimeStatus, runtimeWarnings := c.probeOllamaRuntime(ctx, settings)
	warnings = append(warnings, runtimeWarnings...)
	if runtimeStatus != nil && runtimeStatus.SelectedModelLoaded && runtimeStatus.SelectedModelVRAM > 0 {
		capabilities = append(capabilities, "gpu-offload")
	}

	return ProbeResult{
		OK:           true,
		Message:      message,
		Endpoint:     endpoint,
		ModelCount:   len(modelIDs),
		ModelSample:  sampleModels(modelIDs),
		Capabilities: capabilities,
		Warnings:     warnings,
		Runtime:      runtimeStatus,
	}, nil
}

func (c *Client) probeOllamaRuntime(ctx context.Context, settings storage.LLMSettings) (*RuntimeStatus, []string) {
	endpoint, ok := ollamaRuntimeEndpoint(settings)
	if !ok {
		return nil, []string{}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, []string{err.Error()}
	}
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, []string{"Ollama runtime status is unavailable: " + err.Error()}
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, []string{fmt.Sprintf("Ollama runtime status returned HTTP %d.", response.StatusCode)}
	}

	var payload ollamaPSResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, []string{"Ollama runtime status could not be decoded: " + err.Error()}
	}

	selectedModel := strings.TrimSpace(settings.Model)
	status := &RuntimeStatus{
		Provider:      "ollama",
		Endpoint:      endpoint,
		SelectedModel: selectedModel,
		LoadedModels:  make([]RuntimeModel, 0, len(payload.Models)),
	}

	for _, model := range payload.Models {
		runtimeModel := RuntimeModel{
			Name:          model.Name,
			Model:         model.Model,
			Size:          model.Size,
			SizeVRAM:      model.SizeVRAM,
			ContextLength: model.ContextLength,
		}
		status.LoadedModels = append(status.LoadedModels, runtimeModel)
		if selectedModel != "" && (model.Name == selectedModel || model.Model == selectedModel) {
			status.SelectedModelLoaded = true
			status.SelectedModelVRAM = model.SizeVRAM
		}
	}

	warnings := []string{}
	switch {
	case len(status.LoadedModels) == 0:
		status.Message = "No models are loaded in Ollama runtime yet."
	case selectedModel == "":
		status.Message = fmt.Sprintf("%d Ollama model(s) loaded.", len(status.LoadedModels))
	case !status.SelectedModelLoaded:
		status.Message = "Selected model is not loaded in Ollama runtime yet."
	case status.SelectedModelVRAM > 0:
		status.Message = "Selected model is loaded with GPU VRAM assigned."
	default:
		status.Message = "Selected model is loaded on CPU."
		warnings = append(warnings, "Selected Ollama model is loaded on CPU (size_vram is 0).")
	}

	return status, warnings
}

func ollamaRuntimeEndpoint(settings storage.LLMSettings) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(settings.BaseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}

	provider := strings.ToLower(settings.ProviderName)
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

type ollamaPSResponse struct {
	Models []ollamaPSModel `json:"models"`
}

type ollamaPSModel struct {
	Name          string `json:"name"`
	Model         string `json:"model"`
	Size          int64  `json:"size"`
	SizeVRAM      int64  `json:"size_vram"`
	ContextLength int    `json:"context_length"`
}
