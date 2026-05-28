package editor

import (
	"bytes"
	"encoding/json"
	"fmt"
	goformat "go/format"
	"path/filepath"
	"strings"
)

type FormatResult struct {
	Content string
	Changed bool
	Message string
}

func FormatDocument(fileName string, content string) (FormatResult, error) {
	extension := strings.ToLower(filepath.Ext(fileName))
	lowerName := strings.ToLower(filepath.Base(fileName))
	var next string
	var err error
	switch extension {
	case ".go":
		var formatted []byte
		formatted, err = goformat.Source([]byte(content))
		if err != nil {
			return FormatResult{}, fmt.Errorf("go format failed: %w", err)
		}
		next = string(formatted)
	case ".json", ".code-workspace":
		var buffer bytes.Buffer
		if err := json.Indent(&buffer, []byte(content), "", "  "); err != nil {
			return FormatResult{}, fmt.Errorf("json format failed: %w", err)
		}
		next = strings.TrimRight(buffer.String(), "\n") + "\n"
	case ".md", ".markdown", ".mdx":
		next = formatWhitespaceDocument(content, true)
	case ".yaml", ".yml", ".sql", ".env", ".txt", ".log", ".csv", ".tsv":
		next = formatWhitespaceDocument(content, false)
	default:
		if lowerName == "dockerfile" || strings.HasPrefix(lowerName, "dockerfile.") {
			next = formatWhitespaceDocument(content, false)
			break
		}
		return FormatResult{}, fmt.Errorf("formatting is not available for %s files yet", strings.TrimPrefix(extension, "."))
	}
	if next == content {
		return FormatResult{Content: content, Changed: false, Message: "Document is already formatted."}, nil
	}
	return FormatResult{Content: next, Changed: true, Message: "Formatted document draft."}, nil
}

func formatWhitespaceDocument(content string, preserveMarkdownHardBreaks bool) string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n")
	trimmedDocument := strings.TrimRight(normalized, "\n")
	if trimmedDocument == "" {
		return ""
	}
	lines := strings.Split(trimmedDocument, "\n")
	for index, line := range lines {
		lines[index] = trimFormattingLineRight(line, preserveMarkdownHardBreaks)
	}
	return strings.Join(lines, "\n") + "\n"
}

func trimFormattingLineRight(line string, preserveMarkdownHardBreaks bool) string {
	if !preserveMarkdownHardBreaks {
		return strings.TrimRight(line, " \t")
	}
	withoutTabs := strings.TrimRight(line, "\t")
	spaceCount := len(withoutTabs) - len(strings.TrimRight(withoutTabs, " "))
	trimmed := strings.TrimRight(withoutTabs, " ")
	if spaceCount >= 2 && trimmed != "" {
		return trimmed + "  "
	}
	return trimmed
}
