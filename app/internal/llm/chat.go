package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"NexusDesk/internal/storage"
)

type ChatRequest struct {
	Prompt         string `json:"prompt"`
	ContextRelPath string `json:"contextRelPath"`
	ContextContent string `json:"contextContent"`
}

type ChatResult struct {
	Message        string `json:"message"`
	Model          string `json:"model"`
	Endpoint       string `json:"endpoint"`
	ContextRelPath string `json:"contextRelPath"`
}

func (c *Client) Chat(ctx context.Context, settings storage.LLMSettings, chatRequest ChatRequest) (ChatResult, error) {
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: probeTimeout}
	}

	settings = storage.LLMSettings{
		ProviderName: strings.TrimSpace(settings.ProviderName),
		BaseURL:      strings.TrimSpace(settings.BaseURL),
		Model:        strings.TrimSpace(settings.Model),
		APIKey:       strings.TrimSpace(settings.APIKey),
	}

	prompt := strings.TrimSpace(chatRequest.Prompt)
	if prompt == "" {
		return ChatResult{}, errors.New("prompt is required")
	}
	if settings.Model == "" {
		return ChatResult{}, errors.New("LLM model is required before sending chat")
	}

	endpoint, err := chatCompletionsEndpoint(settings.BaseURL)
	if err != nil {
		return ChatResult{}, err
	}

	body, err := json.Marshal(chatCompletionRequest{
		Model: settings.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt()},
			{Role: "user", Content: buildUserPrompt(prompt, chatRequest)},
		},
		Temperature: 0.2,
	})
	if err != nil {
		return ChatResult{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ChatResult{}, err
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	if settings.APIKey != "" {
		request.Header.Set("Authorization", "Bearer "+settings.APIKey)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return ChatResult{}, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return ChatResult{}, fmt.Errorf("provider returned HTTP %d", response.StatusCode)
	}

	var payload chatCompletionResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return ChatResult{}, err
	}

	if len(payload.Choices) == 0 || strings.TrimSpace(payload.Choices[0].Message.Content) == "" {
		return ChatResult{}, errors.New("provider returned an empty chat response")
	}

	return ChatResult{
		Message:        strings.TrimSpace(payload.Choices[0].Message.Content),
		Model:          settings.Model,
		Endpoint:       endpoint,
		ContextRelPath: strings.TrimSpace(chatRequest.ContextRelPath),
	}, nil
}

func chatCompletionsEndpoint(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("LLM base URL must be a valid HTTP URL")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("LLM base URL must use http or https")
	}

	if strings.HasSuffix(parsed.Path, "/chat/completions") {
		return parsed.String(), nil
	}

	parsed.Path = path.Join(parsed.Path, "chat/completions")
	return parsed.String(), nil
}

func systemPrompt() string {
	return "You are NexusDesk, a local-first AI workbench assistant. Answer from provided workspace context when it is present. If more source context is needed, say what to select or inspect next. Do not claim access to files that were not provided."
}

func buildUserPrompt(prompt string, chatRequest ChatRequest) string {
	contextContent := strings.TrimSpace(chatRequest.ContextContent)
	contextRelPath := strings.TrimSpace(chatRequest.ContextRelPath)
	if contextContent == "" {
		return prompt
	}

	return fmt.Sprintf("Workspace context file: %s\n\n```text\n%s\n```\n\nUser request: %s", contextRelPath, contextContent, prompt)
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}
