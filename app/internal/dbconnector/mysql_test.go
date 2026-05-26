package dbconnector

import (
	"strings"
	"testing"

	"NexusAugenticStudio/internal/storage"
)

func TestRequireMySQLProfile(t *testing.T) {
	err := requireMySQLProfile(storage.ConnectorProfile{ID: "pg", Kind: "postgres", Host: "db", Database: "app"})
	if err == nil || !strings.Contains(err.Error(), "not MySQL/MariaDB") {
		t.Fatalf("expected kind validation error, got %v", err)
	}
	err = requireMySQLProfile(storage.ConnectorProfile{ID: "mysql", Kind: "mysql", Host: "db", Database: "app"})
	if err != nil {
		t.Fatalf("expected valid mysql profile, got %v", err)
	}
	err = requireMySQLProfile(storage.ConnectorProfile{ID: "mariadb", Kind: "mariadb", Host: "db", Database: "app"})
	if err != nil {
		t.Fatalf("expected valid mariadb profile, got %v", err)
	}
}

func TestMySQLDSN(t *testing.T) {
	dsn := mysqlDSN(storage.ConnectorProfile{
		Kind:     "mysql",
		Host:     "db.example.test",
		Port:     3307,
		Database: "analytics",
		Username: "analyst",
		Password: "secret/value",
		SSLMode:  "require",
	}, 12)
	if !strings.Contains(dsn, "analyst:secret/value@tcp(db.example.test:3307)/analytics") {
		t.Fatalf("unexpected dsn: %s", dsn)
	}
	if !strings.Contains(dsn, "tls=true") || !strings.Contains(dsn, "timeout=12s") {
		t.Fatalf("expected tls and timeout params, got %s", dsn)
	}
}

func TestMySQLTLSMode(t *testing.T) {
	cases := map[string]string{
		"disable":     "false",
		"require":     "true",
		"preferred":   "preferred",
		"skip-verify": "skip-verify",
	}
	for input, expected := range cases {
		if actual := mysqlTLSMode(input); actual != expected {
			t.Fatalf("mysqlTLSMode(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestSplitMySQLCSV(t *testing.T) {
	columns := splitMySQLCSV("id, customer_id,created_at")
	if len(columns) != 3 || columns[1] != "customer_id" {
		t.Fatalf("unexpected columns: %#v", columns)
	}
}

func TestMySQLEngineNames(t *testing.T) {
	if mysqlEngine(storage.ConnectorProfile{Kind: "mariadb"}) != "mariadb-readonly" {
		t.Fatal("expected mariadb engine")
	}
	if mysqlDisplayName(storage.ConnectorProfile{Kind: "mysql"}) != "MySQL" {
		t.Fatal("expected MySQL display name")
	}
}
