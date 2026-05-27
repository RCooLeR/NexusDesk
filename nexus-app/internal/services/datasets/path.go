package datasets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func resolveDatasetFile(root string, relPath string) (string, string, os.FileInfo, error) {
	root = strings.TrimSpace(root)
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	if root == "" {
		return "", "", nil, errors.New("dataset root is required")
	}
	if relPath == "" {
		return "", "", nil, errors.New("dataset path is required")
	}
	if filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "../") || relPath == ".." || strings.Contains(relPath, "/../") {
		return "", "", nil, errors.New("dataset path must stay inside the workspace")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", nil, err
	}
	target := filepath.Join(absRoot, filepath.FromSlash(relPath))
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", "", nil, err
	}
	prefix := absRoot + string(filepath.Separator)
	if absTarget != absRoot && !strings.HasPrefix(absTarget, prefix) {
		return "", "", nil, errors.New("dataset path must stay inside the workspace")
	}
	info, err := os.Stat(absTarget)
	if err != nil {
		return "", "", nil, err
	}
	if info.IsDir() {
		return "", "", nil, errors.New("dataset path must be a file")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", "", nil, errors.New("dataset symlinks are not supported")
	}
	return absTarget, relPath, info, nil
}
