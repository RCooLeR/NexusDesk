package documents

import (
	"errors"
	"html"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"nexusdesk/internal/domain"
)

const extractedDocumentMaxChars = 60000

type Service struct {
	previewer Previewer
}

func New(previewer Previewer) *Service {
	return &Service{previewer: previewer}
}

func (s *Service) Extract(root string, relPath string) (ExtractedDocument, error) {
	if s == nil || s.previewer == nil {
		return ExtractedDocument{}, errors.New("document previewer is required")
	}
	preview, err := s.previewer.PreviewFile(root, relPath)
	if err != nil {
		return ExtractedDocument{}, err
	}
	format := documentFormat(preview.RelPath)
	if !supportedDocumentFormat(format) {
		return ExtractedDocument{}, errors.New("document extraction currently supports Markdown, TXT, HTML, XML, DOCX, XLSX, and PDF files")
	}
	if !previewKindMatchesFormat(format, preview.Kind) {
		return ExtractedDocument{}, errors.New("document extraction requires a previewable text, DOCX, or PDF document")
	}
	text := extractReadableText(format, preview.Text)
	if text == "" {
		return ExtractedDocument{}, errors.New("document preview did not contain extractable text")
	}
	pages := previewPageCount(preview)
	truncated := previewTruncated(preview)
	if len(text) > extractedDocumentMaxChars {
		text = truncateDocumentText(text, extractedDocumentMaxChars)
		truncated = true
	}
	return ExtractedDocument{
		RelPath:   preview.RelPath,
		Title:     documentTitle(format, preview.Text, text, preview.RelPath),
		Format:    format,
		MediaType: preview.MediaType,
		Encoding:  preview.Encoding,
		Text:      text,
		Size:      preview.Size,
		Lines:     countDocumentLines(text),
		Words:     len(strings.Fields(text)),
		Pages:     pages,
		Truncated: truncated,
	}, nil
}

func documentFormat(relPath string) string {
	switch strings.ToLower(filepath.Ext(relPath)) {
	case ".md", ".markdown":
		return "markdown"
	case ".txt", ".text":
		return "txt"
	case ".html", ".htm":
		return "html"
	case ".xml":
		return "xml"
	case ".docx":
		return "docx"
	case ".xlsx":
		return "xlsx"
	case ".pdf":
		return "pdf"
	default:
		return ""
	}
}

func supportedDocumentFormat(format string) bool {
	switch format {
	case "markdown", "txt", "html", "xml", "docx", "xlsx", "pdf":
		return true
	default:
		return false
	}
}

func previewKindMatchesFormat(format string, kind domain.PreviewKind) bool {
	switch format {
	case "docx":
		return kind == domain.PreviewDoc
	case "pdf":
		return kind == domain.PreviewPDF
	case "xlsx":
		return kind == domain.PreviewTable
	default:
		return kind == domain.PreviewText
	}
}

func previewTruncated(preview domain.FilePreview) bool {
	if preview.Document != nil && preview.Document.Truncated {
		return true
	}
	if preview.PDF != nil && preview.PDF.Truncated {
		return true
	}
	if preview.Table != nil && preview.Table.Truncated {
		return true
	}
	return false
}

func previewPageCount(preview domain.FilePreview) int {
	if preview.PDF == nil {
		return 0
	}
	return len(preview.PDF.Pages)
}

func extractReadableText(format string, text string) string {
	switch format {
	case "html":
		return normalizeExtractedText(stripHTML(text))
	case "xml":
		return normalizeExtractedText(stripXML(text))
	default:
		return normalizeExtractedText(text)
	}
}

func stripHTML(value string) string {
	value = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`(?i)</(p|div|section|article|header|footer|li|h[1-6]|tr)>`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(value, " ")
	return html.UnescapeString(value)
}

func stripXML(value string) string {
	value = regexp.MustCompile(`(?is)<\?xml[^>]*>`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?is)<!--.*?-->`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(value, " ")
	return html.UnescapeString(value)
}

func normalizeExtractedText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = regexp.MustCompile(`[ \t]+`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`\n[ \t]+`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`\n{3,}`).ReplaceAllString(value, "\n\n")
	return strings.TrimSpace(value)
}

func documentTitle(format string, original string, extracted string, relPath string) string {
	if format == "markdown" {
		for _, line := range strings.Split(original, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") {
				return compactTitle(strings.TrimSpace(strings.TrimLeft(line, "#")))
			}
		}
	}
	if format == "html" {
		matches := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`).FindStringSubmatch(original)
		if len(matches) == 2 {
			return compactTitle(html.UnescapeString(stripHTML(matches[1])))
		}
	}
	for _, line := range strings.Split(extracted, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return compactTitle(line)
		}
	}
	return filepath.Base(relPath)
}

func compactTitle(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 120 {
		return value[:117] + "..."
	}
	return value
}

func truncateDocumentText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	truncated := value[:limit]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return strings.TrimSpace(truncated) + "\n[document extraction truncated]"
}

func countDocumentLines(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}
