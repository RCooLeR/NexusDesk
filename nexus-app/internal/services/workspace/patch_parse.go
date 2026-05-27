package workspace

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

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
		file, err := parsePatchFileHeader(line[4:], lines[index+1][4:])
		if err != nil {
			return nil, err
		}
		index += 2
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
			return nil, fmt.Errorf("unified patch for %s has no hunks", file.relPath)
		}
		if seenFiles[file.relPath] {
			return nil, fmt.Errorf("unified patch contains duplicate file section for %s", file.relPath)
		}
		seenFiles[file.relPath] = true
		files = append(files, file)
	}
	if len(files) == 0 {
		return nil, errors.New("no unified patch file sections found")
	}
	return files, nil
}

func parsePatchFileHeader(oldPath string, newPath string) (parsedPatchFile, error) {
	oldPath = parsePatchPath(oldPath)
	newPath = parsePatchPath(newPath)
	if newPath == "/dev/null" {
		return parsedPatchFile{}, errors.New("unified patch deletes are not supported; use delete_file for explicit deletes")
	}
	action := "update"
	if oldPath == "/dev/null" {
		action = "create"
	}
	cleanRelPath, err := cleanPatchRelPath(newPath)
	if err != nil {
		return parsedPatchFile{}, err
	}
	return parsedPatchFile{relPath: cleanRelPath, action: action}, nil
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
