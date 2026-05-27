package workspace

import (
	"errors"
	"fmt"
	"strings"
)

func applyPatchFile(root string, file parsedPatchFile) (string, string, error) {
	before, trailingNewline, err := readPatchTarget(root, file)
	if err != nil {
		return "", "", err
	}
	lines := splitPatchContent(before)
	for _, hunk := range file.hunks {
		oldLines, newLines := patchHunkLines(hunk)
		index, err := findPatchHunkIndex(lines, oldLines, hunk.oldStart)
		if err != nil {
			return "", "", fmt.Errorf("%s: %w", file.relPath, err)
		}
		replacement := make([]string, 0, len(lines)-len(oldLines)+len(newLines))
		replacement = append(replacement, lines[:index]...)
		replacement = append(replacement, newLines...)
		replacement = append(replacement, lines[index+len(oldLines):]...)
		lines = replacement
	}
	if file.action == "create" {
		trailingNewline = true
	}
	return before, joinPatchContent(lines, trailingNewline), nil
}

func patchHunkLines(hunk parsedPatchHunk) ([]string, []string) {
	oldLines := []string{}
	newLines := []string{}
	for _, line := range hunk.lines {
		if line.kind == ' ' || line.kind == '-' {
			oldLines = append(oldLines, line.text)
		}
		if line.kind == ' ' || line.kind == '+' {
			newLines = append(newLines, line.text)
		}
	}
	return oldLines, newLines
}

func findPatchHunkIndex(lines []string, oldLines []string, oldStart int) (int, error) {
	if len(oldLines) == 0 {
		index := oldStart
		if index < 0 {
			index = 0
		}
		if index > len(lines) {
			index = len(lines)
		}
		return index, nil
	}

	expected := oldStart - 1
	if expected < 0 {
		expected = 0
	}
	if hasLineSequenceAt(lines, oldLines, expected) {
		return expected, nil
	}

	matches := []int{}
	for index := 0; index+len(oldLines) <= len(lines); index++ {
		if hasLineSequenceAt(lines, oldLines, index) {
			matches = append(matches, index)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 {
		return 0, errors.New("patch hunk did not match current file content")
	}
	return 0, errors.New("patch hunk matched multiple locations; add more context")
}

func hasLineSequenceAt(lines []string, needle []string, index int) bool {
	if index < 0 || index+len(needle) > len(lines) {
		return false
	}
	for offset, line := range needle {
		if lines[index+offset] != line {
			return false
		}
	}
	return true
}

func splitPatchContent(content string) []string {
	if content == "" {
		return []string{}
	}
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return []string{}
	}
	return strings.Split(content, "\n")
}

func joinPatchContent(lines []string, trailingNewline bool) string {
	if len(lines) == 0 {
		if trailingNewline {
			return "\n"
		}
		return ""
	}
	content := strings.Join(lines, "\n")
	if trailingNewline {
		content += "\n"
	}
	return content
}
