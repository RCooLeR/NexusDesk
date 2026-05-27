package dbconnector

type SQLiteMetadata struct {
	ID            string
	RelPath       string
	Name          string
	Engine        string
	ReadOnly      bool
	Tables        []SQLiteObject
	Views         []SQLiteObject
	Indexes       []SQLiteIndex
	Relationships []SQLiteRelationship
	Message       string
}

type SQLiteObject struct {
	Name       string
	Type       string
	RowCount   int
	Columns    []SQLiteColumn
	Indexes    []SQLiteIndex
	SampleRows [][]string
}

type SQLiteColumn struct {
	Name       string
	Type       string
	Nullable   bool
	PrimaryKey bool
	Default    string
}

type SQLiteIndex struct {
	Name    string
	Table   string
	Unique  bool
	Columns []string
}

type SQLiteRelationship struct {
	Kind       string
	FromTable  string
	FromColumn string
	ToTable    string
	ToColumn   string
	Confidence string
	Reason     string
}
