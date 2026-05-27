// Package metadata owns the native SQLite metadata store under .nexusdesk.
package metadata

import "time"

type Status struct {
	Path          string
	SchemaPath    string
	SchemaVersion int
	SchemaHash    string
	Tables        []string
	Message       string
	UpdatedAt     time.Time
}

type TaskRunRecord struct {
	ID           string
	JobID        string
	TaskID       string
	Kind         string
	Label        string
	Command      string
	Cwd          string
	Source       string
	Status       string
	ExitCode     int
	Stdout       string
	Stderr       string
	Message      string
	ArtifactPath string
	StartedAt    time.Time
	CompletedAt  time.Time
	DurationMs   int64
}

type AgentPlanStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

type AgentRunRecord struct {
	ID          string
	JobID       string
	Prompt      string
	Status      string
	Message     string
	Iterations  int
	StopReason  string
	Plan        []AgentPlanStep
	SourcePaths []string
	StartedAt   time.Time
	CompletedAt time.Time
	DurationMs  int64
}

type ToolRunRecord struct {
	ID          string
	AgentRunID  string
	JobID       string
	Sequence    int
	ToolName    string
	Risk        string
	Mutated     bool
	Args        map[string]string
	Observation string
	Error       string
	StartedAt   time.Time
	CompletedAt time.Time
}

type ChatMessageRecord struct {
	ID          string
	Role        string
	Content     string
	Model       string
	SourcePaths []string
	CreatedAt   time.Time
}

type ArtifactRecord struct {
	ID           string
	Kind         string
	Title        string
	RelPath      string
	MetadataPath string
	Size         int64
	JobID        string
	TaskID       string
	Source       string
	SourcePaths  []string
	Archived     bool
	CreatedAt    time.Time
	GeneratedAt  time.Time
	UpdatedAt    time.Time
}

type SQLRunRecord struct {
	ID           string
	RelPath      string
	SQL          string
	Engine       string
	Status       string
	RowCount     int
	MatchedRows  int
	ShownRows    int
	Message      string
	Error        string
	ArtifactPath string
	StartedAt    time.Time
	CompletedAt  time.Time
	DurationMs   int64
}

type DatasetDependencyRecord struct {
	ID            string
	SourcePath    string
	DependentKind string
	DependentRef  string
	Relation      string
	Metadata      map[string]string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
