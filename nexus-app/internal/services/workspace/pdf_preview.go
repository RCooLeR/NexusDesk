package workspace

import (
	"bytes"
	"strings"
	"unicode"

	"nexusdesk/internal/domain"
)

func decodePDF(content []byte) *domain.PDFPreview {
	pages := extractPDFPages(content)
	text := joinPDFTextPages(pages)
	return &domain.PDFPreview{
		Pages:     pages,
		Text:      text,
		Truncated: false,
	}
}

func extractPDFPages(content []byte) []domain.TextPage {
	segments := splitPDFPageSegments(content)
	pages := make([]domain.TextPage, 0, len(segments))
	for pageIndex, segment := range segments {
		text := extractPDFLiteralText(segment)
		if text == "" {
			continue
		}
		pages = append(pages, domain.TextPage{Page: pageIndex + 1, Text: text})
	}
	return pages
}

func splitPDFPageSegments(content []byte) [][]byte {
	marker := []byte("/Type /Page")
	positions := []int{}
	offset := 0
	for {
		index := bytes.Index(content[offset:], marker)
		if index < 0 {
			break
		}
		index += offset
		if index+len(marker) >= len(content) || content[index+len(marker)] != 's' {
			positions = append(positions, index)
		}
		offset = index + len(marker)
	}
	if len(positions) == 0 {
		return [][]byte{content}
	}
	segments := make([][]byte, 0, len(positions))
	for index, start := range positions {
		end := len(content)
		if index+1 < len(positions) {
			end = positions[index+1]
		}
		segments = append(segments, content[start:end])
	}
	return segments
}

func extractPDFLiteralText(content []byte) string {
	chunks := []string{}
	for index := 0; index < len(content); index++ {
		if content[index] != '(' {
			continue
		}
		value, next, ok := readPDFLiteralString(content, index+1)
		if !ok {
			continue
		}
		index = next
		if text := cleanExtractedPDFText(value); text != "" {
			chunks = append(chunks, text)
		}
	}
	return strings.TrimSpace(strings.Join(chunks, " "))
}

func readPDFLiteralString(content []byte, start int) (string, int, bool) {
	var builder strings.Builder
	depth := 1
	for index := start; index < len(content); index++ {
		value := content[index]
		if value == '\\' && index+1 < len(content) {
			index++
			builder.WriteByte(unescapePDFByte(content[index]))
			continue
		}
		if value == '(' {
			depth++
		}
		if value == ')' {
			depth--
			if depth == 0 {
				return builder.String(), index, true
			}
		}
		builder.WriteByte(value)
	}
	return "", start, false
}

func unescapePDFByte(value byte) byte {
	switch value {
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case 'b':
		return '\b'
	case 'f':
		return '\f'
	default:
		return value
	}
}

func cleanExtractedPDFText(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return ""
	}
	printable := 0
	total := 0
	for _, char := range value {
		total++
		if char == '\n' || char == '\r' || char == '\t' || unicode.IsPrint(char) {
			printable++
		}
	}
	if total == 0 || printable*100/total < 85 {
		return ""
	}
	return strings.Join(strings.Fields(value), " ")
}

func joinPDFTextPages(pages []domain.TextPage) string {
	chunks := make([]string, 0, len(pages))
	for _, page := range pages {
		if page.Text != "" {
			chunks = append(chunks, page.Text)
		}
	}
	return strings.TrimSpace(strings.Join(chunks, "\n\n"))
}
