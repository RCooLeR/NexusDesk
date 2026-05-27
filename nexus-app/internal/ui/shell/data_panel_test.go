package shell

import (
	"strings"
	"testing"
	"time"

	datasetsSvc "nexusdesk/internal/services/datasets"
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

type errForTest string

func (e errForTest) Error() string {
	return string(e)
}
