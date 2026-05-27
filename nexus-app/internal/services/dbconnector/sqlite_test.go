package dbconnector

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestInspectWorkspaceSQLiteReturnsSchemaSamplesAndRelationships(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "data", "store.sqlite")
	if err := makeSQLiteFixture(dbPath); err != nil {
		t.Fatal(err)
	}

	metadata, err := New().InspectWorkspaceSQLite(root, filepath.Join("data", "store.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.RelPath != "data/store.sqlite" || !metadata.ReadOnly || metadata.Engine != "sqlite-readonly" {
		t.Fatalf("unexpected metadata header: %#v", metadata)
	}
	if len(metadata.Tables) != 2 || len(metadata.Views) != 1 {
		t.Fatalf("unexpected object counts: tables=%d views=%d", len(metadata.Tables), len(metadata.Views))
	}
	orders := findSQLiteObject(metadata.Tables, "orders")
	if orders.Name == "" || orders.RowCount != 2 || len(orders.Columns) != 3 || len(orders.SampleRows) != 2 {
		t.Fatalf("orders metadata incomplete: %#v", orders)
	}
	if len(orders.Indexes) != 1 || orders.Indexes[0].Columns[0] != "customer_id" {
		t.Fatalf("expected customer_id index, got %#v", orders.Indexes)
	}
	if len(metadata.Relationships) == 0 {
		t.Fatalf("expected relationship hints")
	}
}

func TestInspectWorkspaceSQLiteRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	if _, err := New().InspectWorkspaceSQLite(root, "../outside.sqlite"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
	if _, err := New().InspectWorkspaceSQLite(root, "notes.txt"); err == nil {
		t.Fatal("expected non-sqlite file to be rejected")
	}
}

func TestQueryWorkspaceSQLiteReturnsBoundedRows(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "data", "store.sqlite")
	if err := makeSQLiteFixture(dbPath); err != nil {
		t.Fatal(err)
	}
	result, err := New().QueryWorkspaceSQLite(root, SQLiteQueryRequest{
		RelPath:     filepath.Join("data", "store.sqlite"),
		SQL:         "select id, customer_id, total from orders order by id",
		ResultLimit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.RelPath != "data/store.sqlite" || result.Engine != "sqlite-readonly" || result.ResultLimit != 1 || result.TimeoutSeconds != DefaultSQLiteTimeoutSeconds {
		t.Fatalf("unexpected query header: %#v", result)
	}
	if len(result.Columns) != 3 || len(result.Rows) != 1 || result.Rows[0][0] != "10" || !result.Truncated {
		t.Fatalf("unexpected bounded result: %#v", result)
	}
}

func TestQueryWorkspaceSQLiteRejectsMutationsAndMultiStatement(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "data", "store.sqlite")
	if err := makeSQLiteFixture(dbPath); err != nil {
		t.Fatal(err)
	}
	service := New()
	for _, sqlText := range []string{
		"delete from orders",
		"select * from orders; select * from customers",
		"pragma table_info(orders)",
	} {
		if _, err := service.QueryWorkspaceSQLite(root, SQLiteQueryRequest{RelPath: filepath.Join("data", "store.sqlite"), SQL: sqlText}); err == nil {
			t.Fatalf("expected query to be rejected: %s", sqlText)
		}
	}
}

func findSQLiteObject(objects []SQLiteObject, name string) SQLiteObject {
	for _, object := range objects {
		if object.Name == name {
			return object
		}
	}
	return SQLiteObject{}
}

func makeSQLiteFixture(path string) error {
	if err := ensureFixtureDir(path); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`
		create table customers (id integer primary key, name text not null);
		create table orders (id integer primary key, customer_id integer not null references customers(id), total real);
		create index idx_orders_customer on orders(customer_id);
		insert into customers (id, name) values (1, 'Ada'), (2, 'Grace');
		insert into orders (id, customer_id, total) values (10, 1, 42.5), (11, 2, 7.25);
		create view order_totals as select customer_id, sum(total) total from orders group by customer_id;
	`)
	return err
}

func ensureFixtureDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}
