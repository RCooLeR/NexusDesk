package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHistoryTruncatesAtLimit(t *testing.T) {
	output := strings.Join([]string{
		"1111111111111111111111111111111111111111\x1f1111111\x1fAda\x1fada@example.com\x1f2026-01-01T00:00:00Z\x1fAdd core",
		"2222222222222222222222222222222222222222\x1f2222222\x1fGrace\x1fgrace@example.com\x1f2026-01-02T00:00:00Z\x1fTune UI",
	}, "\n")
	entries, truncated := parseHistory(output, 1)
	if !truncated {
		t.Fatal("expected history to report truncation")
	}
	if len(entries) != 1 || entries[0].ShortHash != "1111111" || entries[0].Subject != "Add core" {
		t.Fatalf("unexpected history entries: %#v", entries)
	}
}

func TestParseBlameReadsPorcelainLines(t *testing.T) {
	output := strings.Join([]string{
		"1111111111111111111111111111111111111111 1 7 1",
		"author Ada",
		"author-time 1767225600",
		"summary Add core",
		"\tline seven",
	}, "\n")
	lines, truncated := parseBlame(output, 10)
	if truncated {
		t.Fatal("did not expect blame truncation")
	}
	if len(lines) != 1 {
		t.Fatalf("expected one blame line, got %#v", lines)
	}
	if lines[0].Line != 7 || lines[0].Author != "Ada" || lines[0].ShortHash != "111111111111" || lines[0].Content != "line seven" {
		t.Fatalf("unexpected blame line: %#v", lines[0])
	}
}

func TestHistoryAndBlame(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("line one\nline two\n"), 0o644); err != nil {
		t.Fatalf("write notes file: %v", err)
	}
	runGit(t, root, "add", "notes.txt")
	runGit(t, root, "commit", "-m", "initial notes")

	service := New()
	history, err := service.History(root, "notes.txt", 5)
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	if !history.Available || len(history.Entries) != 1 || history.Entries[0].Subject != "initial notes" {
		t.Fatalf("unexpected history result: %#v", history)
	}

	blame, err := service.Blame(root, "notes.txt", 2, 2)
	if err != nil {
		t.Fatalf("Blame returned error: %v", err)
	}
	if !blame.Available || len(blame.Lines) != 1 || blame.Lines[0].Line != 2 || blame.Lines[0].Content != "line two" {
		t.Fatalf("unexpected blame result: %#v", blame)
	}
}
