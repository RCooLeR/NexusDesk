package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

func parseAction(message string) (ToolCall, bool) {
	re := regexp.MustCompile(`(?is)Action:\s*([a-zA-Z0-9_.-]+)\s*\((.*)\)\s*$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(message))
	if len(matches) != 3 {
		return ToolCall{}, false
	}

	args := map[string]string{}
	rawArgs := strings.TrimSpace(matches[2])
	if strings.HasPrefix(rawArgs, "{") {
		decoded := map[string]any{}
		if err := json.Unmarshal([]byte(rawArgs), &decoded); err == nil {
			for key, value := range decoded {
				args[key] = stringifyActionArgument(value)
			}
			return ToolCall{Name: strings.TrimSpace(matches[1]), Args: args}, true
		}
	}

	for _, pair := range splitArgs(rawArgs) {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		args[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return ToolCall{Name: strings.TrimSpace(matches[1]), Args: args}, true
}

func stringifyActionArgument(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	case map[string]any, []any:
		encoded, err := json.Marshal(typed)
		if err == nil {
			return string(encoded)
		}
	}
	return fmt.Sprint(value)
}

func splitArgs(raw string) []string {
	parts := []string{}
	var current strings.Builder
	inQuote := rune(0)
	escaped := false
	for _, r := range raw {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && inQuote != 0 {
			current.WriteRune(r)
			escaped = true
			continue
		}
		if inQuote != 0 {
			current.WriteRune(r)
			if r == inQuote {
				inQuote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			inQuote = r
			current.WriteRune(r)
			continue
		}
		if r == ',' {
			if value := strings.TrimSpace(current.String()); value != "" {
				parts = append(parts, value)
			}
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	if value := strings.TrimSpace(current.String()); value != "" {
		parts = append(parts, value)
	}
	return parts
}

func parseFinalAnswer(message string) string {
	re := regexp.MustCompile(`(?is)Final Answer:\s*(.*)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(message))
	if len(matches) != 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func parsePlanUpdate(message string) ([]PlanStep, bool) {
	re := regexp.MustCompile(`(?is)Action:\s*update_plan\s*\((.*)\)\s*$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(message))
	if len(matches) != 2 {
		return nil, false
	}
	payload := struct {
		Steps []PlanStep `json:"steps"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(matches[1])), &payload); err != nil {
		return nil, false
	}
	if len(payload.Steps) == 0 {
		return nil, false
	}
	return normalizePlan(payload.Steps), true
}
