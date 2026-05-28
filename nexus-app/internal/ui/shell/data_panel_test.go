package shell

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	datasetsSvc "nexusdesk/internal/services/datasets"
	dbconnectorSvc "nexusdesk/internal/services/dbconnector"
	metadataSvc "nexusdesk/internal/services/metadata"

	_ "modernc.org/sqlite"
)

func testArtifact(relPath string) artifactsSvc.Artifact {
	now := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	return artifactsSvc.Artifact{RelPath: relPath, GeneratedAt: now, CreatedAt: now}
}

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

func TestFormatSavedQueriesListsNewestSnippets(t *testing.T) {
	output := formatSavedQueries("Saved SQLite Queries", []datasetsSvc.SavedQuery{
		{RelPath: "data/store.sqlite", Label: "Orders", Kind: "sqlite-sql", Query: "select * from orders"},
	})
	for _, expected := range []string{"# Saved SQLite Queries", "Orders", "data/store.sqlite", "select * from orders"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("saved query output missing %q:\n%s", expected, output)
		}
	}
}

func TestSQLiteQueryArtifactInputMapsRowsAndCaps(t *testing.T) {
	input := sqliteQueryArtifactInput(dbconnectorSvc.SQLiteQueryResult{
		RelPath:        "data/store.sqlite",
		SQL:            "select id from users",
		Engine:         "sqlite-readonly",
		Columns:        []string{"id"},
		Rows:           [][]string{{"1"}},
		TotalRows:      1,
		ResultLimit:    100,
		TimeoutSeconds: 30,
		DurationMs:     7,
	})
	if input.SourcePath != "data/store.sqlite" || input.SQL != "select id from users" || input.Rows[0][0] != "1" || input.ResultLimit != 100 {
		t.Fatalf("unexpected SQLite artifact input: %#v", input)
	}
}

func TestDatasetArtifactInputsAndDependenciesCaptureRebuildMetadata(t *testing.T) {
	query := datasetsSvc.QueryResult{
		RelPath:     "data/sales.csv",
		Query:       "channel=search",
		Format:      "CSV",
		Columns:     []string{"channel", "spend"},
		Rows:        [][]string{{"search", "42"}},
		TotalRows:   2,
		MatchedRows: 1,
	}
	queryInput := datasetQueryArtifactInput(query)
	if queryInput.SourcePath != "data/sales.csv" || queryInput.Query != "channel=search" || queryInput.Rows[0][1] != "42" {
		t.Fatalf("unexpected dataset query artifact input: %#v", queryInput)
	}
	queryDependency := datasetQueryArtifactDependencyRecord(query, testArtifact(".nexusdesk/artifacts/dataset-queries/query.csv"))
	if queryDependency.DependentKind != "filter-export" || queryDependency.Metadata["query"] != "channel=search" {
		t.Fatalf("unexpected query dependency: %#v", queryDependency)
	}

	sql := datasetsSvc.SQLResult{
		QueryResult: query,
		SQL:         "select channel, spend from dataset where channel = 'search'",
		Engine:      "native-dataset-sql",
		Plan:        []string{"Read selected dataset."},
		DurationMs:  5,
	}
	sqlInput := datasetSQLArtifactInput(sql)
	if sqlInput.SourcePath != "data/sales.csv" || sqlInput.SQL == "" || len(sqlInput.Plan) != 1 {
		t.Fatalf("unexpected dataset SQL artifact input: %#v", sqlInput)
	}
	sqlDependency := datasetSQLArtifactDependencyRecord(sql, metadataSvc.SQLRunRecord{ID: "sql-1"}, testArtifact(".nexusdesk/artifacts/dataset-sql/report.md"))
	if sqlDependency.DependentKind != "sql-report" || sqlDependency.Metadata["sqlRunId"] != "sql-1" || sqlDependency.Metadata["sql"] == "" {
		t.Fatalf("unexpected SQL dependency: %#v", sqlDependency)
	}

	profile := datasetsSvc.Profile{
		RelPath: "data/sales.csv",
		Format:  "CSV",
		Rows:    2,
		Columns: []datasetsSvc.ColumnProfile{
			{Name: "channel", Type: "text", NonEmpty: 2, Samples: []string{"search", "email"}},
			{Name: "spend", Type: "integer", NonEmpty: 2, Samples: []string{"42", "7"}},
		},
	}
	summaryInput := datasetSummaryArtifactInput(profile)
	if summaryInput.SourcePath != "data/sales.csv" || summaryInput.Format != "CSV" || len(summaryInput.Columns) != 2 || summaryInput.Columns[1].Samples[0] != "42" {
		t.Fatalf("unexpected dataset summary artifact input: %#v", summaryInput)
	}
	summaryDependency := datasetSummaryArtifactDependencyRecord(profile, testArtifact(".nexusdesk/artifacts/dataset-summaries/summary.md"))
	if summaryDependency.DependentKind != "dataset-summary" || summaryDependency.Relation != "summarizes" || summaryDependency.Metadata["columns"] != "2" {
		t.Fatalf("unexpected summary dependency: %#v", summaryDependency)
	}
}

func TestLatestRebuildableDatasetDependencyUsesSupportedKinds(t *testing.T) {
	dependency, ok := latestRebuildableDatasetDependency([]metadataSvc.DatasetDependencyRecord{
		{DependentKind: "sql-run", Metadata: map[string]string{"sql": "select * from dataset"}},
		{SourcePath: "data/sales.csv", DependentKind: "filter-export", Metadata: map[string]string{"query": "channel=search"}},
	})
	if !ok || dependency.DependentKind != "filter-export" {
		t.Fatalf("expected filter-export dependency, got %#v ok=%v", dependency, ok)
	}
	if canRebuildDatasetDependency(metadataSvc.DatasetDependencyRecord{DependentKind: "sql-report", Metadata: map[string]string{}}) {
		t.Fatal("expected SQL report without SQL text to be non-rebuildable")
	}
	if !canRebuildDatasetDependency(metadataSvc.DatasetDependencyRecord{SourcePath: "data/sales.csv", DependentKind: "sql-notebook"}) {
		t.Fatal("expected saved SQL notebook dependencies to be rebuildable")
	}
	if !canRebuildDatasetDependency(metadataSvc.DatasetDependencyRecord{SourcePath: "data/store.sqlite", DependentKind: "sqlite-query-artifact", Metadata: map[string]string{"sql": "select id from orders"}}) {
		t.Fatal("expected SQLite query artifact dependencies to be rebuildable")
	}
	if canRebuildDatasetDependency(metadataSvc.DatasetDependencyRecord{SourcePath: "data/store.sqlite", DependentKind: "sqlite-query-artifact"}) {
		t.Fatal("expected SQLite query artifact without SQL text to be non-rebuildable")
	}
	if !canRebuildDatasetDependency(metadataSvc.DatasetDependencyRecord{SourcePath: "data/sales.csv", DependentKind: "dataset-summary"}) {
		t.Fatal("expected dataset summary dependencies to be rebuildable")
	}
}

func TestNotebookDependencyRecordPreservesNotebookIDForRebuilds(t *testing.T) {
	notebook := datasetsSvc.Notebook{
		ID:      "sales-book",
		RelPath: "data/sales.csv",
		Label:   "Sales Notebook",
		Cells:   []datasetsSvc.NotebookCell{{ID: "cell-1", Kind: "sql", SQL: "select * from dataset"}},
	}
	record := notebookDependencyRecord("data/sales.csv", notebook)
	if record.DependentKind != "sql-notebook" || record.DependentRef != "sales-book" || record.Metadata["notebookID"] != "sales-book" {
		t.Fatalf("unexpected notebook dependency record: %#v", record)
	}
	selected, ok := notebookForDatasetDependency([]datasetsSvc.Notebook{notebook}, record)
	if !ok || selected.ID != "sales-book" {
		t.Fatalf("expected notebook dependency to resolve by id, got %#v ok=%v", selected, ok)
	}
	rebuiltRecord := record
	rebuiltRecord.DependentRef = ".nexusdesk/artifacts/notebooks/run.md"
	selected, ok = notebookForDatasetDependency([]datasetsSvc.Notebook{notebook}, rebuiltRecord)
	if !ok || selected.ID != "sales-book" {
		t.Fatalf("expected rebuilt dependency to keep resolving by metadata id, got %#v ok=%v", selected, ok)
	}
}

func TestRebuildDatasetDependencyArtifactSupportsDatasetSummary(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "sales.csv"), []byte("channel,spend\nsearch,42\nemail,7\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	store, err := metadataSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.Ensure(); err != nil {
		t.Fatalf("Ensure failed: %v", err)
	}
	dependency := store.NormalizeDatasetDependencyRecord(metadataSvc.DatasetDependencyRecord{
		SourcePath:    "data/sales.csv",
		DependentKind: "dataset-summary",
		DependentRef:  ".nexusdesk/artifacts/dataset-summaries/old.md",
		Relation:      "summarizes",
		Metadata:      map[string]string{"artifact": ".nexusdesk/artifacts/dataset-summaries/old.md"},
	})
	if err := store.SaveDatasetDependency(dependency); err != nil {
		t.Fatalf("SaveDatasetDependency failed: %v", err)
	}
	view := &View{datasetService: datasetsSvc.New(nil), metadataStore: store}
	artifact, err := view.rebuildDatasetDependencyArtifact(context.Background(), root, dependency)
	if err != nil {
		t.Fatalf("rebuildDatasetDependencyArtifact failed: %v", err)
	}
	if artifact.Kind != "dataset-summary" || !strings.Contains(artifact.RelPath, ".nexusdesk/artifacts/dataset-summaries/") {
		t.Fatalf("unexpected rebuilt summary artifact: %#v", artifact)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile rebuilt summary artifact failed: %v", err)
	}
	if !strings.Contains(string(content), "Dataset Summary - sales.csv") || !strings.Contains(string(content), "| spend | integer |") {
		t.Fatalf("rebuilt summary artifact missing expected content:\n%s", string(content))
	}
	updated, err := store.GetDatasetDependency(dependency.ID)
	if err != nil {
		t.Fatalf("GetDatasetDependency failed: %v", err)
	}
	if updated.DependentRef != artifact.RelPath || updated.Metadata["artifact"] != artifact.RelPath || updated.Metadata["format"] != "CSV" || updated.Metadata["columns"] != "2" {
		t.Fatalf("expected summary dependency to point at rebuilt artifact, got %#v", updated)
	}
}

func TestRebuildDatasetDependencyArtifactSupportsSQLNotebook(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "sales.csv"), []byte("channel,spend\nsearch,42\nemail,7\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	datasetService := datasetsSvc.New(nil)
	notebook, err := datasetService.SaveNotebook(root, datasetsSvc.NotebookSaveRequest{
		ID:      "sales-book",
		RelPath: "data/sales.csv",
		Label:   "Sales Notebook",
		Cells: []datasetsSvc.NotebookCell{{
			ID:    "cell-1",
			Kind:  "sql",
			Label: "Top spend",
			SQL:   "select channel, spend from dataset order by spend desc",
		}},
	})
	if err != nil {
		t.Fatalf("SaveNotebook failed: %v", err)
	}
	store, err := metadataSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.Ensure(); err != nil {
		t.Fatalf("Ensure failed: %v", err)
	}
	dependency := store.NormalizeDatasetDependencyRecord(notebookDependencyRecord("data/sales.csv", notebook))
	if err := store.SaveDatasetDependency(dependency); err != nil {
		t.Fatalf("SaveDatasetDependency failed: %v", err)
	}
	view := &View{datasetService: datasetService, metadataStore: store}
	artifact, err := view.rebuildDatasetDependencyArtifact(context.Background(), root, dependency)
	if err != nil {
		t.Fatalf("rebuildDatasetDependencyArtifact failed: %v", err)
	}
	if artifact.Kind != "sql-notebook-run" || !strings.Contains(artifact.RelPath, ".nexusdesk/artifacts/notebooks/") {
		t.Fatalf("unexpected rebuilt artifact: %#v", artifact)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile rebuilt artifact failed: %v", err)
	}
	if !strings.Contains(string(content), "Sales Notebook") || !strings.Contains(string(content), "| search | 42 |") {
		t.Fatalf("rebuilt notebook artifact missing expected content:\n%s", string(content))
	}
	updated, err := store.GetDatasetDependency(dependency.ID)
	if err != nil {
		t.Fatalf("GetDatasetDependency failed: %v", err)
	}
	if updated.DependentRef != artifact.RelPath || updated.Metadata["artifact"] != artifact.RelPath || updated.Metadata["notebookID"] != "sales-book" {
		t.Fatalf("expected dependency to point at rebuilt artifact with notebook id, got %#v", updated)
	}
}

func TestRebuildDatasetDependencyArtifactSupportsSQLiteQueryArtifacts(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "data", "store.sqlite")
	if err := makeShellSQLiteFixture(dbPath); err != nil {
		t.Fatalf("makeShellSQLiteFixture failed: %v", err)
	}
	store, err := metadataSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.Ensure(); err != nil {
		t.Fatalf("Ensure failed: %v", err)
	}
	originalArtifact := artifactsSvc.Artifact{
		Kind:    "sqlite-query-csv",
		RelPath: ".nexusdesk/artifacts/sqlite-queries/old.csv",
	}
	run := metadataSvc.SQLRunRecord{
		ID:          "sqlite-run-1",
		SQL:         "select id, total from orders order by id",
		Engine:      "sqlite-readonly",
		StartedAt:   time.Now().UTC(),
		CompletedAt: time.Now().UTC(),
	}
	dependency := store.NormalizeDatasetDependencyRecord(sqliteArtifactDependencyRecord("data/store.sqlite", run, originalArtifact))
	if err := store.SaveDatasetDependency(dependency); err != nil {
		t.Fatalf("SaveDatasetDependency failed: %v", err)
	}
	view := &View{dbconnectorService: dbconnectorSvc.New(), metadataStore: store}
	artifact, err := view.rebuildDatasetDependencyArtifact(context.Background(), root, dependency)
	if err != nil {
		t.Fatalf("rebuildDatasetDependencyArtifact failed: %v", err)
	}
	if artifact.Kind != "sqlite-query-csv" || filepath.Ext(artifact.RelPath) != ".csv" {
		t.Fatalf("unexpected rebuilt SQLite artifact: %#v", artifact)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile rebuilt SQLite artifact failed: %v", err)
	}
	if !strings.Contains(string(content), "id,total") || !strings.Contains(string(content), "10,42.5") {
		t.Fatalf("rebuilt SQLite CSV artifact missing expected content:\n%s", string(content))
	}
	updated, err := store.GetDatasetDependency(dependency.ID)
	if err != nil {
		t.Fatalf("GetDatasetDependency failed: %v", err)
	}
	if updated.DependentRef != artifact.RelPath || updated.Metadata["artifact"] != artifact.RelPath || updated.Metadata["format"] != "csv" || updated.Metadata["sql"] == "" {
		t.Fatalf("expected SQLite dependency to point at rebuilt artifact, got %#v", updated)
	}
}

func makeShellSQLiteFixture(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`
		create table orders (id integer primary key, total real);
		insert into orders (id, total) values (10, 42.5), (11, 7.25);
	`)
	return err
}

func TestSQLiteSQLRunRecordCapturesFailure(t *testing.T) {
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	record := sqliteSQLRunRecord(dbconnectorSvc.SQLiteQueryResult{}, "data/store.sqlite", "delete from orders", started, errForTest("blocked"))
	if record.Status != "failed" || record.Error != "blocked" || record.Engine != "sqlite-readonly" || record.RelPath != "data/store.sqlite" {
		t.Fatalf("unexpected SQLite failed SQL record: %#v", record)
	}
}

func TestSQLiteSQLRunRecordCapturesCanceledStatus(t *testing.T) {
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	record := sqliteSQLRunRecord(dbconnectorSvc.SQLiteQueryResult{}, "data/store.sqlite", "select * from orders", started, context.Canceled)
	if record.Status != "canceled" || record.Error == "" {
		t.Fatalf("expected canceled SQLite SQL record, got %#v", record)
	}
}

func TestSQLRunRecordCapturesFailure(t *testing.T) {
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	record := sqlRunRecord(datasetsSvc.SQLResult{StartedAt: started}, "sales.csv", "delete from dataset", errForTest("blocked"))
	if record.Status != "failed" || record.Error != "blocked" || record.RelPath != "sales.csv" {
		t.Fatalf("unexpected failed SQL record: %#v", record)
	}
}

func TestSQLRunRecordCapturesCanceledStatus(t *testing.T) {
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	record := sqlRunRecord(datasetsSvc.SQLResult{StartedAt: started}, "sales.csv", "select * from dataset", context.Canceled)
	if record.Status != "canceled" || record.Error == "" || record.RelPath != "sales.csv" {
		t.Fatalf("expected canceled SQL record: %#v", record)
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

func TestSQLHistorySourcesIncludesDatasetAndConnectorFallback(t *testing.T) {
	sources := sqlHistorySources("data/sales.csv", "warehouse")
	if len(sources) != 2 || sources[0] != "data/sales.csv" || sources[1] != "connector:warehouse" {
		t.Fatalf("unexpected source ordering: %#v", sources)
	}
	connectorOnly := sqlHistorySources("", "warehouse")
	if len(connectorOnly) != 1 || connectorOnly[0] != "connector:warehouse" {
		t.Fatalf("unexpected connector-only sources: %#v", connectorOnly)
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

func TestConnectorRunDetectionParsesSourcePath(t *testing.T) {
	if !isConnectorRun(metadataSvc.SQLRunRecord{RelPath: "connector:warehouse"}) {
		t.Fatal("expected connector run to be detected")
	}
	if got := connectorProfileIDFromSourcePath("connector:warehouse"); got != "warehouse" {
		t.Fatalf("unexpected connector id: %q", got)
	}
	if isConnectorRun(metadataSvc.SQLRunRecord{RelPath: "data/sales.csv"}) {
		t.Fatal("did not expect dataset run to be detected as connector run")
	}
	if got := connectorProfileIDFromSourcePath("data/sales.csv"); got != "" {
		t.Fatalf("expected empty connector id, got %q", got)
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

func TestAppendNotebookCellTemplateAddsSQLAndChartBlocks(t *testing.T) {
	sqlText := appendNotebookCellTemplate("", "cell")
	if !strings.Contains(sqlText, "-- cell: Query 1") || !strings.Contains(sqlText, "select * from dataset limit 50") {
		t.Fatalf("unexpected SQL cell template:\n%s", sqlText)
	}
	chartText := appendNotebookCellTemplate(sqlText, "chart")
	for _, expected := range []string{"-- cell: Query 1", "-- chart: Chart 2", "select * from dataset limit 50"} {
		if !strings.Contains(chartText, expected) {
			t.Fatalf("chart template output missing %q:\n%s", expected, chartText)
		}
	}
	cells := notebookCellsFromEditor(chartText)
	if len(cells) != 2 || cells[0].Kind != "sql" || cells[1].Kind != "chart" {
		t.Fatalf("templates did not parse into notebook cells: %#v", cells)
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

func TestNotebookCellControlsMoveAndDeleteCells(t *testing.T) {
	cells := notebookCellsFromEditor("-- cell: First\nselect id from dataset\n\n-- chart: Chart\nselect channel, spend from dataset\n\n-- cell: Third\nselect * from dataset limit 3")
	moved, activeIndex, ok := moveNotebookCells(cells, 2, -1)
	if !ok || activeIndex != 1 {
		t.Fatalf("expected third cell to move up, active=%d ok=%v", activeIndex, ok)
	}
	if moved[1].Label != "Third" || moved[1].ID != "cell-2" {
		t.Fatalf("unexpected moved cells: %#v", moved)
	}
	deleted, activeIndex, ok := deleteNotebookCell(moved, 1)
	if !ok || activeIndex != 1 || len(deleted) != 2 {
		t.Fatalf("expected moved cell to delete, active=%d ok=%v cells=%#v", activeIndex, ok, deleted)
	}
	if deleted[1].Label != "Chart" || deleted[1].ID != "cell-2" {
		t.Fatalf("unexpected remaining cells: %#v", deleted)
	}
	if _, _, ok := deleteNotebookCell(deleted[:1], 0); ok {
		t.Fatal("expected last notebook cell deletion to be blocked")
	}
}

func TestNotebookCellOptionsAndOutlineMarkActiveCell(t *testing.T) {
	cells := []datasetsSvc.NotebookCell{
		{ID: "cell-1", Kind: "sql", Label: "Top spend", SQL: "select * from dataset limit 5"},
		{ID: "cell-2", Kind: "chart", Label: "Spend chart", SQL: "select channel, spend from dataset"},
	}
	options := notebookCellOptions(cells)
	if len(options) != 2 || notebookCellOptionIndex(options, options[1]) != 1 {
		t.Fatalf("unexpected notebook options: %#v", options)
	}
	outline := formatNotebookCellOutline(cells, 1)
	for _, expected := range []string{"# SQL Notebook Cells", "* 2. Spend chart [chart]", "select channel, spend from dataset"} {
		if !strings.Contains(outline, expected) {
			t.Fatalf("outline missing %q:\n%s", expected, outline)
		}
	}
}

func TestFormatNotebookRunResultIncludesCellRowsPlansAndChart(t *testing.T) {
	result := datasetsSvc.NotebookRunResult{
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
					Mode:    "count",
				},
			},
		},
	}
	output := formatNotebookRunResult(result)
	for _, expected := range []string{"# SQL Notebook Run", "Cell 1: Top spend", "Rows", "search", "Chart", "Points: 1"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("notebook run output missing %q:\n%s", expected, output)
		}
	}
	rows := formatNotebookRowsTab(result)
	plan := formatNotebookPlanTab(result)
	charts := formatNotebookChartsTab(result)
	for _, assertion := range []struct {
		name     string
		output   string
		expected string
	}{
		{name: "rows", output: rows, expected: "channel\nsearch"},
		{name: "plan", output: plan, expected: "Validate SELECT-only native dataset SQL."},
		{name: "charts", output: charts, expected: "<svg/>"},
	} {
		if !strings.Contains(assertion.output, assertion.expected) {
			t.Fatalf("%s tab missing %q:\n%s", assertion.name, assertion.expected, assertion.output)
		}
	}
}

func TestFormatNotebookTabsShowEmptyStates(t *testing.T) {
	result := datasetsSvc.NotebookRunResult{Label: "Empty", Cells: []datasetsSvc.NotebookCellRun{{CellID: "cell-1", Label: "Broken", Error: "blocked"}}}
	if !strings.Contains(formatNotebookRowsTab(result), "No tabular rows") {
		t.Fatalf("rows tab missing empty state")
	}
	if !strings.Contains(formatNotebookPlanTab(result), "Status: failed") {
		t.Fatalf("plan tab should include failed cell details")
	}
	if !strings.Contains(formatNotebookChartsTab(result), "No charts") {
		t.Fatalf("charts tab missing empty state")
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
