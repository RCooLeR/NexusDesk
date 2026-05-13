package llm

import (
	"bufio"
	"bytes"
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

const chatTimeout = 5 * time.Minute

type ChatRequest struct {
	Prompt         string   `json:"prompt"`
	ContextRelPath string   `json:"contextRelPath"`
	ContextContent string   `json:"contextContent"`
	SourcePaths    []string `json:"sourcePaths"`
}

type ChatResult struct {
	Message        string   `json:"message"`
	Model          string   `json:"model"`
	Endpoint       string   `json:"endpoint"`
	ContextRelPath string   `json:"contextRelPath"`
	SourcePaths    []string `json:"sourcePaths"`
}

func (c *Client) Chat(ctx context.Context, settings storage.LLMSettings, chatRequest ChatRequest) (ChatResult, error) {
	return c.chat(ctx, settings, chatRequest, false, nil)
}

func (c *Client) ChatStream(ctx context.Context, settings storage.LLMSettings, chatRequest ChatRequest, onDelta func(string) error) (ChatResult, error) {
	return c.chat(ctx, settings, chatRequest, true, onDelta)
}

func (c *Client) chat(ctx context.Context, settings storage.LLMSettings, chatRequest ChatRequest, stream bool, onDelta func(string) error) (ChatResult, error) {
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
		Stream:      stream,
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

	response, err := c.chatHTTPClient().Do(request)
	if err != nil {
		return ChatResult{}, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return ChatResult{}, fmt.Errorf("provider returned HTTP %d", response.StatusCode)
	}

	if stream {
		message, err := readChatCompletionStream(response, onDelta)
		if err != nil {
			return ChatResult{}, err
		}

		return ChatResult{
			Message:        message,
			Model:          settings.Model,
			Endpoint:       endpoint,
			ContextRelPath: strings.TrimSpace(chatRequest.ContextRelPath),
			SourcePaths:    append([]string{}, chatRequest.SourcePaths...),
		}, nil
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
		SourcePaths:    append([]string{}, chatRequest.SourcePaths...),
	}, nil
}

func (c *Client) chatHTTPClient() *http.Client {
	if c.httpClient == nil {
		return &http.Client{Timeout: chatTimeout}
	}
	if c.httpClient.Timeout > 0 && c.httpClient.Timeout < chatTimeout {
		chatClient := *c.httpClient
		chatClient.Timeout = chatTimeout
		return &chatClient
	}
	return c.httpClient
}

func readChatCompletionStream(response *http.Response, onDelta func(string) error) (string, error) {
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var message strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var payload chatCompletionStreamResponse
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return "", err
		}
		if len(payload.Choices) == 0 {
			continue
		}

		delta := payload.Choices[0].Delta.Content
		if delta == "" {
			continue
		}

		message.WriteString(delta)
		if onDelta != nil {
			if err := onDelta(delta); err != nil {
				return "", err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	result := strings.TrimSpace(message.String())
	if result == "" {
		return "", errors.New("provider returned an empty chat response")
	}

	return result, nil
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
	Stream      bool          `json:"stream,omitempty"`
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

type chatCompletionStreamResponse struct {
	Choices []chatStreamChoice `json:"choices"`
}

type chatStreamChoice struct {
	Delta chatMessage `json:"delta"`
}
