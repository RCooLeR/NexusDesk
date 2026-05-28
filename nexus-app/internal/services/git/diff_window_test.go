package git

import (
	"fmt"
	"strings"
	"testing"
)

func TestWindowUnifiedDiffKeepsWholeHunksAndElidesRemainder(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("diff --git a/large.txt b/large.txt\n")
	builder.WriteString("--- a/large.txt\n")
	builder.WriteString("+++ b/large.txt\n")
	for index := 0; index < diffPreviewMaxHunks+3; index++ {
		builder.WriteString(fmt.Sprintf("@@ -%d +%d @@\n", index+1, index+1))
		builder.WriteString("-old\n")
		builder.WriteString("+new\n")
	}
	diff := builder.String() + strings.Repeat("+tail\n", diffMaxBytes)

	windowed, truncated := windowUnifiedDiff(diff)
	if !truncated {
		t.Fatal("expected large diff to be marked truncated")
	}
	if !strings.Contains(windowed, "diff --git a/large.txt b/large.txt") {
		t.Fatalf("expected diff header to be preserved:\n%s", windowed)
	}
	if strings.Contains(windowed, fmt.Sprintf("@@ -%d +%d @@", diffPreviewMaxHunks+1, diffPreviewMaxHunks+1)) {
		t.Fatalf("expected hunks beyond preview window to be elided:\n%s", windowed)
	}
	if !strings.Contains(windowed, "diff preview elided") {
		t.Fatalf("expected explicit elision marker:\n%s", windowed)
	}
	if len(windowed) > diffMaxBytes+200 {
		t.Fatalf("expected bounded preview, got %d bytes", len(windowed))
	}
}

func TestWindowUnifiedDiffFallsBackForHugeHeaderOnlyDiff(t *testing.T) {
	diff := "diff --git a/blob.bin b/blob.bin\n" + strings.Repeat("index line\n", diffMaxBytes)

	windowed, truncated := windowUnifiedDiff(diff)
	if !truncated {
		t.Fatal("expected huge header-only diff to be marked truncated")
	}
	if !strings.Contains(windowed, "diff preview elided") {
		t.Fatalf("expected fallback elision marker:\n%s", windowed)
	}
	if !strings.Contains(windowed, "diff --git a/blob.bin b/blob.bin") {
		t.Fatalf("expected fallback to keep beginning of diff:\n%s", windowed)
	}
}
