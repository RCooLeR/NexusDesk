package workspace

import (
	"bytes"
	"errors"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"nexusdesk/internal/domain"
)

var textExtensions = map[string]struct{}{
	".css":        {},
	".go":         {},
	".html":       {},
	".js":         {},
	".json":       {},
	".jsx":        {},
	".log":        {},
	".md":         {},
	".ps1":        {},
	".py":         {},
	".rs":         {},
	".sql":        {},
	".toml":       {},
	".ts":         {},
	".tsx":        {},
	".txt":        {},
	".xml":        {},
	".yaml":       {},
	".yml":        {},
	".dockerfile": {},
}

func (s *Service) PreviewFile(root string, relPath string) (domain.FilePreview, error) {
	target, cleanRelPath, info, err := resolveFile(root, relPath)
	if err != nil {
		return domain.FilePreview{}, err
	}
	if info.Size() > s.previewByteLimit {
		return domain.FilePreview{}, errors.New("file is too large for inline preview")
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return domain.FilePreview{}, err
	}
	kind := previewKind(cleanRelPath, content)
	preview := domain.FilePreview{
		RelPath:   cleanRelPath,
		Name:      filepath.Base(cleanRelPath),
		Size:      info.Size(),
		Kind:      kind,
		MediaType: mediaType(cleanRelPath),
	}
	if kind == domain.PreviewImage {
		preview.Bytes = content
		return preview, nil
	}
	if kind != domain.PreviewText {
		return preview, nil
	}
	text, encoding, err := decodeText(content)
	if err != nil {
		return domain.FilePreview{}, err
	}
	preview.Text = text
	preview.Encoding = encoding
	return preview, nil
}

func previewKind(relPath string, content []byte) domain.PreviewKind {
	extension := strings.ToLower(filepath.Ext(relPath))
	if isImageExtension(extension) {
		return domain.PreviewImage
	}
	if _, ok := textExtensions[extension]; ok {
		if looksBinary(content) && !looksLikeUTF16LE(content) && !looksLikeUTF16BE(content) {
			return domain.PreviewBinary
		}
		return domain.PreviewText
	}
	if looksLikeUTF16LE(content) || looksLikeUTF16BE(content) {
		return domain.PreviewText
	}
	if looksBinary(content) {
		return domain.PreviewBinary
	}
	if utf8.Valid(content) {
		return domain.PreviewText
	}
	return domain.PreviewBinary
}

func isImageExtension(extension string) bool {
	switch extension {
	case ".bmp", ".gif", ".jpeg", ".jpg", ".png", ".svg", ".webp":
		return true
	default:
		return false
	}
}

func mediaType(relPath string) string {
	if detected := mime.TypeByExtension(filepath.Ext(relPath)); detected != "" {
		return detected
	}
	return "application/octet-stream"
}

func looksBinary(content []byte) bool {
	sample := content
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	return bytes.IndexByte(sample, 0) >= 0
}
