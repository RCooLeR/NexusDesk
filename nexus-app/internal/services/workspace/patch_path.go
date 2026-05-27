package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func readPatchTarget(root string, file parsedPatchFile) (string, bool, error) {
	_, absTarget, cleanRelPath, err := resolveWriteTarget(root, file.relPath)
	if err != nil {
		return "", false, err
	}
	relPath := filepath.ToSlash(cleanRelPath)
	content, err := os.ReadFile(absTarget)
	if os.IsNotExist(err) {
		if file.action != "create" {
			return "", false, fmt.Errorf("patch target %s does not exist", relPath)
		}
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if file.action == "create" {
		return "", false, fmt.Errorf("patch target %s already exists", relPath)
	}
	if len(content) > writeContentMaxBytes {
		return "", false, errors.New("patch target is too large")
	}
	if looksBinary(content) && !looksLikeUTF16LE(content) && !looksLikeUTF16BE(content) {
		return "", false, errors.New("patch target is not safe text")
	}
	text, _, err := decodeText(content)
	if err != nil {
		return "", false, err
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text, strings.HasSuffix(text, "\n"), nil
}

func parsePatchPath(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.IndexAny(value, "\t "); index >= 0 {
		value = value[:index]
	}
	value = strings.Trim(value, `"'`)
	if value == "/dev/null" {
		return value
	}
	value = strings.TrimPrefix(value, "a/")
	value = strings.TrimPrefix(value, "b/")
	return filepath.ToSlash(value)
}

func cleanPatchRelPath(relPath string) (string, error) {
	cleanRelPath, err := cleanRel(relPath)
	if err != nil {
		return "", err
	}
	if cleanRelPath == "" {
		return "", errors.New("unified patch target path is required")
	}
	if isInternalMetadataPath(cleanRelPath) {
		return "", errors.New("direct patches to Nexus metadata are not allowed")
	}
	return filepath.ToSlash(cleanRelPath), nil
}
