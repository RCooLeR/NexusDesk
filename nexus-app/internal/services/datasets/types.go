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
	JSONProfile *JSONProfile
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

type SQLResult struct {
	QueryResult
	SQL         string
	Engine      string
	Plan        []string
	StartedAt   time.Time
	CompletedAt time.Time
	DurationMs  int64
}
