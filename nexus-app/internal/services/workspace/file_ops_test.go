package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyFileCreateCreatesEmptyFileAndRollbackRemovesIt(t *testing.T) {
	root := t.TempDir()
	service := New()

	proposal, err := service.ApplyFileCreate(root, FileCreateRequest{RelPath: "docs/new.md"})
	if err != nil {
		t.Fatalf("ApplyFileCreate returned error: %v", err)
	}
	if proposal.Action != "create" || proposal.RollbackID == "" {
		t.Fatalf("unexpected create proposal: %#v", proposal)
	}
	assertFileContent(t, filepath.Join(root, "docs", "new.md"), "")

	result, err := service.ApplyRollback(root, proposal.RollbackID)
	if err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if len(result.Removed) != 1 || result.Removed[0] != "docs/new.md" {
		t.Fatalf("expected created file rollback removal, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "new.md")); !os.IsNotExist(err) {
		t.Fatalf("expected rollback to remove created file, got err=%v", err)
	}
}

func TestApplyFileDeleteRemovesFileAndRollbackRestoresIt(t *testing.T) {
	root := t.TempDir()
	service := New()
	writeFile(t, filepath.Join(root, "docs", "old.md"), "# Old\n")

	proposal, err := service.ApplyFileDelete(root, "docs/old.md")
	if err != nil {
		t.Fatalf("ApplyFileDelete returned error: %v", err)
	}
	if proposal.Action != "delete" || proposal.RollbackID == "" {
		t.Fatalf("unexpected delete proposal: %#v", proposal)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "old.md")); !os.IsNotExist(err) {
		t.Fatalf("expected file to be deleted, got err=%v", err)
	}

	result, err := service.ApplyRollback(root, proposal.RollbackID)
	if err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if len(result.Restored) != 1 || result.Restored[0] != "docs/old.md" {
		t.Fatalf("expected deleted file rollback restore, got %#v", result)
	}
	assertFileContent(t, filepath.Join(root, "docs", "old.md"), "# Old\n")
}

func TestApplyFileCopyCopiesFileAndRollbackRemovesTarget(t *testing.T) {
	root := t.TempDir()
	service := New()
	writeFile(t, filepath.Join(root, "docs", "source.md"), "# Source\n")

	proposal, err := service.ApplyFileCopy(root, FileCopyRequest{
		SourceRelPath: "docs/source.md",
		TargetRelPath: "docs/copy.md",
	})
	if err != nil {
		t.Fatalf("ApplyFileCopy returned error: %v", err)
	}
	if proposal.Action != "copy" || proposal.RollbackID == "" {
		t.Fatalf("unexpected copy proposal: %#v", proposal)
	}
	assertFileContent(t, filepath.Join(root, "docs", "copy.md"), "# Source\n")

	result, err := service.ApplyRollback(root, proposal.RollbackID)
	if err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if len(result.Removed) != 1 || result.Removed[0] != "docs/copy.md" {
		t.Fatalf("expected copied target rollback removal, got %#v", result)
	}
	assertFileContent(t, filepath.Join(root, "docs", "source.md"), "# Source\n")
}

func TestApplyFileMoveMovesFileAndRollbackRestoresSource(t *testing.T) {
	root := t.TempDir()
	service := New()
	writeFile(t, filepath.Join(root, "docs", "source.md"), "# Source\n")

	proposal, err := service.ApplyFileMove(root, FileMoveRequest{
		SourceRelPath: "docs/source.md",
		TargetRelPath: "archive/source.md",
	})
	if err != nil {
		t.Fatalf("ApplyFileMove returned error: %v", err)
	}
	if proposal.Action != "move" || proposal.RollbackID == "" {
		t.Fatalf("unexpected move proposal: %#v", proposal)
	}
	assertFileContent(t, filepath.Join(root, "archive", "source.md"), "# Source\n")
	if _, err := os.Stat(filepath.Join(root, "docs", "source.md")); !os.IsNotExist(err) {
		t.Fatalf("expected source to move, got err=%v", err)
	}

	result, err := service.ApplyRollback(root, proposal.RollbackID)
	if err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if len(result.Restored) != 1 || result.Restored[0] != "docs/source.md" || len(result.Removed) != 1 || result.Removed[0] != "archive/source.md" {
		t.Fatalf("expected move rollback restore/remove, got %#v", result)
	}
	assertFileContent(t, filepath.Join(root, "docs", "source.md"), "# Source\n")
}

func TestApplyFileRenameUsesRenameAction(t *testing.T) {
	root := t.TempDir()
	service := New()
	writeFile(t, filepath.Join(root, "docs", "old.md"), "# Old\n")

	proposal, err := service.ApplyFileRename(root, FileMoveRequest{
		SourceRelPath: "docs/old.md",
		TargetRelPath: "docs/new.md",
	})
	if err != nil {
		t.Fatalf("ApplyFileRename returned error: %v", err)
	}
	if proposal.Action != "rename" || !strings.Contains(proposal.Message, "Renamed docs/old.md to docs/new.md") {
		t.Fatalf("unexpected rename proposal: %#v", proposal)
	}
	assertFileContent(t, filepath.Join(root, "docs", "new.md"), "# Old\n")
}

func TestFileOperationsRejectUnsafeTargets(t *testing.T) {
	root := t.TempDir()
	service := New()
	writeFile(t, filepath.Join(root, "docs", "source.md"), "# Source\n")
	writeFile(t, filepath.Join(root, "docs", "existing.md"), "# Existing\n")

	if _, err := service.PreviewFileCreate(root, FileCreateRequest{RelPath: ".nexusdesk/config.json"}); err == nil {
		t.Fatal("expected metadata create to be rejected")
	}
	if _, err := service.PreviewFileCopy(root, FileCopyRequest{SourceRelPath: "docs/source.md", TargetRelPath: "../copy.md"}); err == nil {
		t.Fatal("expected traversal copy target to be rejected")
	}
	if _, err := service.PreviewFileCopy(root, FileCopyRequest{SourceRelPath: "docs/source.md", TargetRelPath: "docs/existing.md"}); err == nil {
		t.Fatal("expected existing copy target to be rejected")
	}
	if _, err := service.PreviewFileMove(root, FileMoveRequest{SourceRelPath: "docs/source.md", TargetRelPath: "docs/"}); err == nil {
		t.Fatal("expected folder-like move target to be rejected")
	}
	if _, err := service.PreviewFileDelete(root, ".nexusdesk/config.json"); err == nil {
		t.Fatal("expected metadata delete to be rejected")
	}
}
