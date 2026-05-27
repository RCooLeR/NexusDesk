package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCollectContextFilesExpandsDirectory(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceTestFile(t, root, "src/main.go", "package main\n")
	writeWorkspaceTestFile(t, root, "src/nested/readme.md", "# Nested\n")
	writeWorkspaceTestFile(t, root, "src/logo.png", "not really png")
	writeWorkspaceTestFile(t, root, "node_modules/pkg/index.js", "ignored")

	collection, err := New().CollectContextFiles(root, []string{"src"}, ContextCollectOptions{MaxFiles: 10})
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
	writeWorkspaceTestFile(t, root, "README.md", "# Project\n")
	writeWorkspaceTestFile(t, root, "app/main.go", "package main\n")

	collection, err := New().CollectContextFiles(root, []string{"."}, ContextCollectOptions{MaxFiles: 10})
	if err != nil {
		t.Fatalf("CollectContextFiles returned error: %v", err)
	}

	assertContextFile(t, collection, "README.md")
	assertContextFile(t, collection, "app/main.go")
	if len(collection.Roots) != 1 || collection.Roots[0] != "." {
		t.Fatalf("unexpected roots: %#v", collection.Roots)
	}
}

func TestBuildContextPackAllowsExplicitArtifactContext(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceTestFile(t, root, ".nexusdesk/artifacts/task-runs/report.md", "# Task Report\n\nDone.\n")

	pack, err := New().BuildContextPack(root, []string{".nexusdesk/artifacts/task-runs/report.md"}, ContextPackOptions{MaxBytes: 4096})
	if err != nil {
		t.Fatalf("BuildContextPack returned error: %v", err)
	}

	if len(pack.SourcePaths) != 1 || pack.SourcePaths[0] != ".nexusdesk/artifacts/task-runs/report.md" {
		t.Fatalf("unexpected artifact source paths: %#v", pack.SourcePaths)
	}
	if !strings.Contains(pack.Content, "Workspace context: .nexusdesk/artifacts/task-runs/report.md") || !strings.Contains(pack.Content, "Done.") {
		t.Fatalf("missing artifact context content: %q", pack.Content)
	}
}

func TestPreviewContextPackSummarizesExpandedFiles(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceTestFile(t, root, "docs/a.md", "a")
	writeWorkspaceTestFile(t, root, "docs/b.md", "b")

	preview, err := New().PreviewContextPack(root, []string{"docs"}, ContextCollectOptions{MaxFiles: 10})
	if err != nil {
		t.Fatalf("PreviewContextPack returned error: %v", err)
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

func TestBuildContextPackIncludesManifestAndSections(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceTestFile(t, root, "src/main.go", "package main\n")
	writeWorkspaceTestFile(t, root, "src/readme.md", "# Source\n")

	pack, err := New().BuildContextPack(root, []string{"src"}, ContextPackOptions{
		ContextCollectOptions: ContextCollectOptions{MaxFiles: 10},
		MaxBytes:              4096,
	})
	if err != nil {
		t.Fatalf("BuildContextPack returned error: %v", err)
	}

	if pack.Label != "context: src" {
		t.Fatalf("unexpected label %q", pack.Label)
	}
	if !strings.Contains(pack.Content, "Requested roots: src") {
		t.Fatalf("missing manifest: %q", pack.Content)
	}
	if !strings.Contains(pack.Content, "Workspace context: src/main.go") || !strings.Contains(pack.Content, "Workspace context: src/readme.md") {
		t.Fatalf("missing file sections: %q", pack.Content)
	}
	if len(pack.SourcePaths) != 2 {
		t.Fatalf("unexpected source paths: %#v", pack.SourcePaths)
	}
}

func TestBuildContextPackCapsBytesAndKeepsUTF8Valid(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceTestFile(t, root, "large.md", strings.Repeat("π", 200))

	pack, err := New().BuildContextPack(root, []string{"large.md"}, ContextPackOptions{MaxBytes: 220})
	if err != nil {
		t.Fatalf("BuildContextPack returned error: %v", err)
	}

	if !pack.Truncated {
		t.Fatal("expected truncation")
	}
	if !utf8.ValidString(pack.Content) {
		t.Fatalf("expected valid UTF-8, got %q", pack.Content)
	}
	if !strings.Contains(pack.Content, "[context pack truncated]") {
		t.Fatalf("missing truncation marker: %q", pack.Content)
	}
}

func TestCollectContextFilesRejectsTraversal(t *testing.T) {
	if _, err := New().CollectContextFiles(t.TempDir(), []string{"../outside.md"}, ContextCollectOptions{}); err == nil {
		t.Fatal("expected traversal error")
	}
}

func writeWorkspaceTestFile(t *testing.T, root string, relPath string, content string) {
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
