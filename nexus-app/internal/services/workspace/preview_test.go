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

func TestPreviewFileReturnsImageBytes(t *testing.T) {
	root := t.TempDir()
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
	writeBytes(t, filepath.Join(root, "pixel.png"), png)

	preview, err := New().PreviewFile(root, "pixel.png")
	if err != nil {
		t.Fatalf("PreviewFile returned error: %v", err)
	}
	if preview.Kind != "image" || preview.MediaType != "image/png" || len(preview.Bytes) != len(png) {
		t.Fatalf("unexpected image preview: %#v", preview)
	}
}
