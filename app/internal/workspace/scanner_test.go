package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanReturnsSafeWorkspaceSnapshot(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "README.md", "hello")
	writeFile(t, root, "src/main.go", "package main")
	writeFile(t, root, "data/report.csv", "name,value")
	writeFile(t, root, "node_modules/pkg/index.js", "ignored")
	writeFile(t, root, ".git/config", "ignored")

	snapshot, err := Scan(root, ScanOptions{MaxDepth: 4, MaxEntries: 20})
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if snapshot.Root == "" {
		t.Fatal("expected absolute root")
	}

	if snapshot.Truncated {
		t.Fatal("did not expect truncated result")
	}

	assertContains(t, snapshot.Nodes, "README.md", "code")
	assertContains(t, snapshot.Nodes, "src/main.go", "code")
	assertContains(t, snapshot.Nodes, "data/report.csv", "data")
	assertNotContains(t, snapshot.Nodes, "node_modules/pkg/index.js")
	assertNotContains(t, snapshot.Nodes, ".git/config")
}

func TestScanReturnsFilesystemTreeOrder(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "zeta.txt", "z")
	writeFile(t, root, "app/main.go", "package main")
	writeFile(t, root, "app/internal/run.go", "package internal")
	writeFile(t, root, "app/README.md", "app")
	writeFile(t, root, "docs/guide.md", "docs")

	snapshot, err := Scan(root, ScanOptions{MaxDepth: 4, MaxEntries: 20})
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	got := make([]string, 0, len(snapshot.Nodes))
	for _, node := range snapshot.Nodes {
		got = append(got, node.RelPath)
	}

	want := []string{
		"app",
		"app/internal",
		"app/internal/run.go",
		"app/main.go",
		"app/README.md",
		"docs",
		"docs/guide.md",
		"zeta.txt",
	}

	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected tree order:\n%s", strings.Join(got, "\n"))
	}
}

func TestScanHonorsDepthAndEntryLimit(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "a/b/c/d/e.txt", "deep")
	writeFile(t, root, "first.txt", "first")
	writeFile(t, root, "second.txt", "second")
	writeFile(t, root, "third.txt", "third")

	snapshot, err := Scan(root, ScanOptions{MaxDepth: 2, MaxEntries: 2})
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if !snapshot.Truncated {
		t.Fatal("expected truncated result")
	}

	assertNotContains(t, snapshot.Nodes, "a/b/c/d/e.txt")

	if len(snapshot.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(snapshot.Nodes))
	}
}

func TestScanDefaultDepthIncludesDeepWorkspaceFiles(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "one/two/three/four/five/six/seven/eight/nine/ten.txt", "deep")

	snapshot, err := Scan(root, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	assertContains(t, snapshot.Nodes, "one/two/three/four/five/six/seven/eight/nine/ten.txt", "document")
}

func writeFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func assertContains(t *testing.T, nodes []FileNode, relPath string, fileType string) {
	t.Helper()

	for _, node := range nodes {
		if node.RelPath == relPath {
			if node.FileType != fileType {
				t.Fatalf("expected %s to have file type %s, got %s", relPath, fileType, node.FileType)
			}
			return
		}
	}

	t.Fatalf("expected nodes to contain %s", relPath)
}

func assertNotContains(t *testing.T, nodes []FileNode, relPath string) {
	t.Helper()

	for _, node := range nodes {
		if node.RelPath == relPath {
			t.Fatalf("expected nodes not to contain %s", relPath)
		}
	}
}
