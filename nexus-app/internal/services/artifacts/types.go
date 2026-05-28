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
	Fingerprints []SourceFingerprint
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

type WorkspaceScanReport struct {
	Title          string
	WorkspaceName  string
	Included       int
	Ignored        int
	DepthSkipped   int
	EntrySkipped   int
	Unreadable     int
	MaxDepth       int
	MaxEntries     int
	Truncated      bool
	IgnoredSamples []string
	SkippedSamples []string
	Message        string
}

type ChartArtifactReport struct {
	Title          string
	SourcePath     string
	Query          string
	Format         string
	Mode           string
	CategoryColumn string
	ValueColumn    string
	SVG            string
	PointCount     int
	Truncated      bool
}

type NotebookRunReport struct {
	Title       string
	SourcePath  string
	NotebookID  string
	Label       string
	Message     string
	StartedAt   time.Time
	CompletedAt time.Time
	DurationMs  int64
	Cells       []NotebookRunCellReport
}

type NotebookRunCellReport struct {
	CellID       string
	Label        string
	Kind         string
	SQL          string
	Status       string
	Error        string
	Engine       string
	Columns      []string
	Rows         [][]string
	MatchedRows  int
	ShownRows    int
	Plan         []string
	ChartMode    string
	ChartMessage string
	ChartSVG     string
	ChartPoints  int
	StartedAt    time.Time
	CompletedAt  time.Time
	DurationMs   int64
}

type SQLiteQueryReport struct {
	Title          string
	SourcePath     string
	SQL            string
	Engine         string
	Columns        []string
	Rows           [][]string
	TotalRows      int
	ResultLimit    int
	TimeoutSeconds int
	DurationMs     int64
	Truncated      bool
	Message        string
}

type DatasetQueryReport struct {
	Title       string
	SourcePath  string
	Query       string
	Format      string
	Columns     []string
	Rows        [][]string
	TotalRows   int
	MatchedRows int
	Truncated   bool
	Message     string
}

type DatasetSQLReport struct {
	Title       string
	SourcePath  string
	SQL         string
	Engine      string
	Columns     []string
	Rows        [][]string
	TotalRows   int
	MatchedRows int
	ShownRows   int
	DurationMs  int64
	Truncated   bool
	Plan        []string
	Message     string
}

type DatasetSummaryReport struct {
	Title      string
	SourcePath string
	Format     string
	MediaType  string
	Size       int64
	Rows       int
	Columns    []DatasetSummaryColumnReport
	Sheet      string
	Sheets     []string
	Truncated  bool
	Notes      []string
}

type DatasetSummaryColumnReport struct {
	Name     string
	Type     string
	NonEmpty int
	Empty    int
	Samples  []string
}

type OperationsRunbookReport struct {
	Title           string
	SourcePath      string
	Kind            string
	Size            int64
	Content         string
	Services        []OperationsServiceSummary
	TopologySummary string
	TopologyEdges   []OperationsTopologyEdge
	ExposedPorts    []OperationsPortExposure
	NamedVolumes    []string
	Warnings        []string
	Truncated       bool
	GeneratedBy     string
}

type ChatAnswerReport struct {
	Title                  string
	Prompt                 string
	Content                string
	Source                 string
	ContextRelPath         string
	Model                  string
	SourcePaths            []string
	CitationRefs           []string
	UnverifiedCitationRefs []string
	CitationSnippets       []string
	CitedSourcePaths       []string
	UncitedSourcePaths     []string
	EvidenceQuality        string
	EvidenceSummary        string
}

type PresentationOutlineReport struct {
	Title       string
	SourcePath  string
	SourceTitle string
	SourceKind  string
	SourcePaths []string
	Content     string
	SlideCount  int
	GeneratedBy string
}

type PresentationPackageReport struct {
	Title       string
	SourcePath  string
	SourceTitle string
	SourceKind  string
	SourcePaths []string
	Outline     string
	SlideCount  int
	GeneratedBy string
}

type OperationsServiceSummary struct {
	Name      string
	Image     string
	Ports     []string
	Volumes   []string
	DependsOn []string
}

type OperationsTopologyEdge struct {
	From     string
	To       string
	Relation string
	Missing  bool
}

type OperationsPortExposure struct {
	Service string
	Port    string
}

type ListOptions struct {
	Query           string
	IncludeArchived bool
}

type Metadata struct {
	Kind                   string              `json:"kind"`
	Title                  string              `json:"title"`
	RelPath                string              `json:"relPath"`
	JobID                  string              `json:"jobId,omitempty"`
	TaskID                 string              `json:"taskId,omitempty"`
	Source                 string              `json:"source,omitempty"`
	ContextRelPath         string              `json:"contextRelPath,omitempty"`
	Prompt                 string              `json:"prompt,omitempty"`
	Model                  string              `json:"model,omitempty"`
	SourcePaths            []string            `json:"sourcePaths,omitempty"`
	CitationRefs           []string            `json:"citationRefs,omitempty"`
	UnverifiedCitationRefs []string            `json:"unverifiedCitationRefs,omitempty"`
	CitationSnippets       []string            `json:"citationSnippets,omitempty"`
	CitedSourcePaths       []string            `json:"citedSourcePaths,omitempty"`
	UncitedSourcePaths     []string            `json:"uncitedSourcePaths,omitempty"`
	EvidenceQuality        string              `json:"evidenceQuality,omitempty"`
	EvidenceSummary        string              `json:"evidenceSummary,omitempty"`
	SourceFingerprints     []SourceFingerprint `json:"sourceFingerprints,omitempty"`
	ExportFormat           string              `json:"exportFormat,omitempty"`
	PackageFiles           []string            `json:"packageFiles,omitempty"`
	GeneratedAt            time.Time           `json:"generatedAt"`
}

type SourceFingerprint struct {
	RelPath    string    `json:"relPath"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modifiedAt"`
	SHA256     string    `json:"sha256,omitempty"`
	Error      string    `json:"error,omitempty"`
}

type Lineage struct {
	Nodes              []LineageNode  `json:"nodes"`
	Edges              []LineageEdge  `json:"edges"`
	RelationshipCounts map[string]int `json:"relationshipCounts,omitempty"`
	Message            string         `json:"message,omitempty"`
}

type LineageNode struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Label   string `json:"label"`
	RelPath string `json:"relPath,omitempty"`
}

type LineageEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

type LineageImport struct {
	Lineage Lineage `json:"lineage"`
	Message string  `json:"message"`
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
	RelPath             string
	Exists              bool
	Changed             bool
	Unknown             bool
	ModifiedAt          time.Time
	Size                int64
	Fingerprint         string
	ExpectedFingerprint string
	Message             string
}
