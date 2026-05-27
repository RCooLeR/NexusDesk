package datasets

import (
	"strings"
	"testing"
)

func TestBuildChartAggregatesNumericColumnByFirstColumn(t *testing.T) {
	chart, err := BuildChart(QueryResult{
		RelPath: "campaigns.csv",
		Format:  "CSV",
		Columns: []string{"channel", "spend"},
		Rows: [][]string{
			{"search", "12.5"},
			{"email", "4"},
			{"search", "7.5"},
		},
	})
	if err != nil {
		t.Fatalf("BuildChart() error = %v", err)
	}
	if chart.Mode != "sum" || chart.CategoryColumn != "channel" || chart.ValueColumn != "spend" {
		t.Fatalf("unexpected chart metadata: %#v", chart)
	}
	if len(chart.Points) != 2 || chart.Points[0].Label != "search" || chart.Points[0].Value != 20 {
		t.Fatalf("unexpected chart points: %#v", chart.Points)
	}
	if !strings.Contains(chart.SVG, "<svg") || !strings.Contains(chart.SVG, "spend by channel") {
		t.Fatalf("unexpected SVG: %s", chart.SVG)
	}
}

func TestBuildChartFallsBackToCounts(t *testing.T) {
	chart, err := BuildChart(QueryResult{
		RelPath: "events.json",
		Format:  "JSON",
		Columns: []string{"channel", "kind"},
		Rows: [][]string{
			{"search", "click"},
			{"search", "view"},
			{"email", "click"},
		},
	})
	if err != nil {
		t.Fatalf("BuildChart() error = %v", err)
	}
	if chart.Mode != "count" || chart.ValueColumn != "" {
		t.Fatalf("unexpected count chart: %#v", chart)
	}
	if chart.Points[0].Label != "search" || chart.Points[0].Value != 2 {
		t.Fatalf("unexpected count points: %#v", chart.Points)
	}
}

func TestBuildChartRejectsEmptyResults(t *testing.T) {
	if _, err := BuildChart(QueryResult{Columns: []string{"name"}}); err == nil {
		t.Fatal("expected empty result rejection")
	}
}
