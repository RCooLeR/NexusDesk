package workspace

import (
	"encoding/base64"
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

func TestPreviewFileWriteUsesLCSForInsertedLines(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/notes.md", "alpha\nbeta\ngamma\n")

	proposal, err := PreviewFileWrite(root, FileWriteRequest{
		RelPath: "docs/notes.md",
		Content: "alpha\ninserted\nbeta\ngamma\n",
	})
	if err != nil {
		t.Fatalf("PreviewFileWrite returned error: %v", err)
	}

	if !strings.Contains(proposal.Diff, " alpha\n+inserted\n beta\n gamma\n") {
		t.Fatalf("expected inserted line with following lines preserved, got %s", proposal.Diff)
	}
	if strings.Contains(proposal.Diff, "-beta\n+inserted") {
		t.Fatalf("expected beta to remain context instead of rewritten, got %s", proposal.Diff)
	}
}

func TestApplyFileAppendDoesNotTruncateLargeFile(t *testing.T) {
	root := t.TempDir()
	large := strings.Repeat("a", 600*1024)
	writeFile(t, root, "docs/large.txt", large)

	proposal, err := ApplyFileAppend(root, FileWriteRequest{
		RelPath: "docs/large.txt",
		Content: "\nappended\n",
	})
	if err != nil {
		t.Fatalf("ApplyFileAppend returned error: %v", err)
	}
	if proposal.Action != "append" {
		t.Fatalf("expected append action, got %s", proposal.Action)
	}

	content, err := os.ReadFile(filepath.Join(root, "docs", "large.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != large+"\nappended\n" {
		t.Fatalf("append changed existing content or failed to append, got %d bytes", len(content))
	}
}

func TestApplyFileWriteCanOverwriteLargeTextFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/large.txt", strings.Repeat("a", 300*1024))

	proposal, err := ApplyFileWrite(root, FileWriteRequest{
		RelPath: "docs/large.txt",
		Content: "replacement\n",
	})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	if proposal.Action != "update" {
		t.Fatalf("expected update action, got %s", proposal.Action)
	}
	if !strings.Contains(proposal.Diff, "larger than inline diff limit") {
		t.Fatalf("expected large-file diff omission marker, got %q", proposal.Diff)
	}

	content, err := os.ReadFile(filepath.Join(root, "docs", "large.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "replacement\n" {
		t.Fatalf("expected replacement content, got %q", string(content))
	}
}

func TestPreviewFileAppendRejectsMetadataWrites(t *testing.T) {
	root := t.TempDir()

	if _, err := PreviewFileAppend(root, FileWriteRequest{RelPath: ".nexusdesk/config.json", Content: "{}"}); err == nil {
		t.Fatal("expected metadata append to be rejected")
	}
}

func TestApplyBinaryFileWriteCreatesBinaryFile(t *testing.T) {
	root := t.TempDir()
	payload := []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd}

	proposal, err := ApplyBinaryFileWrite(root, BinaryFileWriteRequest{
		RelPath:       "assets/blob.bin",
		Base64Content: base64.StdEncoding.EncodeToString(payload),
		ContentType:   "application/octet-stream",
	})
	if err != nil {
		t.Fatalf("ApplyBinaryFileWrite returned error: %v", err)
	}
	if proposal.Action != "create" || proposal.Encoding != "base64" || proposal.SHA256 == "" {
		t.Fatalf("unexpected proposal: %#v", proposal)
	}
	if !strings.Contains(proposal.Diff, "No text diff is available") {
		t.Fatalf("expected binary diff summary, got %q", proposal.Diff)
	}

	content, err := os.ReadFile(filepath.Join(root, "assets", "blob.bin"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != string(payload) {
		t.Fatalf("expected binary payload %#v, got %#v", payload, content)
	}
}

func TestPreviewBinaryFileWriteRejectsInvalidBase64(t *testing.T) {
	root := t.TempDir()

	if _, err := PreviewBinaryFileWrite(root, BinaryFileWriteRequest{RelPath: "assets/blob.bin", Base64Content: "not-base64"}); err == nil {
		t.Fatal("expected invalid base64 to be rejected")
	}
}

func TestPreviewBinaryFileWriteRejectsMetadataWrites(t *testing.T) {
	root := t.TempDir()

	if _, err := PreviewBinaryFileWrite(root, BinaryFileWriteRequest{RelPath: ".nexusdesk/blob.bin", Base64Content: "AA=="}); err == nil {
		t.Fatal("expected metadata binary write to be rejected")
	}
}

func TestPreviewBinaryFileWriteRejectsExecutableTargets(t *testing.T) {
	root := t.TempDir()

	if _, err := PreviewBinaryFileWrite(root, BinaryFileWriteRequest{RelPath: "dist/tool.exe", Base64Content: "AA=="}); err == nil {
		t.Fatal("expected executable binary write to be rejected")
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
