package git

import "time"

type Status struct {
	Available     bool
	RepoRoot      string
	Branch        string
	Head          string
	Dirty         bool
	ChangedFiles  []FileChange
	StagedFiles   []FileChange
	UnstagedFiles []FileChange
	AheadBehind   string
	Message       string
	GeneratedAt   time.Time
}

type FileChange struct {
	Path     string
	OldPath  string
	Index    string
	Worktree string
	Summary  string
}

type FileDiff struct {
	Path                  string
	StagedDiff            string
	StagedDiffTruncated   bool
	UnstagedDiff          string
	UnstagedDiffTruncated bool
	Message               string
	GeneratedAt           time.Time
}

type FileAction string

const (
	FileActionStage   FileAction = "stage"
	FileActionUnstage FileAction = "unstage"
)

type FileActionResult struct {
	Path        string
	Action      FileAction
	Message     string
	Status      Status
	GeneratedAt time.Time
}
