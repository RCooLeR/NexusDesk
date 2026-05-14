package dataset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndListSavedQueries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "leads.csv"), []byte("channel,revenue\nsearch,10\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	saved, err := SaveQuery(root, "leads.csv", "channel=search", "Search leads")
	if err != nil {
		t.Fatalf("SaveQuery returned error: %v", err)
	}
	if saved.Label != "Search leads" || saved.Query != "channel=search" || saved.Kind != "filter" {
		t.Fatalf("unexpected saved query: %#v", saved)
	}
	sql, err := SaveQueryKind(root, "leads.csv", "select * from dataset", "All rows", "sql")
	if err != nil {
		t.Fatalf("SaveQueryKind returned error: %v", err)
	}
	if sql.Kind != "sql" {
		t.Fatalf("expected SQL query kind: %#v", sql)
	}

	queries, err := ListSavedQueries(root, "leads.csv")
	if err != nil {
		t.Fatalf("ListSavedQueries returned error: %v", err)
	}
	if len(queries) != 1 || queries[0].Query != "channel=search" {
		t.Fatalf("unexpected saved queries: %#v", queries)
	}
	sqlQueries, err := ListSavedQueriesKind(root, "leads.csv", "sql")
	if err != nil {
		t.Fatalf("ListSavedQueriesKind returned error: %v", err)
	}
	if len(sqlQueries) != 1 || sqlQueries[0].Query != "select * from dataset" {
		t.Fatalf("unexpected saved SQL queries: %#v", sqlQueries)
	}
}

func TestSaveQueryRejectsTraversal(t *testing.T) {
	if _, err := SaveQuery(t.TempDir(), "../leads.csv", "x", "bad"); err == nil {
		t.Fatal("expected traversal path to fail")
	}
}
