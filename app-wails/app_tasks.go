package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"NexusAugenticStudio/internal/artifact"
	"NexusAugenticStudio/internal/processutil"
)

const workspaceTaskMaxFiles = 1500
const workspaceTaskMaxDepth = 8
const workspaceTaskMaxTasks = 80
const workspaceTaskTimeout = 2 * time.Minute
const workspaceTaskOutputLimit = 24 * 1024

type WorkspaceTask struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Label   string `json:"label"`
	Command string `json:"command"`
	Cwd     string `json:"cwd"`
	Source  string `json:"source"`
}

type WorkspaceTaskSummary struct {
	Tasks       []WorkspaceTask `json:"tasks"`
	Message     string          `json:"message"`
	GeneratedAt string          `json:"generatedAt"`
}

type WorkspaceTaskRunRequest struct {
	TaskID string `json:"taskId"`
}

type WorkspaceTaskRunResult struct {
	Task            WorkspaceTask `json:"task"`
	Status          string        `json:"status"`
	ExitCode        int           `json:"exitCode"`
	Stdout          string        `json:"stdout"`
	Stderr          string        `json:"stderr"`
	StartedAt       string        `json:"startedAt"`
	CompletedAt     string        `json:"completedAt"`
	DurationMs      int64         `json:"durationMs"`
	ArtifactRelPath string        `json:"artifactRelPath"`
	Message         string        `json:"message"`
}

func (a *App) ListWorkspaceTasks() (WorkspaceTaskSummary, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return WorkspaceTaskSummary{}, errors.New("open a workspace before listing tasks")
	}
	return discoverWorkspaceTasks(root)
}

func (a *App) RunWorkspaceTask(request WorkspaceTaskRunRequest) (WorkspaceTaskRunResult, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return WorkspaceTaskRunResult{}, errors.New("open a workspace before running tasks")
	}
	result, err := runDiscoveredWorkspaceTask(root, request.TaskID)
	if err != nil {
		return WorkspaceTaskRunResult{}, err
	}

	report, err := a.artifactSvc.CreateGeneratedMarkdown(artifact.MarkdownArtifactRequest{
		Title:          "Task Run: " + result.Task.Label,
		Content:        taskRunMarkdown(result),
		Kind:           "task-run",
		ContextRelPath: result.Task.Source,
		Prompt:         result.Task.Command,
		Source:         "Workspace task runner",
		SourcePaths:    []string{result.Task.Source},
	})
	if err != nil {
		return WorkspaceTaskRunResult{}, err
	}
	result.ArtifactRelPath = report.RelPath
	result.Message = fmt.Sprintf("%s Artifact saved to %s.", result.Message, report.RelPath)
	a.recordApproval("workspace.task.run", result.Task.Command, taskRunRisk(result.Task), result.Message)
	return result, nil
}

func discoverWorkspaceTasks(root string) (WorkspaceTaskSummary, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return WorkspaceTaskSummary{}, err
	}

	packageFiles := []string{}
	goModules := []string{}
	goTestFilesByModule := map[string]map[string]bool{}
	composeFiles := []string{}
	visited := 0

	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path == root {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		depth := taskPathDepth(rel)
		if entry.IsDir() {
			if shouldSkipTaskDir(entry.Name(), rel) || depth > workspaceTaskMaxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		visited++
		if visited > workspaceTaskMaxFiles {
			return filepath.SkipAll
		}
		switch entry.Name() {
		case "package.json":
			packageFiles = append(packageFiles, path)
		case "go.mod":
			goModules = append(goModules, path)
		default:
			if isComposeTaskFile(entry.Name()) {
				composeFiles = append(composeFiles, path)
			}
			if strings.HasSuffix(entry.Name(), "_test.go") {
				moduleRoot := nearestGoModuleRoot(root, filepath.Dir(path))
				if moduleRoot != "" {
					if goTestFilesByModule[moduleRoot] == nil {
						goTestFilesByModule[moduleRoot] = map[string]bool{}
					}
					goTestFilesByModule[moduleRoot][filepath.Dir(path)] = true
				}
			}
		}
		return nil
	})
	if err != nil {
		return WorkspaceTaskSummary{}, err
	}

	tasks := []WorkspaceTask{}
	for _, packageFile := range packageFiles {
		tasks = append(tasks, npmScriptTasks(root, packageFile)...)
	}
	sort.Strings(goModules)
	for _, goMod := range goModules {
		moduleRoot := filepath.Dir(goMod)
		tasks = append(tasks, WorkspaceTask{
			ID:      taskID("go", relDir(root, moduleRoot), "test-all"),
			Kind:    "go-test",
			Label:   "go test ./...",
			Command: "go test ./...",
			Cwd:     relDir(root, moduleRoot),
			Source:  relFile(root, goMod),
		})
	}
	for moduleRoot, packageDirs := range goTestFilesByModule {
		dirs := make([]string, 0, len(packageDirs))
		for dir := range packageDirs {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)
		for _, dir := range dirs {
			if len(tasks) >= workspaceTaskMaxTasks {
				break
			}
			relPackage, err := filepath.Rel(moduleRoot, dir)
			if err != nil || relPackage == "." {
				continue
			}
			relPackage = filepath.ToSlash(relPackage)
			command := "go test ./" + relPackage
			tasks = append(tasks, WorkspaceTask{
				ID:      taskID("go", relDir(root, moduleRoot), relPackage),
				Kind:    "go-test",
				Label:   command,
				Command: command,
				Cwd:     relDir(root, moduleRoot),
				Source:  relFile(root, filepath.Join(moduleRoot, "go.mod")),
			})
		}
	}
	sort.Strings(composeFiles)
	for _, composeFile := range composeFiles {
		if len(tasks) >= workspaceTaskMaxTasks {
			break
		}
		cwd := relDir(root, filepath.Dir(composeFile))
		source := relFile(root, composeFile)
		tasks = append(tasks, WorkspaceTask{
			ID:      taskID("compose", cwd, source),
			Kind:    "compose",
			Label:   "docker compose config",
			Command: "docker compose -f " + quoteTaskPath(filepath.Base(composeFile)) + " config",
			Cwd:     cwd,
			Source:  source,
		})
	}

	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].Cwd == tasks[j].Cwd {
			return tasks[i].Label < tasks[j].Label
		}
		return tasks[i].Cwd < tasks[j].Cwd
	})
	if len(tasks) > workspaceTaskMaxTasks {
		tasks = tasks[:workspaceTaskMaxTasks]
	}
	message := fmt.Sprintf("%d tasks detected from package scripts, Go tests, and Compose files.", len(tasks))
	if len(tasks) == 0 {
		message = "No package scripts, Go tests, or Compose files detected."
	}
	return WorkspaceTaskSummary{
		Tasks:       tasks,
		Message:     message,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func runDiscoveredWorkspaceTask(root string, taskID string) (WorkspaceTaskRunResult, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return WorkspaceTaskRunResult{}, errors.New("task id is required")
	}
	summary, err := discoverWorkspaceTasks(root)
	if err != nil {
		return WorkspaceTaskRunResult{}, err
	}
	var selected WorkspaceTask
	for _, task := range summary.Tasks {
		if task.ID == taskID {
			selected = task
			break
		}
	}
	if selected.ID == "" {
		return WorkspaceTaskRunResult{}, errors.New("task is no longer available in this workspace")
	}
	if err := validateRunnableTask(selected); err != nil {
		return WorkspaceTaskRunResult{}, err
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return WorkspaceTaskRunResult{}, err
	}
	cwd := filepath.Clean(filepath.Join(absRoot, filepath.FromSlash(selected.Cwd)))
	if selected.Cwd == "." || selected.Cwd == "" {
		cwd = absRoot
	}
	if err := ensureTaskInsideRoot(absRoot, cwd); err != nil {
		return WorkspaceTaskRunResult{}, err
	}

	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), workspaceTaskTimeout)
	defer cancel()

	cmd := taskExecCommand(ctx, selected.Command)
	cmd.Dir = cwd
	processutil.ConfigureHiddenCommand(cmd)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = limitWriter{buffer: &stdout, limit: workspaceTaskOutputLimit}
	cmd.Stderr = limitWriter{buffer: &stderr, limit: workspaceTaskOutputLimit}
	err = cmd.Run()
	completed := time.Now()

	exitCode := 0
	status := "success"
	message := fmt.Sprintf("Task %q completed.", selected.Label)
	if ctx.Err() == context.DeadlineExceeded {
		exitCode = -1
		status = "timeout"
		message = fmt.Sprintf("Task %q timed out after %s.", selected.Label, workspaceTaskTimeout)
	} else if err != nil {
		status = "failed"
		message = fmt.Sprintf("Task %q failed.", selected.Label)
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
			message = err.Error()
		}
	}

	return WorkspaceTaskRunResult{
		Task:        selected,
		Status:      status,
		ExitCode:    exitCode,
		Stdout:      stdout.String(),
		Stderr:      stderr.String(),
		StartedAt:   started.UTC().Format(time.RFC3339),
		CompletedAt: completed.UTC().Format(time.RFC3339),
		DurationMs:  completed.Sub(started).Milliseconds(),
		Message:     message,
	}, nil
}

func ensureTaskInsideRoot(root string, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("task working directory must stay inside workspace")
	}
	return nil
}

func validateRunnableTask(task WorkspaceTask) error {
	switch task.Kind {
	case "npm-script":
		if strings.HasPrefix(task.Command, "npm run ") {
			return nil
		}
	case "go-test":
		if strings.HasPrefix(task.Command, "go test ") {
			return nil
		}
	case "compose":
		if strings.HasPrefix(task.Command, "docker compose -f ") && strings.HasSuffix(task.Command, " config") {
			return nil
		}
	}
	return fmt.Errorf("task %q is not runnable by the safe task runner", task.Label)
}

func taskRunRisk(task WorkspaceTask) string {
	if task.Kind == "compose" {
		return "medium"
	}
	return "low"
}

func taskRunMarkdown(result WorkspaceTaskRunResult) string {
	var builder strings.Builder
	builder.WriteString("## Summary\n\n")
	builder.WriteString("- Task: `")
	builder.WriteString(strings.ReplaceAll(result.Task.Label, "`", "'"))
	builder.WriteString("`\n")
	builder.WriteString("- Status: ")
	builder.WriteString(result.Status)
	builder.WriteString("\n")
	builder.WriteString("- Exit code: ")
	builder.WriteString(fmt.Sprintf("%d", result.ExitCode))
	builder.WriteString("\n")
	builder.WriteString("- Working directory: `")
	builder.WriteString(strings.ReplaceAll(result.Task.Cwd, "`", "'"))
	builder.WriteString("`\n")
	builder.WriteString("- Command: `")
	builder.WriteString(strings.ReplaceAll(result.Task.Command, "`", "'"))
	builder.WriteString("`\n")
	builder.WriteString("- Started: ")
	builder.WriteString(result.StartedAt)
	builder.WriteString("\n")
	builder.WriteString("- Completed: ")
	builder.WriteString(result.CompletedAt)
	builder.WriteString("\n\n")
	builder.WriteString("## Stdout\n\n```text\n")
	builder.WriteString(result.Stdout)
	if !strings.HasSuffix(result.Stdout, "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString("```\n\n## Stderr\n\n```text\n")
	builder.WriteString(result.Stderr)
	if !strings.HasSuffix(result.Stderr, "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString("```\n")
	return builder.String()
}

type limitWriter struct {
	buffer *bytes.Buffer
	limit  int
}

func (w limitWriter) Write(p []byte) (int, error) {
	if w.buffer.Len() < w.limit {
		remaining := w.limit - w.buffer.Len()
		if len(p) <= remaining {
			_, _ = w.buffer.Write(p)
		} else {
			_, _ = w.buffer.Write(p[:remaining])
			_, _ = w.buffer.WriteString("\n[output truncated]\n")
		}
	}
	return len(p), nil
}

func isComposeTaskFile(name string) bool {
	lower := strings.ToLower(name)
	return lower == "compose.yml" ||
		lower == "compose.yaml" ||
		lower == "docker-compose.yml" ||
		lower == "docker-compose.yaml"
}

func quoteTaskPath(path string) string {
	if strings.ContainsAny(path, " \t\"") {
		return `"` + strings.ReplaceAll(path, `"`, `\"`) + `"`
	}
	return path
}

func npmScriptTasks(root string, packageFile string) []WorkspaceTask {
	content, err := os.ReadFile(packageFile)
	if err != nil {
		return nil
	}
	var manifest struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(content, &manifest); err != nil || len(manifest.Scripts) == 0 {
		return nil
	}
	names := make([]string, 0, len(manifest.Scripts))
	for name := range manifest.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)
	cwd := relDir(root, filepath.Dir(packageFile))
	tasks := make([]WorkspaceTask, 0, len(names))
	for _, name := range names {
		tasks = append(tasks, WorkspaceTask{
			ID:      taskID("npm", cwd, name),
			Kind:    "npm-script",
			Label:   "npm run " + name,
			Command: "npm run " + name,
			Cwd:     cwd,
			Source:  relFile(root, packageFile),
		})
	}
	return tasks
}

func nearestGoModuleRoot(root string, dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		if dir == root {
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func shouldSkipTaskDir(name string, relPath string) bool {
	switch name {
	case ".git", ".idea", ".nexusdesk", "node_modules", "dist", "build", "vendor":
		return true
	}
	normalized := strings.ToLower(filepath.ToSlash(relPath))
	return strings.HasPrefix(normalized, "app/frontend/node_modules") ||
		strings.HasPrefix(normalized, "app/frontend/dist") ||
		strings.HasPrefix(normalized, "app/build")
}

func relDir(root string, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return "."
	}
	return filepath.ToSlash(rel)
}

func relFile(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return filepath.ToSlash(filepath.Base(path))
	}
	return filepath.ToSlash(rel)
}

func taskPathDepth(relPath string) int {
	if relPath == "." || relPath == "" {
		return 0
	}
	return strings.Count(filepath.ToSlash(relPath), "/") + 1
}

func taskID(kind string, cwd string, name string) string {
	value := strings.ToLower(kind + ":" + cwd + ":" + name)
	value = strings.NewReplacer("\\", "/", " ", "-", ":", "-", "@", "-", ".", "-").Replace(value)
	value = strings.Trim(value, "-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return value
}
