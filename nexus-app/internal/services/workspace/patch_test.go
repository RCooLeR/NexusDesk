package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyUnifiedPatchUpdatesSingleFileAndRollsBack(t *testing.T) {
	root := t.TempDir()
	service := New()
	writeFile(t, filepath.Join(root, "docs", "notes.md"), "alpha\nbeta\ngamma\n")

	proposal, err := service.ApplyUnifiedPatch(root, UnifiedPatchRequest{Patch: `--- a/docs/notes.md
+++ b/docs/notes.md
@@ -1,3 +1,4 @@
 alpha
+inserted
 beta
 gamma
`})
	if err != nil {
		t.Fatalf("ApplyUnifiedPatch returned error: %v", err)
	}
	if proposal.FileCount != 1 || proposal.RollbackID == "" || proposal.Files[0].RelPath != "docs/notes.md" {
		t.Fatalf("unexpected proposal: %#v", proposal)
	}
	assertFileContent(t, filepath.Join(root, "docs", "notes.md"), "alpha\ninserted\nbeta\ngamma\n")

	result, err := service.ApplyRollback(root, proposal.RollbackID)
	if err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if len(result.Restored) != 1 || result.Restored[0] != "docs/notes.md" {
		t.Fatalf("unexpected rollback result: %#v", result)
	}
	assertFileContent(t, filepath.Join(root, "docs", "notes.md"), "alpha\nbeta\ngamma\n")
}

func TestApplyUnifiedPatchCreatesNewFileAndRollsBack(t *testing.T) {
	root := t.TempDir()
	service := New()

	proposal, err := service.ApplyUnifiedPatch(root, UnifiedPatchRequest{Patch: `--- /dev/null
+++ b/docs/new.md
@@ -0,0 +1,2 @@
+# New
+content
`})
	if err != nil {
		t.Fatalf("ApplyUnifiedPatch returned error: %v", err)
	}
	if proposal.Files[0].Action != "create" || proposal.RollbackID == "" {
		t.Fatalf("unexpected proposal: %#v", proposal)
	}
	assertFileContent(t, filepath.Join(root, "docs", "new.md"), "# New\ncontent\n")

	result, err := service.ApplyRollback(root, proposal.RollbackID)
	if err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if len(result.Removed) != 1 || result.Removed[0] != "docs/new.md" {
		t.Fatalf("unexpected rollback result: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "new.md")); !os.IsNotExist(err) {
		t.Fatalf("expected created file to be removed, got err=%v", err)
	}
}

func TestPreviewUnifiedPatchRejectsUnsafeTargets(t *testing.T) {
	root := t.TempDir()
	service := New()

	_, traversalErr := service.PreviewUnifiedPatch(root, UnifiedPatchRequest{Patch: `--- a/../outside.txt
+++ b/../outside.txt
@@ -1,1 +1,1 @@
-old
+new
`})
	if traversalErr == nil {
		t.Fatal("expected traversal patch to be rejected")
	}

	_, deleteErr := service.PreviewUnifiedPatch(root, UnifiedPatchRequest{Patch: `--- a/notes.txt
+++ /dev/null
@@ -1,1 +0,0 @@
-old
`})
	if deleteErr == nil || !strings.Contains(deleteErr.Error(), "deletes are not supported") {
		t.Fatalf("expected delete patch to be rejected, got %v", deleteErr)
	}
}

func TestPreviewUnifiedPatchRejectsAmbiguousAndBinaryTargets(t *testing.T) {
	root := t.TempDir()
	service := New()
	writeFile(t, filepath.Join(root, "notes.txt"), "same\nvalue\nsame\nvalue\n")
	writeBytes(t, filepath.Join(root, "blob.bin"), []byte{0, 1, 2})

	_, ambiguousErr := service.PreviewUnifiedPatch(root, UnifiedPatchRequest{Patch: `--- a/notes.txt
+++ b/notes.txt
@@ -2,2 +2,2 @@
 same
-value
+changed
`})
	if ambiguousErr == nil || !strings.Contains(ambiguousErr.Error(), "multiple locations") {
		t.Fatalf("expected ambiguous hunk error, got %v", ambiguousErr)
	}

	_, binaryErr := service.PreviewUnifiedPatch(root, UnifiedPatchRequest{Patch: `--- a/blob.bin
+++ b/blob.bin
@@ -1,1 +1,1 @@
-old
+new
`})
	if binaryErr == nil || !strings.Contains(binaryErr.Error(), "not safe text") {
		t.Fatalf("expected binary target error, got %v", binaryErr)
	}
}
