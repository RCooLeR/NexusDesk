package datasets

import (
	"strings"
	"testing"
)

func TestRunNotebookExecutesSQLAndChartCells(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "sales.csv", "channel,spend\nsearch,12\nemail,4\nsearch,20\n")
	notebook := Notebook{
		ID:      "sales-book",
		RelPath: "sales.csv",
		Label:   "Sales Notebook",
		Cells: []NotebookCell{
			{ID: "top", Kind: "sql", Label: "Top spend", SQL: "select channel, spend from dataset order by spend desc limit 1"},
			{ID: "chart", Kind: "chart", Label: "Spend chart", SQL: "select channel, spend from dataset"},
		},
	}
	result, err := New(nil).RunNotebook(root, notebook)
	if err != nil {
		t.Fatalf("RunNotebook returned error: %v", err)
	}
	if len(result.Cells) != 2 || !strings.Contains(result.Message, "Ran 2 notebook cell") {
		t.Fatalf("unexpected run result: %#v", result)
	}
	if result.Cells[0].Error != "" || len(result.Cells[0].SQLResult.Rows) != 1 || result.Cells[0].SQLResult.Rows[0][1] != "20" {
		t.Fatalf("unexpected SQL cell result: %#v", result.Cells[0])
	}
	if result.Cells[1].Error != "" || result.Cells[1].ChartResult.SVG == "" || len(result.Cells[1].ChartResult.Points) == 0 {
		t.Fatalf("unexpected chart cell result: %#v", result.Cells[1])
	}
}

func TestRunNotebookKeepsCellFailuresIsolated(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "sales.csv", "channel,spend\nsearch,12\n")
	notebook := Notebook{
		RelPath: "sales.csv",
		Label:   "Broken Notebook",
		Cells: []NotebookCell{
			{ID: "bad", Kind: "sql", Label: "Bad", SQL: "delete from dataset"},
			{ID: "good", Kind: "sql", Label: "Good", SQL: "select * from dataset limit 1"},
		},
	}
	result, err := New(nil).RunNotebook(root, notebook)
	if err != nil {
		t.Fatalf("RunNotebook returned error: %v", err)
	}
	if len(result.Cells) != 2 || result.Cells[0].Error == "" || result.Cells[1].Error != "" {
		t.Fatalf("expected isolated cell failure: %#v", result.Cells)
	}
	if !strings.Contains(result.Message, "1 failure") {
		t.Fatalf("expected failure summary, got %q", result.Message)
	}
}
