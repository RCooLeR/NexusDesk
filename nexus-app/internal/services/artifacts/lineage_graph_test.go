package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLineageGraphExportImportCreatesJSONArtifact(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"test":"go test ./..."}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "input.md"), []byte("# Input\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	taskArtifact, err := store.WriteTaskRunReport(TaskRunReport{
		ID:        "task-run-1",
		JobID:     "job-1",
		TaskID:    "task-1",
		Label:     "Test task",
		Command:   "go test ./...",
		Cwd:       ".",
		Source:    "package.json",
		Status:    "success",
		StartedAt: time.Date(2026, 5, 28, 9, 0, 0, 0, time.UTC),
		Message:   "ok",
	})
	if err != nil {
		t.Fatalf("WriteTaskRunReport returned error: %v", err)
	}
	docArtifact, err := store.WriteDocumentSetReport(DocumentSetReport{
		Title:       "Docs",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/input.md"},
		Content:     "# Input\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport returned error: %v", err)
	}

	lineage, err := store.LineageGraph(ListOptions{})
	if err != nil {
		t.Fatalf("LineageGraph returned error: %v", err)
	}
	if len(lineage.Nodes) < 6 || len(lineage.Edges) < 4 {
		t.Fatalf("expected graph to include artifact, source, job, and task nodes, got %#v", lineage)
	}
	if lineage.RelationshipCounts["generated"] != 1 || lineage.RelationshipCounts["cited"] != 2 || lineage.RelationshipCounts["ran"] != 1 {
		t.Fatalf("unexpected relationship counts: %#v", lineage.RelationshipCounts)
	}

	exported, err := store.WriteLineageGraphArtifact(lineage)
	if err != nil {
		t.Fatalf("WriteLineageGraphArtifact returned error: %v", err)
	}
	if exported.Kind != "artifact-lineage" || !strings.HasSuffix(exported.RelPath, ".json") {
		t.Fatalf("unexpected exported lineage artifact: %#v", exported)
	}
	listed, err := store.ListArtifacts(ListOptions{Query: "kind:artifact-lineage"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(listed) != 1 || listed[0].RelPath != exported.RelPath {
		t.Fatalf("expected JSON lineage artifact to be listed, got %#v", listed)
	}
	text, err := store.ReadArtifactText(exported.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText returned error: %v", err)
	}
	imported, err := ParseLineageJSON(text, exported.RelPath)
	if err != nil {
		t.Fatalf("ParseLineageJSON returned error: %v", err)
	}
	if len(imported.Lineage.Nodes) != len(lineage.Nodes) || !strings.Contains(imported.Message, exported.RelPath) {
		t.Fatalf("unexpected imported lineage: %#v", imported)
	}
	if !containsLineageRelPath(exported.SourcePaths, taskArtifact.RelPath) || !containsLineageRelPath(exported.SourcePaths, docArtifact.RelPath) {
		t.Fatalf("expected lineage export metadata to cite source artifacts, got %#v", exported.SourcePaths)
	}
}

func containsLineageRelPath(paths []string, want string) bool {
	for _, path := range paths {
		if path == want {
			return true
		}
	}
	return false
}
