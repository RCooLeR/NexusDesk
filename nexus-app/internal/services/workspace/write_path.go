package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func resolveWriteTarget(root string, relPath string) (string, string, string, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return "", "", "", err
	}
	cleanRelPath, err := cleanRel(relPath)
	if err != nil {
		return "", "", "", err
	}
	if cleanRelPath == "" {
		return "", "", "", errors.New("workspace file path is required")
	}
	if isInternalMetadataPath(cleanRelPath) {
		return "", "", "", errors.New("direct writes to Nexus metadata are not allowed")
	}
	absTarget, err := filepath.Abs(filepath.Join(absRoot, filepath.FromSlash(cleanRelPath)))
	if err != nil {
		return "", "", "", err
	}
	if !isInside(absRoot, absTarget) {
		return "", "", "", errors.New("workspace path must stay inside the root")
	}
	if info, err := os.Lstat(absTarget); err == nil {
		if info.IsDir() {
			return "", "", "", errors.New("file write target must be a file")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", "", "", errors.New("file write target cannot be a symlink")
		}
	} else if !os.IsNotExist(err) {
		return "", "", "", err
	}
	if err := ensureWriteParentInsideRoot(absRoot, absTarget); err != nil {
		return "", "", "", err
	}
	return absRoot, absTarget, cleanRelPath, nil
}

func ensureWriteParentInsideRoot(absRoot string, absTarget string) error {
	evaluatedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return err
	}
	parent := filepath.Dir(absTarget)
	for {
		if info, err := os.Lstat(parent); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return errors.New("file write parent cannot be a symlink")
			}
			evaluatedParent, err := filepath.EvalSymlinks(parent)
			if err != nil {
				return err
			}
			if !isInside(evaluatedRoot, evaluatedParent) {
				return errors.New("workspace path must stay inside the root")
			}
			return nil
		}
		next := filepath.Dir(parent)
		if next == parent {
			return errors.New("file write parent path is invalid")
		}
		parent = next
	}
}

func isInternalMetadataPath(relPath string) bool {
	relPath = strings.Trim(filepath.ToSlash(relPath), "/")
	return relPath == ".nexusdesk" || strings.HasPrefix(relPath, ".nexusdesk/")
}
