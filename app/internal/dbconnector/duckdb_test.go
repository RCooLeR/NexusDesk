package dbconnector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"NexusAugenticStudio/internal/storage"
)

func TestRequireDuckDBProfile(t *testing.T) {
	err := requireDuckDBProfile(storage.ConnectorProfile{ID: "pg", Kind: "postgres", Host: "db.duckdb"})
	if err == nil || !strings.Contains(err.Error(), "not DuckDB") {
		t.Fatalf("expected kind validation error, got %v", err)
	}

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "warehouse.duckdb")
	if err := os.WriteFile(dbPath, []byte("duckdb placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = requireDuckDBProfile(storage.ConnectorProfile{ID: "duck", Kind: "duckdb", Host: dbPath})
	if err != nil {
		t.Fatalf("expected valid DuckDB profile, got %v", err)
	}
}

func TestDuckDBProfilePathPrefersDatabase(t *testing.T) {
	profile := storage.ConnectorProfile{
		Kind:     "duckdb",
		Host:     "host-file.duckdb",
		Database: "database-file.duckdb",
	}
	if got := duckDBProfilePath(profile); got != filepath.Clean("database-file.duckdb") {
		t.Fatalf("expected database path to win, got %q", got)
	}
}

func TestDuckDBDSNUsesReadOnlyAccessMode(t *testing.T) {
	profile := storage.ConnectorProfile{Kind: "duckdb", Database: filepath.Join("data", "main.duckdb")}
	dsn := duckDBDSN(profile)
	if !strings.Contains(dsn, "access_mode=read_only") {
		t.Fatalf("expected read-only access mode, got %q", dsn)
	}
}
