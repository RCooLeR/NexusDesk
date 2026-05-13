package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewReadsTextFileInsideWorkspace(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/readme.md", "hello workspace")

	preview, err := Preview(root, "docs/readme.md", PreviewOptions{})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Content != "hello workspace" {
		t.Fatalf("expected file content, got %q", preview.Content)
	}
	if preview.RelPath != "docs/readme.md" {
		t.Fatalf("expected slash relative path, got %q", preview.RelPath)
	}
	if preview.Kind != "file" || preview.FileType != "code" {
		t.Fatalf("expected code file preview, got kind=%s type=%s", preview.Kind, preview.FileType)
	}
}

func TestPreviewRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	parentFile := filepath.Join(filepath.Dir(root), "outside.txt")
	if err := os.WriteFile(parentFile, []byte("outside"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if _, err := Preview(root, "../outside.txt", PreviewOptions{}); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestPreviewRejectsBinaryContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "image.bin")
	if err := os.WriteFile(path, []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	preview, err := Preview(root, "image.bin", PreviewOptions{})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Kind != "unsupported" {
		t.Fatalf("expected unsupported binary preview, got %s", preview.Kind)
	}
	if preview.Content != "" {
		t.Fatalf("expected no binary content, got %q", preview.Content)
	}
}

func TestPreviewTruncatesLargeText(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "large.txt", strings.Repeat("a", 20))

	preview, err := Preview(root, "large.txt", PreviewOptions{MaxBytes: 8})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if !preview.Truncated {
		t.Fatal("expected truncated preview")
	}
	if len(preview.Content) != 8 {
		t.Fatalf("expected 8 preview bytes, got %d", len(preview.Content))
	}
}

func TestPreviewTrimsPartialUTF8RuneAtLimit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "utf8.txt")
	if err := os.WriteFile(path, []byte("hello \xd0\x96"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	preview, err := Preview(root, "utf8.txt", PreviewOptions{MaxBytes: 7})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Kind == "unsupported" {
		t.Fatal("expected partial UTF-8 truncation to stay previewable")
	}
	if preview.Content != "hello " {
		t.Fatalf("expected valid prefix, got %q", preview.Content)
	}
}
