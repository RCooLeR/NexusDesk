package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenListsOnlyTopLevel(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "src", "deep"))
	writeFile(t, filepath.Join(root, "src", "deep", "main.go"), "package main\n")
	writeFile(t, filepath.Join(root, "README.md"), "# project\n")

	service := New()
	workspace, err := service.Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if workspace.Root == "" || workspace.Name == "" {
		t.Fatalf("workspace identity was not populated: %#v", workspace)
	}
	if len(workspace.Tree) != 2 {
		t.Fatalf("expected top-level nodes only, got %#v", workspace.Tree)
	}
	for _, node := range workspace.Tree {
		if len(node.Children) != 0 {
			t.Fatalf("expected lazy tree nodes without eager children: %#v", node)
		}
	}
}

func TestListChildrenSortsDirectoriesFirstAndSkipsIgnored(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "b-dir"))
	mkdir(t, filepath.Join(root, "a-dir"))
	mkdir(t, filepath.Join(root, ".git"))
	writeFile(t, filepath.Join(root, "z.txt"), "z")
	writeFile(t, filepath.Join(root, "a.txt"), "a")

	result, err := New().ListChildren(root, "")
	if err != nil {
		t.Fatalf("ListChildren returned error: %v", err)
	}
	got := names(result)
	want := []string{"a-dir", "b-dir", "a.txt", "z.txt"}
	if !sameStrings(got, want) {
		t.Fatalf("unexpected order: got %#v want %#v", got, want)
	}
	if result.Summary.Ignored != 1 {
		t.Fatalf("expected ignored count for .git, got %#v", result.Summary)
	}
}

func TestListChildrenCanIncludeIgnoredExceptMetadata(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, ".git"))
	mkdir(t, filepath.Join(root, ".nexusdesk"))
	mkdir(t, filepath.Join(root, "node_modules"))
	writeFile(t, filepath.Join(root, "README.md"), "# project\n")

	result, err := New().ListChildrenWithOptions(root, "", ListOptions{IncludeIgnored: true})
	if err != nil {
		t.Fatalf("ListChildrenWithOptions returned error: %v", err)
	}
	got := names(result)
	want := []string{".git", "node_modules", "README.md"}
	if !sameStrings(got, want) {
		t.Fatalf("unexpected visible nodes: got %#v want %#v", got, want)
	}
	if result.Summary.Ignored != 3 {
		t.Fatalf("expected ignored count to include metadata, got %#v", result.Summary)
	}
	for _, node := range result.Nodes {
		if node.Name == ".git" || node.Name == "node_modules" {
			if !node.Ignored {
				t.Fatalf("expected %s to be marked ignored", node.Name)
			}
		}
		if node.Name == ".nexusdesk" {
			t.Fatal("metadata directory should stay hidden even when ignored paths are shown")
		}
	}
}

func TestListChildrenRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := New().ListChildren(root, "../outside"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestListChildrenAppliesEntryLimit(t *testing.T) {
	root := t.TempDir()
	for index := 0; index < 5; index++ {
		writeFile(t, filepath.Join(root, string(rune('a'+index))+".txt"), "x")
	}
	service := &Service{entryLimit: 3}
	result, err := service.ListChildren(root, "")
	if err != nil {
		t.Fatalf("ListChildren returned error: %v", err)
	}
	if len(result.Nodes) != 3 {
		t.Fatalf("expected capped nodes, got %#v", result.Nodes)
	}
	if result.Summary.EntryCap != 1 {
		t.Fatalf("expected entry cap marker, got %#v", result.Summary)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	writeBytes(t, path, []byte(content))
}

func writeBytes(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func names(result ListResult) []string {
	values := []string{}
	for _, node := range result.Nodes {
		values = append(values, node.Name)
	}
	return values
}

func sameStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
