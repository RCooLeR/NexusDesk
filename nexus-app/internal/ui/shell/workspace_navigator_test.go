package shell

import "testing"

func TestDefaultCreatePathUsesSelectedDirectory(t *testing.T) {
	if got := defaultCreatePath("docs"); got != "docs/new-file.txt" {
		t.Fatalf("unexpected create path: %q", got)
	}
	if got := defaultCreatePath("docs/readme.md"); got != "docs/new-file.txt" {
		t.Fatalf("unexpected create path near selected file: %q", got)
	}
}

func TestDefaultCopyPathAddsCopySuffix(t *testing.T) {
	if got := defaultCopyPath("docs/readme.md"); got != "docs/readme-copy.md" {
		t.Fatalf("unexpected copy path: %q", got)
	}
	if got := defaultCopyPath("README"); got != "README-copy" {
		t.Fatalf("unexpected root copy path: %q", got)
	}
}
