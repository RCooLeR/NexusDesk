package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewFileWriteBuildsCreateDiff(t *testing.T) {
	root := t.TempDir()

	proposal, err := PreviewFileWrite(root, FileWriteRequest{
		RelPath: "docs/new.md",
		Content: "# New\n",
	})
	if err != nil {
		t.Fatalf("PreviewFileWrite returned error: %v", err)
	}

	if proposal.Action != "create" {
		t.Fatalf("expected create action, got %s", proposal.Action)
	}
	if !strings.Contains(proposal.Diff, "+++ b/docs/new.md") || !strings.Contains(proposal.Diff, "+# New") {
		t.Fatalf("unexpected diff: %s", proposal.Diff)
	}
}

func TestApplyFileWriteUpdatesTextFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/notes.md", "old\n")

	proposal, err := ApplyFileWrite(root, FileWriteRequest{
		RelPath: "docs/notes.md",
		Content: "new\n",
	})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}

	if proposal.Action != "update" {
		t.Fatalf("expected update action, got %s", proposal.Action)
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "notes.md"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "new\n" {
		t.Fatalf("expected updated content, got %q", string(content))
	}
}

func TestPreviewFileWriteRejectsTraversal(t *testing.T) {
	root := t.TempDir()

	if _, err := PreviewFileWrite(root, FileWriteRequest{RelPath: "../outside.txt", Content: "x"}); err == nil {
		t.Fatal("expected traversal write to be rejected")
	}
}

func TestPreviewFileWriteRejectsMetadataWrites(t *testing.T) {
	root := t.TempDir()

	if _, err := PreviewFileWrite(root, FileWriteRequest{RelPath: ".nexusdesk/config.json", Content: "{}"}); err == nil {
		t.Fatal("expected metadata write to be rejected")
	}
}
