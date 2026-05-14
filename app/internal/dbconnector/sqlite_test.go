package dbconnector

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestQuerySQLiteReadOnlyWorkspaceDatabase(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "sample.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE leads (channel TEXT, revenue INTEGER); INSERT INTO leads VALUES ('search', 10), ('email', 7);`); err != nil {
		t.Fatalf("seed sqlite failed: %v", err)
	}
	_ = db.Close()

	result, err := QuerySQLite(root, SQLiteQueryRequest{RelPath: "sample.sqlite", SQL: "select channel, revenue from leads order by revenue desc"})
	if err != nil {
		t.Fatalf("QuerySQLite returned error: %v", err)
	}
	if result.Engine != "sqlite-readonly" || len(result.Rows) != 2 || result.Rows[0][0] != "search" {
		t.Fatalf("unexpected query result: %#v", result)
	}
}

func TestQuerySQLiteBlocksMutation(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "sample.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	_ = db.Close()
	if _, err := QuerySQLite(root, SQLiteQueryRequest{RelPath: "sample.db", SQL: "drop table leads"}); err == nil {
		t.Fatal("expected mutation query to fail")
	}
}
