package main

import (
	"strings"
	"testing"

	"NexusDesk/internal/appmeta"
)

func TestSearchIncludesSanitizedSQLRunRows(t *testing.T) {
	root := t.TempDir()
	if _, err := appmeta.Ensure(root); err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}

	app := NewApp()
	rawError := "provider returned HTTP 500: error api_key=super-secret-token in request body " + strings.Repeat("x", 500)
	app.recordSQLRun(root, "data.csv", "select * from events", "duckdb", 0, "", "failed", sanitizeProviderMessage(rawError))

	results, err := appmeta.Search(root, "data.csv", 10)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected SQL run search result")
	}

	found := false
	for _, result := range results {
		if result.Kind != "sql-run" {
			continue
		}
		found = true
		if !strings.Contains(result.Snippet, "[redacted]") {
			t.Fatalf("expected redacted marker in SQL run snippet, got %q", result.Snippet)
		}
		if strings.Contains(result.Snippet, "super-secret-token") {
			t.Fatalf("expected SQL run snippet to be redacted, got %q", result.Snippet)
		}
		if len(result.Snippet) > 220 {
			t.Fatalf("expected metadata snippet to stay bounded, got len=%d", len(result.Snippet))
		}
	}
	if !found {
		t.Fatalf("expected to find sql-run kind result, got %#v", results)
	}
}
