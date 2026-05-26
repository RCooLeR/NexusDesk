package git

import (
	"errors"
	"path/filepath"
	"strings"
)

func cleanRelPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	value = filepath.ToSlash(value)
	value = strings.TrimPrefix(value, "/")
	if value == "" || value == "." {
		return "", errors.New("Git path is required")
	}
	if strings.HasPrefix(value, "-") {
		return "", errors.New("Git path cannot start with a dash")
	}
	if filepath.IsAbs(value) || value == ".." || strings.HasPrefix(value, "../") || strings.Contains(value, "/../") || strings.HasSuffix(value, "/..") {
		return "", errors.New("Git path must stay inside the repository")
	}
	return value, nil
}
