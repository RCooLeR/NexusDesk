package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyFileCopyCopiesFile(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "docs/source.md", "hello")

	proposal, err := ApplyFileCopy(root, FileCopyRequest{
		SourceRelPath: "docs/source.md",
		TargetRelPath: "docs/copy.md",
	})
	if err != nil {
		t.Fatalf("ApplyFileCopy returned error: %v", err)
	}
	if proposal.Action != "copy" || proposal.SourceRelPath != "docs/source.md" || proposal.TargetRelPath != "docs/copy.md" {
		t.Fatalf("unexpected proposal: %#v", proposal)
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "copy.md"))
	if err != nil {
		t.Fatalf("copy was not created: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected copy content: %q", string(content))
	}
}

func TestPreviewFileCopyRejectsOverwrite(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "docs/source.md", "hello")
	writeTestFile(t, root, "docs/copy.md", "existing")

	if _, err := PreviewFileCopy(root, FileCopyRequest{SourceRelPath: "docs/source.md", TargetRelPath: "docs/copy.md"}); err == nil {
		t.Fatal("expected overwrite rejection")
	}
}

func TestPreviewFileCopyRejectsMetadata(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "docs/source.md", "hello")

	if _, err := PreviewFileCopy(root, FileCopyRequest{SourceRelPath: "docs/source.md", TargetRelPath: ".nexusdesk/copy.md"}); err == nil {
		t.Fatal("expected metadata target rejection")
	}
}

func TestPreviewFileCopyRejectsLargeSource(t *testing.T) {
	root := t.TempDir()
	large := strings.Repeat("x", fileCopyMaxBytes+1)
	writeTestFile(t, root, "large.txt", large)

	if _, err := PreviewFileCopy(root, FileCopyRequest{SourceRelPath: "large.txt", TargetRelPath: "large-copy.txt"}); err == nil {
		t.Fatal("expected large source rejection")
	}
}
