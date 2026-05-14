package workspace

import "testing"

func TestQueryCSVFiltersAcrossRows(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data/campaigns.csv", "channel,conversions\nOrganic,42\nPaid,7\n")

	result, err := QueryCSV(root, "data/campaigns.csv", "organic")
	if err != nil {
		t.Fatalf("QueryCSV returned error: %v", err)
	}

	if result.MatchedRows != 1 || len(result.Rows) != 1 {
		t.Fatalf("expected one matching row, got %#v", result)
	}
	if result.Rows[0][0] != "Organic" {
		t.Fatalf("unexpected row: %#v", result.Rows[0])
	}
}

func TestQueryCSVFiltersByColumn(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data/campaigns.csv", "channel,conversions\nOrganic,42\nPaid,7\n")

	result, err := QueryCSV(root, "data/campaigns.csv", "channel=Paid")
	if err != nil {
		t.Fatalf("QueryCSV returned error: %v", err)
	}

	if result.MatchedRows != 1 || result.Rows[0][0] != "Paid" {
		t.Fatalf("unexpected query result: %#v", result)
	}
}

func TestQueryCSVSupportsNumericComparisonOrderAndLimit(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data/campaigns.csv", "campaign,spend,channel\nA,10,Paid\nB,30,Organic\nC,20,Paid\n")

	result, err := QueryCSV(root, "data/campaigns.csv", "spend>10 order by spend desc limit 1")
	if err != nil {
		t.Fatalf("QueryCSV returned error: %v", err)
	}

	if result.MatchedRows != 2 {
		t.Fatalf("expected two matched rows before limit, got %#v", result)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "B" || result.Rows[0][1] != "30" {
		t.Fatalf("unexpected ordered limited result: %#v", result.Rows)
	}
}

func TestQueryCSVSupportsContainsAndRangeOperators(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data/campaigns.csv", "campaign,spend,channel\nA,10,Paid\nB,30,Organic\nC,20,Paid social\n")

	result, err := QueryCSV(root, "data/campaigns.csv", "channel contains Paid")
	if err != nil {
		t.Fatalf("QueryCSV returned error: %v", err)
	}
	if result.MatchedRows != 2 {
		t.Fatalf("expected two contains matches, got %#v", result)
	}

	result, err = QueryCSV(root, "data/campaigns.csv", "spend<=20")
	if err != nil {
		t.Fatalf("QueryCSV returned error: %v", err)
	}
	if result.MatchedRows != 2 {
		t.Fatalf("expected two range matches, got %#v", result)
	}
}
