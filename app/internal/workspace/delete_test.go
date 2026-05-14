package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewFileDeleteBuildsDiff(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/old.md", "# Old\n")

	proposal, err := PreviewFileDelete(root, "docs/old.md")
	if err != nil {
		t.Fatalf("PreviewFileDelete returned error: %v", err)
	}

	if proposal.Action != "delete" {
		t.Fatalf("expected delete action, got %s", proposal.Action)
	}
	if !strings.Contains(proposal.Diff, "--- a/docs/old.md") || !strings.Contains(proposal.Diff, "-# Old") {
		t.Fatalf("unexpected delete diff: %s", proposal.Diff)
	}
}

func TestApplyFileDeleteRemovesFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/old.md", "# Old\n")

	proposal, err := ApplyFileDelete(root, "docs/old.md")
	if err != nil {
		t.Fatalf("ApplyFileDelete returned error: %v", err)
	}

	if proposal.Action != "delete" {
		t.Fatalf("expected delete action, got %s", proposal.Action)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "old.md")); !os.IsNotExist(err) {
		t.Fatalf("expected file to be deleted, got err=%v", err)
	}
}

func TestPreviewFileDeleteRejectsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	if _, err := PreviewFileDelete(root, "docs"); err == nil {
		t.Fatal("expected directory delete to be rejected")
	}
}

func TestPreviewFileDeleteRejectsMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".nexusdesk/config.json", "{}")

	if _, err := PreviewFileDelete(root, ".nexusdesk/config.json"); err == nil {
		t.Fatal("expected metadata delete to be rejected")
	}
}
