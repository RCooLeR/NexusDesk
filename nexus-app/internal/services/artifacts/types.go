// Package artifacts owns deterministic generated files under .nexusdesk/artifacts.
package artifacts

import "time"

type Artifact struct {
	Kind         string
	Title        string
	RelPath      string
	AbsPath      string
	MetadataPath string
	Message      string
	Size         int64
	CreatedAt    time.Time
	GeneratedAt  time.Time
	JobID        string
	TaskID       string
	Source       string
	SourcePaths  []string
	Archived     bool
}

type TaskRunReport struct {
	ID          string
	JobID       string
	TaskID      string
	Kind        string
	Label       string
	Command     string
	Cwd         string
	Source      string
	Status      string
	ExitCode    int
	Stdout      string
	Stderr      string
	Message     string
	StartedAt   time.Time
	CompletedAt time.Time
	DurationMs  int64
}

type DocumentSetReport struct {
	Title       string
	Roots       []string
	SourcePaths []string
	Content     string
	Truncated   bool
	GeneratedBy string
}

type DocumentExtractionReport struct {
	Title     string
	RelPath   string
	Format    string
	MediaType string
	Encoding  string
	Content   string
	Size      int64
	Lines     int
	Words     int
	Pages     int
	Truncated bool
}

type ListOptions struct {
	Query           string
	IncludeArchived bool
}

type Metadata struct {
	Kind        string    `json:"kind"`
	Title       string    `json:"title"`
	RelPath     string    `json:"relPath"`
	JobID       string    `json:"jobId,omitempty"`
	TaskID      string    `json:"taskId,omitempty"`
	Source      string    `json:"source,omitempty"`
	SourcePaths []string  `json:"sourcePaths,omitempty"`
	GeneratedAt time.Time `json:"generatedAt"`
}

type Lineage struct {
	Nodes []LineageNode
	Edges []LineageEdge
}

type LineageNode struct {
	ID    string
	Kind  string
	Label string
}

type LineageEdge struct {
	From  string
	To    string
	Label string
}

type ArtifactComparison struct {
	Kind       string
	LeftPath   string
	RightPath  string
	LeftTitle  string
	RightTitle string
	Diff       string
	Same       bool
	Message    string
}

type SourceFreshness struct {
	ArtifactRelPath string
	GeneratedAt     time.Time
	Sources         []SourceFreshnessStatus
	ChangedCount    int
	MissingCount    int
	UnknownCount    int
	Stale           bool
	Message         string
}

type SourceFreshnessStatus struct {
	RelPath    string
	Exists     bool
	Changed    bool
	Unknown    bool
	ModifiedAt time.Time
	Message    string
}
