package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRollbackRestoresUpdatedFile(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "docs/notes.md", "before\n")

	record, err := PrepareRollback(root, "file.write", "docs/notes.md", []string{"docs/notes.md"})
	if err != nil {
		t.Fatalf("PrepareRollback returned error: %v", err)
	}
	if _, err := ApplyFileWrite(root, FileWriteRequest{RelPath: "docs/notes.md", Content: "after\n"}); err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	if _, err := CommitRollback(root, record); err != nil {
		t.Fatalf("CommitRollback returned error: %v", err)
	}

	result, err := ApplyRollback(root, record.ID)
	if err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if len(result.Restored) != 1 || result.Restored[0] != "docs/notes.md" {
		t.Fatalf("unexpected rollback result: %#v", result)
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "notes.md"))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(content) != "before\n" {
		t.Fatalf("expected original content, got %q", string(content))
	}
}

func TestRollbackRemovesCreatedFile(t *testing.T) {
	root := t.TempDir()

	record, err := PrepareRollback(root, "file.write", "new.md", []string{"new.md"})
	if err != nil {
		t.Fatalf("PrepareRollback returned error: %v", err)
	}
	if _, err := ApplyFileWrite(root, FileWriteRequest{RelPath: "new.md", Content: "created\n"}); err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	if _, err := CommitRollback(root, record); err != nil {
		t.Fatalf("CommitRollback returned error: %v", err)
	}

	result, err := ApplyRollback(root, record.ID)
	if err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if len(result.Removed) != 1 || result.Removed[0] != "new.md" {
		t.Fatalf("unexpected rollback result: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "new.md")); !os.IsNotExist(err) {
		t.Fatalf("expected created file to be removed, got %v", err)
	}
}

func TestRollbackRestoresMove(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "docs/source.md", "content\n")

	record, err := PrepareRollback(root, "file.move", "docs/target.md", []string{"docs/source.md", "docs/target.md"})
	if err != nil {
		t.Fatalf("PrepareRollback returned error: %v", err)
	}
	if _, err := ApplyFileMove(root, FileMoveRequest{SourceRelPath: "docs/source.md", TargetRelPath: "docs/target.md"}); err != nil {
		t.Fatalf("ApplyFileMove returned error: %v", err)
	}
	if _, err := CommitRollback(root, record); err != nil {
		t.Fatalf("CommitRollback returned error: %v", err)
	}
	if _, err := ApplyRollback(root, record.ID); err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "target.md")); !os.IsNotExist(err) {
		t.Fatalf("expected target to be removed, got %v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "source.md"))
	if err != nil {
		t.Fatalf("read restored source: %v", err)
	}
	if string(content) != "content\n" {
		t.Fatalf("unexpected restored content: %q", string(content))
	}
}

func TestRollbackRejectsMetadataPath(t *testing.T) {
	root := t.TempDir()
	if _, err := PrepareRollback(root, "file.write", ".nexusdesk/log.json", []string{".nexusdesk/log.json"}); err == nil {
		t.Fatal("expected metadata rollback path to be rejected")
	}
}
