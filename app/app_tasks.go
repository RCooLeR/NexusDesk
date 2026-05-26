package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const workspaceTaskMaxFiles = 1500
const workspaceTaskMaxDepth = 8
const workspaceTaskMaxTasks = 80

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

func (a *App) ListWorkspaceTasks() (WorkspaceTaskSummary, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return WorkspaceTaskSummary{}, errors.New("open a workspace before listing tasks")
	}
	return discoverWorkspaceTasks(root)
}

func discoverWorkspaceTasks(root string) (WorkspaceTaskSummary, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return WorkspaceTaskSummary{}, err
	}

	packageFiles := []string{}
	goModules := []string{}
	goTestFilesByModule := map[string]map[string]bool{}
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

	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].Cwd == tasks[j].Cwd {
			return tasks[i].Label < tasks[j].Label
		}
		return tasks[i].Cwd < tasks[j].Cwd
	})
	if len(tasks) > workspaceTaskMaxTasks {
		tasks = tasks[:workspaceTaskMaxTasks]
	}
	message := fmt.Sprintf("%d tasks detected from package scripts and Go tests.", len(tasks))
	if len(tasks) == 0 {
		message = "No package scripts or Go tests detected."
	}
	return WorkspaceTaskSummary{
		Tasks:       tasks,
		Message:     message,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}, nil
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
