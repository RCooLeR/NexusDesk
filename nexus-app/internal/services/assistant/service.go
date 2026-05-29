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
const defaultContextMaxBytes = 96 * 1024

const askSystemPrompt = "You are Nexus Ask mode inside NexusDesk. Answer directly from the user request and provided workspace context. If more source context is needed, say what to select or inspect next. Do not claim access to files that were not provided, and do not request or describe tool execution."

type SettingsStore interface {
	Load() (settingssvc.Settings, error)
}

type ContextPacker interface {
	BuildContextPack(root string, relPaths []string, options workspacesvc.ContextPackOptions) (workspacesvc.ContextPack, error)
}

type StreamClient interface {
	ChatStream(ctx context.Context, config llm.Config, chatRequest llm.ChatRequest, onDelta func(string) error) (llm.ChatResult, error)
}

type ProfileReader interface {
	Get() (Profile, error)
}

type Service struct {
	settingsStore SettingsStore
	contextPacker ContextPacker
	client        StreamClient
	profileStore  ProfileReader
}

type Request struct {
	Prompt        string
	WorkspaceRoot string
	SelectedPath  string
	ContextPaths  []string
	Conversation  []llm.ChatTurn
	ModelRouteID  string
}

type Result struct {
	Message        string
	Model          string
	Endpoint       string
	ContextRelPath string
	SourcePaths    []string
	ContextWarning string
	ModelRouteID   string
	ModelRoute     string
	RouteWarning   string
}

func New(settingsStore *settingssvc.Store, previewer *workspacesvc.Service, client *llm.Client) *Service {
	return &Service{settingsStore: settingsStore, contextPacker: previewer, client: client}
}

func NewWithDependencies(settingsStore SettingsStore, contextPacker ContextPacker, client StreamClient) *Service {
	return &Service{settingsStore: settingsStore, contextPacker: contextPacker, client: client}
}

func (s *Service) SetProfileStore(store ProfileReader) {
	s.profileStore = store
}

func (s *Service) AskStream(ctx context.Context, request Request, onDelta func(string) error) (Result, error) {
	settings, err := s.settingsStore.Load()
	if err != nil {
		return Result{}, err
	}
	settings, routeInfo := s.settingsForRequest(settings, request)
	config := llm.ConfigFromSettings(settings)
	chatRequest := llm.ChatRequest{
		SystemPrompt: askSystemPrompt,
		Prompt:       s.promptWithProfile(request.Prompt),
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
		ModelRouteID:   routeInfo.ID,
		ModelRoute:     routeInfo.Label,
		RouteWarning:   routeInfo.Warning,
	}, nil
}

type routeResolution struct {
	ID      string
	Label   string
	Warning string
}

func (s *Service) settingsForRequest(settings settingssvc.Settings, request Request) (settingssvc.Settings, routeResolution) {
	routeID := strings.TrimSpace(request.ModelRouteID)
	if routeID == "" {
		return settings, routeResolution{}
	}
	route, ok := settingssvc.ModelRouteByID(settings, routeID)
	if !ok {
		return settings, routeResolution{
			ID:      routeID,
			Warning: "Model route " + routeID + " was not found; using the global model.",
		}
	}
	routed, ok := settingssvc.SettingsForModelRoute(settings, routeID)
	if !ok {
		return settings, routeResolution{
			ID:      routeID,
			Label:   route.Label,
			Warning: "Model route " + firstNonEmptyString(route.Label, routeID) + " could not be resolved; using the global model.",
		}
	}
	return routed, routeResolution{
		ID:    route.ID,
		Label: route.Label,
	}
}

func (s *Service) promptWithProfile(prompt string) string {
	if s.profileStore == nil {
		return prompt
	}
	profile, err := s.profileStore.Get()
	if err != nil {
		return prompt
	}
	return ApplyProfileToPrompt(prompt, profile)
}

func (s *Service) attachSelectedContext(config llm.Config, request Request, chatRequest *llm.ChatRequest) string {
	root := strings.TrimSpace(request.WorkspaceRoot)
	contextPaths := requestedContextPaths(request)
	if root == "" || len(contextPaths) == 0 || s.contextPacker == nil {
		return ""
	}
	pack, err := s.contextPacker.BuildContextPack(root, contextPaths, workspacesvc.ContextPackOptions{
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

func requestedContextPaths(request Request) []string {
	seen := map[string]bool{}
	paths := make([]string, 0, len(request.ContextPaths)+1)
	for _, relPath := range request.ContextPaths {
		relPath = strings.TrimSpace(relPath)
		if relPath == "" || seen[relPath] {
			continue
		}
		seen[relPath] = true
		paths = append(paths, relPath)
	}
	selectedPath := strings.TrimSpace(request.SelectedPath)
	if len(paths) == 0 && selectedPath != "" {
		paths = append(paths, selectedPath)
	}
	return paths
}

func contextBudgetBytes(config llm.Config) int {
	budgetTokens := config.ContextTokens - config.ResponseReserveTokens
	if budgetTokens <= 0 {
		return defaultContextMaxBytes
	}
	return budgetTokens * charsPerTokenEstimate
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
