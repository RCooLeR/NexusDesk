// Package assistant orchestrates user-facing assistant requests without
// coupling provider transport to Fyne widgets.
package assistant

import (
	"context"
	"strings"

	"nexusdesk/internal/domain"
	"nexusdesk/internal/services/llm"
	settingssvc "nexusdesk/internal/services/settings"
	workspacesvc "nexusdesk/internal/services/workspace"
)

const charsPerTokenEstimate = 4

type SettingsStore interface {
	Load() (settingssvc.Settings, error)
}

type Previewer interface {
	PreviewFile(root string, relPath string) (domain.FilePreview, error)
}

type StreamClient interface {
	ChatStream(ctx context.Context, config llm.Config, chatRequest llm.ChatRequest, onDelta func(string) error) (llm.ChatResult, error)
}

type Service struct {
	settingsStore SettingsStore
	previewer     Previewer
	client        StreamClient
}

type Request struct {
	Prompt        string
	WorkspaceRoot string
	SelectedPath  string
	Conversation  []llm.ChatTurn
}

type Result struct {
	Message        string
	Model          string
	Endpoint       string
	ContextRelPath string
	SourcePaths    []string
	ContextWarning string
}

func New(settingsStore *settingssvc.Store, previewer *workspacesvc.Service, client *llm.Client) *Service {
	return &Service{settingsStore: settingsStore, previewer: previewer, client: client}
}

func NewWithDependencies(settingsStore SettingsStore, previewer Previewer, client StreamClient) *Service {
	return &Service{settingsStore: settingsStore, previewer: previewer, client: client}
}

func (s *Service) AskStream(ctx context.Context, request Request, onDelta func(string) error) (Result, error) {
	settings, err := s.settingsStore.Load()
	if err != nil {
		return Result{}, err
	}
	config := llm.ConfigFromSettings(settings)
	chatRequest := llm.ChatRequest{
		Prompt:       request.Prompt,
		Conversation: append([]llm.ChatTurn{}, request.Conversation...),
	}
	contextWarning := s.attachSelectedContext(config, request, &chatRequest)
	result, err := s.client.ChatStream(ctx, config, chatRequest, onDelta)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Message:        result.Message,
		Model:          result.Model,
		Endpoint:       result.Endpoint,
		ContextRelPath: result.ContextRelPath,
		SourcePaths:    result.SourcePaths,
		ContextWarning: contextWarning,
	}, nil
}

func (s *Service) attachSelectedContext(config llm.Config, request Request, chatRequest *llm.ChatRequest) string {
	root := strings.TrimSpace(request.WorkspaceRoot)
	selectedPath := strings.TrimSpace(request.SelectedPath)
	if root == "" || selectedPath == "" || s.previewer == nil {
		return ""
	}
	preview, err := s.previewer.PreviewFile(root, selectedPath)
	if err != nil {
		return "Selected file was not included: " + err.Error()
	}
	content := previewContextText(preview)
	if strings.TrimSpace(content) == "" {
		return "Selected file has no previewable text context."
	}
	content, truncated := fitContext(content, config)
	chatRequest.ContextRelPath = preview.RelPath
	chatRequest.ContextContent = content
	chatRequest.SourcePaths = []string{preview.RelPath}
	if truncated {
		return "Selected file context was capped to fit the model budget."
	}
	return ""
}

func previewContextText(preview domain.FilePreview) string {
	switch preview.Kind {
	case domain.PreviewText, domain.PreviewTable, domain.PreviewDoc, domain.PreviewPDF:
		return preview.Text
	default:
		return ""
	}
}

func fitContext(content string, config llm.Config) (string, bool) {
	budgetTokens := config.ContextTokens - config.ResponseReserveTokens
	if budgetTokens <= 0 {
		return "", true
	}
	limit := budgetTokens * charsPerTokenEstimate
	if limit <= 0 || len(content) <= limit {
		return content, false
	}
	return string([]rune(content)[:maxRunePrefix(content, limit)]) + "\n[context truncated]", true
}

func maxRunePrefix(content string, byteLimit int) int {
	totalBytes := 0
	for index, value := range []rune(content) {
		totalBytes += len(string(value))
		if totalBytes > byteLimit {
			return index
		}
	}
	return len([]rune(content))
}
