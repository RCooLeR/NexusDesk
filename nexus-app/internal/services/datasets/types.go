package datasets

import "time"

type Profile struct {
	RelPath     string
	Format      string
	MediaType   string
	Size        int64
	Rows        int
	Columns     []ColumnProfile
	Sheet       string
	Sheets      []string
	Truncated   bool
	Notes       []string
	JSONProfile *JSONProfile
	Parquet     *ParquetProfile
}

type ColumnProfile struct {
	Name     string
	Type     string
	NonEmpty int
	Empty    int
	Samples  []string
}

type JSONProfile struct {
	TopLevel string
	Count    int
	Notes    []string
}

type ParquetProfile struct {
	Version         int
	CreatedBy       string
	FooterLength    int64
	DataBytes       int64
	SchemaColumns   []ParquetColumn
	RowGroups       []ParquetRowGroup
	MetadataDecoded bool
	Truncated       bool
}

type ParquetColumn struct {
	Path           string
	Type           string
	RepetitionType string
	ConvertedType  string
	TypeLength     int
	Precision      int
	Scale          int
}

type ParquetRowGroup struct {
	Index                 int
	Rows                  int64
	Columns               int
	TotalByteSize         int64
	TotalCompressedSize   int64
	TotalUncompressedSize int64
	ColumnChunks          []ParquetColumnChunk
}

type ParquetColumnChunk struct {
	Path             string
	Type             string
	Codec            string
	Values           int64
	CompressedSize   int64
	UncompressedSize int64
}

type QueryResult struct {
	RelPath     string
	Query       string
	Format      string
	Columns     []string
	Rows        [][]string
	TotalRows   int
	MatchedRows int
	Truncated   bool
	Message     string
}

type ChartResult struct {
	RelPath        string
	Query          string
	Format         string
	Mode           string
	CategoryColumn string
	ValueColumn    string
	Points         []ChartPoint
	SVG            string
	Truncated      bool
	Message        string
}

type ChartPoint struct {
	Label string
	Value float64
}

type DashboardResult struct {
	RelPath   string
	Query     string
	Format    string
	Metrics   []DashboardMetric
	Chart     ChartResult
	SVG       string
	Truncated bool
	Message   string
}

type DashboardMetric struct {
	Label  string
	Value  string
	Detail string
}

type SQLResult struct {
	QueryResult
	SQL         string
	Engine      string
	Plan        []string
	StartedAt   time.Time
	CompletedAt time.Time
	DurationMs  int64
}

type NotebookCell struct {
	ID        string
	Kind      string
	Label     string
	SQL       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Notebook struct {
	ID        string
	RelPath   string
	Label     string
	Cells     []NotebookCell
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NotebookSaveRequest struct {
	ID      string
	RelPath string
	Label   string
	Cells   []NotebookCell
}

type NotebookRunResult struct {
	RelPath     string
	NotebookID  string
	Label       string
	Cells       []NotebookCellRun
	StartedAt   time.Time
	CompletedAt time.Time
	DurationMs  int64
	Message     string
}

type NotebookCellRun struct {
	CellID      string
	Label       string
	Kind        string
	SQL         string
	SQLResult   SQLResult
	ChartResult ChartResult
	Error       string
	StartedAt   time.Time
	CompletedAt time.Time
	DurationMs  int64
}
