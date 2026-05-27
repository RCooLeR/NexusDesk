package datasets

type Profile struct {
	RelPath     string
	Format      string
	MediaType   string
	Size        int64
	Rows        int
	Columns     []ColumnProfile
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
