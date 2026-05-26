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

func TestApplyFileWritePreservesRequestedUTF16LEEncoding(t *testing.T) {
	root := t.TempDir()

	proposal, err := ApplyFileWrite(root, FileWriteRequest{
		RelPath:  "docs/notes.txt",
		Content:  "hello\n",
		Encoding: "utf-16le",
	})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	if proposal.Encoding != "utf-16le" {
		t.Fatalf("expected utf-16le proposal encoding, got %q", proposal.Encoding)
	}

	content, err := os.ReadFile(filepath.Join(root, "docs", "notes.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(content) < 4 || content[0] != 0xff || content[1] != 0xfe || content[2] != 'h' || content[3] != 0x00 {
		t.Fatalf("expected utf-16le content with BOM, got %#v", content)
	}

	preview, err := Preview(root, "docs/notes.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if preview.Encoding != "utf-16le" || preview.Content != "hello\n" {
		t.Fatalf("expected decoded utf-16le preview, got encoding=%q content=%q", preview.Encoding, preview.Content)
	}
}

func TestApplyFileWriteEncodesWindows1251(t *testing.T) {
	root := t.TempDir()

	proposal, err := ApplyFileWrite(root, FileWriteRequest{
		RelPath:  "docs/cyrillic.txt",
		Content:  "привіт\n",
		Encoding: "windows-1251",
	})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	if proposal.Encoding != "windows-1251" {
		t.Fatalf("expected windows-1251 proposal encoding, got %q", proposal.Encoding)
	}

	content, err := os.ReadFile(filepath.Join(root, "docs", "cyrillic.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) == "привіт\n" {
		t.Fatalf("expected non-utf8 windows-1251 bytes")
	}

	preview, err := Preview(root, "docs/cyrillic.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if preview.Encoding != "windows-1251" || preview.Content != "привіт\n" {
		t.Fatalf("expected decoded windows-1251 preview, got encoding=%q content=%q", preview.Encoding, preview.Content)
	}
}

func TestPreviewFileWriteRejectsUnsupportedEncoding(t *testing.T) {
	root := t.TempDir()

	if _, err := PreviewFileWrite(root, FileWriteRequest{RelPath: "bad.txt", Content: "x", Encoding: "koi8-r"}); err == nil {
		t.Fatal("expected unsupported encoding to be rejected")
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
