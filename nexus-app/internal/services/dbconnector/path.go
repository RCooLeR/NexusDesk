package dbconnector

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func resolveSQLitePath(root string, relPath string) (string, string, string, error) {
	if strings.TrimSpace(root) == "" {
		return "", "", "", errors.New("open a workspace before inspecting SQLite files")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", "", err
	}
	cleanRel := filepath.Clean(strings.TrimSpace(relPath))
	if cleanRel == "." || filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, "..") {
		return "", "", "", errors.New("SQLite path must stay inside the workspace")
	}
	ext := strings.ToLower(filepath.Ext(cleanRel))
	if ext != ".sqlite" && ext != ".sqlite3" && ext != ".db" {
		return "", "", "", errors.New("selected file is not a SQLite database")
	}
	absPath := filepath.Join(absRoot, cleanRel)
	resolved, err := filepath.Abs(absPath)
	if err != nil {
		return "", "", "", err
	}
	rootPrefix := strings.ToLower(absRoot) + string(filepath.Separator)
	if !strings.HasPrefix(strings.ToLower(resolved), rootPrefix) && !strings.EqualFold(resolved, absRoot) {
		return "", "", "", errors.New("SQLite path must stay inside the workspace")
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return "", "", "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", "", "", errors.New("SQLite connector target must not be a symlink")
	}
	if info.IsDir() {
		return "", "", "", errors.New("SQLite connector target must be a file")
	}
	return absRoot, resolved, filepath.ToSlash(cleanRel), nil
}
