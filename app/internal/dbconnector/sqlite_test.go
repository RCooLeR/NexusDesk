package dbconnector

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
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
	if _, err := db.Exec(`CREATE TABLE leads (channel TEXT); INSERT INTO leads VALUES ('search');`); err != nil {
		_ = db.Close()
		t.Fatalf("seed sqlite failed: %v", err)
	}
	_ = db.Close()
	if _, err := QuerySQLite(root, SQLiteQueryRequest{RelPath: "sample.db", SQL: "drop table leads"}); err == nil {
		t.Fatal("expected mutation query to fail")
	}
}

func TestQuerySQLiteRejectsMultiStatementSQL(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "sample.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE leads (channel TEXT); INSERT INTO leads VALUES ('search');`); err != nil {
		_ = db.Close()
		t.Fatalf("seed sqlite failed: %v", err)
	}
	_ = db.Close()
	if _, err := QuerySQLite(root, SQLiteQueryRequest{
		RelPath: "sample.sqlite",
		SQL:     "select channel from leads where 1=1; select channel from leads",
	}); err == nil || !strings.Contains(err.Error(), "single SQL statement") {
		t.Fatalf("expected multi-statement query to fail, got %v", err)
	}
}

func TestQuerySQLiteAllowsQuotedSemicolon(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "sample.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE t (name TEXT); INSERT INTO t VALUES ('a;b'), ('c');`)
	if err != nil {
		t.Fatalf("seed sqlite failed: %v", err)
	}
	_ = db.Close()

	result, err := QuerySQLite(root, SQLiteQueryRequest{RelPath: "sample.db", SQL: "select name from t where name='a;b'"})
	if err != nil {
		t.Fatalf("QuerySQLite returned error: %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "a;b" {
		t.Fatalf("unexpected quoted-semicolon query result: %#v", result.Rows)
	}
}

func TestQuerySQLiteCapsReturnedRows(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "sample.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE hits (value INTEGER);`); err != nil {
		_ = db.Close()
		t.Fatalf("seed sqlite failed: %v", err)
	}
	for index := 1; index <= 150; index++ {
		if _, err := db.Exec(fmt.Sprintf("INSERT INTO hits VALUES (%d)", index)); err != nil {
			_ = db.Close()
			t.Fatalf("seed sqlite row %d failed: %v", index, err)
		}
	}
	_ = db.Close()

	result, err := QuerySQLite(root, SQLiteQueryRequest{RelPath: "sample.db", SQL: "select value from hits order by value"})
	if err != nil {
		t.Fatalf("QuerySQLite returned error: %v", err)
	}
	if len(result.Rows) != 100 {
		t.Fatalf("expected row cap to limit output to 100, got %d", len(result.Rows))
	}
	if result.TotalRows != 150 {
		t.Fatalf("expected TotalRows to include all rows, got %d", result.TotalRows)
	}
	if !strings.Contains(result.Message, "showing 100") {
		t.Fatalf("expected message to include showing count, got %q", result.Message)
	}
}
