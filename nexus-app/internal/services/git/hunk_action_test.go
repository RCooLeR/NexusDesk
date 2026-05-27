package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHunkActionArgs(t *testing.T) {
	stage, err := hunkActionArgs(HunkActionStage, DiffKindUnstaged)
	if err != nil {
		t.Fatalf("unexpected stage error: %v", err)
	}
	if strings.Join(stage, " ") != "--cached --whitespace=nowarn" {
		t.Fatalf("unexpected stage args: %#v", stage)
	}
	unstage, err := hunkActionArgs(HunkActionUnstage, DiffKindStaged)
	if err != nil {
		t.Fatalf("unexpected unstage error: %v", err)
	}
	if strings.Join(unstage, " ") != "--cached --reverse --whitespace=nowarn" {
		t.Fatalf("unexpected unstage args: %#v", unstage)
	}
	if _, err := hunkActionArgs(HunkActionStage, DiffKindStaged); err == nil {
		t.Fatal("expected invalid action/kind pair to fail")
	}
}

func TestExtractHunkPatchUsesZeroBasedIndex(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/notes.txt b/notes.txt",
		"index 111..222 100644",
		"--- a/notes.txt",
		"+++ b/notes.txt",
		"@@ -1,3 +1,3 @@",
		" line 1",
		"-line 2",
		"+changed 2",
		"@@ -20,3 +20,3 @@",
		" line 20",
		"-line 21",
		"+changed 21",
	}, "\n")

	patch, err := extractHunkPatch(diff, 1)
	if err != nil {
		t.Fatalf("extractHunkPatch returned error: %v", err)
	}
	if !strings.Contains(patch, "@@ -20,3 +20,3 @@") || strings.Contains(patch, "@@ -1,3 +1,3 @@") {
		t.Fatalf("unexpected patch: %q", patch)
	}
}

func TestApplyHunkActionStagesOneUnstagedHunk(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := createTwoHunkRepo(t)

	result, err := New().ApplyHunkAction(root, "notes.txt", DiffKindUnstaged, 0, HunkActionStage)
	if err != nil {
		t.Fatalf("ApplyHunkAction returned error: %v", err)
	}
	if result.Message != "Staged hunk 1 in notes.txt." {
		t.Fatalf("unexpected message: %q", result.Message)
	}
	staged := runGitOutput(t, root, "diff", "--cached", "--", "notes.txt")
	unstaged := runGitOutput(t, root, "diff", "--", "notes.txt")
	if !strings.Contains(staged, "changed 2") || strings.Contains(staged, "changed 25") {
		t.Fatalf("unexpected staged diff: %q", staged)
	}
	if strings.Contains(unstaged, "\n+changed 2\n") || !strings.Contains(unstaged, "\n+changed 25\n") {
		t.Fatalf("unexpected unstaged diff: %q", unstaged)
	}
}

func TestApplyHunkActionUnstagesOneStagedHunk(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := createTwoHunkRepo(t)
	if _, err := New().ApplyHunkAction(root, "notes.txt", DiffKindUnstaged, 0, HunkActionStage); err != nil {
		t.Fatalf("stage hunk failed: %v", err)
	}

	result, err := New().ApplyHunkAction(root, "notes.txt", DiffKindStaged, 0, HunkActionUnstage)
	if err != nil {
		t.Fatalf("ApplyHunkAction returned error: %v", err)
	}
	if result.Message != "Unstaged hunk 1 in notes.txt." {
		t.Fatalf("unexpected message: %q", result.Message)
	}
	staged := runGitOutput(t, root, "diff", "--cached", "--", "notes.txt")
	if strings.Contains(staged, "changed 2") {
		t.Fatalf("expected staged hunk to be removed: %q", staged)
	}
}

func createTwoHunkRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	initial := []string{}
	for index := 1; index <= 30; index++ {
		initial = append(initial, "line "+itoa(index))
	}
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte(strings.Join(initial, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	runGit(t, root, "add", "notes.txt")
	runGit(t, root, "commit", "-m", "initial")
	modified := append([]string{}, initial...)
	modified[1] = "changed 2"
	modified[24] = "changed 25"
	if err := os.WriteFile(path, []byte(strings.Join(modified, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write modified file: %v", err)
	}
	return root
}

func runGitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	output, err := gitOutput(root, args...)
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return output
}

func itoa(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var out []byte
	for value > 0 {
		out = append([]byte{digits[value%10]}, out...)
		value /= 10
	}
	return string(out)
}
