package datasets

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
