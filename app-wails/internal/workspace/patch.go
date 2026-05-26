package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const unifiedPatchMaxBytes = 2 * 1024 * 1024

type UnifiedPatchRequest struct {
	Patch string `json:"patch"`
}

type UnifiedPatchFileResult struct {
	RelPath string `json:"relPath"`
	Action  string `json:"action"`
	Diff    string `json:"diff"`
	Message string `json:"message"`
}

type UnifiedPatchProposal struct {
	Files     []UnifiedPatchFileResult `json:"files"`
	FileCount int                      `json:"fileCount"`
	Message   string                   `json:"message"`
}

type parsedPatchFile struct {
	relPath string
	action  string
	hunks   []parsedPatchHunk
}

type parsedPatchHunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []parsedPatchLine
}

type parsedPatchLine struct {
	kind byte
	text string
}

var unifiedHunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

func PreviewUnifiedPatch(root string, request UnifiedPatchRequest) (UnifiedPatchProposal, error) {
	files, err := parseUnifiedPatch(request.Patch)
	if err != nil {
		return UnifiedPatchProposal{}, err
	}

	results := make([]UnifiedPatchFileResult, 0, len(files))
	for _, file := range files {
		content, err := applyPatchFile(root, file)
		if err != nil {
			return UnifiedPatchProposal{}, err
		}
		proposal, err := PreviewFileWrite(root, FileWriteRequest{RelPath: file.relPath, Content: content})
		if err != nil {
			return UnifiedPatchProposal{}, err
		}
		results = append(results, UnifiedPatchFileResult{
			RelPath: proposal.RelPath,
			Action:  proposal.Action,
			Diff:    proposal.Diff,
			Message: proposal.Message,
		})
	}

	return UnifiedPatchProposal{
		Files:     results,
		FileCount: len(results),
		Message:   fmt.Sprintf("Preview ready to apply unified patch to %d file(s).", len(results)),
	}, nil
}

func ApplyUnifiedPatch(root string, request UnifiedPatchRequest) (UnifiedPatchProposal, error) {
	proposal, err := PreviewUnifiedPatch(root, request)
	if err != nil {
		return UnifiedPatchProposal{}, err
	}

	files, err := parseUnifiedPatch(request.Patch)
	if err != nil {
		return UnifiedPatchProposal{}, err
	}
	for _, file := range files {
		content, err := applyPatchFile(root, file)
		if err != nil {
			return UnifiedPatchProposal{}, err
		}
		if _, err := ApplyFileWrite(root, FileWriteRequest{RelPath: file.relPath, Content: content}); err != nil {
			return UnifiedPatchProposal{}, err
		}
	}

	proposal.Message = fmt.Sprintf("Applied unified patch to %d file(s).", proposal.FileCount)
	for index := range proposal.Files {
		proposal.Files[index].Message = fmt.Sprintf("Patch applied for %s.", proposal.Files[index].RelPath)
	}
	return proposal, nil
}

func parseUnifiedPatch(patch string) ([]parsedPatchFile, error) {
	patch = strings.ReplaceAll(patch, "\r\n", "\n")
	patch = strings.ReplaceAll(patch, "\r", "\n")
	if strings.TrimSpace(patch) == "" {
		return nil, errors.New("unified patch content is required")
	}
	if len(patch) > unifiedPatchMaxBytes {
		return nil, errors.New("unified patch is too large")
	}

	lines := strings.Split(patch, "\n")
	files := []parsedPatchFile{}
	seenFiles := map[string]bool{}
	for index := 0; index < len(lines); {
		line := lines[index]
		if !strings.HasPrefix(line, "--- ") {
			index++
			continue
		}
		if index+1 >= len(lines) || !strings.HasPrefix(lines[index+1], "+++ ") {
			return nil, errors.New("unified patch file header must include old and new paths")
		}

		oldPath := parsePatchPath(line[4:])
		newPath := parsePatchPath(lines[index+1][4:])
		if newPath == "/dev/null" {
			return nil, errors.New("unified patch deletes are not supported; use delete_file for explicit deletes")
		}
		relPath := newPath
		action := "update"
		if oldPath == "/dev/null" {
			action = "create"
		}
		cleanRelPath, err := cleanPatchRelPath(relPath)
		if err != nil {
			return nil, err
		}
		relPath = cleanRelPath

		index += 2
		file := parsedPatchFile{relPath: relPath, action: action}
		for index < len(lines) {
			if strings.HasPrefix(lines[index], "--- ") {
				break
			}
			if !strings.HasPrefix(lines[index], "@@ ") {
				index++
				continue
			}
			hunk, next, err := parsePatchHunk(lines, index)
			if err != nil {
				return nil, err
			}
			file.hunks = append(file.hunks, hunk)
			index = next
		}
		if len(file.hunks) == 0 {
			return nil, fmt.Errorf("unified patch for %s has no hunks", relPath)
		}
		if seenFiles[relPath] {
			return nil, fmt.Errorf("unified patch contains duplicate file section for %s", relPath)
		}
		seenFiles[relPath] = true
		files = append(files, file)
	}
	if len(files) == 0 {
		return nil, errors.New("no unified patch file sections found")
	}
	return files, nil
}

func parsePatchHunk(lines []string, start int) (parsedPatchHunk, int, error) {
	match := unifiedHunkHeaderPattern.FindStringSubmatch(lines[start])
	if match == nil {
		return parsedPatchHunk{}, start, fmt.Errorf("invalid unified patch hunk header: %s", lines[start])
	}
	hunk := parsedPatchHunk{
		oldStart: parsePatchCount(match[1], 0),
		oldCount: parsePatchCount(match[2], 1),
		newStart: parsePatchCount(match[3], 0),
		newCount: parsePatchCount(match[4], 1),
	}
	index := start + 1
	oldSeen := 0
	newSeen := 0
	for index < len(lines) {
		line := lines[index]
		if oldSeen == hunk.oldCount && newSeen == hunk.newCount {
			break
		}
		if strings.HasPrefix(line, `\ No newline at end of file`) || (line == "" && index == len(lines)-1) {
			index++
			continue
		}
		if line == "" {
			return parsedPatchHunk{}, start, errors.New("unified patch hunk line must start with space, +, or -")
		}
		kind := line[0]
		if kind != ' ' && kind != '+' && kind != '-' {
			return parsedPatchHunk{}, start, errors.New("unified patch hunk line must start with space, +, or -")
		}
		hunk.lines = append(hunk.lines, parsedPatchLine{kind: kind, text: line[1:]})
		if kind == ' ' || kind == '-' {
			oldSeen++
		}
		if kind == ' ' || kind == '+' {
			newSeen++
		}
		index++
	}
	if len(hunk.lines) == 0 {
		return parsedPatchHunk{}, start, errors.New("unified patch hunk is empty")
	}
	if err := validatePatchHunkCounts(hunk); err != nil {
		return parsedPatchHunk{}, start, err
	}
	return hunk, index, nil
}

func validatePatchHunkCounts(hunk parsedPatchHunk) error {
	oldCount := 0
	newCount := 0
	for _, line := range hunk.lines {
		if line.kind == ' ' || line.kind == '-' {
			oldCount++
		}
		if line.kind == ' ' || line.kind == '+' {
			newCount++
		}
	}
	if oldCount != hunk.oldCount || newCount != hunk.newCount {
		return fmt.Errorf("unified patch hunk counts do not match header: expected -%d +%d, got -%d +%d", hunk.oldCount, hunk.newCount, oldCount, newCount)
	}
	return nil
}

func applyPatchFile(root string, file parsedPatchFile) (string, error) {
	before, trailingNewline, err := readPatchTarget(root, file)
	if err != nil {
		return "", err
	}
	lines := splitPatchContent(before)
	for _, hunk := range file.hunks {
		oldLines, newLines := patchHunkLines(hunk)
		index, err := findPatchHunkIndex(lines, oldLines, hunk.oldStart)
		if err != nil {
			return "", fmt.Errorf("%s: %w", file.relPath, err)
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
	return joinPatchContent(lines, trailingNewline), nil
}

func readPatchTarget(root string, file parsedPatchFile) (string, bool, error) {
	_, absTarget, cleanRel, err := resolveWriteTarget(root, file.relPath)
	if err != nil {
		return "", false, err
	}
	relPath := filepath.ToSlash(cleanRel)
	content, err := os.ReadFile(absTarget)
	if os.IsNotExist(err) {
		if file.action != "create" {
			return "", false, fmt.Errorf("patch target %s does not exist", relPath)
		}
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if file.action == "create" {
		return "", false, fmt.Errorf("patch target %s already exists", relPath)
	}
	if len(content) > writeContentMaxBytes {
		return "", false, errors.New("patch target is too large")
	}
	normalized, _, ok := normalizePreviewText(content)
	if !ok || isLikelyBinary(normalized) {
		return "", false, errors.New("patch target is not safe text")
	}
	text := strings.ReplaceAll(string(normalized), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text, strings.HasSuffix(text, "\n"), nil
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

func parsePatchCount(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parsePatchPath(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.IndexAny(value, "\t "); index >= 0 {
		value = value[:index]
	}
	value = strings.Trim(value, `"'`)
	if value == "/dev/null" {
		return value
	}
	value = strings.TrimPrefix(value, "a/")
	value = strings.TrimPrefix(value, "b/")
	return filepath.ToSlash(value)
}

func cleanPatchRelPath(relPath string) (string, error) {
	cleanRel, err := cleanPreviewRelPath(relPath)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(filepath.ToSlash(cleanRel), ".nexusdesk/") {
		return "", errors.New("direct patches to Nexus metadata are not allowed")
	}
	return filepath.ToSlash(cleanRel), nil
}
