package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildCSVChartCountsCategories(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "leads.csv"), []byte("channel,revenue\nsearch,10\nsocial,4\nsearch,6\nemail,3\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	chart, err := BuildCSVChart(root, DatasetChartRequest{
		RelPath:        "leads.csv",
		CategoryColumn: "channel",
	})
	if err != nil {
		t.Fatalf("BuildCSVChart returned error: %v", err)
	}

	if chart.Mode != "count" {
		t.Fatalf("expected count mode, got %s", chart.Mode)
	}
	if len(chart.Points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(chart.Points))
	}
	if chart.Points[0].Label != "search" || chart.Points[0].Value != 2 || chart.Points[0].Count != 2 {
		t.Fatalf("unexpected top point: %#v", chart.Points[0])
	}
}

func TestBuildCSVChartSumsNumericValues(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "leads.csv"), []byte("channel,revenue\nsearch,10\nsocial,4\nsearch,6\nemail,3\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	chart, err := BuildCSVChart(root, DatasetChartRequest{
		RelPath:        "leads.csv",
		CategoryColumn: "channel",
		ValueColumn:    "revenue",
	})
	if err != nil {
		t.Fatalf("BuildCSVChart returned error: %v", err)
	}

	if chart.Mode != "sum" {
		t.Fatalf("expected sum mode, got %s", chart.Mode)
	}
	if chart.Points[0].Label != "search" || chart.Points[0].Value != 16 || chart.Points[0].Count != 2 {
		t.Fatalf("unexpected top point: %#v", chart.Points[0])
	}
}

func TestBuildCSVChartSupportsLineType(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "daily.csv"), []byte("day,revenue\nMon,10\nTue,4\nWed,6\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	chart, err := BuildCSVChart(root, DatasetChartRequest{
		RelPath:        "daily.csv",
		ChartType:      "line",
		CategoryColumn: "day",
		ValueColumn:    "revenue",
	})
	if err != nil {
		t.Fatalf("BuildCSVChart returned error: %v", err)
	}

	if chart.ChartType != "line" {
		t.Fatalf("expected line chart, got %s", chart.ChartType)
	}
	if chart.Points[0].Label != "Mon" || chart.Points[1].Label != "Tue" {
		t.Fatalf("expected line chart to preserve input order, got %#v", chart.Points)
	}
}

func TestBuildCSVChartRejectsMissingCategory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "leads.csv"), []byte("channel,revenue\nsearch,10\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if _, err := BuildCSVChart(root, DatasetChartRequest{RelPath: "leads.csv", CategoryColumn: "missing"}); err == nil {
		t.Fatal("expected missing category column to fail")
	}
}
