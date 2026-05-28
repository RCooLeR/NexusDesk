package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQuickOpenFilesRanksBasenameMatches(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceTestFile(t, root, "src/app/main.go", "package main\n")
	writeWorkspaceTestFile(t, root, "docs/main-notes.md", "# Main\n")
	writeWorkspaceTestFile(t, root, "src/other.go", "package main\n")

	files, err := New().QuickOpenFiles(root, "main.go", 10)
	if err != nil {
		t.Fatalf("QuickOpenFiles returned error: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("expected quick-open matches, got %#v", files)
	}
	if files[0].RelPath != "src/app/main.go" {
		t.Fatalf("expected exact basename match first, got %#v", files)
	}
}

func TestQuickOpenFilesSkipsIgnoredAndSymlinkedEntries(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceTestFile(t, root, "README.md", "# Project\n")
	writeWorkspaceTestFile(t, root, "node_modules/pkg/index.js", "ignored\n")
	writeWorkspaceTestFile(t, root, ".nexusdesk/log.json", "{}")
	if err := os.Symlink(filepath.Join(root, "README.md"), filepath.Join(root, "linked.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	files, err := New().QuickOpenFiles(root, "", 10)
	if err != nil {
		t.Fatalf("QuickOpenFiles returned error: %v", err)
	}
	for _, file := range files {
		switch file.RelPath {
		case "node_modules/pkg/index.js", ".nexusdesk/log.json", "linked.md":
			t.Fatalf("quick-open included unsafe/ignored path: %#v", files)
		}
	}
}

func TestQuickOpenFilesHonorsLimit(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceTestFile(t, root, "a.txt", "a")
	writeWorkspaceTestFile(t, root, "b.txt", "b")

	files, err := New().QuickOpenFiles(root, "", 1)
	if err != nil {
		t.Fatalf("QuickOpenFiles returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected limit to be honored, got %#v", files)
	}
}
