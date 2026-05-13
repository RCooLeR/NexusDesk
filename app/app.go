package main

import (
	"context"
	"errors"
	"os"
	"sync"

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

type App struct {
	ctx           context.Context
	llmClient     *llm.Client
	llmStore      *storage.LLMSettingsStore
	recentStore   *storage.RecentWorkspaceStore
	workspaceMu   sync.RWMutex
	workspaceRoot string
}

func NewApp() *App {
	return &App{
		llmClient:   llm.NewClient(),
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

func (a *App) GetRecentWorkspaces() ([]storage.RecentWorkspace, error) {
	return a.recentStore.List()
}

func (a *App) GetLLMSettings() (storage.LLMSettings, error) {
	return a.llmStore.Get()
}

func (a *App) SaveLLMSettings(settings storage.LLMSettings) (storage.LLMSettings, error) {
	return a.llmStore.Save(settings)
}

func (a *App) TestLLMConnection(settings storage.LLMSettings) (llm.ProbeResult, error) {
	return a.llmClient.Probe(context.Background(), settings)
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
