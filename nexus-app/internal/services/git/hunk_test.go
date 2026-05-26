package git

import (
	"strings"
	"testing"
)

func TestParseDiffHunks(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/app.go b/app.go",
		"index 111..222 100644",
		"--- a/app.go",
		"+++ b/app.go",
		"@@ -1,2 +1,3 @@",
		" package main",
		"-old",
		"+new",
		"+added",
		"@@ -20 +21 @@",
		"-next old",
		"+next new",
	}, "\n")

	hunks := parseDiffHunks(DiffKindUnstaged, diff)
	if len(hunks) != 2 {
		t.Fatalf("expected two hunks, got %#v", hunks)
	}
	first := hunks[0]
	if first.Kind != DiffKindUnstaged || first.Index != 0 || first.OldStart != 1 || first.OldLines != 2 || first.NewStart != 1 || first.NewLines != 3 {
		t.Fatalf("unexpected first hunk metadata: %#v", first)
	}
	if first.DeletedLines != 1 || first.AddedLines != 2 {
		t.Fatalf("unexpected first hunk counts: %#v", first)
	}
	second := hunks[1]
	if second.OldStart != 20 || second.OldLines != 1 || second.NewStart != 21 || second.NewLines != 1 {
		t.Fatalf("unexpected single-line hunk metadata: %#v", second)
	}
}

func TestParseHunkHeaderRejectsNonHunk(t *testing.T) {
	if _, ok := parseHunkHeader(DiffKindStaged, 0, "diff --git a/app.go b/app.go"); ok {
		t.Fatal("expected non-hunk line to be rejected")
	}
}
