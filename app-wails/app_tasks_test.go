package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverWorkspaceTasksListsNpmScriptsAndGoTests(t *testing.T) {
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

	summary, err := discoverWorkspaceTasks(root)
	if err != nil {
		t.Fatalf("discoverWorkspaceTasks() error = %v", err)
	}

	assertTask(t, summary.Tasks, "npm-script", "npm run build", "app/frontend", "app/frontend/package.json")
	assertTask(t, summary.Tasks, "npm-script", "npm run smoke", "app/frontend", "app/frontend/package.json")
	assertTask(t, summary.Tasks, "go-test", "go test ./...", "app", "app/go.mod")
	assertTask(t, summary.Tasks, "go-test", "go test ./internal/widget", "app", "app/go.mod")
	assertTask(t, summary.Tasks, "compose", "docker compose config", "services", "services/docker-compose.yml")
	if summary.Message == "" || summary.GeneratedAt == "" {
		t.Fatalf("expected message and timestamp, got %#v", summary)
	}
}

func TestDiscoverWorkspaceTasksSkipsNoisyDirectories(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "node_modules", "package.json"), `{"scripts":{"bad":"bad"}}`)
	mustWrite(t, filepath.Join(root, ".git", "package.json"), `{"scripts":{"bad":"bad"}}`)

	summary, err := discoverWorkspaceTasks(root)
	if err != nil {
		t.Fatalf("discoverWorkspaceTasks() error = %v", err)
	}
	if len(summary.Tasks) != 0 {
		t.Fatalf("expected no tasks from ignored directories, got %#v", summary.Tasks)
	}
}

func TestRunDiscoveredWorkspaceTaskCapturesGoTestOutput(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module taskfixture\n\ngo 1.24\n")
	mustWrite(t, filepath.Join(root, "main_test.go"), "package taskfixture\n\nimport \"testing\"\n\nfunc TestOK(t *testing.T) {}\n")

	summary, err := discoverWorkspaceTasks(root)
	if err != nil {
		t.Fatalf("discoverWorkspaceTasks() error = %v", err)
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

	result, err := runDiscoveredWorkspaceTask(root, taskID)
	if err != nil {
		t.Fatalf("runDiscoveredWorkspaceTask() error = %v", err)
	}
	if result.Status != "success" || result.ExitCode != 0 {
		t.Fatalf("expected successful task result, got %#v", result)
	}
	if result.Stdout == "" {
		t.Fatalf("expected captured stdout, got %#v", result)
	}
}

func TestRunDiscoveredWorkspaceTaskRejectsUnknownTask(t *testing.T) {
	root := t.TempDir()
	if _, err := runDiscoveredWorkspaceTask(root, "missing-task"); err == nil {
		t.Fatal("expected missing task to be rejected")
	}
}

func assertTask(t *testing.T, tasks []WorkspaceTask, kind string, label string, cwd string, source string) {
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
