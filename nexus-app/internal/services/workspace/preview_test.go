package workspace

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewFileReadsUTF8Text(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "# Hello\n")

	preview, err := New().PreviewFile(root, "README.md")
	if err != nil {
		t.Fatalf("PreviewFile returned error: %v", err)
	}
	if preview.RelPath != "README.md" || preview.Text != "# Hello\n" || preview.Kind != "text" || preview.Encoding != encodingUTF8 {
		t.Fatalf("unexpected preview: %#v", preview)
	}
}

func TestPreviewFileReadsUTF16LEText(t *testing.T) {
	root := t.TempDir()
	writeBytes(t, filepath.Join(root, "notes.txt"), []byte{0xff, 0xfe, 'H', 0, 'i', 0})

	preview, err := New().PreviewFile(root, "notes.txt")
	if err != nil {
		t.Fatalf("PreviewFile returned error: %v", err)
	}
	if preview.Text != "Hi" || preview.Encoding != encodingUTF16LE {
		t.Fatalf("unexpected UTF-16 preview: %#v", preview)
	}
}

func TestPreviewFileReadsWindows1251Text(t *testing.T) {
	root := t.TempDir()
	writeBytes(t, filepath.Join(root, "notes.txt"), []byte{0xcf, 0xf0, 0xe8, 0xe2, 0xb3, 0xf2})

	preview, err := New().PreviewFile(root, "notes.txt")
	if err != nil {
		t.Fatalf("PreviewFile returned error: %v", err)
	}
	if preview.Text != "Привіт" || preview.Encoding != encodingWindows1251 {
		t.Fatalf("unexpected Windows-1251 preview: %#v", preview)
	}
}

func TestPreviewFileRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := New().PreviewFile(root, "../secrets.txt"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestPreviewFileRejectsOversizedFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "large.txt"), strings.Repeat("x", 8))
	service := &Service{entryLimit: 10, previewByteLimit: 4}
	if _, err := service.PreviewFile(root, "large.txt"); err == nil {
		t.Fatal("expected large file preview to be rejected")
	}
}

func TestPreviewFileMarksBinaryWithoutText(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "blob.bin"), "a\x00b")

	preview, err := New().PreviewFile(root, "blob.bin")
	if err != nil {
		t.Fatalf("PreviewFile returned error: %v", err)
	}
	if preview.Kind != "binary" || preview.Text != "" {
		t.Fatalf("unexpected binary preview: %#v", preview)
	}
}
