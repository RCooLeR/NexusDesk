package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"nexusdesk/internal/services/agent"
	editorSvc "nexusdesk/internal/services/editor"
	workspacesvc "nexusdesk/internal/services/workspace"
)

func (h defaultHandlers) formatFile(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	if !request.ApproveWrites {
		err := errors.New("approval is required before formatting workspace files")
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: err.Error(), Error: err.Error()}, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	relPath := firstArg(call, "relPath", "path")
	if relPath == "" {
		err := errors.New("relPath is required")
		return toolError(call, "high", err), err
	}
	formatter := strings.ToLower(strings.TrimSpace(firstArg(call, "formatter")))
	if formatter != "" && formatter != "native" {
		err := fmt.Errorf("unsupported formatter %q; only native formatting is available", formatter)
		return toolError(call, "high", err), err
	}
	read, err := h.deps.Workspace.ReadTextFile(root, relPath)
	if err != nil {
		return toolError(call, "high", err), err
	}
	result, err := editorSvc.FormatDocument(read.RelPath, read.Content)
	if err != nil {
		return toolError(call, "high", err), err
	}
	if !result.Changed {
		return agent.ToolResult{
			Name:        call.Name,
			Args:        call.Args,
			Risk:        "high",
			Observation: fmt.Sprintf("Document is already formatted.\nPath: %s\nFormatter: native\nEncoding: %s\nSize: %d", read.RelPath, read.Encoding, read.Size),
			Mutated:     false,
		}, nil
	}
	proposal, err := h.deps.Workspace.ApplyFileWrite(root, workspacesvc.FileWriteRequest{RelPath: read.RelPath, Content: result.Content, Encoding: read.Encoding})
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{
		Name:        call.Name,
		Args:        call.Args,
		Risk:        "high",
		Observation: formatFileObservation(read, proposal),
		Mutated:     true,
	}, nil
}

func formatFileObservation(read workspacesvc.TextFileRead, proposal workspacesvc.FileWriteProposal) string {
	lines := []string{
		"Formatted workspace file.",
		"Path: " + read.RelPath,
		"Formatter: native",
		"Encoding: " + proposal.Encoding,
		fmt.Sprintf("Size: %d", proposal.Size),
		"Rollback: " + proposal.RollbackID,
	}
	if proposal.Diff != "" {
		lines = append(lines, "Diff:\n"+proposal.Diff)
	}
	return strings.Join(lines, "\n")
}
