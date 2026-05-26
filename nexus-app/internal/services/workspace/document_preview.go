package workspace

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"nexusdesk/internal/domain"
)

const documentPreviewMaxChars = 20000

func decodeDocument(content []byte, relPath string) (*domain.DocumentPreview, error) {
	if strings.EqualFold(filepath.Ext(relPath), ".docx") {
		return decodeDOCX(content)
	}
	return nil, errors.New("unsupported document preview format")
}

func decodeDOCX(content []byte) (*domain.DocumentPreview, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, err
	}
	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			return extractDOCXText(file)
		}
	}
	return nil, errors.New("docx document body was not found")
}

func extractDOCXText(file *zip.File) (*domain.DocumentPreview, error) {
	body, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer body.Close()

	decoder := xml.NewDecoder(body)
	var builder strings.Builder
	truncated := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return &domain.DocumentPreview{Text: previewDocumentText(builder.String()), Truncated: truncated}, nil
		}
		if err != nil {
			return nil, err
		}
		if truncated {
			continue
		}
		if err := appendDOCXTokenText(&builder, token); err != nil {
			return nil, err
		}
		if builder.Len() > documentPreviewMaxChars {
			truncated = true
		}
	}
}

func previewDocumentText(text string) string {
	if len(text) > documentPreviewMaxChars {
		text = text[:documentPreviewMaxChars]
	}
	return strings.TrimSpace(text)
}

func appendDOCXTokenText(builder *strings.Builder, token xml.Token) error {
	switch value := token.(type) {
	case xml.StartElement:
		switch value.Name.Local {
		case "tab":
			_, err := builder.WriteString("\t")
			return err
		case "br":
			return builder.WriteByte('\n')
		}
	case xml.EndElement:
		if value.Name.Local == "p" {
			return builder.WriteByte('\n')
		}
	case xml.CharData:
		_, err := builder.Write([]byte(value))
		return err
	}
	return nil
}
