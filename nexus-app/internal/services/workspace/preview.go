package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"nexusdesk/internal/domain"
)

const structuredPreviewMaxBytes = 32 * 1024 * 1024

func (s *Service) PreviewFile(root string, relPath string) (domain.FilePreview, error) {
	target, cleanRelPath, info, err := resolveFile(root, relPath)
	if err != nil {
		return domain.FilePreview{}, err
	}

	sampleLimit := int64(4096)
	if s.previewByteLimit > 0 && s.previewByteLimit < sampleLimit {
		sampleLimit = s.previewByteLimit
	}
	sample, err := readFilePrefix(target, sampleLimit)
	if err != nil {
		return domain.FilePreview{}, err
	}
	isOversized := s.previewByteLimit > 0 && info.Size() > s.previewByteLimit

	content := sample
	kind := previewKind(cleanRelPath, content)
	if kind == domain.PreviewBinary && isOversized {
		return domain.FilePreview{}, errors.New("file is too large for inline preview")
	}
	if isStructuredPreviewKind(kind) {
		if info.Size() > structuredPreviewMaxBytes {
			return domain.FilePreview{}, fmt.Errorf("file is too large for structured preview: %d bytes exceeds %d byte cap", info.Size(), structuredPreviewMaxBytes)
		}
		content, err = os.ReadFile(target)
		if err != nil {
			return domain.FilePreview{}, err
		}
	}
	if kind == domain.PreviewText && isOversized {
		content, err = readFilePrefix(target, s.previewByteLimit)
		if err != nil {
			return domain.FilePreview{}, err
		}
	}
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
	if kind == domain.PreviewPDF {
		pdf := decodePDF(content)
		preview.Bytes = content
		preview.Text = pdf.Text
		preview.TextBytes = int64(len([]byte(pdf.Text)))
		preview.Truncated = pdf.Truncated
		preview.PDF = pdf
		return preview, nil
	}
	if kind == domain.PreviewTable {
		text, encoding, table, err := decodeTable(content, cleanRelPath)
		if err != nil {
			return domain.FilePreview{}, err
		}
		preview.Text = text
		preview.TextBytes = int64(len(content))
		if table != nil {
			preview.Truncated = table.Truncated
		}
		preview.Encoding = encoding
		preview.Table = table
		return preview, nil
	}
	if kind == domain.PreviewDoc {
		document, err := decodeDocument(content, cleanRelPath)
		if err != nil {
			return domain.FilePreview{}, err
		}
		preview.Text = document.Text
		preview.TextBytes = int64(len([]byte(document.Text)))
		preview.Truncated = document.Truncated
		preview.Document = document
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
	preview.TextBytes = int64(len(content))
	preview.Truncated = isOversized
	preview.Encoding = encoding
	return preview, nil
}

func isStructuredPreviewKind(kind domain.PreviewKind) bool {
	switch kind {
	case domain.PreviewImage, domain.PreviewPDF, domain.PreviewTable, domain.PreviewDoc:
		return true
	default:
		return false
	}
}

func readFilePrefix(path string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return os.ReadFile(path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return nil, err
	}
	return content, nil
}

func previewKind(relPath string, content []byte) domain.PreviewKind {
	extension := strings.ToLower(filepath.Ext(relPath))
	if isImageExtension(extension) {
		return domain.PreviewImage
	}
	if isPDFExtension(extension) {
		return domain.PreviewPDF
	}
	if isTableExtension(extension) {
		return domain.PreviewTable
	}
	if isDocumentExtension(extension) {
		return domain.PreviewDoc
	}
	if isTextLikePath(relPath) {
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

func isPDFExtension(extension string) bool {
	return extension == ".pdf"
}

func isDocumentExtension(extension string) bool {
	return extension == ".docx"
}

func isTableExtension(extension string) bool {
	switch extension {
	case ".csv", ".tsv", ".xlsx":
		return true
	default:
		return false
	}
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
