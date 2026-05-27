package datasets

import (
	"strings"
	"testing"
)

func TestBuildDashboardCreatesMetricsAndSVG(t *testing.T) {
	dashboard, err := BuildDashboard(QueryResult{
		RelPath:     "campaigns.csv",
		Query:       "channel=search",
		Format:      "CSV",
		Columns:     []string{"channel", "spend"},
		TotalRows:   4,
		MatchedRows: 3,
		Rows: [][]string{
			{"search", "12.5"},
			{"email", "4"},
			{"search", "7.5"},
		},
	})
	if err != nil {
		t.Fatalf("BuildDashboard() error = %v", err)
	}
	if dashboard.RelPath != "campaigns.csv" || dashboard.Chart.Mode != "sum" || len(dashboard.Metrics) < 4 {
		t.Fatalf("unexpected dashboard metadata: %#v", dashboard)
	}
	for _, expected := range []string{"<svg", "Dataset Dashboard", "Shown rows", "Total spend", "spend by channel", "Query: channel=search"} {
		if !strings.Contains(dashboard.SVG, expected) {
			t.Fatalf("dashboard SVG missing %q:\n%s", expected, dashboard.SVG)
		}
	}
}

func TestBuildDashboardRejectsEmptyResults(t *testing.T) {
	if _, err := BuildDashboard(QueryResult{Columns: []string{"name"}}); err == nil {
		t.Fatal("expected empty dashboard result rejection")
	}
}
