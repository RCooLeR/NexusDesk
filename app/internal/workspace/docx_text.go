package workspace

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"strings"
)

func extractDOCXText(path string, maxBytes int) (string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != "word/document.xml" {
			continue
		}
		content, err := readZipTextFile(file, maxBytes)
		if err != nil {
			return "", err
		}
		return parseDOCXDocumentText(content), nil
	}

	return "", nil
}

func readZipTextFile(file *zip.File, maxBytes int) ([]byte, error) {
	handle, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	if maxBytes <= 0 {
		maxBytes = defaultPreviewMaxBytes
	}
	return io.ReadAll(io.LimitReader(handle, int64(maxBytes)+1))
}

func parseDOCXDocumentText(content []byte) string {
	decoder := xml.NewDecoder(strings.NewReader(string(content)))
	var builder strings.Builder
	inText := false

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "t":
				inText = true
			case "tab":
				builder.WriteString("\t")
			case "br", "p":
				writeDOCXSpace(&builder, "\n")
			}
		case xml.EndElement:
			switch value.Name.Local {
			case "t":
				inText = false
			case "p":
				writeDOCXSpace(&builder, "\n")
			}
		case xml.CharData:
			if inText {
				builder.WriteString(string(value))
			}
		}
	}

	return strings.TrimSpace(strings.Join(strings.Fields(builder.String()), " "))
}

func writeDOCXSpace(builder *strings.Builder, value string) {
	if builder.Len() == 0 {
		return
	}
	builder.WriteString(value)
}
