// Package llm owns OpenAI-compatible model transport for the native app.
// It stays UI-agnostic so shell panels, future agents, and background jobs can
// share the same provider gateway without coupling to Fyne widgets.
package llm

import (
	"net/http"
	"strings"

	settingssvc "nexusdesk/internal/services/settings"
)

type Config struct {
	Provider              string
	BaseURL               string
	Model                 string
	APIKey                string
	ContextTokens         int
	ResponseReserveTokens int
}

type ChatRequest struct {
	Prompt         string
	ContextRelPath string
	ContextContent string
	SourcePaths    []string
	Conversation   []ChatTurn
}

type ChatTurn struct {
	Role    string
	Content string
}

type ChatResult struct {
	Message        string
	Model          string
	Endpoint       string
	ContextRelPath string
	SourcePaths    []string
}

type ProbeResult struct {
	OK           bool
	Message      string
	Endpoint     string
	ModelCount   int
	ModelSample  []string
	Capabilities []string
	Warnings     []string
	Runtime      *RuntimeStatus
}

type RuntimeStatus struct {
	Provider            string
	Endpoint            string
	Message             string
	SelectedModel       string
	SelectedModelLoaded bool
	SelectedModelVRAM   int64
	LoadedModels        []RuntimeModel
}

type RuntimeModel struct {
	Name          string
	Model         string
	Size          int64
	SizeVRAM      int64
	ContextLength int
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: probeTimeout}}
}

func NewClientWithHTTPClient(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

func ConfigFromSettings(settings settingssvc.Settings) Config {
	return Config{
		Provider:              settings.Provider,
		BaseURL:               settings.BaseURL,
		Model:                 settings.Model,
		APIKey:                settings.APIKey,
		ContextTokens:         settings.ContextTokens,
		ResponseReserveTokens: settings.ResponseReserveTokens,
	}
}

func normalizeConfig(config Config) Config {
	defaults := settingssvc.Defaults()
	config.Provider = strings.TrimSpace(config.Provider)
	config.BaseURL = strings.TrimSpace(config.BaseURL)
	config.Model = strings.TrimSpace(config.Model)
	config.APIKey = strings.TrimSpace(config.APIKey)
	if config.Provider == "" {
		config.Provider = defaults.Provider
	}
	if config.BaseURL == "" {
		config.BaseURL = defaults.BaseURL
	}
	if config.ContextTokens <= 0 {
		config.ContextTokens = defaults.ContextTokens
	}
	if config.ResponseReserveTokens <= 0 {
		config.ResponseReserveTokens = defaults.ResponseReserveTokens
	}
	if config.ResponseReserveTokens >= config.ContextTokens {
		config.ResponseReserveTokens = config.ContextTokens / 4
	}
	return config
}
