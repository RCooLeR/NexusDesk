package workspace

import (
	"regexp"
	"strings"
)

type WorkspaceSymbol struct {
	RelPath string `json:"relPath"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Line    int    `json:"line"`
}

var (
	jsSymbolPattern        = regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?(?:function|class|interface|type|enum)\s+([A-Za-z_$][\w$]*)`)
	jsArrowSymbolPattern   = regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>`)
	goFuncSymbolPattern    = regexp.MustCompile(`^func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\(`)
	goTypeSymbolPattern    = regexp.MustCompile(`^type\s+([A-Za-z_]\w*)\s+(?:struct|interface|func|map|\[\])`)
	cssSelectorPattern     = regexp.MustCompile(`^([.#]?[A-Za-z_][^{]+)\s*\{`)
	jsonKeySymbolPattern   = regexp.MustCompile(`^"([^"]+)"\s*:`)
	yamlKeySymbolPattern   = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:`)
	markdownHeadingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
)

func searchFileSymbols(root string, relPath string, matcher searchMatcher) []SearchResult {
	preview, err := Preview(root, relPath, PreviewOptions{MaxBytes: searchPreviewMaxBytes})
	if err != nil || strings.TrimSpace(preview.Content) == "" {
		return nil
	}

	symbols := BuildSymbols(preview.RelPath, preview.Name, preview.Content)
	results := []SearchResult{}
	for _, symbol := range symbols {
		if !matcher.matches(symbol.Name) {
			continue
		}
		results = append(results, SearchResult{
			RelPath:   preview.RelPath,
			Name:      preview.Name,
			Kind:      preview.Kind,
			FileType:  preview.FileType,
			MatchType: matcher.matchType("symbol"),
			Line:      symbol.Line,
			Snippet:   symbol.Kind + " " + symbol.Name,
		})
		if len(results) >= 3 {
			return results
		}
	}
	return results
}

func BuildSymbols(relPath string, name string, content string) []WorkspaceSymbol {
	extension := lowerExtension(name)
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	symbols := []WorkspaceSymbol{}
	add := func(kind string, label string, line int) {
		label = strings.TrimSpace(label)
		if label == "" {
			return
		}
		symbols = append(symbols, WorkspaceSymbol{
			RelPath: relPath,
			Name:    trimSymbolLabel(label),
			Kind:    kind,
			Line:    line,
		})
	}

	for index, line := range lines {
		if index >= 4000 || len(symbols) >= 120 {
			break
		}
		lineNumber := index + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if match := markdownHeadingPattern.FindStringSubmatch(trimmed); match != nil {
			add("heading", match[2], lineNumber)
			continue
		}

		switch extension {
		case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
			if match := jsSymbolPattern.FindStringSubmatch(trimmed); match != nil {
				add(jsSymbolKind(trimmed), match[1], lineNumber)
			} else if match := jsArrowSymbolPattern.FindStringSubmatch(trimmed); match != nil {
				add("func", match[1], lineNumber)
			}
		case ".go":
			if match := goFuncSymbolPattern.FindStringSubmatch(trimmed); match != nil {
				add("func", match[1], lineNumber)
			} else if match := goTypeSymbolPattern.FindStringSubmatch(trimmed); match != nil {
				add("type", match[1], lineNumber)
			}
		case ".css", ".scss", ".sass":
			if match := cssSelectorPattern.FindStringSubmatch(trimmed); match != nil {
				add("selector", match[1], lineNumber)
			}
		case ".json", ".jsonc":
			if match := jsonKeySymbolPattern.FindStringSubmatch(trimmed); match != nil && leadingWhitespace(line) <= 8 {
				add("key", match[1], lineNumber)
			}
		case ".yaml", ".yml":
			if match := yamlKeySymbolPattern.FindStringSubmatch(trimmed); match != nil && leadingWhitespace(line) <= 8 {
				add("key", match[1], lineNumber)
			}
		}
	}
	return symbols
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

func lowerExtension(name string) string {
	index := strings.LastIndex(name, ".")
	if index < 0 {
		return ""
	}
	return strings.ToLower(name[index:])
}

func leadingWhitespace(value string) int {
	return len(value) - len(strings.TrimLeft(value, " \t"))
}

func trimSymbolLabel(value string) string {
	if len(value) <= 120 {
		return value
	}
	return value[:117] + "..."
}
