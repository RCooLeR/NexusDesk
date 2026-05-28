package tasks

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

func discover(root string) (Summary, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return Summary{}, errors.New("open a workspace before listing tasks")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Summary{}, err
	}

	packageFiles := []string{}
	goModules := []string{}
	goTestFilesByModule := map[string]map[string]bool{}
	cargoFiles := []string{}
	pythonConfigs := []string{}
	composeFiles := []string{}
	visited := 0

	err = filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || path == absRoot {
			return nil
		}
		rel, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if shouldSkipDir(entry.Name(), rel) || pathDepth(rel) > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		visited++
		if visited > maxFiles {
			return filepath.SkipAll
		}
		switch entry.Name() {
		case "package.json":
			packageFiles = append(packageFiles, path)
		case "go.mod":
			goModules = append(goModules, path)
		case "Cargo.toml":
			cargoFiles = append(cargoFiles, path)
		case "pyproject.toml", "pytest.ini", "tox.ini", "setup.cfg":
			pythonConfigs = append(pythonConfigs, path)
		default:
			if isComposeFile(entry.Name()) {
				composeFiles = append(composeFiles, path)
			}
			if strings.HasSuffix(entry.Name(), "_test.go") {
				moduleRoot := nearestGoModuleRoot(absRoot, filepath.Dir(path))
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
		return Summary{}, err
	}

	tasks := []Task{}
	for _, packageFile := range packageFiles {
		tasks = append(tasks, npmScriptTasks(absRoot, packageFile)...)
	}
	sort.Strings(goModules)
	for _, goMod := range goModules {
		moduleRoot := filepath.Dir(goMod)
		tasks = append(tasks, Task{
			ID:      taskID("go", relDir(absRoot, moduleRoot), "test-all"),
			Kind:    "go-test",
			Label:   "go test ./...",
			Command: "go test ./...",
			Cwd:     relDir(absRoot, moduleRoot),
			Source:  relFile(absRoot, goMod),
		})
	}
	tasks = append(tasks, packageGoTestTasks(absRoot, goTestFilesByModule)...)
	tasks = append(tasks, pythonPytestTasks(absRoot, pythonConfigs)...)
	sort.Strings(cargoFiles)
	for _, cargoFile := range cargoFiles {
		if len(tasks) >= maxTasks {
			break
		}
		cwd := relDir(absRoot, filepath.Dir(cargoFile))
		tasks = append(tasks, Task{
			ID:      taskID("cargo", cwd, "test"),
			Kind:    "cargo-test",
			Label:   "cargo test",
			Command: "cargo test",
			Cwd:     cwd,
			Source:  relFile(absRoot, cargoFile),
		})
	}
	sort.Strings(composeFiles)
	for _, composeFile := range composeFiles {
		if len(tasks) >= maxTasks {
			break
		}
		cwd := relDir(absRoot, filepath.Dir(composeFile))
		source := relFile(absRoot, composeFile)
		tasks = append(tasks, Task{
			ID:      taskID("compose", cwd, source),
			Kind:    "compose",
			Label:   "docker compose config",
			Command: "docker compose -f " + quotePath(filepath.Base(composeFile)) + " config",
			Cwd:     cwd,
			Source:  source,
		})
	}

	sort.SliceStable(tasks, func(left int, right int) bool {
		if tasks[left].Cwd == tasks[right].Cwd {
			return tasks[left].Label < tasks[right].Label
		}
		return tasks[left].Cwd < tasks[right].Cwd
	})
	if len(tasks) > maxTasks {
		tasks = tasks[:maxTasks]
	}
	message := fmt.Sprintf("%d tasks detected from package scripts, Go tests, Python pytest, Cargo tests, and Compose files.", len(tasks))
	if len(tasks) == 0 {
		message = "No package scripts, Go tests, Python pytest, Cargo tests, or Compose files detected."
	}
	return Summary{Tasks: tasks, Message: message, GeneratedAt: time.Now().UTC()}, nil
}

func pythonPytestTasks(root string, configFiles []string) []Task {
	sort.Strings(configFiles)
	seenDirs := map[string]bool{}
	tasks := []Task{}
	for _, configFile := range configFiles {
		if len(tasks) >= maxTasks {
			break
		}
		dir := filepath.Dir(configFile)
		if seenDirs[dir] {
			continue
		}
		seenDirs[dir] = true
		cwd := relDir(root, dir)
		tasks = append(tasks, Task{
			ID:      taskID("pytest", cwd, "test"),
			Kind:    "python-pytest",
			Label:   "python -m pytest",
			Command: "python -m pytest",
			Cwd:     cwd,
			Source:  relFile(root, configFile),
		})
	}
	return tasks
}

func packageGoTestTasks(root string, goTestFilesByModule map[string]map[string]bool) []Task {
	tasks := []Task{}
	for moduleRoot, packageDirs := range goTestFilesByModule {
		dirs := make([]string, 0, len(packageDirs))
		for dir := range packageDirs {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)
		for _, dir := range dirs {
			if len(tasks) >= maxTasks {
				break
			}
			relPackage, err := filepath.Rel(moduleRoot, dir)
			if err != nil || relPackage == "." {
				continue
			}
			relPackage = filepath.ToSlash(relPackage)
			command := "go test ./" + relPackage
			tasks = append(tasks, Task{
				ID:      taskID("go", relDir(root, moduleRoot), relPackage),
				Kind:    "go-test",
				Label:   command,
				Command: command,
				Cwd:     relDir(root, moduleRoot),
				Source:  relFile(root, filepath.Join(moduleRoot, "go.mod")),
			})
		}
	}
	return tasks
}

func npmScriptTasks(root string, packageFile string) []Task {
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
	tasks := make([]Task, 0, len(names))
	for _, name := range names {
		tasks = append(tasks, Task{
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
