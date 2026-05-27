package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveExistingFile(root string, relPath string, action string) (string, string, os.FileInfo, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return "", "", nil, err
	}
	cleanRelPath, err := cleanRel(relPath)
	if err != nil {
		return "", "", nil, err
	}
	if cleanRelPath == "" {
		return "", "", nil, fmt.Errorf("%s target must name a file", action)
	}
	if isInternalMetadataPath(cleanRelPath) {
		return "", "", nil, fmt.Errorf("direct %ss involving Nexus metadata are not allowed", action)
	}
	absTarget, err := filepath.Abs(filepath.Join(absRoot, filepath.FromSlash(cleanRelPath)))
	if err != nil {
		return "", "", nil, err
	}
	if !isInside(absRoot, absTarget) {
		return "", "", nil, errors.New("workspace path must stay inside the root")
	}
	info, err := os.Lstat(absTarget)
	if os.IsNotExist(err) {
		return "", "", nil, fmt.Errorf("%s target does not exist", action)
	}
	if err != nil {
		return "", "", nil, err
	}
	if info.IsDir() {
		return "", "", nil, fmt.Errorf("%s target must be a file", action)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", "", nil, fmt.Errorf("%s target cannot be a symlink", action)
	}
	return absTarget, cleanRelPath, info, nil
}

func resolveNewFileTarget(root string, relPath string, action string) (string, string, string, error) {
	absRoot, absTarget, cleanRelPath, err := resolveWriteTarget(root, relPath)
	if err != nil {
		return "", "", "", err
	}
	if hasTrailingSeparator(relPath) {
		return "", "", "", fmt.Errorf("%s target must include a file name", action)
	}
	if _, err := os.Lstat(absTarget); err == nil {
		return "", "", "", fmt.Errorf("%s target already exists", action)
	} else if !os.IsNotExist(err) {
		return "", "", "", err
	}
	return absRoot, absTarget, cleanRelPath, nil
}

func resolveNewDirectoryTarget(root string, relPath string, action string) (string, string, string, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return "", "", "", err
	}
	cleanRelPath, err := cleanRel(relPath)
	if err != nil {
		return "", "", "", err
	}
	if cleanRelPath == "" {
		return "", "", "", fmt.Errorf("%s target must name a folder", action)
	}
	if isInternalMetadataPath(cleanRelPath) {
		return "", "", "", fmt.Errorf("direct %ss involving Nexus metadata are not allowed", action)
	}
	absTarget, err := filepath.Abs(filepath.Join(absRoot, filepath.FromSlash(cleanRelPath)))
	if err != nil {
		return "", "", "", err
	}
	if !isInside(absRoot, absTarget) {
		return "", "", "", errors.New("workspace path must stay inside the root")
	}
	if _, err := os.Lstat(absTarget); err == nil {
		return "", "", "", fmt.Errorf("%s target already exists", action)
	} else if !os.IsNotExist(err) {
		return "", "", "", err
	}
	return absRoot, absTarget, cleanRelPath, nil
}

func resolveTransferTargets(root string, sourceRelPath string, targetRelPath string, action string) (string, string, string, string, os.FileInfo, error) {
	absSource, cleanSource, info, err := resolveExistingFile(root, sourceRelPath, action+" source")
	if err != nil {
		return "", "", "", "", nil, err
	}
	if info.Size() > fileOperationMaxBytes {
		return "", "", "", "", nil, fmt.Errorf("%s source is too large for interactive file operations", action)
	}
	_, absTarget, cleanTarget, err := resolveNewFileTarget(root, targetRelPath, action)
	if err != nil {
		return "", "", "", "", nil, err
	}
	if filepath.ToSlash(cleanSource) == filepath.ToSlash(cleanTarget) {
		return "", "", "", "", nil, fmt.Errorf("%s target must be different from the source", action)
	}
	return absSource, absTarget, cleanSource, cleanTarget, info, nil
}

func hasTrailingSeparator(path string) bool {
	path = strings.TrimSpace(path)
	return strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\")
}
