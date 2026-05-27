package artifacts

import (
	"strings"
	"testing"
)

func TestCompareArtifactsBuildsSameKindDiff(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	left, err := store.WriteDocumentSetReport(DocumentSetReport{
		Title:       "Left",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "alpha\nbeta\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport left returned error: %v", err)
	}
	right, err := store.WriteDocumentSetReport(DocumentSetReport{
		Title:       "Right",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "alpha\nBETA\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport right returned error: %v", err)
	}

	comparison, err := store.CompareArtifacts(left.RelPath, right.RelPath)
	if err != nil {
		t.Fatalf("CompareArtifacts returned error: %v", err)
	}
	if comparison.Kind != "document-report" || comparison.Same {
		t.Fatalf("unexpected comparison: %#v", comparison)
	}
	for _, expected := range []string{"--- " + left.RelPath, "+++ " + right.RelPath, "-beta", "+BETA"} {
		if !strings.Contains(comparison.Diff, expected) {
			t.Fatalf("diff missing %q:\n%s", expected, comparison.Diff)
		}
	}
}

func TestCompareArtifactsRejectsSamePathAndCrossKind(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	task, err := store.WriteTaskRunReport(TaskRunReport{ID: "task", Label: "Task", Command: "go test", Cwd: ".", Status: "success", Message: "ok"})
	if err != nil {
		t.Fatalf("WriteTaskRunReport returned error: %v", err)
	}
	doc, err := store.WriteDocumentSetReport(DocumentSetReport{Title: "Doc", Roots: []string{"docs"}, Content: "hello"})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport returned error: %v", err)
	}
	if _, err := store.CompareArtifacts(task.RelPath, task.RelPath); err == nil {
		t.Fatal("expected same path comparison to be rejected")
	}
	if _, err := store.CompareArtifacts(task.RelPath, doc.RelPath); err == nil || !strings.Contains(err.Error(), "kinds must match") {
		t.Fatalf("expected kind mismatch, got %v", err)
	}
}

func TestBuildArtifactDiffGuardsLineDenseInputs(t *testing.T) {
	left := strings.Repeat("a\n", artifactDiffMaxTotalLines)
	diff := buildArtifactDiff("left.md", "right.md", left, "b\n")
	for _, expected := range []string{"too line-dense", "Left lines: 10000", "Right lines: 1"} {
		if !strings.Contains(diff, expected) {
			t.Fatalf("diff missing %q:\n%s", expected, diff)
		}
	}
}
