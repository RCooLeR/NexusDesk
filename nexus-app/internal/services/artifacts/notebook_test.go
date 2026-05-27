package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteNotebookRunReportCreatesMarkdownAndMetadata(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sales.csv"), []byte("channel,spend\nsearch,12\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	artifact, err := store.WriteNotebookRunReport(NotebookRunReport{
		SourcePath:  "sales.csv",
		NotebookID:  "book-1",
		Label:       "Sales Notebook",
		Message:     "Ran 2 notebook cell(s).",
		StartedAt:   time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC),
		CompletedAt: time.Date(2026, 5, 27, 10, 0, 1, 0, time.UTC),
		DurationMs:  1000,
		Cells: []NotebookRunCellReport{
			{
				CellID:      "cell-1",
				Label:       "Top spend",
				Kind:        "sql",
				SQL:         "select channel from dataset",
				Status:      "success",
				Engine:      "native-dataset-sql",
				Columns:     []string{"channel"},
				Rows:        [][]string{{"search"}},
				MatchedRows: 1,
				ShownRows:   1,
				Plan:        []string{"Validate SELECT-only native dataset SQL."},
				DurationMs:  8,
			},
			{
				CellID:       "chart-1",
				Label:        "Spend chart",
				Kind:         "chart",
				SQL:          "select channel, spend from dataset",
				Status:       "success",
				Engine:       "native-dataset-sql",
				ChartMode:    "sum",
				ChartMessage: "Bar chart.",
				ChartSVG:     `<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
				ChartPoints:  1,
				DurationMs:   10,
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteNotebookRunReport() error = %v", err)
	}
	if artifact.Kind != "sql-notebook-run" || !strings.Contains(artifact.RelPath, "/notebooks/") || !strings.HasSuffix(artifact.RelPath, ".md") {
		t.Fatalf("unexpected notebook artifact: %#v", artifact)
	}
	text, err := store.ReadArtifactText(artifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText() error = %v", err)
	}
	for _, expected := range []string{"# SQL Notebook Run - Sales Notebook", "sales.csv", "Top spend", "```sql", "| channel |", "Spend chart", "```svg"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("notebook artifact missing %q:\n%s", expected, text)
		}
	}
	matches, err := store.ListArtifacts(ListOptions{Query: "kind:sql-notebook-run"})
	if err != nil {
		t.Fatalf("ListArtifacts() error = %v", err)
	}
	if len(matches) != 1 || matches[0].RelPath != artifact.RelPath {
		t.Fatalf("expected searchable notebook artifact, got %#v", matches)
	}
}

func TestWriteNotebookRunReportRequiresSourceAndCells(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if _, err := store.WriteNotebookRunReport(NotebookRunReport{Cells: []NotebookRunCellReport{{CellID: "cell"}}}); err == nil {
		t.Fatal("expected missing source error")
	}
	if _, err := store.WriteNotebookRunReport(NotebookRunReport{SourcePath: "sales.csv"}); err == nil {
		t.Fatal("expected missing cells error")
	}
}
