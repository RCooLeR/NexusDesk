package spreadsheets

type Options struct {
	MaxRows    int
	MaxColumns int
}

type Workbook struct {
	Sheets []Sheet
}

type Sheet struct {
	Name      string
	Path      string
	Rows      [][]string
	Truncated bool
}

func DefaultOptions() Options {
	return Options{MaxRows: 50, MaxColumns: 30}
}
