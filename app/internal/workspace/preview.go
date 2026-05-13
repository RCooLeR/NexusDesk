package workspace

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const defaultPreviewMaxBytes = 64 * 1024
const defaultImagePreviewMaxBytes = 2 * 1024 * 1024
const defaultDocumentPreviewMaxBytes = 8 * 1024 * 1024

type PreviewOptions struct {
	MaxBytes int
}

type FilePreview struct {
	RelPath   string `json:"relPath"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FileType  string `json:"fileType"`
	Content   string `json:"content"`
	Encoding  string `json:"encoding"`
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

	if strings.EqualFold(filepath.Ext(info.Name()), ".pdf") {
		content, err := readBinaryDataURLContent(absTarget, info.Size(), binaryPreviewLimit(options.MaxBytes, defaultDocumentPreviewMaxBytes), "application/pdf")
		if err != nil {
			return FilePreview{}, err
		}
		if content == "" {
			preview.Kind = "unsupported"
			preview.Message = "PDF is too large to preview inline."
			return preview, nil
		}

		preview.Kind = "pdf"
		preview.Content = content
		preview.Message = "PDF preview rendered from the approved workspace root."
		return preview, nil
	}

	if preview.FileType == "image" {
		mimeType, ok := imageMimeType(absTarget)
		if !ok {
			preview.Kind = "unsupported"
			preview.Message = "Image type is not supported for inline preview."
			return preview, nil
		}

		content, err := readBinaryDataURLContent(absTarget, info.Size(), binaryPreviewLimit(options.MaxBytes, defaultImagePreviewMaxBytes), mimeType)
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

	normalized, encoding, ok := normalizePreviewText(content)
	if !ok || isLikelyBinary(normalized) {
		preview.Kind = "unsupported"
		preview.Message = "Binary or non-UTF-8 files are not previewed yet."
		return preview, nil
	}

	preview.Content = string(normalized)
	preview.Encoding = encoding
	preview.Truncated = truncated
	if truncated {
		preview.Message = "Preview truncated to keep the app responsive."
	} else if encoding != "utf-8" {
		preview.Message = fmt.Sprintf("Decoded as %s.", encoding)
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

func binaryPreviewLimit(maxBytes int, defaultMaxBytes int) int {
	if maxBytes <= 0 {
		return defaultMaxBytes
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

func readBinaryDataURLContent(path string, size int64, maxBytes int, mimeType string) (string, error) {
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

func normalizePreviewText(content []byte) ([]byte, string, bool) {
	if bytes.HasPrefix(content, []byte{0xef, 0xbb, 0xbf}) {
		content = bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf})
	}
	if utf8.Valid(content) {
		return content, "utf-8", true
	}

	if decoded, ok := decodeUTF16(content); ok {
		return decoded.content, decoded.encoding, true
	}

	trimmed := content
	for i := 0; i < utf8.UTFMax-1; i++ {
		if len(trimmed) == 0 {
			return nil, "", false
		}
		trimmed = trimmed[:len(trimmed)-1]
		if utf8.Valid(trimmed) {
			return trimmed, "utf-8", true
		}
	}

	return nil, "", false
}

type decodedText struct {
	content  []byte
	encoding string
}

func decodeUTF16(content []byte) (decodedText, bool) {
	var byteOrder binary.ByteOrder
	encoding := ""

	switch {
	case bytes.HasPrefix(content, []byte{0xff, 0xfe}):
		byteOrder = binary.LittleEndian
		encoding = "utf-16le"
		content = content[2:]
	case bytes.HasPrefix(content, []byte{0xfe, 0xff}):
		byteOrder = binary.BigEndian
		encoding = "utf-16be"
		content = content[2:]
	default:
		return decodedText{}, false
	}

	if len(content) < 2 {
		return decodedText{encoding: encoding}, true
	}
	if len(content)%2 != 0 {
		content = content[:len(content)-1]
	}

	values := make([]uint16, 0, len(content)/2)
	for index := 0; index < len(content); index += 2 {
		values = append(values, byteOrder.Uint16(content[index:index+2]))
	}

	return decodedText{
		content:  []byte(string(utf16.Decode(values))),
		encoding: encoding,
	}, true
}
