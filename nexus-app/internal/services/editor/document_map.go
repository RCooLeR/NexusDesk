package editor

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	documentMapMaxLines = 4000
	documentMapMaxItems = 120
)

var (
	documentMapMarkerPattern   = regexp.MustCompile(`(?i)\b(TODO|FIXME|HACK|BUG)\b[:\s-]*(.*)$`)
	documentMapConflictPattern = regexp.MustCompile(`^(<<<<<<<|=======|>>>>>>>)`)
)

type DocumentMapItem struct {
	Kind  string
	Label string
	Line  int
}

func BuildDocumentMap(fileName string, content string) []DocumentMapItem {
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	outlineByLine := map[int][]OutlineItem{}
	for _, item := range BuildOutline(fileName, content) {
		outlineByLine[item.Line] = append(outlineByLine[item.Line], item)
	}
	items := make([]DocumentMapItem, 0)
	for index, line := range lines {
		if index >= documentMapMaxLines || len(items) >= documentMapMaxItems {
			break
		}
		lineNumber := index + 1
		for _, outline := range outlineByLine[lineNumber] {
			items = appendDocumentMapItem(items, outline.Kind, outline.Label, lineNumber)
		}
		trimmed := strings.TrimSpace(line)
		if match := documentMapConflictPattern.FindStringSubmatch(trimmed); match != nil {
			items = appendDocumentMapItem(items, "conflict", conflictMarkerLabel(match[1]), lineNumber)
			continue
		}
		if match := documentMapMarkerPattern.FindStringSubmatch(trimmed); match != nil {
			items = appendDocumentMapItem(items, strings.ToLower(match[1]), documentMapMarkerLabel(match[1], match[2]), lineNumber)
		}
	}
	if len(items) < 8 && len(lines) >= 80 {
		items = append(items, documentMapAnchors(len(lines), items)...)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Line == items[j].Line {
			return items[i].Kind < items[j].Kind
		}
		return items[i].Line < items[j].Line
	})
	if len(items) > documentMapMaxItems {
		items = items[:documentMapMaxItems]
	}
	return items
}

func appendDocumentMapItem(items []DocumentMapItem, kind string, label string, line int) []DocumentMapItem {
	if len(items) >= documentMapMaxItems {
		return items
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return items
	}
	if len(label) > 120 {
		label = label[:120]
	}
	return append(items, DocumentMapItem{Kind: kind, Label: label, Line: line})
}

func documentMapAnchors(lineCount int, existing []DocumentMapItem) []DocumentMapItem {
	existingLines := map[int]bool{}
	for _, item := range existing {
		existingLines[item.Line] = true
	}
	anchorCount := 8
	step := lineCount / anchorCount
	if step < 20 {
		step = 20
	}
	anchors := make([]DocumentMapItem, 0, anchorCount)
	for line := step; line < lineCount && len(anchors) < anchorCount; line += step {
		if existingLines[line] {
			continue
		}
		anchors = append(anchors, DocumentMapItem{
			Kind:  "anchor",
			Label: fmt.Sprintf("Around line %d", line),
			Line:  line,
		})
	}
	if !existingLines[lineCount] && len(anchors) < anchorCount {
		anchors = append(anchors, DocumentMapItem{Kind: "anchor", Label: "End of file", Line: lineCount})
	}
	return anchors
}

func documentMapMarkerLabel(kind string, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return strings.ToUpper(kind)
	}
	return strings.ToUpper(kind) + ": " + text
}

func conflictMarkerLabel(marker string) string {
	switch marker {
	case "<<<<<<<":
		return "Merge conflict start"
	case "=======":
		return "Merge conflict separator"
	case ">>>>>>>":
		return "Merge conflict end"
	default:
		return "Merge conflict"
	}
}
