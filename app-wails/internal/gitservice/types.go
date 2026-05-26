package gitservice

type Status struct {
	Available             bool         `json:"available"`
	RepoRoot              string       `json:"repoRoot"`
	Branch                string       `json:"branch"`
	Head                  string       `json:"head"`
	Dirty                 bool         `json:"dirty"`
	ChangedFiles          []FileChange `json:"changedFiles"`
	StagedFiles           []FileChange `json:"stagedFiles"`
	UnstagedFiles         []FileChange `json:"unstagedFiles"`
	Diff                  string       `json:"diff"`
	DiffTruncated         bool         `json:"diffTruncated"`
	StagedDiff            string       `json:"stagedDiff"`
	StagedDiffTruncated   bool         `json:"stagedDiffTruncated"`
	UnstagedDiff          string       `json:"unstagedDiff"`
	UnstagedDiffTruncated bool         `json:"unstagedDiffTruncated"`
	AheadBehind           string       `json:"aheadBehind"`
	Message               string       `json:"message"`
	GeneratedAt           string       `json:"generatedAt"`
}

type FileChange struct {
	Path     string `json:"path"`
	OldPath  string `json:"oldPath"`
	Index    string `json:"index"`
	Worktree string `json:"worktree"`
	Summary  string `json:"summary"`
}

type FileDiff struct {
	Path                  string `json:"path"`
	StagedDiff            string `json:"stagedDiff"`
	StagedDiffTruncated   bool   `json:"stagedDiffTruncated"`
	UnstagedDiff          string `json:"unstagedDiff"`
	UnstagedDiffTruncated bool   `json:"unstagedDiffTruncated"`
	Message               string `json:"message"`
	GeneratedAt           string `json:"generatedAt"`
}

type FileActionRequest struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

type FileActionPreview struct {
	Path              string   `json:"path"`
	Action            string   `json:"action"`
	Command           []string `json:"command"`
	RequiresApproval  bool     `json:"requiresApproval"`
	MutatesRepository bool     `json:"mutatesRepository"`
	Message           string   `json:"message"`
	Status            Status   `json:"status"`
	GeneratedAt       string   `json:"generatedAt"`
}

type HunkActionRequest struct {
	Path      string `json:"path"`
	Action    string `json:"action"`
	DiffKind  string `json:"diffKind"`
	HunkIndex int    `json:"hunkIndex"`
}

type HunkActionPreview struct {
	Path              string   `json:"path"`
	Action            string   `json:"action"`
	DiffKind          string   `json:"diffKind"`
	HunkIndex         int      `json:"hunkIndex"`
	Command           []string `json:"command"`
	Patch             string   `json:"patch"`
	RequiresApproval  bool     `json:"requiresApproval"`
	MutatesRepository bool     `json:"mutatesRepository"`
	Message           string   `json:"message"`
	Status            Status   `json:"status"`
	GeneratedAt       string   `json:"generatedAt"`
}

type HistoryRequest struct {
	Path  string `json:"path"`
	Limit int    `json:"limit"`
}

type HistoryEntry struct {
	Hash      string `json:"hash"`
	ShortHash string `json:"shortHash"`
	Author    string `json:"author"`
	Email     string `json:"email"`
	Date      string `json:"date"`
	Subject   string `json:"subject"`
}

type HistoryResult struct {
	Available   bool           `json:"available"`
	Path        string         `json:"path"`
	Limit       int            `json:"limit"`
	Entries     []HistoryEntry `json:"entries"`
	Truncated   bool           `json:"truncated"`
	Message     string         `json:"message"`
	GeneratedAt string         `json:"generatedAt"`
}

type BlameRequest struct {
	Path      string `json:"path"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
}

type BlameLine struct {
	Line      int    `json:"line"`
	Hash      string `json:"hash"`
	ShortHash string `json:"shortHash"`
	Author    string `json:"author"`
	Date      string `json:"date"`
	Summary   string `json:"summary"`
	Content   string `json:"content"`
}

type BlameResult struct {
	Available   bool        `json:"available"`
	Path        string      `json:"path"`
	StartLine   int         `json:"startLine"`
	EndLine     int         `json:"endLine"`
	Lines       []BlameLine `json:"lines"`
	Truncated   bool        `json:"truncated"`
	Message     string      `json:"message"`
	GeneratedAt string      `json:"generatedAt"`
}
