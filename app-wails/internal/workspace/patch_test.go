package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyUnifiedPatchUpdatesSingleFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/notes.md", "alpha\nbeta\ngamma\n")

	proposal, err := ApplyUnifiedPatch(root, UnifiedPatchRequest{Patch: `--- a/docs/notes.md
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
	if proposal.FileCount != 1 || proposal.Files[0].RelPath != "docs/notes.md" {
		t.Fatalf("unexpected proposal: %#v", proposal)
	}

	content, err := os.ReadFile(filepath.Join(root, "docs", "notes.md"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "alpha\ninserted\nbeta\ngamma\n" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestApplyUnifiedPatchUpdatesMultipleFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.txt", "one\ntwo\n")
	writeFile(t, root, "b.txt", "red\nblue\n")

	_, err := ApplyUnifiedPatch(root, UnifiedPatchRequest{Patch: `diff --git a/a.txt b/a.txt
--- a/a.txt
+++ b/a.txt
@@ -1,2 +1,2 @@
-one
+ONE
 two
diff --git a/b.txt b/b.txt
--- a/b.txt
+++ b/b.txt
@@ -1,2 +1,2 @@
 red
-blue
+BLUE
`})
	if err != nil {
		t.Fatalf("ApplyUnifiedPatch returned error: %v", err)
	}
	assertFileContent(t, root, "a.txt", "ONE\ntwo\n")
	assertFileContent(t, root, "b.txt", "red\nBLUE\n")
}

func TestApplyUnifiedPatchCreatesNewFile(t *testing.T) {
	root := t.TempDir()

	proposal, err := ApplyUnifiedPatch(root, UnifiedPatchRequest{Patch: `--- /dev/null
+++ b/docs/new.md
@@ -0,0 +1,2 @@
+# New
+content
`})
	if err != nil {
		t.Fatalf("ApplyUnifiedPatch returned error: %v", err)
	}
	if proposal.Files[0].Action != "create" {
		t.Fatalf("expected create action, got %#v", proposal.Files[0])
	}
	assertFileContent(t, root, "docs/new.md", "# New\ncontent\n")
}

func TestPreviewUnifiedPatchRejectsAmbiguousHunk(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "notes.txt", "same\nvalue\nsame\nvalue\n")

	_, err := PreviewUnifiedPatch(root, UnifiedPatchRequest{Patch: `--- a/notes.txt
+++ b/notes.txt
@@ -2,2 +2,2 @@
 same
-value
+changed
`})
	if err == nil || !strings.Contains(err.Error(), "multiple locations") {
		t.Fatalf("expected ambiguous hunk error, got %v", err)
	}
}

func TestPreviewUnifiedPatchRejectsTraversal(t *testing.T) {
	root := t.TempDir()

	_, err := PreviewUnifiedPatch(root, UnifiedPatchRequest{Patch: `--- a/../outside.txt
+++ b/../outside.txt
@@ -1,1 +1,1 @@
-old
+new
`})
	if err == nil {
		t.Fatal("expected traversal patch to be rejected")
	}
}

func TestPreviewUnifiedPatchRejectsBinaryTarget(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "blob.bin"), []byte{0, 1, 2}, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err := PreviewUnifiedPatch(root, UnifiedPatchRequest{Patch: `--- a/blob.bin
+++ b/blob.bin
@@ -1,1 +1,1 @@
-old
+new
`})
	if err == nil || !strings.Contains(err.Error(), "not safe text") {
		t.Fatalf("expected binary target error, got %v", err)
	}
}

func assertFileContent(t *testing.T, root string, relPath string, expected string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("ReadFile %s failed: %v", relPath, err)
	}
	if string(content) != expected {
		t.Fatalf("expected %s content %q, got %q", relPath, expected, string(content))
	}
}
