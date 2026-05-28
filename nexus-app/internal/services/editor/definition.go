package editor

import (
	"strings"
	"unicode"
)

type DefinitionResult struct {
	Query string
	Item  OutlineItem
}

func ResolveDefinition(fileName string, content string, cursorRow int, cursorColumn int) (DefinitionResult, bool) {
	query := identifierAtCursor(content, cursorRow, cursorColumn)
	if query == "" {
		return DefinitionResult{}, false
	}
	queryKeys := definitionLookupKeys(query)
	for _, item := range BuildOutline(fileName, content) {
		if item.Line == cursorRow+1 {
			continue
		}
		if definitionKeysMatch(queryKeys, definitionLookupKeys(item.Label)) {
			return DefinitionResult{Query: query, Item: item}, true
		}
	}
	return DefinitionResult{Query: query}, false
}

func identifierAtCursor(content string, cursorRow int, cursorColumn int) string {
	if cursorRow < 0 {
		cursorRow = 0
	}
	if cursorColumn < 0 {
		cursorColumn = 0
	}
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n"), "\n")
	if cursorRow >= len(lines) {
		return ""
	}
	line := []rune(lines[cursorRow])
	if len(line) == 0 {
		return ""
	}
	if cursorColumn >= len(line) {
		cursorColumn = len(line) - 1
	}
	if !isDefinitionIdentifierRune(line[cursorColumn]) && cursorColumn > 0 && isDefinitionIdentifierRune(line[cursorColumn-1]) {
		cursorColumn--
	}
	if !isDefinitionIdentifierRune(line[cursorColumn]) {
		return ""
	}
	start := cursorColumn
	for start > 0 && isDefinitionIdentifierRune(line[start-1]) {
		start--
	}
	end := cursorColumn + 1
	for end < len(line) && isDefinitionIdentifierRune(line[end]) {
		end++
	}
	return strings.Trim(string(line[start:end]), `"'`)
}

func isDefinitionIdentifierRune(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsDigit(value) || value == '_' || value == '$' || value == '-' || value == '.' || value == '#'
}

func definitionLookupKeys(value string) map[string]bool {
	keys := map[string]bool{}
	add := func(candidate string) {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		candidate = strings.Trim(candidate, ".#")
		if candidate != "" {
			keys[candidate] = true
		}
	}
	add(value)
	for _, separator := range []string{".", "#"} {
		if index := strings.LastIndex(value, separator); index >= 0 && index+1 < len(value) {
			add(value[index+1:])
		}
	}
	return keys
}

func definitionKeysMatch(left map[string]bool, right map[string]bool) bool {
	for key := range left {
		if right[key] {
			return true
		}
	}
	return false
}
