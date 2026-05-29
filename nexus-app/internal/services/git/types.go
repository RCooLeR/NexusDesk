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
	StagedHunks           []DiffHunk
	UnstagedDiff          string
	UnstagedDiffTruncated bool
	UnstagedHunks         []DiffHunk
	Message               string
	GeneratedAt           time.Time
}

type DiffKind string

const (
	DiffKindStaged   DiffKind = "staged"
	DiffKindUnstaged DiffKind = "unstaged"
)

type DiffHunk struct {
	Kind         DiffKind
	Index        int
	Header       string
	OldStart     int
	OldLines     int
	NewStart     int
	NewLines     int
	AddedLines   int
	DeletedLines int
}

type FileAction string

const (
	FileActionStage   FileAction = "stage"
	FileActionUnstage FileAction = "unstage"
)

type HunkAction string

const (
	HunkActionStage   HunkAction = "stage"
	HunkActionUnstage HunkAction = "unstage"
)

type FileActionResult struct {
	Path        string
	Action      FileAction
	Message     string
	Status      Status
	GeneratedAt time.Time
}

type HunkActionResult struct {
	Path        string
	Action      HunkAction
	DiffKind    DiffKind
	HunkIndex   int
	Patch       string
	Message     string
	Status      Status
	GeneratedAt time.Time
}

type CommitResult struct {
	Hash        string
	ShortHash   string
	Subject     string
	Body        string
	StagedStat  string
	Message     string
	Status      Status
	GeneratedAt time.Time
}

type HistoryEntry struct {
	Hash      string
	ShortHash string
	Author    string
	Email     string
	Date      string
	Subject   string
}

type HistoryResult struct {
	Available   bool
	Path        string
	Limit       int
	Entries     []HistoryEntry
	Truncated   bool
	Message     string
	GeneratedAt time.Time
}

type BlameLine struct {
	Line      int
	Hash      string
	ShortHash string
	Author    string
	Date      string
	Summary   string
	Content   string
}

type BlameResult struct {
	Available   bool
	Path        string
	StartLine   int
	EndLine     int
	Lines       []BlameLine
	Truncated   bool
	Message     string
	GeneratedAt time.Time
}
