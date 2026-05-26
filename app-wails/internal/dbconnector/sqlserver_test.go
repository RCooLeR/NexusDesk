package dbconnector

import (
	"strings"
	"testing"

	"NexusAugenticStudio/internal/storage"
)

func TestRequireSQLServerProfile(t *testing.T) {
	err := requireSQLServerProfile(storage.ConnectorProfile{ID: "mysql", Kind: "mysql", Host: "db", Database: "app"})
	if err == nil || !strings.Contains(err.Error(), "not SQL Server") {
		t.Fatalf("expected kind validation error, got %v", err)
	}
	err = requireSQLServerProfile(storage.ConnectorProfile{ID: "mssql", Kind: "sqlserver", Host: "db", Database: "app"})
	if err != nil {
		t.Fatalf("expected valid sqlserver profile, got %v", err)
	}
}

func TestSQLServerDSN(t *testing.T) {
	dsn := sqlServerDSN(storage.ConnectorProfile{
		Kind:     "sqlserver",
		Host:     "db.example.test",
		Port:     1434,
		Database: "analytics",
		Username: "analyst",
		Password: "secret/value",
		SSLMode:  "skip-verify",
	}, 12)
	if !strings.Contains(dsn, "sqlserver://analyst:secret%2Fvalue@db.example.test:1434") {
		t.Fatalf("unexpected dsn: %s", dsn)
	}
	if !strings.Contains(dsn, "database=analytics") || !strings.Contains(dsn, "connection+timeout=12") {
		t.Fatalf("expected database and timeout params, got %s", dsn)
	}
	if !strings.Contains(dsn, "encrypt=true") || !strings.Contains(dsn, "TrustServerCertificate=true") {
		t.Fatalf("expected encryption skip-verify params, got %s", dsn)
	}
}

func TestSQLServerEncryptMode(t *testing.T) {
	cases := map[string]string{
		"disable":     "disable",
		"require":     "true",
		"prefer":      "false",
		"skip-verify": "true",
	}
	for input, expected := range cases {
		if actual := sqlServerEncryptMode(input); actual != expected {
			t.Fatalf("sqlServerEncryptMode(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestSplitSQLServerCSV(t *testing.T) {
	columns := splitSQLServerCSV("id, customer_id,created_at")
	if len(columns) != 3 || columns[1] != "customer_id" {
		t.Fatalf("unexpected columns: %#v", columns)
	}
}
