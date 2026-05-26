package dbconnector

import (
	"context"
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
	if result.TotalRows != 100 {
		t.Fatalf("expected TotalRows to count returned rows up to cap, got %d", result.TotalRows)
	}
	if !result.Truncated || !strings.Contains(result.Message, "row cap") {
		t.Fatalf("expected capped result message, got truncated=%t message=%q", result.Truncated, result.Message)
	}
}

func TestQuerySQLiteHonorsCustomRowCap(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "sample.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE hits (value INTEGER); INSERT INTO hits VALUES (1), (2), (3);`); err != nil {
		_ = db.Close()
		t.Fatalf("seed sqlite failed: %v", err)
	}
	_ = db.Close()

	result, err := QuerySQLite(root, SQLiteQueryRequest{RelPath: "sample.db", SQL: "select value from hits order by value", ResultLimit: 2, TimeoutSeconds: 5})
	if err != nil {
		t.Fatalf("QuerySQLite returned error: %v", err)
	}
	if len(result.Rows) != 2 || !result.Truncated {
		t.Fatalf("expected custom row cap to truncate to 2 rows, got %#v", result)
	}
	if result.ResultLimit != 2 || result.TimeoutSeconds != 5 {
		t.Fatalf("expected query controls in result, got %#v", result)
	}
}

func TestQuerySQLiteContextCancellation(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "sample.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE hits (value INTEGER); INSERT INTO hits VALUES (1);`); err != nil {
		_ = db.Close()
		t.Fatalf("seed sqlite failed: %v", err)
	}
	_ = db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = QuerySQLiteContext(ctx, root, SQLiteQueryRequest{RelPath: "sample.db", SQL: "select value from hits"})
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected canceled query error, got %v", err)
	}
}

func TestRedactConnectorError(t *testing.T) {
	redacted := RedactConnectorError(`database rejected password=secret-token for user`)
	if strings.Contains(redacted, "secret-token") || !strings.Contains(redacted, "[redacted]") {
		t.Fatalf("expected connector error to be redacted, got %q", redacted)
	}
}

func TestInspectSQLiteReturnsConnectorMetadata(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "sample.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE leads (id INTEGER PRIMARY KEY, channel TEXT NOT NULL, revenue INTEGER DEFAULT 0);
		CREATE INDEX leads_channel_idx ON leads(channel);
		INSERT INTO leads (channel, revenue) VALUES ('search', 10), ('email', 7);
		CREATE VIEW lead_channels AS SELECT channel FROM leads;
	`)
	if err != nil {
		_ = db.Close()
		t.Fatalf("seed sqlite failed: %v", err)
	}
	_ = db.Close()

	metadata, err := InspectSQLite(root, "sample.sqlite")
	if err != nil {
		t.Fatalf("InspectSQLite returned error: %v", err)
	}
	if metadata.Kind != "sqlite" || !metadata.ReadOnly || metadata.Engine != "sqlite-readonly" {
		t.Fatalf("unexpected metadata header: %#v", metadata)
	}
	if len(metadata.Tables) != 1 || metadata.Tables[0].Name != "leads" {
		t.Fatalf("unexpected tables: %#v", metadata.Tables)
	}
	table := metadata.Tables[0]
	if table.RowCount != 2 || len(table.Columns) != 3 || len(table.SampleRows) != 2 {
		t.Fatalf("unexpected table metadata: %#v", table)
	}
	if table.Columns[0].Name != "id" || !table.Columns[0].PrimaryKey {
		t.Fatalf("expected primary key column metadata, got %#v", table.Columns[0])
	}
	if table.Columns[1].Name != "channel" || table.Columns[1].Nullable {
		t.Fatalf("expected non-null channel column metadata, got %#v", table.Columns[1])
	}
	if len(table.Indexes) != 1 || table.Indexes[0].Columns[0] != "channel" {
		t.Fatalf("unexpected index metadata: %#v", table.Indexes)
	}
	if len(metadata.Views) != 1 || metadata.Views[0].Name != "lead_channels" {
		t.Fatalf("unexpected views: %#v", metadata.Views)
	}
}

func TestInspectSQLiteReturnsRelationshipHints(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "sample.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	_, err = db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE customers (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY,
			customer_id INTEGER NOT NULL,
			owner_id INTEGER,
			FOREIGN KEY(customer_id) REFERENCES customers(id)
		);
		CREATE TABLE owners (id INTEGER PRIMARY KEY, name TEXT);
	`)
	if err != nil {
		_ = db.Close()
		t.Fatalf("seed sqlite failed: %v", err)
	}
	_ = db.Close()

	metadata, err := InspectSQLite(root, "sample.sqlite")
	if err != nil {
		t.Fatalf("InspectSQLite returned error: %v", err)
	}

	if len(metadata.Relationships) != 2 {
		t.Fatalf("expected explicit and inferred relationships, got %#v", metadata.Relationships)
	}
	if !hasConnectorRelationship(metadata.Relationships, "foreign-key", "orders", "customer_id", "customers", "id") {
		t.Fatalf("expected foreign key relationship, got %#v", metadata.Relationships)
	}
	if !hasConnectorRelationship(metadata.Relationships, "inferred", "orders", "owner_id", "owners", "id") {
		t.Fatalf("expected inferred relationship, got %#v", metadata.Relationships)
	}
}

func hasConnectorRelationship(relationships []ConnectorRelationship, kind string, fromTable string, fromColumn string, toTable string, toColumn string) bool {
	for _, relationship := range relationships {
		if relationship.Kind == kind &&
			relationship.FromTable == fromTable &&
			relationship.FromColumn == fromColumn &&
			relationship.ToTable == toTable &&
			relationship.ToColumn == toColumn {
			return true
		}
	}
	return false
}
