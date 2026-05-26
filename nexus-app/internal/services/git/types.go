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
