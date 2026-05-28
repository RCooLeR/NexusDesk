package artifacts

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateUniqueArtifactFileRetriesWhenNameExists(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, 5, 28, 20, 30, 0, 123, time.UTC)
	existingRel := store.relPath("document-briefs", artifactTimestamp(createdAt)+"-"+safeName("Document Brief")+".md")
	existingAbs := store.absPath(existingRel)
	if err := os.MkdirAll(filepath.Dir(existingAbs), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(existingAbs, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	relPath, absPath, file, err := store.createUniqueArtifactFile("document-briefs", "Document Brief", "md", createdAt)
	if err != nil {
		t.Fatalf("createUniqueArtifactFile returned error: %v", err)
	}
	if file == nil {
		t.Fatal("expected created file handle")
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if relPath == existingRel || absPath == existingAbs {
		t.Fatalf("expected retry path, got original rel=%q abs=%q", relPath, absPath)
	}
	if _, err := os.Stat(absPath); err != nil {
		t.Fatalf("expected retry file to exist: %v", err)
	}
}
