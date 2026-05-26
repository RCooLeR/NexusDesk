package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestApplyFileActionStagesAndUnstagesFile(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runGit(t, root, "init")
	path := filepath.Join(root, "app.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "--", "app.go")
	runGit(t, root, "-c", "user.email=nexus@example.test", "-c", "user.name=Nexus", "commit", "-m", "initial")

	if err := os.WriteFile(path, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := New()
	staged, err := service.ApplyFileAction(root, "app.go", FileActionStage)
	if err != nil {
		t.Fatalf("stage failed: %v", err)
	}
	if staged.Message != "Staged app.go." {
		t.Fatalf("unexpected stage message: %q", staged.Message)
	}
	if len(staged.Status.StagedFiles) != 1 || len(staged.Status.UnstagedFiles) != 0 {
		t.Fatalf("expected staged-only status after stage, got %#v", staged.Status)
	}

	unstaged, err := service.ApplyFileAction(root, "app.go", FileActionUnstage)
	if err != nil {
		t.Fatalf("unstage failed: %v", err)
	}
	if unstaged.Message != "Unstaged app.go." {
		t.Fatalf("unexpected unstage message: %q", unstaged.Message)
	}
	if len(unstaged.Status.StagedFiles) != 0 || len(unstaged.Status.UnstagedFiles) != 1 {
		t.Fatalf("expected unstaged-only status after unstage, got %#v", unstaged.Status)
	}
}

func TestRunFileActionRejectsUnsupportedAction(t *testing.T) {
	if err := runFileAction("", "app.go", FileAction("discard")); err == nil {
		t.Fatal("expected unsupported action to be rejected")
	}
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	if _, err := gitOutput(root, args...); err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
}
