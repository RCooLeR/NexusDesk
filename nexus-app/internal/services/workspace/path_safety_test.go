package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceReadRejectsSymlinkParentEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "secret.txt"), "secret")
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	_, err := New().PreviewFile(root, "linked/secret.txt")
	if err == nil || !strings.Contains(err.Error(), "inside the root") {
		t.Fatalf("expected symlink parent escape rejection, got %v", err)
	}
}

func TestWorkspaceReadRejectsSymlinkFileTarget(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	writeFile(t, outside, "secret")
	if err := os.Symlink(outside, filepath.Join(root, "secret.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	_, err := New().PreviewFile(root, "secret.txt")
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink file target rejection, got %v", err)
	}
}

func TestWorkspaceWriteRejectsSymlinkParentComponent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	_, err := New().PreviewFileWrite(root, FileWriteRequest{RelPath: "linked/escape.txt", Content: "escape"})
	if err == nil || !strings.Contains(err.Error(), "parent cannot be a symlink") {
		t.Fatalf("expected symlink parent write rejection, got %v", err)
	}
}

func TestWorkspacePathRejectsWindowsDangerousForms(t *testing.T) {
	for _, relPath := range []string{
		"notes.txt:Zone.Identifier",
		"CON",
		"CON.txt",
		"docs/NUL.md",
		"COM1",
		"LPT9.log",
		"trailing-dot.",
		"docs/trailing-space /file.txt",
	} {
		if _, err := cleanRel(relPath); err == nil {
			t.Fatalf("expected %q to be rejected", relPath)
		}
	}
}
