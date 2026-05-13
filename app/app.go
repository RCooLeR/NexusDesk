package main

import "context"

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
	Capabilities   []Capability   `json:"capabilities"`
	WorkspaceItems []WorkspaceItem `json:"workspaceItems"`
	ToolEvents     []ToolEvent     `json:"toolEvents"`
}

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
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
