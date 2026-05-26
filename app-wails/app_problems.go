package main

import "NexusAugenticStudio/internal/workspace"

func (a *App) ListWorkspaceProblems() (workspace.ProblemSummary, error) {
	return a.workspaceSvc.Problems()
}
