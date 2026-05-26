package main

import "NexusAugenticStudio/internal/gitservice"

type GitStatus = gitservice.Status
type GitFileChange = gitservice.FileChange
type GitFileDiff = gitservice.FileDiff
type GitFileActionRequest = gitservice.FileActionRequest
type GitFileActionPreview = gitservice.FileActionPreview
type GitHunkActionRequest = gitservice.HunkActionRequest
type GitHunkActionPreview = gitservice.HunkActionPreview
type GitHistoryRequest = gitservice.HistoryRequest
type GitHistoryEntry = gitservice.HistoryEntry
type GitHistoryResult = gitservice.HistoryResult
type GitBlameRequest = gitservice.BlameRequest
type GitBlameLine = gitservice.BlameLine
type GitBlameResult = gitservice.BlameResult

func (a *App) GetGitStatus() (GitStatus, error) {
	return gitservice.New(a.getWorkspaceRoot).Status()
}

func (a *App) GetGitFileDiff(relPath string) (GitFileDiff, error) {
	return gitservice.New(a.getWorkspaceRoot).FileDiff(relPath)
}

func (a *App) PreviewGitFileAction(request GitFileActionRequest) (GitFileActionPreview, error) {
	return gitservice.New(a.getWorkspaceRoot).PreviewFileAction(request)
}

func (a *App) ApplyGitFileAction(request GitFileActionRequest) (GitFileActionPreview, error) {
	preview, err := gitservice.New(a.getWorkspaceRoot).ApplyFileAction(request)
	if err != nil {
		return GitFileActionPreview{}, err
	}
	a.recordApproval("git.file."+preview.Action, preview.Path, "medium", preview.Message)
	return preview, nil
}

func (a *App) PreviewGitHunkAction(request GitHunkActionRequest) (GitHunkActionPreview, error) {
	return gitservice.New(a.getWorkspaceRoot).PreviewHunkAction(request)
}

func (a *App) ApplyGitHunkAction(request GitHunkActionRequest) (GitHunkActionPreview, error) {
	preview, err := gitservice.New(a.getWorkspaceRoot).ApplyHunkAction(request)
	if err != nil {
		return GitHunkActionPreview{}, err
	}
	a.recordApproval("git.hunk."+preview.Action, preview.Path, "high", preview.Message)
	return preview, nil
}

func (a *App) GetGitHistory(request GitHistoryRequest) (GitHistoryResult, error) {
	return gitservice.New(a.getWorkspaceRoot).History(request)
}

func (a *App) GetGitBlame(request GitBlameRequest) (GitBlameResult, error) {
	return gitservice.New(a.getWorkspaceRoot).Blame(request)
}
