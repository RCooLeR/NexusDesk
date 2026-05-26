package workspace

import (
	"errors"
	"os"
	"strings"
)

func readExistingWriteTarget(absTarget string) (string, string, error) {
	content, err := os.ReadFile(absTarget)
	if os.IsNotExist(err) {
		return "", "create", nil
	}
	if err != nil {
		return "", "", err
	}
	if len(content) > writeDiffMaxBytes {
		return "[existing file omitted: larger than inline diff limit]", "update", nil
	}
	if looksBinary(content) && !looksLikeUTF16LE(content) && !looksLikeUTF16BE(content) {
		return "", "", errors.New("existing file is not safe text")
	}
	text, _, err := decodeText(content)
	if err != nil {
		return "", "", err
	}
	return text, "update", nil
}

func buildUnifiedDiff(relPath string, before string, after string) string {
	var builder strings.Builder
	builder.WriteString("--- a/")
	builder.WriteString(relPath)
	builder.WriteString("\n+++ b/")
	builder.WriteString(relPath)
	builder.WriteString("\n")
	for _, line := range lcsDiffLines(splitDiffLines(before), splitDiffLines(after)) {
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return builder.String()
}

func buildAppendDiff(relPath string, appended string) string {
	var builder strings.Builder
	builder.WriteString("--- a/")
	builder.WriteString(relPath)
	builder.WriteString("\n+++ b/")
	builder.WriteString(relPath)
	builder.WriteString("\n@@ append @@\n")
	for _, line := range splitDiffLines(appended) {
		builder.WriteString("+")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	if appended == "" {
		builder.WriteString("+\n")
	}
	return builder.String()
}

func lcsDiffLines(beforeLines []string, afterLines []string) []string {
	table := make([][]int, len(beforeLines)+1)
	for index := range table {
		table[index] = make([]int, len(afterLines)+1)
	}
	for left := len(beforeLines) - 1; left >= 0; left-- {
		for right := len(afterLines) - 1; right >= 0; right-- {
			if beforeLines[left] == afterLines[right] {
				table[left][right] = table[left+1][right+1] + 1
			} else if table[left+1][right] >= table[left][right+1] {
				table[left][right] = table[left+1][right]
			} else {
				table[left][right] = table[left][right+1]
			}
		}
	}

	diff := []string{}
	left := 0
	right := 0
	for left < len(beforeLines) && right < len(afterLines) {
		if beforeLines[left] == afterLines[right] {
			diff = append(diff, " "+beforeLines[left])
			left++
			right++
			continue
		}
		if table[left+1][right] >= table[left][right+1] {
			diff = append(diff, "-"+beforeLines[left])
			left++
		} else {
			diff = append(diff, "+"+afterLines[right])
			right++
		}
	}
	for left < len(beforeLines) {
		diff = append(diff, "-"+beforeLines[left])
		left++
	}
	for right < len(afterLines) {
		diff = append(diff, "+"+afterLines[right])
		right++
	}
	return diff
}

func splitDiffLines(content string) []string {
	if content == "" {
		return []string{}
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return []string{}
	}
	return strings.Split(content, "\n")
}
