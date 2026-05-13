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

func TestPreviewReturnsImageDataURL(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "logo.png")
	if err := os.WriteFile(path, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	preview, err := Preview(root, "logo.png", PreviewOptions{})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Kind != "image" || preview.FileType != "image" {
		t.Fatalf("expected image preview, got kind=%s type=%s", preview.Kind, preview.FileType)
	}
	if !strings.HasPrefix(preview.Content, "data:image/png;base64,") {
		t.Fatalf("expected PNG data URL, got %q", preview.Content)
	}
}

func TestPreviewRejectsOversizedImage(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "large.png")
	if err := os.WriteFile(path, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	preview, err := Preview(root, "large.png", PreviewOptions{MaxBytes: 2})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Kind != "unsupported" {
		t.Fatalf("expected oversized image to be unsupported, got %s", preview.Kind)
	}
	if preview.Content != "" {
		t.Fatalf("expected oversized image content to be empty, got %q", preview.Content)
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

func TestPreviewDecodesUTF16LEText(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	content := []byte{0xff, 0xfe, 'h', 0x00, 'i', 0x00}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	preview, err := Preview(root, "notes.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Content != "hi" {
		t.Fatalf("expected decoded UTF-16 content, got %q", preview.Content)
	}
	if preview.Encoding != "utf-16le" {
		t.Fatalf("expected utf-16le encoding, got %q", preview.Encoding)
	}
}

func TestPreviewDecodesUTF16LETextWithoutBOM(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	content := []byte{'h', 0x00, 'i', 0x00, '\n', 0x00}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	preview, err := Preview(root, "notes.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Content != "hi\n" {
		t.Fatalf("expected decoded UTF-16 content, got %q", preview.Content)
	}
	if preview.Encoding != "utf-16le" {
		t.Fatalf("expected utf-16le encoding, got %q", preview.Encoding)
	}
}

func TestPreviewDecodesWindows1251Text(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	content := []byte{0xcf, 0xf0, 0xe8, 0xe2, 0xe5, 0xf2}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	preview, err := Preview(root, "notes.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Content != "Привет" {
		t.Fatalf("expected decoded Windows-1251 content, got %q", preview.Content)
	}
	if preview.Encoding != "windows-1251" {
		t.Fatalf("expected windows-1251 encoding, got %q", preview.Encoding)
	}
}

func TestPreviewParsesCSVTable(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data/report.csv", "name,value\nalpha,10\nbeta,20\n")

	preview, err := Preview(root, "data/report.csv", PreviewOptions{})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Table == nil {
		t.Fatal("expected CSV table preview")
	}
	if preview.Table.TotalRows != 2 {
		t.Fatalf("expected 2 data rows, got %d", preview.Table.TotalRows)
	}
	if got := strings.Join(preview.Table.Columns, ","); got != "name,value" {
		t.Fatalf("unexpected columns: %s", got)
	}
	if preview.Table.Rows[1][0] != "beta" {
		t.Fatalf("expected second row beta, got %q", preview.Table.Rows[1][0])
	}
	if preview.Content == "" {
		t.Fatal("expected raw CSV content to remain available")
	}
}

func TestPreviewTruncatesCSVTableRows(t *testing.T) {
	root := t.TempDir()
	var builder strings.Builder
	builder.WriteString("name,value\n")
	for index := 0; index < csvPreviewMaxRows+2; index++ {
		builder.WriteString("row,1\n")
	}
	writeFile(t, root, "data/report.csv", builder.String())

	preview, err := Preview(root, "data/report.csv", PreviewOptions{})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Table == nil {
		t.Fatal("expected CSV table preview")
	}
	if len(preview.Table.Rows) != csvPreviewMaxRows {
		t.Fatalf("expected %d rendered rows, got %d", csvPreviewMaxRows, len(preview.Table.Rows))
	}
	if !preview.Table.Truncated {
		t.Fatal("expected CSV table preview truncation")
	}
}

func TestPreviewReturnsPDFDataURL(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "brief.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	preview, err := Preview(root, "brief.pdf", PreviewOptions{})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Kind != "pdf" {
		t.Fatalf("expected PDF preview, got %s", preview.Kind)
	}
	if !strings.HasPrefix(preview.Content, "data:application/pdf;base64,") {
		t.Fatalf("expected PDF data URL, got %q", preview.Content)
	}
}

func TestPreviewRejectsOversizedPDF(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "large.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	preview, err := Preview(root, "large.pdf", PreviewOptions{MaxBytes: 2})
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.Kind != "unsupported" {
		t.Fatalf("expected oversized PDF to be unsupported, got %s", preview.Kind)
	}
	if preview.Content != "" {
		t.Fatalf("expected oversized PDF content to be empty, got %q", preview.Content)
	}
}
