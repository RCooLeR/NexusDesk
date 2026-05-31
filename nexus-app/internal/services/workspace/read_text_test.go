package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTextFileReadsFullSafeText(t *testing.T) {
	root := t.TempDir()
	content := strings.Repeat("line with text\n", 500)
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	read, err := New().ReadTextFile(root, "notes.txt")
	if err != nil {
		t.Fatalf("ReadTextFile returned error: %v", err)
	}
	if read.RelPath != "notes.txt" || read.Content != content || read.Encoding != "utf-8" || read.Size != int64(len(content)) {
		t.Fatalf("unexpected text read: %#v", read)
	}
}

func TestReadTextFileSurfacesAmbiguousLatin1Fallback(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte{'c', 'a', 'f', 0xe9}, 0o644); err != nil {
		t.Fatal(err)
	}

	read, err := New().ReadTextFile(root, "notes.txt")
	if err != nil {
		t.Fatalf("ReadTextFile returned error: %v", err)
	}
	if read.Content != "café" || read.Encoding != encodingLatin1 || !read.EncodingAmbiguous || !strings.Contains(read.EncodingWarning, "Low-confidence") {
		t.Fatalf("expected ambiguous Latin-1 fallback, got %#v", read)
	}
}

func TestReadTextFileDetectsWindows1251WithCyrillicSignal(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte{0xcf, 0xf0, 0xe8, 0xe2, 0xb3, 0xf2}, 0o644); err != nil {
		t.Fatal(err)
	}

	read, err := New().ReadTextFile(root, "notes.txt")
	if err != nil {
		t.Fatalf("ReadTextFile returned error: %v", err)
	}
	if read.Content != "Привіт" || read.Encoding != encodingWindows1251 || read.EncodingAmbiguous {
		t.Fatalf("expected confident Windows-1251 detection, got %#v", read)
	}
}

func TestReadTextFileReadsUTF16BEText(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte{0xfe, 0xff, 0, 'H', 0, 'i'}, 0o644); err != nil {
		t.Fatal(err)
	}

	read, err := New().ReadTextFile(root, "notes.txt")
	if err != nil {
		t.Fatalf("ReadTextFile returned error: %v", err)
	}
	if read.Content != "Hi" || read.Encoding != encodingUTF16BE || read.EncodingAmbiguous {
		t.Fatalf("expected UTF-16BE text read, got %#v", read)
	}
}

func TestReadTextFileRejectsUnsafeTargets(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "blob.bin"), []byte{'a', 0x00, 'b'}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New().ReadTextFile(root, "../outside.txt"); err == nil {
		t.Fatal("expected traversal rejection")
	}
	if _, err := New().ReadTextFile(root, "blob.bin"); err == nil {
		t.Fatal("expected binary rejection")
	}
}
