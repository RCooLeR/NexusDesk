package dataset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndListNotebooks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "campaigns.csv"), []byte("campaign,spend\nA,10\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	saved, err := SaveNotebook(root, NotebookSaveRequest{
		RelPath: "campaigns.csv",
		Label:   "Spend Analysis",
		Cells: []NotebookCell{
			{ID: "cell-a", Kind: "sql", Label: "Top spend", SQL: "select * from dataset order by spend desc"},
			{ID: "cell-b", Kind: "chart", Label: "Spend chart"},
		},
	})
	if err != nil {
		t.Fatalf("SaveNotebook returned error: %v", err)
	}
	if saved.ID == "" || saved.Label != "Spend Analysis" || len(saved.Cells) != 2 {
		t.Fatalf("unexpected saved notebook: %#v", saved)
	}

	notebooks, err := ListNotebooks(root, "campaigns.csv")
	if err != nil {
		t.Fatalf("ListNotebooks returned error: %v", err)
	}
	if len(notebooks) != 1 || notebooks[0].Cells[0].SQL != "select * from dataset order by spend desc" {
		t.Fatalf("unexpected notebooks: %#v", notebooks)
	}
}

func TestSaveNotebookRejectsTraversal(t *testing.T) {
	if _, err := SaveNotebook(t.TempDir(), NotebookSaveRequest{RelPath: "../campaigns.csv"}); err == nil {
		t.Fatal("expected traversal path to fail")
	}
}
