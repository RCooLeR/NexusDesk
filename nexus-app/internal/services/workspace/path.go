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
	relPath = filepath.Clean(relPath)
	relPath = filepath.ToSlash(relPath)
	if relPath == "." {
		return "", nil
	}
	if relPath == ".." || strings.HasPrefix(relPath, "../") || strings.Contains(relPath, "/../") || strings.HasSuffix(relPath, "/..") {
		return "", errors.New("workspace path must stay inside the root")
	}
	if filepath.IsAbs(relPath) {
		return "", errors.New("workspace path must be relative")
	}
	if err := validateWorkspacePathComponents(relPath); err != nil {
		return "", err
	}
	return relPath, nil
}

func validateWorkspacePathComponents(relPath string) error {
	for _, component := range strings.Split(filepath.ToSlash(relPath), "/") {
		if component == "" || component == "." {
			continue
		}
		if strings.TrimSpace(component) != component {
			return errors.New("workspace path component cannot start or end with whitespace")
		}
		if strings.Contains(component, ":") {
			return errors.New("workspace path cannot use Windows drive or alternate-data-stream syntax")
		}
		if strings.TrimRight(component, " .") != component {
			return errors.New("workspace path component cannot end with a space or dot")
		}
		if isWindowsReservedPathComponent(component) {
			return errors.New("workspace path cannot use a Windows reserved device name")
		}
	}
	return nil
}

func isWindowsReservedPathComponent(component string) bool {
	base := strings.ToUpper(strings.TrimRight(component, " ."))
	if dot := strings.IndexByte(base, '.'); dot >= 0 {
		base = base[:dot]
	}
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CLOCK$":
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) {
		return base[3] >= '1' && base[3] <= '9'
	}
	return false
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
	if _, err := os.Lstat(filepath.Join(absRoot, filepath.FromSlash(cleanRelPath))); err != nil {
		return "", "", nil, err
	}
	resolvedTarget, err := ensureResolvedReadPathInsideRoot(absRoot, target)
	if err != nil {
		return "", "", nil, err
	}
	info, err := os.Stat(resolvedTarget)
	if err != nil {
		return "", "", nil, err
	}
	if info.IsDir() {
		return "", "", nil, errors.New("workspace path must be a file")
	}
	return resolvedTarget, cleanRelPath, info, nil
}

func resolveDirectory(root string, relPath string) (string, string, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return "", "", err
	}
	cleanRelPath, err := cleanRel(relPath)
	if err != nil {
		return "", "", err
	}
	target := filepath.Join(absRoot, filepath.FromSlash(cleanRelPath))
	if _, err := os.Lstat(target); err != nil {
		return "", "", err
	}
	resolvedTarget, err := ensureResolvedReadPathInsideRoot(absRoot, target)
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(resolvedTarget)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() {
		return "", "", errors.New("workspace path must be a directory")
	}
	return resolvedTarget, cleanRelPath, nil
}

func ensureResolvedReadPathInsideRoot(absRoot string, absTarget string) (string, error) {
	if absTarget == "" {
		return "", errors.New("workspace file path is required")
	}
	evaluatedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", err
	}
	targetInfo, err := os.Lstat(absTarget)
	if err != nil {
		return "", err
	}
	if targetInfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("workspace path cannot be a symlink")
	}
	resolvedTarget, err := filepath.EvalSymlinks(absTarget)
	if err != nil {
		return "", err
	}
	if !isInside(evaluatedRoot, resolvedTarget) {
		return "", errors.New("workspace path must stay inside the root")
	}
	return resolvedTarget, nil
}

func isInside(root string, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
