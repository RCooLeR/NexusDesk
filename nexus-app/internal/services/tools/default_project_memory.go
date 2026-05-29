package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"nexusdesk/internal/services/agent"
	workspacesvc "nexusdesk/internal/services/workspace"
)

func (h defaultHandlers) updateProjectMemory(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	if !request.ApproveWrites {
		err := errors.New("approval is required before updating project memory")
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "medium", Observation: err.Error(), Error: err.Error()}, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "medium", err), err
	}
	key := firstArg(call, "key", "name")
	content := firstArg(call, "content", "value", "fact")
	if content != "" {
		content = redactSensitiveText(content)
	}
	sources, err := projectMemorySourceArgs(call)
	if err != nil {
		return toolError(call, "medium", err), err
	}
	result, err := h.deps.Workspace.UpdateProjectMemory(root, workspacesvc.ProjectMemoryUpdateRequest{
		Key:            key,
		Content:        content,
		SourceRelPaths: sources,
	})
	if err != nil {
		return toolError(call, "medium", err), err
	}
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "medium", Observation: formatProjectMemoryObservation(result), Mutated: true}, nil
}

func projectMemorySourceArgs(call agent.ToolCall) ([]string, error) {
	if value := strings.TrimSpace(firstArg(call, "sourceRelPaths", "sourcesJson")); value != "" && strings.HasPrefix(value, "[") {
		var sources []string
		if err := json.Unmarshal([]byte(value), &sources); err != nil {
			return nil, err
		}
		return sources, nil
	}
	sources := listArg(call, "sourceRelPaths")
	if len(sources) == 0 {
		sources = listArg(call, "sources")
	}
	if len(sources) == 0 {
		source := firstArg(call, "sourceRelPath", "relPath", "path")
		if source != "" {
			sources = []string{source}
		}
	}
	return sources, nil
}

func formatProjectMemoryObservation(result workspacesvc.ProjectMemoryUpdateResult) string {
	action := "Updated"
	if result.Created {
		action = "Created"
	}
	lines := []string{
		action + " project memory.",
		"Key: " + result.Record.Key,
		fmt.Sprintf("Total records: %d", result.Count),
		fmt.Sprintf("Sources: %d", len(result.Record.SourceRelPaths)),
	}
	if result.Record.SourceSHA256 != "" {
		lines = append(lines, "Source fingerprint: "+result.Record.SourceSHA256)
	}
	if !result.Record.UpdatedAt.IsZero() {
		lines = append(lines, "Updated: "+result.Record.UpdatedAt.UTC().Format("2006-01-02 15:04:05Z"))
	}
	lines = append(lines, "", result.Record.Content)
	return strings.Join(lines, "\n")
}
