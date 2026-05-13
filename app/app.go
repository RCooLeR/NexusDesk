package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"NexusDesk/internal/artifact"
	"NexusDesk/internal/dataset"
	"NexusDesk/internal/llm"
	"NexusDesk/internal/storage"
	"NexusDesk/internal/workspace"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Capability struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type WorkspaceItem struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Meta string `json:"meta"`
}

type ToolEvent struct {
	Time   string `json:"time"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type StartupState struct {
	ProductName    string          `json:"productName"`
	Tagline        string          `json:"tagline"`
	BuildStage     string          `json:"buildStage"`
	Capabilities   []Capability    `json:"capabilities"`
	WorkspaceItems []WorkspaceItem `json:"workspaceItems"`
	ToolEvents     []ToolEvent     `json:"toolEvents"`
}

type WorkspaceOpenResult struct {
	Selected bool                        `json:"selected"`
	Snapshot workspace.WorkspaceSnapshot `json:"snapshot"`
}

type ChatStreamEvent struct {
	RequestID      string `json:"requestId"`
	Type           string `json:"type"`
	Delta          string `json:"delta"`
	Message        string `json:"message"`
	Model          string `json:"model"`
	Endpoint       string `json:"endpoint"`
	ContextRelPath string `json:"contextRelPath"`
}

const chatContextMaxBytes = 16 * 1024
const chatCSVContextMaxRows = 20
const chatContextPackMaxFiles = 6
const chatContextPackMaxBytes = 32 * 1024
const chatStreamEventName = "nexusdesk:chat-stream"

type App struct {
	ctx           context.Context
	llmClient     *llm.Client
	chatStore     *storage.ChatHistoryStore
	llmStore      *storage.LLMSettingsStore
	recentStore   *storage.RecentWorkspaceStore
	workspaceMu   sync.RWMutex
	workspaceRoot string
}

func NewApp() *App {
	return &App{
		llmClient:   llm.NewClient(),
		chatStore:   storage.NewDefaultChatHistoryStore(),
		llmStore:    storage.NewDefaultLLMSettingsStore(),
		recentStore: storage.NewDefaultRecentWorkspaceStore(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetStartupState() StartupState {
	return StartupState{
		ProductName: "NexusDesk",
		Tagline:     "Local-first AI workbench for code, data, documents, and ops.",
		BuildStage:  "Workspace MVP scaffold",
		Capabilities: []Capability{
			{
				Title:       "Workspace browser",
				Description: "Open local folders, inspect files, and keep access inside approved roots.",
				Status:      "planned",
			},
			{
				Title:       "Configurable LLM chat",
				Description: "Connect a local or remote model and ground answers in selected context.",
				Status:      "planned",
			},
			{
				Title:       "Artifacts and approvals",
				Description: "Create reports, charts, and file changes through visible approval flows.",
				Status:      "planned",
			},
		},
		WorkspaceItems: []WorkspaceItem{
			{Name: "app", Kind: "folder", Meta: "Wails desktop shell"},
			{Name: "docs", Kind: "folder", Meta: "Product and engineering source of truth"},
			{Name: "services", Kind: "folder", Meta: "Development helper services"},
		},
		ToolEvents: []ToolEvent{
			{Time: "now", Title: "Scaffold ready", Detail: "React + TypeScript frontend bound to Go backend."},
			{Time: "next", Title: "Workspace opening", Detail: "Add a safe folder picker and file tree."},
			{Time: "then", Title: "LLM settings", Detail: "Store provider URL, model, key, and capabilities."},
		},
	}
}

func (a *App) SelectWorkspace() (WorkspaceOpenResult, error) {
	if a.ctx == nil {
		return WorkspaceOpenResult{}, errors.New("application is not ready")
	}

	root, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open NexusDesk Workspace",
	})
	if err != nil {
		return WorkspaceOpenResult{}, err
	}

	if root == "" {
		return WorkspaceOpenResult{Selected: false}, nil
	}

	return a.openWorkspace(root)
}

func (a *App) OpenWorkspace(root string) (WorkspaceOpenResult, error) {
	if root == "" {
		return WorkspaceOpenResult{Selected: false}, nil
	}

	return a.openWorkspace(root)
}

func (a *App) RefreshWorkspace() (WorkspaceOpenResult, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return WorkspaceOpenResult{Selected: false}, nil
	}

	snapshot, err := workspace.Scan(root, workspace.ScanOptions{})
	if err != nil {
		return WorkspaceOpenResult{}, err
	}

	return WorkspaceOpenResult{
		Selected: true,
		Snapshot: snapshot,
	}, nil
}

func (a *App) ReadWorkspaceFile(relPath string) (workspace.FilePreview, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FilePreview{}, errors.New("open a workspace before reading files")
	}

	return workspace.Preview(root, relPath, workspace.PreviewOptions{})
}

func (a *App) PreviewFileWrite(request workspace.FileWriteRequest) (workspace.FileWriteProposal, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FileWriteProposal{}, errors.New("open a workspace before previewing file writes")
	}

	return workspace.PreviewFileWrite(root, request)
}

func (a *App) ApplyFileWrite(request workspace.FileWriteRequest) (workspace.FileWriteProposal, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FileWriteProposal{}, errors.New("open a workspace before applying file writes")
	}

	return workspace.ApplyFileWrite(root, request)
}

func (a *App) CreateMarkdownReport(relPath string) (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before creating reports")
	}

	source := workspace.FilePreview{
		RelPath: relPath,
		Name:    "workspace-report",
	}
	if relPath != "" {
		preview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: chatContextMaxBytes})
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		source = preview
	}

	return artifact.CreateMarkdownReport(root, source, time.Now())
}

func (a *App) ListArtifacts() ([]artifact.WorkspaceArtifact, error) {
	return artifact.List(a.getWorkspaceRoot())
}

func (a *App) ProfileDataset(relPath string) (dataset.Profile, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return dataset.Profile{}, errors.New("open a workspace before profiling datasets")
	}
	return dataset.Build(root, relPath)
}

func (a *App) ListDatasetProfiles() ([]dataset.Profile, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []dataset.Profile{}, nil
	}
	return dataset.List(root)
}

func (a *App) GetRecentWorkspaces() ([]storage.RecentWorkspace, error) {
	return a.recentStore.List()
}

func (a *App) RemoveRecentWorkspace(path string) ([]storage.RecentWorkspace, error) {
	return a.recentStore.Remove(path)
}

func (a *App) ClearRecentWorkspaces() ([]storage.RecentWorkspace, error) {
	return a.recentStore.Clear()
}

func (a *App) GetLLMSettings() (storage.LLMSettings, error) {
	return a.llmStore.Get()
}

func (a *App) SaveLLMSettings(settings storage.LLMSettings) (storage.LLMSettings, error) {
	return a.llmStore.Save(settings)
}

func (a *App) TestLLMConnection(settings storage.LLMSettings) (llm.ProbeResult, error) {
	resolvedSettings, err := a.llmStore.ResolveForUse(settings)
	if err != nil {
		return llm.ProbeResult{}, err
	}

	return a.llmClient.Probe(context.Background(), resolvedSettings)
}

func (a *App) GetChatHistory() ([]storage.ChatMessage, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []storage.ChatMessage{}, nil
	}

	return a.chatStore.List(root)
}

func (a *App) ClearChatHistory() ([]storage.ChatMessage, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []storage.ChatMessage{}, nil
	}

	return a.chatStore.Clear(root)
}

func (a *App) AskLLM(prompt string, relPath string) (llm.ChatResult, error) {
	chatRequest, settings, err := a.prepareChat(prompt, []string{relPath})
	if err != nil {
		return llm.ChatResult{}, err
	}

	result, err := a.llmClient.Chat(context.Background(), settings, chatRequest)
	if err != nil {
		return llm.ChatResult{}, err
	}

	if err := a.persistChatPair(prompt, chatRequest, result); err != nil {
		return llm.ChatResult{}, err
	}

	return result, nil
}

func (a *App) AskLLMStream(prompt string, relPath string, requestID string) (llm.ChatResult, error) {
	chatRequest, settings, err := a.prepareChat(prompt, []string{relPath})
	if err != nil {
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: err.Error()})
		return llm.ChatResult{}, err
	}

	result, err := a.llmClient.ChatStream(context.Background(), settings, chatRequest, func(delta string) error {
		a.emitChatStreamEvent(ChatStreamEvent{
			RequestID:      requestID,
			Type:           "delta",
			Delta:          delta,
			ContextRelPath: chatRequest.ContextRelPath,
		})
		return nil
	})
	if err != nil {
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: err.Error()})
		return llm.ChatResult{}, err
	}

	if err := a.persistChatPair(prompt, chatRequest, result); err != nil {
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: err.Error()})
		return llm.ChatResult{}, err
	}

	a.emitChatStreamEvent(ChatStreamEvent{
		RequestID:      requestID,
		Type:           "done",
		Message:        result.Message,
		Model:          result.Model,
		Endpoint:       result.Endpoint,
		ContextRelPath: result.ContextRelPath,
	})

	return result, nil
}

func (a *App) AskLLMContextPack(prompt string, relPaths []string) (llm.ChatResult, error) {
	chatRequest, settings, err := a.prepareChat(prompt, relPaths)
	if err != nil {
		return llm.ChatResult{}, err
	}

	result, err := a.llmClient.Chat(context.Background(), settings, chatRequest)
	if err != nil {
		return llm.ChatResult{}, err
	}
	if err := a.persistChatPair(prompt, chatRequest, result); err != nil {
		return llm.ChatResult{}, err
	}
	return result, nil
}

func (a *App) AskLLMStreamContextPack(prompt string, relPaths []string, requestID string) (llm.ChatResult, error) {
	chatRequest, settings, err := a.prepareChat(prompt, relPaths)
	if err != nil {
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: err.Error()})
		return llm.ChatResult{}, err
	}

	result, err := a.llmClient.ChatStream(context.Background(), settings, chatRequest, func(delta string) error {
		a.emitChatStreamEvent(ChatStreamEvent{
			RequestID:      requestID,
			Type:           "delta",
			Delta:          delta,
			ContextRelPath: chatRequest.ContextRelPath,
		})
		return nil
	})
	if err != nil {
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: err.Error()})
		return llm.ChatResult{}, err
	}
	if err := a.persistChatPair(prompt, chatRequest, result); err != nil {
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: err.Error()})
		return llm.ChatResult{}, err
	}
	a.emitChatStreamEvent(ChatStreamEvent{
		RequestID:      requestID,
		Type:           "done",
		Message:        result.Message,
		Model:          result.Model,
		Endpoint:       result.Endpoint,
		ContextRelPath: result.ContextRelPath,
	})
	return result, nil
}

func (a *App) prepareChat(prompt string, relPaths []string) (llm.ChatRequest, storage.LLMSettings, error) {
	settings, err := a.llmStore.Get()
	if err != nil {
		return llm.ChatRequest{}, storage.LLMSettings{}, err
	}

	resolvedSettings, err := a.llmStore.ResolveForUse(settings)
	if err != nil {
		return llm.ChatRequest{}, storage.LLMSettings{}, err
	}

	chatRequest := llm.ChatRequest{
		Prompt: prompt,
	}

	contextPaths := cleanContextPaths(relPaths)
	if len(contextPaths) == 1 {
		contextPreview, err := a.previewChatContext(contextPaths[0])
		if err != nil {
			return llm.ChatRequest{}, storage.LLMSettings{}, err
		}
		chatRequest.ContextRelPath = contextPreview.RelPath
		chatRequest.ContextContent = contextPreview.Content
	} else if len(contextPaths) > 1 {
		contextRelPath, contextContent, err := a.buildContextPack(contextPaths)
		if err != nil {
			return llm.ChatRequest{}, storage.LLMSettings{}, err
		}
		chatRequest.ContextRelPath = contextRelPath
		chatRequest.ContextContent = contextContent
	}

	return chatRequest, resolvedSettings, nil
}

func (a *App) persistChatPair(prompt string, chatRequest llm.ChatRequest, result llm.ChatResult) error {
	root := a.getWorkspaceRoot()
	if root != "" {
		_, err := a.chatStore.AppendPair(root, storage.ChatMessage{
			Role:           "user",
			Content:        prompt,
			ContextRelPath: chatRequest.ContextRelPath,
		}, storage.ChatMessage{
			Role:           "assistant",
			Content:        result.Message,
			ContextRelPath: result.ContextRelPath,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *App) emitChatStreamEvent(event ChatStreamEvent) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, chatStreamEventName, event)
}

func (a *App) previewChatContext(relPath string) (workspace.FilePreview, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FilePreview{}, errors.New("open a workspace before sending selected file context")
	}

	contextPreview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: chatContextMaxBytes})
	if err != nil {
		return workspace.FilePreview{}, err
	}
	if contextPreview.Content == "" {
		return workspace.FilePreview{}, errors.New("selected file cannot be sent as text context")
	}
	if contextPreview.Kind == "pdf" && strings.TrimSpace(contextPreview.Text) != "" {
		contextPreview.Content = contextPreview.Text
		return contextPreview, nil
	}
	if contextPreview.Kind != "file" {
		return workspace.FilePreview{}, errors.New("selected file context must be a text preview")
	}

	contextPreview.Content = buildChatContextContent(contextPreview)
	if strings.TrimSpace(contextPreview.Content) == "" {
		return workspace.FilePreview{}, errors.New("selected file cannot be sent as text context")
	}

	return contextPreview, nil
}

func buildChatContextContent(preview workspace.FilePreview) string {
	if preview.Table == nil {
		return preview.Content
	}

	var builder strings.Builder
	builder.WriteString("CSV context summary\n\n")
	builder.WriteString("Columns:\n")
	for _, profile := range preview.Table.Profiles {
		builder.WriteString("- ")
		builder.WriteString(profile.Name)
		builder.WriteString(": ")
		builder.WriteString(profile.Type)
		builder.WriteString(fmt.Sprintf(", distinct=%d, missing=%d", profile.Distinct, profile.Missing))
		if profile.Min != "" || profile.Max != "" {
			builder.WriteString(", range=")
			builder.WriteString(profile.Min)
			builder.WriteString("..")
			builder.WriteString(profile.Max)
		}
		builder.WriteString("\n")
	}

	builder.WriteString("\nSample rows:\n")
	csvWriter := csv.NewWriter(&builder)
	_ = csvWriter.Write(preview.Table.Columns)
	for index, row := range preview.Table.Rows {
		if index >= chatCSVContextMaxRows {
			break
		}
		_ = csvWriter.Write(row)
	}
	csvWriter.Flush()
	if preview.Table.Truncated || len(preview.Table.Rows) > chatCSVContextMaxRows {
		builder.WriteString("\nCSV context sample was truncated.\n")
	}

	return builder.String()
}

func (a *App) buildContextPack(relPaths []string) (string, string, error) {
	if len(relPaths) > chatContextPackMaxFiles {
		relPaths = relPaths[:chatContextPackMaxFiles]
	}

	var builder strings.Builder
	usedPaths := []string{}
	for _, relPath := range relPaths {
		preview, err := a.previewChatContext(relPath)
		if err != nil {
			return "", "", err
		}

		entry := "\n\n# Workspace context: " + preview.RelPath + "\n\n" + preview.Content
		remaining := chatContextPackMaxBytes - builder.Len()
		if remaining <= 0 {
			break
		}
		truncated := len(entry) > remaining
		if truncated {
			entry = truncateContextString(entry, remaining)
		}

		builder.WriteString(entry)
		usedPaths = append(usedPaths, preview.RelPath)
		if truncated {
			builder.WriteString("\n\n_Context pack truncated._\n")
			break
		}
	}
	if len(usedPaths) == 0 {
		return "", "", errors.New("context pack did not include usable text")
	}
	return "pack: " + strings.Join(usedPaths, ", "), strings.TrimSpace(builder.String()), nil
}

func truncateContextString(content string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(content) <= maxBytes {
		return content
	}

	truncated := content[:maxBytes]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}

func cleanContextPaths(relPaths []string) []string {
	seen := map[string]bool{}
	cleaned := []string{}
	for _, relPath := range relPaths {
		relPath = strings.TrimSpace(relPath)
		if relPath == "" || seen[relPath] {
			continue
		}
		seen[relPath] = true
		cleaned = append(cleaned, relPath)
	}
	return cleaned
}

func (a *App) openWorkspace(root string) (WorkspaceOpenResult, error) {
	info, err := os.Stat(root)
	if err != nil {
		return WorkspaceOpenResult{}, err
	}
	if !info.IsDir() {
		return WorkspaceOpenResult{}, errors.New("workspace root must be a directory")
	}

	snapshot, err := workspace.Scan(root, workspace.ScanOptions{})
	if err != nil {
		return WorkspaceOpenResult{}, err
	}

	a.setWorkspaceRoot(snapshot.Root)
	if _, err := a.recentStore.Add(snapshot.Root); err != nil {
		return WorkspaceOpenResult{}, err
	}

	return WorkspaceOpenResult{
		Selected: true,
		Snapshot: snapshot,
	}, nil
}

func (a *App) setWorkspaceRoot(root string) {
	a.workspaceMu.Lock()
	defer a.workspaceMu.Unlock()
	a.workspaceRoot = root
}

func (a *App) getWorkspaceRoot() string {
	a.workspaceMu.RLock()
	defer a.workspaceMu.RUnlock()
	return a.workspaceRoot
}
