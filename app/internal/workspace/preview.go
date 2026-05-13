package workspace

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const defaultPreviewMaxBytes = 64 * 1024
const defaultImagePreviewMaxBytes = 2 * 1024 * 1024

type PreviewOptions struct {
	MaxBytes int
}

type FilePreview struct {
	RelPath   string `json:"relPath"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FileType  string `json:"fileType"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
	Message   string `json:"message"`
	Size      int64  `json:"size"`
}

func Preview(root string, relPath string, options PreviewOptions) (FilePreview, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return FilePreview{}, err
	}

	cleanRel, err := cleanPreviewRelPath(relPath)
	if err != nil {
		return FilePreview{}, err
	}

	target := filepath.Join(absRoot, cleanRel)
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return FilePreview{}, err
	}

	if err := ensureInsideRoot(absRoot, absTarget); err != nil {
		return FilePreview{}, err
	}

	info, err := os.Lstat(absTarget)
	if err != nil {
		return FilePreview{}, err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return FilePreview{}, errors.New("workspace preview cannot follow symlinks")
	}

	evalRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return FilePreview{}, err
	}
	evalTarget, err := filepath.EvalSymlinks(absTarget)
	if err != nil {
		return FilePreview{}, err
	}
	if err := ensureInsideRoot(evalRoot, evalTarget); err != nil {
		return FilePreview{}, err
	}

	preview := FilePreview{
		RelPath:  filepath.ToSlash(cleanRel),
		Name:     info.Name(),
		Kind:     "file",
		FileType: detectFileTypeName(info.Name(), info.IsDir()),
		Size:     info.Size(),
	}

	if info.IsDir() {
		preview.Kind = "directory"
		preview.Message = "Select a file inside this folder to preview its contents."
		return preview, nil
	}

	if preview.FileType == "image" {
		content, err := readImagePreviewContent(absTarget, info.Size(), imagePreviewLimit(options.MaxBytes))
		if err != nil {
			return FilePreview{}, err
		}
		if content == "" {
			preview.Kind = "unsupported"
			preview.Message = "Image is too large to preview inline."
			return preview, nil
		}

		preview.Kind = "image"
		preview.Content = content
		preview.Message = "Image preview rendered from the approved workspace root."
		return preview, nil
	}

	content, truncated, err := readPreviewContent(absTarget, previewLimit(options.MaxBytes))
	if err != nil {
		return FilePreview{}, err
	}

	normalized, ok := normalizePreviewText(content)
	if !ok || isLikelyBinary(normalized) {
		preview.Kind = "unsupported"
		preview.Message = "Binary or non-UTF-8 files are not previewed yet."
		return preview, nil
	}

	preview.Content = string(normalized)
	preview.Truncated = truncated
	if truncated {
		preview.Message = "Preview truncated to keep the app responsive."
	}

	return preview, nil
}

func cleanPreviewRelPath(relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", errors.New("workspace preview path is required")
	}

	cleanRel := filepath.Clean(filepath.FromSlash(relPath))
	if cleanRel == "." || filepath.IsAbs(cleanRel) {
		return "", errors.New("workspace preview path must be relative")
	}

	parts := strings.Split(cleanRel, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return "", errors.New("workspace preview path must stay inside the workspace")
		}
	}

	return cleanRel, nil
}

func ensureInsideRoot(root string, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}

	if rel == "." {
		return nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return errors.New("workspace preview path must stay inside the workspace")
	}

	return nil
}

func previewLimit(maxBytes int) int {
	if maxBytes <= 0 {
		return defaultPreviewMaxBytes
	}
	return maxBytes
}

func imagePreviewLimit(maxBytes int) int {
	if maxBytes <= 0 {
		return defaultImagePreviewMaxBytes
	}
	return maxBytes
}

func readPreviewContent(path string, maxBytes int) ([]byte, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err != nil {
		return nil, false, err
	}

	if len(content) <= maxBytes {
		return content, false, nil
	}

	return content[:maxBytes], true, nil
}

func readImagePreviewContent(path string, size int64, maxBytes int) (string, error) {
	if size > int64(maxBytes) {
		return "", nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(content) > maxBytes {
		return "", nil
	}

	mimeType, ok := imageMimeType(path)
	if !ok {
		return "", nil
	}

	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(content)), nil
}

func imageMimeType(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png", true
	case ".jpg", ".jpeg":
		return "image/jpeg", true
	case ".gif":
		return "image/gif", true
	case ".webp":
		return "image/webp", true
	case ".svg":
		return "image/svg+xml", true
	case ".ico":
		return "image/x-icon", true
	default:
		return "", false
	}
}

func isLikelyBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	for _, value := range content {
		if value == 0 {
			return true
		}
	}

	return false
}

func normalizePreviewText(content []byte) ([]byte, bool) {
	if utf8.Valid(content) {
		return content, true
	}

	trimmed := content
	for i := 0; i < utf8.UTFMax-1; i++ {
		if len(trimmed) == 0 {
			return nil, false
		}
		trimmed = trimmed[:len(trimmed)-1]
		if utf8.Valid(trimmed) {
			return trimmed, true
		}
	}

	return nil, false
}
