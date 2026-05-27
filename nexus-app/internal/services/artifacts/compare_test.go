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

func TestWriteArtifactComparisonReportCreatesSearchableArtifact(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	comparison := ArtifactComparison{
		Kind:       "document-report",
		LeftPath:   ".nexusdesk/artifacts/document-sets/left.md",
		RightPath:  ".nexusdesk/artifacts/document-sets/right.md",
		LeftTitle:  "Left",
		RightTitle: "Right",
		Diff:       "--- left\n+++ right\n-old\n+new\n",
		Message:    "Compared left with right.",
	}

	artifact, err := store.WriteArtifactComparisonReport(comparison)
	if err != nil {
		t.Fatalf("WriteArtifactComparisonReport returned error: %v", err)
	}
	if artifact.Kind != "artifact-comparison" || artifact.MetadataPath == "" || len(artifact.SourcePaths) != 2 {
		t.Fatalf("unexpected comparison artifact: %#v", artifact)
	}
	text, err := store.ReadArtifactText(artifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText returned error: %v", err)
	}
	for _, expected := range []string{"# Artifact Comparison - Left vs Right", "```diff", "-old", "+new"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("comparison report missing %q:\n%s", expected, text)
		}
	}
	matches, err := store.ListArtifacts(ListOptions{Query: "kind:artifact-comparison"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 1 || matches[0].RelPath != artifact.RelPath {
		t.Fatalf("expected searchable comparison artifact, got %#v", matches)
	}
}

func TestWriteArtifactComparisonReportRequiresPathsAndDiff(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteArtifactComparisonReport(ArtifactComparison{LeftPath: "left.md", Diff: "diff"}); err == nil {
		t.Fatal("expected missing right path to be rejected")
	}
	if _, err := store.WriteArtifactComparisonReport(ArtifactComparison{LeftPath: "left.md", RightPath: "right.md"}); err == nil {
		t.Fatal("expected missing diff to be rejected")
	}
}
