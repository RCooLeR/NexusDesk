package tasks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func ensureInsideRoot(root string, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("task working directory must stay inside workspace")
	}
	return nil
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

func shouldSkipDir(name string, relPath string) bool {
	switch name {
	case ".git", ".idea", ".nexusdesk", "node_modules", "dist", "build", "vendor":
		return true
	}
	return false
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

func pathDepth(relPath string) int {
	if relPath == "." || relPath == "" {
		return 0
	}
	return strings.Count(filepath.ToSlash(relPath), "/") + 1
}

func isComposeFile(name string) bool {
	lower := strings.ToLower(name)
	return lower == "compose.yml" ||
		lower == "compose.yaml" ||
		lower == "docker-compose.yml" ||
		lower == "docker-compose.yaml"
}

func quotePath(path string) string {
	if strings.ContainsAny(path, " \t\"") {
		return `"` + strings.ReplaceAll(path, `"`, `\"`) + `"`
	}
	return path
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
