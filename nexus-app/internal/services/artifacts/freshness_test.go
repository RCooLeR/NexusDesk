package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSourceFreshnessFlagsChangedAndMissingSources(t *testing.T) {
	root := t.TempDir()
	writeArtifactSource(t, root, "docs/a.md", "old")
	writeArtifactSource(t, root, "docs/b.md", "old")

	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.WriteDocumentSetReport(DocumentSetReport{
		Title:       "Docs",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md", "docs/b.md", "docs/missing.md"},
		Content:     "snapshot",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport returned error: %v", err)
	}
	generatedAt := artifact.GeneratedAt
	if generatedAt.IsZero() {
		t.Fatal("expected generated timestamp")
	}
	future := generatedAt.Add(2 * time.Second)
	if err := os.Chtimes(filepath.Join(root, "docs", "b.md"), future, future); err != nil {
		t.Fatalf("Chtimes returned error: %v", err)
	}

	report, err := store.SourceFreshness(artifact.RelPath)
	if err != nil {
		t.Fatalf("SourceFreshness returned error: %v", err)
	}
	if !report.Stale || report.ChangedCount != 1 || report.MissingCount != 1 || len(report.Sources) != 3 {
		t.Fatalf("unexpected freshness report: %#v", report)
	}
	if !strings.Contains(report.Message, "changed source") || !strings.Contains(report.Message, "missing source") {
		t.Fatalf("unexpected freshness message: %q", report.Message)
	}
}

func TestSourceFreshnessRejectsUnsafeSourcePath(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.WriteDocumentSetReport(DocumentSetReport{
		Title:       "Unsafe",
		Roots:       []string{"docs"},
		SourcePaths: []string{"../outside.md"},
		Content:     "snapshot",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport returned error: %v", err)
	}

	report, err := store.SourceFreshness(artifact.RelPath)
	if err != nil {
		t.Fatalf("SourceFreshness returned error: %v", err)
	}
	if report.UnknownCount != 1 || !report.Sources[0].Unknown {
		t.Fatalf("expected unsafe source to be unknown, got %#v", report)
	}
}

func TestSourceFreshnessDetectsSameTimestampContentChange(t *testing.T) {
	root := t.TempDir()
	writeArtifactSource(t, root, "docs/a.md", "old")

	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.WriteDocumentSetReport(DocumentSetReport{
		Title:       "Docs",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "snapshot",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport returned error: %v", err)
	}
	metadata, _ := store.readMetadata(artifact.RelPath)
	if len(metadata.SourceFingerprints) != 1 || metadata.SourceFingerprints[0].SHA256 == "" {
		t.Fatalf("expected source fingerprint in metadata: %#v", metadata)
	}
	sourcePath := filepath.Join(root, "docs", "a.md")
	originalModifiedAt := metadata.SourceFingerprints[0].ModifiedAt
	if err := os.WriteFile(sourcePath, []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.Chtimes(sourcePath, originalModifiedAt, originalModifiedAt); err != nil {
		t.Fatalf("Chtimes returned error: %v", err)
	}

	report, err := store.SourceFreshness(artifact.RelPath)
	if err != nil {
		t.Fatalf("SourceFreshness returned error: %v", err)
	}
	if !report.Stale || report.ChangedCount != 1 || !strings.Contains(report.Sources[0].Message, "fingerprint changed") {
		t.Fatalf("expected fingerprint freshness change, got %#v", report)
	}
}

func writeArtifactSource(t *testing.T, root string, relPath string, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(target, past, past); err != nil {
		t.Fatalf("Chtimes returned error: %v", err)
	}
}
