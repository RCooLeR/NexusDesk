package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectContextFilesExpandsDirectory(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "src/main.go", "package main\n")
	writeTestFile(t, root, "src/nested/readme.md", "# Nested\n")
	writeTestFile(t, root, "src/logo.png", "not really png")
	writeTestFile(t, root, "node_modules/pkg/index.js", "ignored")

	collection, err := CollectContextFiles(root, []string{"src"}, ContextCollectOptions{MaxFiles: 10})
	if err != nil {
		t.Fatalf("CollectContextFiles returned error: %v", err)
	}

	assertContextFile(t, collection, "src/main.go")
	assertContextFile(t, collection, "src/nested/readme.md")
	assertNoContextFile(t, collection, "src/logo.png")
	assertNoContextFile(t, collection, "node_modules/pkg/index.js")
	if collection.Truncated {
		t.Fatal("did not expect truncation")
	}
}

func TestCollectContextFilesSupportsWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "README.md", "# Project\n")
	writeTestFile(t, root, "app/main.go", "package main\n")

	collection, err := CollectContextFiles(root, []string{"."}, ContextCollectOptions{MaxFiles: 10})
	if err != nil {
		t.Fatalf("CollectContextFiles returned error: %v", err)
	}

	assertContextFile(t, collection, "README.md")
	assertContextFile(t, collection, "app/main.go")
	if len(collection.Roots) != 1 || collection.Roots[0] != "." {
		t.Fatalf("unexpected roots: %#v", collection.Roots)
	}
}

func TestCollectContextFilesCapsExpandedDirectories(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "a.md", "a")
	writeTestFile(t, root, "b.md", "b")
	writeTestFile(t, root, "c.md", "c")

	collection, err := CollectContextFiles(root, []string{"."}, ContextCollectOptions{MaxFiles: 2})
	if err != nil {
		t.Fatalf("CollectContextFiles returned error: %v", err)
	}

	if len(collection.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(collection.Files))
	}
	if !collection.Truncated {
		t.Fatal("expected truncation")
	}
}

func TestPreviewContextFilesSummarizesExpandedFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "docs/a.md", "a")
	writeTestFile(t, root, "docs/b.md", "b")

	preview, err := PreviewContextFiles(root, []string{"docs"}, ContextCollectOptions{MaxFiles: 10})
	if err != nil {
		t.Fatalf("PreviewContextFiles returned error: %v", err)
	}

	if preview.FileCount != 2 {
		t.Fatalf("expected 2 preview files, got %d", preview.FileCount)
	}
	if preview.Message != "Context pack will include 2 files." {
		t.Fatalf("unexpected preview message: %s", preview.Message)
	}
	if len(preview.Files) != 2 || preview.Files[0].RelPath != "docs/a.md" || preview.Files[1].RelPath != "docs/b.md" {
		t.Fatalf("unexpected preview files: %#v", preview.Files)
	}
}

func TestPreviewContextFilesReportsTruncation(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "a.md", "a")
	writeTestFile(t, root, "b.md", "b")

	preview, err := PreviewContextFiles(root, []string{"."}, ContextCollectOptions{MaxFiles: 1})
	if err != nil {
		t.Fatalf("PreviewContextFiles returned error: %v", err)
	}

	if !preview.Truncated {
		t.Fatal("expected preview truncation")
	}
	if preview.Message != "Context pack will include 1 file. Some matching files were skipped by safety or size limits." {
		t.Fatalf("unexpected preview message: %s", preview.Message)
	}
}

func writeTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func assertContextFile(t *testing.T, collection ContextCollection, relPath string) {
	t.Helper()

	for _, file := range collection.Files {
		if file.RelPath == relPath {
			return
		}
	}
	t.Fatalf("expected context file %s in %#v", relPath, collection.Files)
}

func assertNoContextFile(t *testing.T, collection ContextCollection, relPath string) {
	t.Helper()

	for _, file := range collection.Files {
		if file.RelPath == relPath {
			t.Fatalf("did not expect context file %s in %#v", relPath, collection.Files)
		}
	}
}
