// Package assistant orchestrates user-facing assistant requests without
// coupling provider transport to Fyne widgets.
package assistant

import (
	"context"
	"strings"

	"nexusdesk/internal/services/llm"
	settingssvc "nexusdesk/internal/services/settings"
	workspacesvc "nexusdesk/internal/services/workspace"
)

const charsPerTokenEstimate = 4

type SettingsStore interface {
	Load() (settingssvc.Settings, error)
}

type ContextPacker interface {
	BuildContextPack(root string, relPaths []string, options workspacesvc.ContextPackOptions) (workspacesvc.ContextPack, error)
}

type StreamClient interface {
	ChatStream(ctx context.Context, config llm.Config, chatRequest llm.ChatRequest, onDelta func(string) error) (llm.ChatResult, error)
}

type Service struct {
	settingsStore SettingsStore
	contextPacker ContextPacker
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
	return &Service{settingsStore: settingsStore, contextPacker: previewer, client: client}
}

func NewWithDependencies(settingsStore SettingsStore, contextPacker ContextPacker, client StreamClient) *Service {
	return &Service{settingsStore: settingsStore, contextPacker: contextPacker, client: client}
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
	if root == "" || selectedPath == "" || s.contextPacker == nil {
		return ""
	}
	pack, err := s.contextPacker.BuildContextPack(root, []string{selectedPath}, workspacesvc.ContextPackOptions{
		MaxBytes: contextBudgetBytes(config),
	})
	if err != nil {
		return "Selected context was not included: " + err.Error()
	}
	chatRequest.ContextRelPath = pack.Label
	chatRequest.ContextContent = pack.Content
	chatRequest.SourcePaths = append([]string{}, pack.SourcePaths...)
	if pack.Truncated {
		return "Selected context pack was capped to fit the model budget."
	}
	return ""
}

func contextBudgetBytes(config llm.Config) int {
	budgetTokens := config.ContextTokens - config.ResponseReserveTokens
	if budgetTokens <= 0 {
		return charsPerTokenEstimate
	}
	return budgetTokens * charsPerTokenEstimate
}
