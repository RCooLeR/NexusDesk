package git

import (
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderPattern = regexp.MustCompile(`^@@ -([0-9]+)(?:,([0-9]+))? \+([0-9]+)(?:,([0-9]+))? @@`)

func parseDiffHunks(kind DiffKind, diff string) []DiffHunk {
	hunks := []DiffHunk{}
	active := -1
	for _, line := range strings.Split(strings.ReplaceAll(diff, "\r\n", "\n"), "\n") {
		hunk, ok := parseHunkHeader(kind, len(hunks), line)
		if ok {
			hunks = append(hunks, hunk)
			active = len(hunks) - 1
			continue
		}
		if active < 0 {
			continue
		}
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			continue
		case strings.HasPrefix(line, "+"):
			hunks[active].AddedLines++
		case strings.HasPrefix(line, "-"):
			hunks[active].DeletedLines++
		}
	}
	return hunks
}

func parseHunkHeader(kind DiffKind, index int, line string) (DiffHunk, bool) {
	matches := hunkHeaderPattern.FindStringSubmatch(strings.TrimSpace(line))
	if matches == nil {
		return DiffHunk{}, false
	}
	return DiffHunk{
		Kind:     kind,
		Index:    index,
		Header:   strings.TrimSpace(line),
		OldStart: mustAtoi(matches[1]),
		OldLines: hunkLineCount(matches[2]),
		NewStart: mustAtoi(matches[3]),
		NewLines: hunkLineCount(matches[4]),
	}, true
}

func hunkLineCount(value string) int {
	if value == "" {
		return 1
	}
	return mustAtoi(value)
}

func mustAtoi(value string) int {
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return number
}
