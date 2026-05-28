package editor

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	outlineMaxLines = 4000
	outlineMaxItems = 120
)

var (
	markdownHeadingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	jsSymbolPattern        = regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?(?:function|class|interface|type|enum)\s+([A-Za-z_$][\w$]*)`)
	jsArrowPattern         = regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>`)
	goFuncPattern          = regexp.MustCompile(`^func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\(`)
	goTypePattern          = regexp.MustCompile(`^type\s+([A-Za-z_]\w*)\s+(?:struct|interface|func|map|\[\])`)
	cssSelectorPattern     = regexp.MustCompile(`^([.#]?[A-Za-z_][^{]+)\s*\{`)
	jsonKeyPattern         = regexp.MustCompile(`^"([^"]+)"\s*:`)
	yamlKeyPattern         = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:`)
)

type OutlineItem struct {
	ID    string
	Kind  string
	Label string
	Level int
	Line  int
}

func BuildOutline(fileName string, content string) []OutlineItem {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	items := make([]OutlineItem, 0)
	add := func(kind string, label string, line int, level int) {
		cleanLabel := strings.TrimSpace(label)
		if cleanLabel == "" {
			return
		}
		if len(cleanLabel) > 120 {
			cleanLabel = cleanLabel[:120]
		}
		items = append(items, OutlineItem{
			ID:    outlineItemID(line, kind, cleanLabel),
			Kind:  kind,
			Label: cleanLabel,
			Level: level,
			Line:  line,
		})
	}
	for index, line := range lines {
		if index >= outlineMaxLines || len(items) >= outlineMaxItems {
			break
		}
		lineNumber := index + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if match := markdownHeadingPattern.FindStringSubmatch(trimmed); match != nil {
			add("heading", match[2], lineNumber, len(match[1])-1)
			continue
		}
		switch {
		case isJavaScriptOutlineExtension(extension):
			if match := jsSymbolPattern.FindStringSubmatch(trimmed); match != nil {
				add(jsSymbolKind(trimmed), match[1], lineNumber, 0)
				continue
			}
			if match := jsArrowPattern.FindStringSubmatch(trimmed); match != nil {
				add("func", match[1], lineNumber, 0)
			}
		case extension == "go":
			if match := goFuncPattern.FindStringSubmatch(trimmed); match != nil {
				add("func", match[1], lineNumber, 0)
				continue
			}
			if match := goTypePattern.FindStringSubmatch(trimmed); match != nil {
				add("type", match[1], lineNumber, 0)
			}
		case extension == "css" || extension == "scss" || extension == "sass":
			if match := cssSelectorPattern.FindStringSubmatch(trimmed); match != nil {
				add("selector", match[1], lineNumber, 0)
			}
		case extension == "json" || extension == "jsonc":
			if match := jsonKeyPattern.FindStringSubmatch(trimmed); match != nil && leadingSpaceCount(line) <= 8 {
				add("key", match[1], lineNumber, leadingSpaceCount(line)/2)
			}
		case extension == "yaml" || extension == "yml":
			if match := yamlKeyPattern.FindStringSubmatch(trimmed); match != nil && leadingSpaceCount(line) <= 8 {
				add("key", match[1], lineNumber, leadingSpaceCount(line)/2)
			}
		}
	}
	return items
}

func outlineItemID(line int, kind string, label string) string {
	return strings.Join([]string{strconv.Itoa(line), kind, label}, "-")
}

func isJavaScriptOutlineExtension(extension string) bool {
	switch extension {
	case "ts", "tsx", "js", "jsx", "mjs", "cjs":
		return true
	default:
		return false
	}
}

func jsSymbolKind(line string) string {
	switch {
	case strings.Contains(line, " class ") || strings.HasPrefix(line, "class ") || strings.HasPrefix(line, "export class "):
		return "class"
	case strings.Contains(line, " interface ") || strings.HasPrefix(line, "interface ") || strings.HasPrefix(line, "export interface "):
		return "interface"
	case strings.Contains(line, " type ") || strings.HasPrefix(line, "type ") || strings.HasPrefix(line, "export type "):
		return "type"
	case strings.Contains(line, " enum ") || strings.HasPrefix(line, "enum ") || strings.HasPrefix(line, "export enum "):
		return "enum"
	default:
		return "func"
	}
}

func leadingSpaceCount(value string) int {
	return len(value) - len(strings.TrimLeft(value, " \t"))
}
