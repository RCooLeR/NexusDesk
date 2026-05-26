package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func cleanRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", errors.New("workspace root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("workspace root must be a directory")
	}
	return absRoot, nil
}

func cleanRel(relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	relPath = strings.Trim(relPath, `"'`)
	relPath = filepath.ToSlash(relPath)
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "." {
		return "", nil
	}
	if relPath == ".." || strings.HasPrefix(relPath, "../") || strings.Contains(relPath, "/../") || strings.HasSuffix(relPath, "/..") {
		return "", errors.New("workspace path must stay inside the root")
	}
	if filepath.IsAbs(relPath) {
		return "", errors.New("workspace path must be relative")
	}
	return relPath, nil
}

func resolveFile(root string, relPath string) (string, string, os.FileInfo, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return "", "", nil, err
	}
	cleanRelPath, err := cleanRel(relPath)
	if err != nil {
		return "", "", nil, err
	}
	if cleanRelPath == "" {
		return "", "", nil, errors.New("workspace file path is required")
	}
	target := filepath.Join(absRoot, filepath.FromSlash(cleanRelPath))
	if !isInside(absRoot, target) {
		return "", "", nil, errors.New("workspace path must stay inside the root")
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", "", nil, err
	}
	if info.IsDir() {
		return "", "", nil, errors.New("workspace path must be a file")
	}
	return target, cleanRelPath, info, nil
}

func isInside(root string, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
