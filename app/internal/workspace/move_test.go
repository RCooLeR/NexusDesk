package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyFileMoveMovesFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/old.md", "# Old\n")

	proposal, err := ApplyFileMove(root, FileMoveRequest{
		SourceRelPath: "docs/old.md",
		TargetRelPath: "docs/new.md",
	})
	if err != nil {
		t.Fatalf("ApplyFileMove returned error: %v", err)
	}

	if proposal.TargetRelPath != "docs/new.md" {
		t.Fatalf("unexpected target: %s", proposal.TargetRelPath)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "old.md")); !os.IsNotExist(err) {
		t.Fatalf("expected old file to be moved, got err=%v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "new.md"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "# Old\n" {
		t.Fatalf("unexpected moved content: %q", string(content))
	}
}

func TestPreviewFileMoveRejectsOverwrite(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/old.md", "# Old\n")
	writeFile(t, root, "docs/new.md", "# New\n")

	if _, err := PreviewFileMove(root, FileMoveRequest{SourceRelPath: "docs/old.md", TargetRelPath: "docs/new.md"}); err == nil {
		t.Fatal("expected overwrite move to be rejected")
	}
}

func TestPreviewFileMoveRejectsMetadataTarget(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/old.md", "# Old\n")

	if _, err := PreviewFileMove(root, FileMoveRequest{SourceRelPath: "docs/old.md", TargetRelPath: ".nexusdesk/old.md"}); err == nil {
		t.Fatal("expected metadata move to be rejected")
	}
}

func TestPreviewFileMoveRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/old.md", "# Old\n")

	if _, err := PreviewFileMove(root, FileMoveRequest{SourceRelPath: "docs/old.md", TargetRelPath: "../old.md"}); err == nil {
		t.Fatal("expected traversal move to be rejected")
	}
}

func TestPreviewFileMoveRejectsDirectoryLikeTarget(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/old.md", "# Old\n")

	if _, err := PreviewFileMove(root, FileMoveRequest{SourceRelPath: "docs/old.md", TargetRelPath: "docs/new/"}); err == nil {
		t.Fatal("expected directory-like target to be rejected")
	}
}
