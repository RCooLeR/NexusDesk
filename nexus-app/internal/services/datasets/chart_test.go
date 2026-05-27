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

func TestBuildChartBuildsLineChartForOrderedNumericSeries(t *testing.T) {
	chart, err := BuildChart(QueryResult{
		RelPath: "daily.csv",
		Format:  "CSV",
		Columns: []string{"date", "visits"},
		Rows: [][]string{
			{"2026-05-03", "19"},
			{"2026-05-01", "11"},
			{"2026-05-02", "15"},
		},
	})
	if err != nil {
		t.Fatalf("BuildChart() error = %v", err)
	}
	if chart.Mode != "line" || chart.CategoryColumn != "date" || chart.ValueColumn != "visits" {
		t.Fatalf("unexpected line chart metadata: %#v", chart)
	}
	if len(chart.Points) != 3 || chart.Points[0].Label != "2026-05-01" || chart.Points[2].Value != 19 {
		t.Fatalf("unexpected ordered points: %#v", chart.Points)
	}
	if !strings.Contains(chart.SVG, "<polyline") || !strings.Contains(chart.SVG, "visits over date") {
		t.Fatalf("unexpected line SVG: %s", chart.SVG)
	}
}

func TestBuildChartLineChartHandlesNegativeValues(t *testing.T) {
	chart, err := BuildChart(QueryResult{
		RelPath: "temperatures.csv",
		Format:  "CSV",
		Columns: []string{"hour", "temperature"},
		Rows: [][]string{
			{"1", "-4"},
			{"2", "-2"},
			{"3", "-7"},
		},
	})
	if err != nil {
		t.Fatalf("BuildChart() error = %v", err)
	}
	if chart.Mode != "line" || !strings.Contains(chart.SVG, "<polyline") {
		t.Fatalf("unexpected negative-value line chart: %#v", chart)
	}
}

func TestBuildChartRejectsEmptyResults(t *testing.T) {
	if _, err := BuildChart(QueryResult{Columns: []string{"name"}}); err == nil {
		t.Fatal("expected empty result rejection")
	}
}
