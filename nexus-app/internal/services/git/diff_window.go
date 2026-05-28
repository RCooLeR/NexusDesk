package git

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const diffPreviewMaxHunks = 40

func windowUnifiedDiff(output string) (string, bool) {
	if len(output) <= diffMaxBytes {
		return output, false
	}
	normalized := strings.ReplaceAll(output, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	var builder strings.Builder
	hunksKept := 0
	hunksElided := 0
	inHunk := false
	truncatedHunk := false

	for index, line := range lines {
		isLastEmpty := index == len(lines)-1 && line == ""
		if isLastEmpty {
			continue
		}
		if strings.HasPrefix(line, "@@") {
			inHunk = true
			if hunksKept >= diffPreviewMaxHunks || builder.Len()+len(line)+1 > diffMaxBytes {
				hunksElided++
				continue
			}
			hunksKept++
			writeDiffWindowLine(&builder, line)
			continue
		}
		if inHunk && hunksKept >= diffPreviewMaxHunks {
			continue
		}
		if builder.Len()+len(line)+1 > diffMaxBytes {
			if inHunk {
				truncatedHunk = true
			}
			break
		}
		writeDiffWindowLine(&builder, line)
	}

	if hunksElided > 0 {
		builder.WriteString(fmt.Sprintf("[diff preview elided: %d hunk(s) omitted]\n", hunksElided))
	} else if truncatedHunk || builder.Len() < len(normalized) {
		builder.WriteString("[diff preview elided: output exceeds preview window]\n")
	}
	text := builder.String()
	if text == "" {
		text, _ = truncateDiffUTF8(output, diffMaxBytes)
		text += "\n[diff preview elided: output exceeds preview window]\n"
	}
	return text, true
}

func writeDiffWindowLine(builder *strings.Builder, line string) {
	builder.WriteString(line)
	builder.WriteString("\n")
}

func truncateDiffUTF8(value string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value, false
	}
	cut := value[:maxBytes]
	for !utf8.ValidString(cut) && len(cut) > 0 {
		cut = cut[:len(cut)-1]
	}
	return cut, true
}
