package tools

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"nexusdesk/internal/services/agent"
)

func workspaceRoot(request agent.Request) (string, error) {
	root := strings.TrimSpace(request.WorkspaceRoot)
	if root == "" {
		return "", errors.New("workspace root is required")
	}
	return root, nil
}

func jsonListArg(call agent.ToolCall, key string) ([]string, error) {
	value := strings.TrimSpace(call.Args[key])
	if value == "" {
		return nil, nil
	}
	var items []string
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func firstArg(call agent.ToolCall, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(call.Args[key]); value != "" {
			return value
		}
	}
	return ""
}

func boolArg(call agent.ToolCall, key string) bool {
	value := strings.ToLower(strings.TrimSpace(call.Args[key]))
	return value == "1" || value == "true" || value == "yes"
}

func intArg(call agent.ToolCall, key string, fallback int) int {
	value := strings.TrimSpace(call.Args[key])
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func listArg(call agent.ToolCall, key string) []string {
	value := strings.TrimSpace(call.Args[key])
	if value == "" {
		return nil
	}
	value = strings.Trim(value, "[]")
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t'
	})
	items := []string{}
	seen := map[string]bool{}
	for _, part := range parts {
		item := strings.Trim(strings.TrimSpace(part), `"'`)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		items = append(items, item)
	}
	return items
}

func toolOK(call agent.ToolCall, risk string, observation string) agent.ToolResult {
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: risk, Observation: strings.TrimSpace(observation)}
}

func toolError(call agent.ToolCall, risk string, err error) agent.ToolResult {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: risk, Observation: message, Error: message}
}
