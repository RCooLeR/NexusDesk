package artifacts

import (
	"os"
	"strings"
	"testing"
)

func TestRestoreArtifactMovesArchivedArtifactToOriginalPath(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	report, err := store.WriteDocumentSetReport(DocumentSetReport{
		Title:       "Docs",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "snapshot",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport returned error: %v", err)
	}
	archived, err := store.ArchiveArtifact(report.RelPath)
	if err != nil {
		t.Fatalf("ArchiveArtifact returned error: %v", err)
	}
	restored, err := store.RestoreArtifact(archived.RelPath)
	if err != nil {
		t.Fatalf("RestoreArtifact returned error: %v", err)
	}
	if restored.RelPath != report.RelPath || restored.Archived {
		t.Fatalf("unexpected restored artifact: %#v", restored)
	}
	if _, err := os.Stat(archived.AbsPath); !os.IsNotExist(err) {
		t.Fatalf("expected archived path to be moved away, stat err=%v", err)
	}
	if _, err := os.Stat(restored.AbsPath); err != nil {
		t.Fatalf("expected restored artifact file: %v", err)
	}
}

func TestRestoreArtifactAvoidsOverwritingExistingTarget(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	report, err := store.WriteDocumentSetReport(DocumentSetReport{
		Title:   "Docs",
		Roots:   []string{"docs"},
		Content: "snapshot",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport returned error: %v", err)
	}
	archived, err := store.ArchiveArtifact(report.RelPath)
	if err != nil {
		t.Fatalf("ArchiveArtifact returned error: %v", err)
	}
	if err := os.WriteFile(report.AbsPath, []byte("new artifact at original path"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	restored, err := store.RestoreArtifact(archived.RelPath)
	if err != nil {
		t.Fatalf("RestoreArtifact returned error: %v", err)
	}
	if restored.RelPath == report.RelPath || !strings.Contains(restored.RelPath, "restored") {
		t.Fatalf("expected collision-safe restore path, got %#v", restored)
	}
}

func TestRestoreArtifactRejectsActiveArtifacts(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	report, err := store.WriteTaskRunReport(TaskRunReport{ID: "task", Label: "Task", Command: "go test", Cwd: ".", Status: "success"})
	if err != nil {
		t.Fatalf("WriteTaskRunReport returned error: %v", err)
	}
	if _, err := store.RestoreArtifact(report.RelPath); err == nil {
		t.Fatal("expected active artifact restore to be rejected")
	}
}

func TestCleanRestoreRelPathRejectsArchiveAndMetadataTargets(t *testing.T) {
	for _, relPath := range []string{
		"",
		"../outside.md",
		".nexusdesk/artifacts/archive/report.md",
		".nexusdesk/artifacts/report.md.json",
	} {
		if got := cleanRestoreRelPath(relPath); got != "" {
			t.Fatalf("expected %q to be rejected, got %q", relPath, got)
		}
	}
	if got := cleanRestoreRelPath(".nexusdesk/artifacts/task-runs/report.md"); got == "" {
		t.Fatal("expected active artifact path to be accepted")
	}
}

func TestReadArtifactMetadataLoadsSidecar(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.WriteChatAnswer(ChatAnswerReport{Prompt: "Q", Content: "A", Model: "model-a"})
	if err != nil {
		t.Fatalf("WriteChatAnswer returned error: %v", err)
	}
	metadata, err := store.ReadArtifactMetadata(artifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactMetadata returned error: %v", err)
	}
	if metadata.Kind != "chat-answer" || metadata.Prompt != "Q" || metadata.Model != "model-a" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
}

func TestReadArtifactMetadataRejectsMissingSidecar(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	report, err := store.WriteTaskRunReport(TaskRunReport{ID: "task", Label: "Task", Command: "go test", Cwd: ".", Status: "success"})
	if err != nil {
		t.Fatalf("WriteTaskRunReport returned error: %v", err)
	}
	if err := os.Remove(report.AbsPath + ".json"); err != nil {
		t.Fatalf("remove metadata sidecar: %v", err)
	}
	if _, err := store.ReadArtifactMetadata(report.RelPath); err == nil {
		t.Fatal("expected missing sidecar to be rejected")
	}
}
