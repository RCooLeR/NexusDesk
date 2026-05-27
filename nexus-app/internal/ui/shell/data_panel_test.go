package shell

import (
	"strings"
	"testing"
	"time"

	datasetsSvc "nexusdesk/internal/services/datasets"
	dbconnectorSvc "nexusdesk/internal/services/dbconnector"
	metadataSvc "nexusdesk/internal/services/metadata"
)

func TestFormatDatasetProfileIncludesColumnsAndJSONNotes(t *testing.T) {
	output := formatDatasetProfile(datasetsSvc.Profile{
		RelPath:   "data/events.json",
		Format:    "JSON",
		MediaType: "application/json",
		Size:      42,
		Rows:      2,
		Columns: []datasetsSvc.ColumnProfile{
			{Name: "channel", Type: "text", NonEmpty: 2, Samples: []string{"search", "email"}},
		},
		JSONProfile: &datasetsSvc.JSONProfile{TopLevel: "array", Count: 2, Notes: []string{"Array object fields are profiled across object elements."}},
	})
	if !strings.Contains(output, "Path: data/events.json") || !strings.Contains(output, "- channel | text") || !strings.Contains(output, "Top level: array") {
		t.Fatalf("profile output missing expected details:\n%s", output)
	}
}

func TestProfileStatusMarksTruncatedSample(t *testing.T) {
	status := profileStatus(datasetsSvc.Profile{RelPath: "data.csv", Format: "CSV", Rows: 50, Columns: make([]datasetsSvc.ColumnProfile, 3), Truncated: true})
	if !strings.Contains(status, "CSV sample") || !strings.Contains(status, "3 columns") {
		t.Fatalf("unexpected profile status: %q", status)
	}
}

func TestFormatDatasetProfileIncludesParquetFooterDetails(t *testing.T) {
	output := formatDatasetProfile(datasetsSvc.Profile{
		RelPath: "events.parquet",
		Format:  "PARQUET",
		Size:    256,
		Rows:    3,
		Columns: []datasetsSvc.ColumnProfile{{Name: "id", Type: "int64", NonEmpty: 3}},
		Parquet: &datasetsSvc.ParquetProfile{
			Version:         1,
			CreatedBy:       "writer",
			FooterLength:    128,
			DataBytes:       120,
			MetadataDecoded: true,
			SchemaColumns:   []datasetsSvc.ParquetColumn{{Path: "id", Type: "INT64"}},
			RowGroups: []datasetsSvc.ParquetRowGroup{{
				Index:                 1,
				Rows:                  3,
				Columns:               1,
				TotalCompressedSize:   40,
				TotalUncompressedSize: 80,
			}},
		},
	})
	for _, expected := range []string{"Parquet", "Footer metadata: 128 bytes", "Schema columns: 1", "row group 1: rows 3", "- id | int64"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("Parquet profile output missing %q:\n%s", expected, output)
		}
	}
}

func TestFormatDatasetQueryResultIncludesRows(t *testing.T) {
	output := formatDatasetQueryResult(datasetsSvc.QueryResult{
		RelPath:     "sales.csv",
		Format:      "CSV",
		Query:       "channel=search",
		Columns:     []string{"channel", "spend"},
		Rows:        [][]string{{"search", "20"}},
		TotalRows:   4,
		MatchedRows: 2,
		Truncated:   true,
		Message:     "2 matching rows from sales.csv; showing 1.",
	})
	for _, expected := range []string{"# Dataset Query", "Query: channel=search", "Matched rows: 2", "channel\tspend", "search\t20", "Scope: result is bounded"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("query output missing %q:\n%s", expected, output)
		}
	}
}

func TestQueryStatusMarksBoundedResults(t *testing.T) {
	status := queryStatus(datasetsSvc.QueryResult{RelPath: "sales.csv", Format: "CSV", Rows: [][]string{{"a"}}, MatchedRows: 10, Truncated: true})
	if !strings.Contains(status, "CSV bounded query") || !strings.Contains(status, "1/10 rows shown") {
		t.Fatalf("unexpected query status: %q", status)
	}
}

func TestFormatDatasetChartIncludesSVGAndPoints(t *testing.T) {
	output := formatDatasetChart(datasetsSvc.ChartResult{
		RelPath:        "sales.csv",
		Format:         "CSV",
		Mode:           "sum",
		CategoryColumn: "channel",
		ValueColumn:    "spend",
		Query:          "channel=search",
		Points:         []datasetsSvc.ChartPoint{{Label: "search", Value: 20}},
		SVG:            `<svg></svg>`,
		Message:        "Bar chart: spend by channel.",
	})
	for _, expected := range []string{"# Dataset Chart", "Value column: spend", "search: 20", "<svg></svg>"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("chart output missing %q:\n%s", expected, output)
		}
	}
}

func TestFormatDatasetDashboardIncludesMetricsAndSVG(t *testing.T) {
	output := formatDatasetDashboard(datasetsSvc.DashboardResult{
		RelPath: "sales.csv",
		Format:  "CSV",
		Query:   "channel=search",
		Metrics: []datasetsSvc.DashboardMetric{
			{Label: "Shown rows", Value: "2", Detail: "2 matched"},
			{Label: "Total spend", Value: "20", Detail: "channel"},
		},
		Chart: datasetsSvc.ChartResult{
			Mode:           "sum",
			CategoryColumn: "channel",
			ValueColumn:    "spend",
			Points:         []datasetsSvc.ChartPoint{{Label: "search", Value: 20}},
		},
		SVG:     `<svg></svg>`,
		Message: "Dashboard: 2 metric(s), Bar chart: spend by channel.",
	})
	for _, expected := range []string{"# Dataset Dashboard", "Shown rows: 2", "Total spend: 20", "Value: spend", "<svg></svg>"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("dashboard output missing %q:\n%s", expected, output)
		}
	}
}

func TestFormatDatasetSQLResultIncludesPlanAndRows(t *testing.T) {
	output := formatDatasetSQLResult(datasetsSvc.SQLResult{
		QueryResult: datasetsSvc.QueryResult{
			RelPath:     "sales.csv",
			Format:      "CSV",
			Columns:     []string{"channel", "spend"},
			Rows:        [][]string{{"search", "20"}},
			TotalRows:   4,
			MatchedRows: 1,
		},
		SQL:        "select * from dataset",
		Engine:     "native-dataset-sql",
		Plan:       []string{"Validate SELECT-only native dataset SQL."},
		DurationMs: 2,
	})
	for _, expected := range []string{"# Dataset SQL", "native-dataset-sql", "Plan", "search\t20"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("SQL output missing %q:\n%s", expected, output)
		}
	}
}

func TestFormatSQLiteMetadataIncludesSchemaIndexesSamplesAndRelationships(t *testing.T) {
	output := formatSQLiteMetadata(dbconnectorSvc.SQLiteMetadata{
		RelPath:  "data/store.sqlite",
		Engine:   "sqlite-readonly",
		ReadOnly: true,
		Tables: []dbconnectorSvc.SQLiteObject{{
			Name:     "orders",
			Type:     "table",
			RowCount: 2,
			Columns: []dbconnectorSvc.SQLiteColumn{
				{Name: "id", Type: "INTEGER", PrimaryKey: true},
				{Name: "customer_id", Type: "INTEGER", Nullable: false},
			},
			Indexes: []dbconnectorSvc.SQLiteIndex{{Name: "idx_orders_customer", Columns: []string{"customer_id"}}},
			SampleRows: [][]string{
				{"10", "1"},
			},
		}},
		Views: []dbconnectorSvc.SQLiteObject{{Name: "order_totals", Type: "view", Columns: []dbconnectorSvc.SQLiteColumn{{Name: "customer_id"}}}},
		Relationships: []dbconnectorSvc.SQLiteRelationship{{
			Kind:       "foreign-key",
			FromTable:  "orders",
			FromColumn: "customer_id",
			ToTable:    "customers",
			ToColumn:   "id",
			Confidence: "high",
			Reason:     "Declared by SQLite foreign_key_list metadata.",
		}},
	})
	for _, expected := range []string{"# SQLite Workspace Connector", "Path: data/store.sqlite", "orders | table | 2 row(s)", "Index: idx_orders_customer on customer_id", "Sample: id\tcustomer_id", "orders.customer_id -> customers.id"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("SQLite metadata output missing %q:\n%s", expected, output)
		}
	}
}

func TestFormatSQLiteQueryResultIncludesCapsAndRows(t *testing.T) {
	output := formatSQLiteQueryResult(dbconnectorSvc.SQLiteQueryResult{
		RelPath:        "data/store.sqlite",
		Engine:         "sqlite-readonly",
		SQL:            "select id, total from orders",
		Columns:        []string{"id", "total"},
		Rows:           [][]string{{"10", "42.5"}},
		TotalRows:      1,
		ResultLimit:    100,
		TimeoutSeconds: 30,
		DurationMs:     4,
		Message:        "Read-only SQLite query returned 1 row(s).",
	})
	for _, expected := range []string{"# SQLite Query Preview", "Path: data/store.sqlite", "Row cap: 100", "Timeout: 30 seconds", "id\ttotal", "10\t42.5"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("SQLite query output missing %q:\n%s", expected, output)
		}
	}
}

func TestSQLiteSQLRunRecordCapturesFailure(t *testing.T) {
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	record := sqliteSQLRunRecord(dbconnectorSvc.SQLiteQueryResult{}, "data/store.sqlite", "delete from orders", started, errForTest("blocked"))
	if record.Status != "failed" || record.Error != "blocked" || record.Engine != "sqlite-readonly" || record.RelPath != "data/store.sqlite" {
		t.Fatalf("unexpected SQLite failed SQL record: %#v", record)
	}
}

func TestSQLRunRecordCapturesFailure(t *testing.T) {
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	record := sqlRunRecord(datasetsSvc.SQLResult{StartedAt: started}, "sales.csv", "delete from dataset", errForTest("blocked"))
	if record.Status != "failed" || record.Error != "blocked" || record.RelPath != "sales.csv" {
		t.Fatalf("unexpected failed SQL record: %#v", record)
	}
}

func TestFormatDatasetHistoryFiltersSelectedRunsAndListsDependencies(t *testing.T) {
	when := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	output := formatDatasetHistory("data/sales.csv", []metadataSvc.SQLRunRecord{
		{RelPath: "data/sales.csv", SQL: "select channel from dataset", Status: "success", MatchedRows: 2, ShownRows: 2, CompletedAt: when, DurationMs: 4},
		{RelPath: "data/other.csv", SQL: "select * from dataset", Status: "success", MatchedRows: 1, ShownRows: 1, CompletedAt: when, DurationMs: 3},
	}, []metadataSvc.DatasetDependencyRecord{
		{SourcePath: "data/sales.csv", DependentKind: "sql-run", DependentRef: "sql-1", Relation: "reads", Metadata: map[string]string{"engine": "native"}, UpdatedAt: when},
	})
	for _, expected := range []string{"# Dataset SQL History", "Selected dataset: data/sales.csv", "select channel from dataset", "data/sales.csv reads sql-run:sql-1", "Metadata: engine=native"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("dataset history missing %q:\n%s", expected, output)
		}
	}
	if strings.Contains(output, "data/other.csv") {
		t.Fatalf("selected dataset history included another dataset:\n%s", output)
	}
}

func TestLatestReusableSQLRunFiltersSelectedSource(t *testing.T) {
	runs := []metadataSvc.SQLRunRecord{
		{RelPath: "data/other.csv", SQL: "select * from dataset"},
		{RelPath: "data/sales.csv"},
		{RelPath: "data/sales.csv", SQL: "select channel from dataset order by spend desc", Engine: "native-dataset-sql"},
	}
	run, ok := latestReusableSQLRun(runs, "data/sales.csv")
	if !ok {
		t.Fatal("expected reusable SQL run")
	}
	if run.SQL != "select channel from dataset order by spend desc" || run.RelPath != "data/sales.csv" {
		t.Fatalf("unexpected reusable SQL run: %#v", run)
	}
	if _, ok := latestReusableSQLRun(runs, "data/missing.csv"); ok {
		t.Fatal("expected no reusable run for missing dataset")
	}
}

func TestFormatSQLRunReuseIncludesRunnableSQL(t *testing.T) {
	output := formatSQLRunReuse("Loaded latest SQL for editing", metadataSvc.SQLRunRecord{
		RelPath:     "data/sales.csv",
		SQL:         "select channel from dataset",
		Engine:      "native-dataset-sql",
		Status:      "success",
		MatchedRows: 2,
		ShownRows:   2,
		DurationMs:  5,
		Message:     "OK",
	})
	for _, expected := range []string{"# Loaded latest SQL for editing", "data/sales.csv", "native-dataset-sql", "select channel from dataset"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("reuse output missing %q:\n%s", expected, output)
		}
	}
}

func TestIsSQLiteRunUsesEngineOrPath(t *testing.T) {
	if !isSQLiteRun(metadataSvc.SQLRunRecord{Engine: "sqlite-readonly", RelPath: "data/store.bin"}) {
		t.Fatal("expected sqlite engine to be detected")
	}
	if !isSQLiteRun(metadataSvc.SQLRunRecord{Engine: "native-dataset-sql", RelPath: "data/store.sqlite3"}) {
		t.Fatal("expected sqlite path to be detected")
	}
	if isSQLiteRun(metadataSvc.SQLRunRecord{Engine: "native-dataset-sql", RelPath: "data/sales.csv"}) {
		t.Fatal("did not expect CSV run to be detected as sqlite")
	}
}

func TestFormatDatasetNotebooksListsCells(t *testing.T) {
	when := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	output := formatDatasetNotebooks([]datasetsSvc.Notebook{{
		ID:        "sales-20260527120000",
		RelPath:   "data/sales.csv",
		Label:     "Sales SQL",
		UpdatedAt: when,
		Cells: []datasetsSvc.NotebookCell{
			{ID: "cell-1", Kind: "sql", Label: "Top spend", SQL: "select * from dataset order by spend desc"},
			{ID: "chart-1", Kind: "chart", Label: "Spend chart"},
		},
	}})
	for _, expected := range []string{"# Dataset SQL Notebooks", "Sales SQL", "2 cell(s)", "Top spend [sql]: select * from dataset", "Spend chart [chart]"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("notebook output missing %q:\n%s", expected, output)
		}
	}
}

func TestNotebookCellsFromEditorParsesSQLAndChartCells(t *testing.T) {
	cells := notebookCellsFromEditor("-- cell: Top spend\nselect * from dataset limit 5\n\n-- chart: Spend chart\nselect channel, spend from dataset")
	if len(cells) != 2 {
		t.Fatalf("expected two cells, got %#v", cells)
	}
	if cells[0].Kind != "sql" || cells[0].Label != "Top spend" || !strings.Contains(cells[0].SQL, "limit 5") {
		t.Fatalf("unexpected first cell: %#v", cells[0])
	}
	if cells[1].Kind != "chart" || cells[1].Label != "Spend chart" || !strings.Contains(cells[1].SQL, "channel") {
		t.Fatalf("unexpected second cell: %#v", cells[1])
	}
}

func TestFormatNotebookForEditorRoundTripsCellDirectives(t *testing.T) {
	output := formatNotebookForEditor(datasetsSvc.Notebook{Cells: []datasetsSvc.NotebookCell{
		{ID: "cell-1", Kind: "sql", Label: "Top spend", SQL: "select * from dataset limit 5"},
		{ID: "chart-1", Kind: "chart", Label: "Spend chart", SQL: "select channel, spend from dataset"},
	}})
	for _, expected := range []string{"-- cell: Top spend", "select * from dataset limit 5", "-- chart: Spend chart"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("notebook editor output missing %q:\n%s", expected, output)
		}
	}
}

func TestFormatNotebookRunResultIncludesCellRowsPlansAndChart(t *testing.T) {
	output := formatNotebookRunResult(datasetsSvc.NotebookRunResult{
		RelPath:    "sales.csv",
		NotebookID: "book",
		Label:      "Sales Notebook",
		DurationMs: 8,
		Message:    "Ran 2 notebook cell(s).",
		Cells: []datasetsSvc.NotebookCellRun{
			{
				CellID: "cell-1",
				Label:  "Top spend",
				Kind:   "sql",
				SQL:    "select channel from dataset",
				SQLResult: datasetsSvc.SQLResult{
					QueryResult: datasetsSvc.QueryResult{
						RelPath:     "sales.csv",
						Columns:     []string{"channel"},
						Rows:        [][]string{{"search"}},
						MatchedRows: 1,
					},
					Plan: []string{"Validate SELECT-only native dataset SQL."},
				},
			},
			{
				CellID: "chart-1",
				Label:  "Chart",
				Kind:   "chart",
				ChartResult: datasetsSvc.ChartResult{
					SVG:     "<svg/>",
					Points:  []datasetsSvc.ChartPoint{{Label: "search", Value: 1}},
					Message: "Bar chart.",
				},
			},
		},
	})
	for _, expected := range []string{"# SQL Notebook Run", "Cell 1: Top spend", "Rows", "search", "Chart", "Points: 1"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("notebook run output missing %q:\n%s", expected, output)
		}
	}
}

func TestFirstNotebookSQLUsesFirstSQLCell(t *testing.T) {
	sqlText := firstNotebookSQL(datasetsSvc.Notebook{Cells: []datasetsSvc.NotebookCell{
		{Kind: "chart", Label: "Chart"},
		{Kind: "sql", SQL: " select * from dataset limit 5 "},
	}})
	if sqlText != "select * from dataset limit 5" {
		t.Fatalf("unexpected notebook SQL: %q", sqlText)
	}
}

func TestChartArtifactInputPreservesSourceAndSVG(t *testing.T) {
	input := chartArtifactInput(datasetsSvc.ChartResult{
		RelPath:        "sales.csv",
		Format:         "CSV",
		Mode:           "count",
		CategoryColumn: "channel",
		SVG:            "<svg/>",
		Points:         []datasetsSvc.ChartPoint{{Label: "search", Value: 2}},
	})
	if input.SourcePath != "sales.csv" || input.SVG != "<svg/>" || input.PointCount != 1 {
		t.Fatalf("unexpected chart artifact input: %#v", input)
	}
}

func TestDashboardArtifactInputUsesDashboardMode(t *testing.T) {
	input := dashboardArtifactInput(datasetsSvc.DashboardResult{
		RelPath: "sales.csv",
		Format:  "CSV",
		Chart: datasetsSvc.ChartResult{
			CategoryColumn: "channel",
			ValueColumn:    "spend",
			Points:         []datasetsSvc.ChartPoint{{Label: "search", Value: 20}},
		},
		SVG: "<svg/>",
	})
	if input.SourcePath != "sales.csv" || input.Mode != "dashboard" || input.SVG != "<svg/>" || input.PointCount != 1 {
		t.Fatalf("unexpected dashboard artifact input: %#v", input)
	}
}

func TestNotebookRunArtifactInputPreservesCellsRowsAndCharts(t *testing.T) {
	input := notebookRunArtifactInput(datasetsSvc.NotebookRunResult{
		RelPath:    "sales.csv",
		NotebookID: "book-1",
		Label:      "Sales Notebook",
		Message:    "Ran 2 notebook cell(s).",
		Cells: []datasetsSvc.NotebookCellRun{
			{
				CellID: "cell-1",
				Label:  "Top spend",
				Kind:   "sql",
				SQL:    "select channel from dataset",
				SQLResult: datasetsSvc.SQLResult{
					QueryResult: datasetsSvc.QueryResult{
						Columns:     []string{"channel"},
						Rows:        [][]string{{"search"}},
						MatchedRows: 1,
					},
					Engine: "native-dataset-sql",
					Plan:   []string{"Validate SELECT-only native dataset SQL."},
				},
			},
			{
				CellID: "chart-1",
				Label:  "Spend chart",
				Kind:   "chart",
				ChartResult: datasetsSvc.ChartResult{
					Mode:    "sum",
					SVG:     "<svg/>",
					Points:  []datasetsSvc.ChartPoint{{Label: "search", Value: 2}},
					Message: "Bar chart.",
				},
			},
		},
	})
	if input.SourcePath != "sales.csv" || input.NotebookID != "book-1" || len(input.Cells) != 2 {
		t.Fatalf("unexpected notebook artifact input: %#v", input)
	}
	if input.Cells[0].Columns[0] != "channel" || input.Cells[0].Rows[0][0] != "search" || input.Cells[0].Plan[0] == "" {
		t.Fatalf("unexpected SQL cell artifact input: %#v", input.Cells[0])
	}
	if input.Cells[1].ChartSVG != "<svg/>" || input.Cells[1].ChartPoints != 1 {
		t.Fatalf("unexpected chart cell artifact input: %#v", input.Cells[1])
	}
}

type errForTest string

func (e errForTest) Error() string {
	return string(e)
}
