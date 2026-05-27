package datasets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndListSavedSQLiteQueries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "data.sqlite"), []byte("placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}

	first, err := SaveQuery(root, "data.sqlite", "select * from users", "Users", "sqlite-sql")
	if err != nil {
		t.Fatal(err)
	}
	if first.Label != "Users" || first.Kind != "sqlite-sql" || first.RelPath != "data.sqlite" {
		t.Fatalf("unexpected saved query: %#v", first)
	}
	if _, err := SaveQuery(root, "data.sqlite", "select * from orders", "", "sqlite-sql"); err != nil {
		t.Fatal(err)
	}
	queries, err := ListSavedQueries(root, "data.sqlite", "sqlite-sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 || !strings.Contains(queries[0].Query, "orders") {
		t.Fatalf("expected newest SQLite query first, got %#v", queries)
	}
}

func TestSavedQueriesAreKindScopedAndTrimmed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "data.csv"), []byte("id\n1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < maxSavedQueriesPerSource+2; index++ {
		if _, err := SaveQuery(root, "data.csv", "select "+string(rune('a'+index)), "", "sql"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := SaveQuery(root, "data.csv", "filter", "", "filter"); err != nil {
		t.Fatal(err)
	}
	sqlQueries, err := ListSavedQueries(root, "data.csv", "sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(sqlQueries) != maxSavedQueriesPerSource {
		t.Fatalf("expected trimmed SQL queries, got %d", len(sqlQueries))
	}
	filterQueries, err := ListSavedQueries(root, "data.csv", "filter")
	if err != nil {
		t.Fatal(err)
	}
	if len(filterQueries) != 1 || filterQueries[0].Kind != "filter" {
		t.Fatalf("expected filter query to stay kind scoped, got %#v", filterQueries)
	}
}
