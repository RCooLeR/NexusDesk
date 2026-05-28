package shell

import "testing"

func TestEditorFindNextOffset(t *testing.T) {
	text := "alpha\nBeta\nbeta"
	if got := editorFindNextOffset(text, "Beta", 0, true); got != 6 {
		t.Fatalf("expected case-sensitive match at 6, got %d", got)
	}
	if got := editorFindNextOffset(text, "beta", 0, true); got != 11 {
		t.Fatalf("expected lower-case match at 11, got %d", got)
	}
	if got := editorFindNextOffset(text, "beta", 0, false); got != 6 {
		t.Fatalf("expected first case-insensitive match at 6, got %d", got)
	}
}

func TestEditorReplaceNext(t *testing.T) {
	next, offset, replaced := editorReplaceNext("one two two", "two", "THREE", 0, true)
	if !replaced {
		t.Fatal("expected replacement")
	}
	if offset != 4 {
		t.Fatalf("expected offset 4, got %d", offset)
	}
	if next != "one THREE two" {
		t.Fatalf("unexpected replacement result: %q", next)
	}
}

func TestEditorReplaceAllCaseInsensitive(t *testing.T) {
	next, count := editorReplaceAll("Beta beta BETA", "beta", "x", false)
	if count != 3 {
		t.Fatalf("expected 3 replacements, got %d", count)
	}
	if next != "x x x" {
		t.Fatalf("unexpected replace all result: %q", next)
	}
}

func TestEditorCursorOffsetRoundTrip(t *testing.T) {
	text := "alpha\nzeta\nomega"
	offset := editorCursorToOffset(text, 1, 2)
	if offset != 8 {
		t.Fatalf("expected offset 8, got %d", offset)
	}
	row, column := editorOffsetToCursor(text, offset)
	if row != 1 || column != 2 {
		t.Fatalf("expected row 1 column 2, got row %d column %d", row, column)
	}
}

func TestEditorWriteEncodingNormalizesWailsOptions(t *testing.T) {
	tests := map[string]string{
		"":             "utf-8",
		"UTF8":         "utf-8",
		"utf-8 bom":    "utf-8-bom",
		"utf16le":      "utf-16le",
		"utf-16 be":    "utf-16be",
		"cp1251":       "windows-1251",
		"windows1252":  "windows-1252",
		"custom-value": "custom-value",
	}
	for input, want := range tests {
		if got := editorWriteEncoding(input); got != want {
			t.Fatalf("editorWriteEncoding(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTextEditorBindingTracksEncodingDirty(t *testing.T) {
	binding := &textEditorBinding{sourceEncoding: "utf-8", saveEncoding: "utf-8-bom"}
	if !binding.encodingDirty() {
		t.Fatalf("expected changed save encoding to mark editor dirty")
	}
	binding.markEncodingSaved("utf8-bom")
	if binding.encodingDirty() {
		t.Fatalf("expected saved encoding to become clean")
	}
}
