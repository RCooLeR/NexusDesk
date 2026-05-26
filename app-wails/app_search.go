package main

import "NexusAugenticStudio/internal/workspace"

type WorkspaceSearchRequest struct {
	Query   string `json:"query"`
	Regex   bool   `json:"regex"`
	Symbols bool   `json:"symbols"`
}

func (a *App) SearchWorkspaceAdvanced(request WorkspaceSearchRequest) ([]workspace.SearchResult, error) {
	return a.workspaceSvc.SearchAdvanced(request)
}
