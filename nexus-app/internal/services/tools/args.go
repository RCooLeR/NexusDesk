package tools

import (
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
