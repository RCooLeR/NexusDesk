package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTextFileReadsFullSafeText(t *testing.T) {
	root := t.TempDir()
	content := strings.Repeat("line with text\n", 500)
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	read, err := New().ReadTextFile(root, "notes.txt")
	if err != nil {
		t.Fatalf("ReadTextFile returned error: %v", err)
	}
	if read.RelPath != "notes.txt" || read.Content != content || read.Encoding != "utf-8" || read.Size != int64(len(content)) {
		t.Fatalf("unexpected text read: %#v", read)
	}
}

func TestReadTextFileRejectsUnsafeTargets(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "blob.bin"), []byte{'a', 0x00, 'b'}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New().ReadTextFile(root, "../outside.txt"); err == nil {
		t.Fatal("expected traversal rejection")
	}
	if _, err := New().ReadTextFile(root, "blob.bin"); err == nil {
		t.Fatal("expected binary rejection")
	}
}
