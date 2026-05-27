package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSQLiteQueryArtifactsCreateCSVMarkdownAndMetadata(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "store.sqlite"), []byte("placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	report := SQLiteQueryReport{
		SourcePath:     "store.sqlite",
		SQL:            "select id, name from users",
		Engine:         "sqlite-readonly",
		Columns:        []string{"id", "name"},
		Rows:           [][]string{{"1", "Ada"}},
		TotalRows:      1,
		ResultLimit:    100,
		TimeoutSeconds: 30,
		Message:        "ok",
	}
	csvArtifact, err := store.WriteSQLiteQueryCSVArtifact(report)
	if err != nil {
		t.Fatal(err)
	}
	markdownArtifact, err := store.WriteSQLiteQueryMarkdownArtifact(report)
	if err != nil {
		t.Fatal(err)
	}
	if csvArtifact.Kind != "sqlite-query-csv" || markdownArtifact.Kind != "sqlite-query-report" {
		t.Fatalf("unexpected artifact kinds: %#v %#v", csvArtifact, markdownArtifact)
	}
	csvText, err := store.ReadArtifactText(csvArtifact.RelPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csvText, "id,name") || !strings.Contains(csvText, "1,Ada") {
		t.Fatalf("unexpected CSV artifact:\n%s", csvText)
	}
	markdownText, err := store.ReadArtifactText(markdownArtifact.RelPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# SQLite Query Report", "```sql", "| id | name |", "| 1 | Ada |"} {
		if !strings.Contains(markdownText, expected) {
			t.Fatalf("Markdown artifact missing %q:\n%s", expected, markdownText)
		}
	}
	matches, err := store.ListArtifacts(ListOptions{Query: "kind:sqlite-query-report"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].SourcePaths[0] != "store.sqlite" {
		t.Fatalf("expected searchable metadata, got %#v", matches)
	}
}

func TestWriteSQLiteQueryCSVRequiresColumns(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteSQLiteQueryCSVArtifact(SQLiteQueryReport{SourcePath: "store.sqlite"}); err == nil {
		t.Fatal("expected missing columns to be rejected")
	}
}
