package datasets

import (
	"regexp"
	"strconv"
	"strings"
)

var logLevelPattern = regexp.MustCompile(`(?i)\b(trace|debug|info|warn|warning|error|fatal|panic)\b`)

func profileLog(relPath string, text string, mediaType string, size int64) Profile {
	columns, rows := parseLogRows(text)
	levels := map[string]int{}
	for _, row := range rows {
		if len(row) > 1 && row[1] != "" {
			levels[row[1]]++
		}
	}
	notes := []string{"Log files are profiled as line-level datasets with detected level and message columns."}
	if len(levels) > 0 {
		notes = append(notes, "Detected levels: "+levelSummary(levels))
	}
	return Profile{
		RelPath:   relPath,
		Format:    "LOG",
		MediaType: mediaType,
		Size:      size,
		Rows:      len(rows),
		Columns:   profileRows(columns, rows),
		Notes:     notes,
	}
}

func parseLogRows(text string) ([]string, [][]string) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	rows := make([][]string, 0, len(lines))
	for index, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		level := detectedLogLevel(line)
		rows = append(rows, []string{strconv.Itoa(index + 1), level, line})
	}
	return []string{"line", "level", "message"}, rows
}

func detectedLogLevel(line string) string {
	match := logLevelPattern.FindString(line)
	if match == "" {
		return ""
	}
	match = strings.ToLower(match)
	if match == "warning" {
		return "warn"
	}
	return match
}

func levelSummary(levels map[string]int) string {
	order := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	parts := []string{}
	for _, level := range order {
		if count := levels[level]; count > 0 {
			parts = append(parts, level+"="+strconv.Itoa(count))
		}
	}
	return strings.Join(parts, ", ")
}
