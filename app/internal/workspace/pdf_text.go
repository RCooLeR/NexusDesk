package workspace

import (
	"os"
	"strings"
	"unicode"
)

func extractPDFText(path string, maxBytes int) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(content) > maxBytes {
		content = content[:maxBytes]
	}

	var chunks []string
	for index := 0; index < len(content); index++ {
		if content[index] != '(' {
			continue
		}
		value, next, ok := readPDFLiteralString(content, index+1)
		if !ok {
			continue
		}
		index = next
		value = cleanExtractedPDFText(value)
		if value != "" {
			chunks = append(chunks, value)
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
