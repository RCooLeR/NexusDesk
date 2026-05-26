package main

type GitStatus struct {
	Available             bool            `json:"available"`
	RepoRoot              string          `json:"repoRoot"`
	Branch                string          `json:"branch"`
	Head                  string          `json:"head"`
	Dirty                 bool            `json:"dirty"`
	ChangedFiles          []GitFileChange `json:"changedFiles"`
	StagedFiles           []GitFileChange `json:"stagedFiles"`
	UnstagedFiles         []GitFileChange `json:"unstagedFiles"`
	Diff                  string          `json:"diff"`
	DiffTruncated         bool            `json:"diffTruncated"`
	StagedDiff            string          `json:"stagedDiff"`
	StagedDiffTruncated   bool            `json:"stagedDiffTruncated"`
	UnstagedDiff          string          `json:"unstagedDiff"`
	UnstagedDiffTruncated bool            `json:"unstagedDiffTruncated"`
	AheadBehind           string          `json:"aheadBehind"`
	Message               string          `json:"message"`
	GeneratedAt           string          `json:"generatedAt"`
}

type GitFileChange struct {
	Path     string `json:"path"`
	OldPath  string `json:"oldPath"`
	Index    string `json:"index"`
	Worktree string `json:"worktree"`
	Summary  string `json:"summary"`
}

type GitFileDiff struct {
	Path                  string `json:"path"`
	StagedDiff            string `json:"stagedDiff"`
	StagedDiffTruncated   bool   `json:"stagedDiffTruncated"`
	UnstagedDiff          string `json:"unstagedDiff"`
	UnstagedDiffTruncated bool   `json:"unstagedDiffTruncated"`
	Message               string `json:"message"`
	GeneratedAt           string `json:"generatedAt"`
}

type GitFileActionRequest struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

type GitFileActionPreview struct {
	Path              string    `json:"path"`
	Action            string    `json:"action"`
	Command           []string  `json:"command"`
	RequiresApproval  bool      `json:"requiresApproval"`
	MutatesRepository bool      `json:"mutatesRepository"`
	Message           string    `json:"message"`
	Status            GitStatus `json:"status"`
	GeneratedAt       string    `json:"generatedAt"`
}

type GitHunkActionRequest struct {
	Path      string `json:"path"`
	Action    string `json:"action"`
	DiffKind  string `json:"diffKind"`
	HunkIndex int    `json:"hunkIndex"`
}

type GitHunkActionPreview struct {
	Path              string    `json:"path"`
	Action            string    `json:"action"`
	DiffKind          string    `json:"diffKind"`
	HunkIndex         int       `json:"hunkIndex"`
	Command           []string  `json:"command"`
	Patch             string    `json:"patch"`
	RequiresApproval  bool      `json:"requiresApproval"`
	MutatesRepository bool      `json:"mutatesRepository"`
	Message           string    `json:"message"`
	Status            GitStatus `json:"status"`
	GeneratedAt       string    `json:"generatedAt"`
}

func (a *App) GetGitStatus() (GitStatus, error) {
	return newGitService(a.getWorkspaceRoot).Status()
}

func (a *App) GetGitFileDiff(relPath string) (GitFileDiff, error) {
	return newGitService(a.getWorkspaceRoot).FileDiff(relPath)
}

func (a *App) PreviewGitFileAction(request GitFileActionRequest) (GitFileActionPreview, error) {
	return newGitService(a.getWorkspaceRoot).PreviewFileAction(request)
}

func (a *App) ApplyGitFileAction(request GitFileActionRequest) (GitFileActionPreview, error) {
	preview, err := newGitService(a.getWorkspaceRoot).ApplyFileAction(request)
	if err != nil {
		return GitFileActionPreview{}, err
	}
	a.recordApproval("git.file."+preview.Action, preview.Path, "medium", preview.Message)
	return preview, nil
}

func (a *App) PreviewGitHunkAction(request GitHunkActionRequest) (GitHunkActionPreview, error) {
	return newGitService(a.getWorkspaceRoot).PreviewHunkAction(request)
}

func (a *App) ApplyGitHunkAction(request GitHunkActionRequest) (GitHunkActionPreview, error) {
	preview, err := newGitService(a.getWorkspaceRoot).ApplyHunkAction(request)
	if err != nil {
		return GitHunkActionPreview{}, err
	}
	a.recordApproval("git.hunk."+preview.Action, preview.Path, "high", preview.Message)
	return preview, nil
}
