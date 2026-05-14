package appmeta

import (
	"os"
	"testing"
)

func TestEnsureWritesSQLiteSchemaManifest(t *testing.T) {
	status, err := Ensure(t.TempDir())
	if err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	if status.SchemaVersion != 1 || status.SchemaHash == "" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if _, err := os.Stat(status.SchemaPath); err != nil {
		t.Fatalf("expected schema file: %v", err)
	}
	for _, table := range status.Tables {
		if !HasSchemaTable(SchemaSQL(), table) {
			t.Fatalf("schema missing table %s", table)
		}
	}
}
