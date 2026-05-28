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
	var formatted []byte
	var err error
	switch extension {
	case ".go":
		formatted, err = goformat.Source([]byte(content))
		if err != nil {
			return FormatResult{}, fmt.Errorf("go format failed: %w", err)
		}
	case ".json":
		var buffer bytes.Buffer
		if err := json.Indent(&buffer, []byte(content), "", "  "); err != nil {
			return FormatResult{}, fmt.Errorf("json format failed: %w", err)
		}
		formatted = []byte(strings.TrimRight(buffer.String(), "\n") + "\n")
	default:
		return FormatResult{}, fmt.Errorf("formatting is not available for %s files yet", strings.TrimPrefix(extension, "."))
	}
	next := string(formatted)
	if next == content {
		return FormatResult{Content: content, Changed: false, Message: "Document is already formatted."}, nil
	}
	return FormatResult{Content: next, Changed: true, Message: "Formatted document draft."}, nil
}
