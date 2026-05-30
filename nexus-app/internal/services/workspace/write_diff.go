package workspace

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	writeDiffMaxTotalLines = 5000
	writeDiffMaxCells      = 2_000_000
	writeDiffContextLines  = 3
	writeDiffChangedLimit  = 120
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
	beforeLines := splitDiffLines(before)
	afterLines := splitDiffLines(after)
	for _, line := range diffLines(beforeLines, afterLines) {
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

func diffLines(beforeLines []string, afterLines []string) []string {
	if useBoundedDiff(beforeLines, afterLines) {
		return boundedHunkDiffLines(beforeLines, afterLines)
	}
	return lcsDiffLines(beforeLines, afterLines)
}

func useBoundedDiff(beforeLines []string, afterLines []string) bool {
	if len(beforeLines)+len(afterLines) > writeDiffMaxTotalLines {
		return true
	}
	return len(beforeLines) > 0 && len(afterLines) > 0 && len(beforeLines) > writeDiffMaxCells/len(afterLines)
}

func boundedHunkDiffLines(beforeLines []string, afterLines []string) []string {
	prefix := commonPrefixLines(beforeLines, afterLines)
	suffix := commonSuffixLines(beforeLines, afterLines, prefix)
	beforeChangeEnd := len(beforeLines) - suffix
	afterChangeEnd := len(afterLines) - suffix
	startContext := maxDiffInt(0, prefix-writeDiffContextLines)
	endContext := minDiffInt(len(beforeLines), beforeChangeEnd+writeDiffContextLines)
	lines := []string{
		fmt.Sprintf("@@ bounded diff: large input, %d removed line(s), %d added line(s) @@", beforeChangeEnd-prefix, afterChangeEnd-prefix),
	}
	for index := startContext; index < prefix; index++ {
		lines = append(lines, " "+beforeLines[index])
	}
	lines = appendChangedDiffLines(lines, "-", beforeLines[prefix:beforeChangeEnd])
	lines = appendChangedDiffLines(lines, "+", afterLines[prefix:afterChangeEnd])
	for index := beforeChangeEnd; index < endContext; index++ {
		lines = append(lines, " "+beforeLines[index])
	}
	return lines
}

func appendChangedDiffLines(lines []string, prefix string, changed []string) []string {
	limit := len(changed)
	if limit > writeDiffChangedLimit {
		limit = writeDiffChangedLimit
	}
	for index := 0; index < limit; index++ {
		lines = append(lines, prefix+changed[index])
	}
	if len(changed) > limit {
		lines = append(lines, prefix+fmt.Sprintf("... %d more line(s) omitted by bounded diff", len(changed)-limit))
	}
	return lines
}

func commonPrefixLines(beforeLines []string, afterLines []string) int {
	limit := minDiffInt(len(beforeLines), len(afterLines))
	index := 0
	for index < limit && beforeLines[index] == afterLines[index] {
		index++
	}
	return index
}

func commonSuffixLines(beforeLines []string, afterLines []string, prefix int) int {
	limit := minDiffInt(len(beforeLines)-prefix, len(afterLines)-prefix)
	count := 0
	for count < limit && beforeLines[len(beforeLines)-1-count] == afterLines[len(afterLines)-1-count] {
		count++
	}
	return count
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

func minDiffInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxDiffInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
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
