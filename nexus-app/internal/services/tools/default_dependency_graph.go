package tools

import (
	"context"
	"fmt"
	"strings"

	"nexusdesk/internal/services/agent"
	workspacesvc "nexusdesk/internal/services/workspace"
)

func (h defaultHandlers) readDependencyGraph(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	graph, err := h.deps.Workspace.DependencyGraph(root, workspacesvc.DependencyGraphOptions{
		RelPath:  firstArg(call, "relPath", "path"),
		MaxFiles: intArg(call, "maxFiles", 80),
		MaxEdges: intArg(call, "maxEdges", 160),
	})
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", formatDependencyGraphObservation(graph)), nil
}

func formatDependencyGraphObservation(graph workspacesvc.DependencyGraph) string {
	lines := []string{
		"Native dependency graph.",
		"Scope: " + graph.RootRelPath,
		graph.Message,
	}
	if len(graph.Edges) == 0 {
		lines = append(lines, "No supported code dependency edges found.")
		return strings.Join(lines, "\n")
	}
	for index, edge := range graph.Edges {
		if index >= 120 {
			lines = append(lines, "[dependency edges truncated]")
			break
		}
		resolution := "external"
		if edge.Resolved {
			resolution = "resolved"
		}
		lines = append(lines, fmt.Sprintf("- %s:%d -> %s [%s/%s] %s", edge.From, edge.Line, edge.To, edge.Kind, resolution, edge.Spec))
	}
	return strings.Join(lines, "\n")
}
