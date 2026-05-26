package tasks

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDiscoverListsNpmScriptsGoTestsAndCompose(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "app", "frontend", "package.json"), `{
		"scripts": {
			"build": "vite build",
			"smoke": "node scripts/smoke.mjs"
		}
	}`)
	mustWrite(t, filepath.Join(root, "app", "go.mod"), "module fixture\n\ngo 1.24\n")
	mustWrite(t, filepath.Join(root, "app", "internal", "widget", "widget_test.go"), "package widget\n")
	mustWrite(t, filepath.Join(root, "services", "docker-compose.yml"), "services:\n  web:\n    image: nginx\n")

	summary, err := New().Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	assertTask(t, summary.Tasks, "npm-script", "npm run build", "app/frontend", "app/frontend/package.json")
	assertTask(t, summary.Tasks, "npm-script", "npm run smoke", "app/frontend", "app/frontend/package.json")
	assertTask(t, summary.Tasks, "go-test", "go test ./...", "app", "app/go.mod")
	assertTask(t, summary.Tasks, "go-test", "go test ./internal/widget", "app", "app/go.mod")
	assertTask(t, summary.Tasks, "compose", "docker compose config", "services", "services/docker-compose.yml")
	if summary.Message == "" || summary.GeneratedAt.IsZero() {
		t.Fatalf("expected message and timestamp, got %#v", summary)
	}
}

func TestDiscoverSkipsNoisyDirectories(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "node_modules", "package.json"), `{"scripts":{"bad":"bad"}}`)
	mustWrite(t, filepath.Join(root, ".git", "package.json"), `{"scripts":{"bad":"bad"}}`)

	summary, err := New().Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(summary.Tasks) != 0 {
		t.Fatalf("expected no tasks from ignored directories, got %#v", summary.Tasks)
	}
}

func TestRunRejectsUnknownTask(t *testing.T) {
	if _, err := New().Run(t.TempDir(), "missing-task"); err == nil {
		t.Fatal("expected missing task to be rejected")
	}
}

func TestRunCapturesGoTestOutput(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go executable is not available")
	}
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module taskfixture\n\ngo 1.24\n")
	mustWrite(t, filepath.Join(root, "main_test.go"), "package taskfixture\n\nimport \"testing\"\n\nfunc TestOK(t *testing.T) {}\n")

	service := New()
	summary, err := service.Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	var taskID string
	for _, task := range summary.Tasks {
		if task.Kind == "go-test" && task.Command == "go test ./..." {
			taskID = task.ID
			break
		}
	}
	if taskID == "" {
		t.Fatalf("expected go test task in %#v", summary.Tasks)
	}

	result, err := service.Run(root, taskID)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != "success" || result.ExitCode != 0 {
		t.Fatalf("expected successful task result, got %#v", result)
	}
	if result.Stdout == "" {
		t.Fatalf("expected captured stdout, got %#v", result)
	}
}

func TestServiceRequiresWorkspaceRoot(t *testing.T) {
	service := New()
	if _, err := service.Discover(""); err == nil {
		t.Fatal("expected empty discover root to be rejected")
	}
	if _, err := service.Run("", "task"); err == nil {
		t.Fatal("expected empty run root to be rejected")
	}
}

func TestValidateRunnableTask(t *testing.T) {
	for _, task := range []Task{
		{Kind: "npm-script", Command: "npm run build", Label: "npm run build"},
		{Kind: "go-test", Command: "go test ./...", Label: "go test ./..."},
		{Kind: "compose", Command: "docker compose -f compose.yml config", Label: "docker compose config"},
	} {
		if err := validateRunnableTask(task); err != nil {
			t.Fatalf("expected %s to be runnable: %v", task.Command, err)
		}
	}
	if err := validateRunnableTask(Task{Kind: "shell", Command: "rm -rf .", Label: "bad"}); err == nil {
		t.Fatal("expected unsafe task to be rejected")
	}
}

func assertTask(t *testing.T, tasks []Task, kind string, label string, cwd string, source string) {
	t.Helper()
	for _, task := range tasks {
		if task.Kind == kind && task.Label == label && task.Cwd == cwd && task.Source == source {
			return
		}
	}
	t.Fatalf("task %s %s in %s from %s not found in %#v", kind, label, cwd, source, tasks)
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
