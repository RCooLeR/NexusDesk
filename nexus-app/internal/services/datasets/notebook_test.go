package datasets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndListNotebooks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "data/sales.csv", "channel,spend\nsearch,12\n")

	saved, err := New(nil).SaveNotebook(root, NotebookSaveRequest{
		RelPath: "data/sales.csv",
		Label:   "Spend Analysis",
		Cells: []NotebookCell{
			{ID: "main cell", Kind: "sql", Label: "Top spend", SQL: "select channel, spend from dataset order by spend desc"},
			{ID: "chart", Kind: "chart", Label: "Spend chart"},
		},
	})
	if err != nil {
		t.Fatalf("SaveNotebook returned error: %v", err)
	}
	if saved.ID == "" || saved.Label != "Spend Analysis" || len(saved.Cells) != 2 || saved.Cells[0].ID != "main-cell" {
		t.Fatalf("unexpected saved notebook: %#v", saved)
	}

	notebooks, err := New(nil).ListNotebooks(root, "data/sales.csv")
	if err != nil {
		t.Fatalf("ListNotebooks returned error: %v", err)
	}
	if len(notebooks) != 1 || notebooks[0].Cells[0].SQL != "select channel, spend from dataset order by spend desc" {
		t.Fatalf("unexpected notebooks: %#v", notebooks)
	}
	if _, err := os.Stat(filepath.Join(root, ".nexusdesk", "datasets", "notebooks.json")); err != nil {
		t.Fatalf("expected notebook store to exist: %v", err)
	}
}

func TestSaveNotebookRejectsTraversal(t *testing.T) {
	if _, err := SaveNotebook(t.TempDir(), NotebookSaveRequest{RelPath: "../sales.csv"}); err == nil {
		t.Fatal("expected traversal path to fail")
	}
}

func TestSaveNotebookCleansAndCapsCells(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "sales.csv", "channel,spend\nsearch,12\n")
	cells := make([]NotebookCell, 0, maxNotebookCells+3)
	for index := 0; index < maxNotebookCells+3; index++ {
		cells = append(cells, NotebookCell{Kind: "sql", SQL: "select * from dataset limit 1"})
	}
	cells[0].SQL = strings.Repeat("x", maxNotebookSQLLength+10)

	saved, err := SaveNotebook(root, NotebookSaveRequest{RelPath: "sales.csv", Cells: cells})
	if err != nil {
		t.Fatalf("SaveNotebook returned error: %v", err)
	}
	if len(saved.Cells) != maxNotebookCells {
		t.Fatalf("expected cells to be capped, got %d", len(saved.Cells))
	}
	if len(saved.Cells[0].SQL) != maxNotebookSQLLength {
		t.Fatalf("expected SQL to be capped, got %d", len(saved.Cells[0].SQL))
	}
}
