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
