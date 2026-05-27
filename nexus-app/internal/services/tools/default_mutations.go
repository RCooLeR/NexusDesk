package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"nexusdesk/internal/services/agent"
	workspacesvc "nexusdesk/internal/services/workspace"
)

func (h defaultHandlers) writeFile(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	if !request.ApproveWrites {
		err := errors.New("approval is required before writing workspace files")
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
	content, ok := call.Args["content"]
	if !ok {
		err := errors.New("content is required")
		return toolError(call, "high", err), err
	}
	proposal, err := h.deps.Workspace.ApplyFileWrite(root, workspacesvc.FileWriteRequest{
		RelPath:  relPath,
		Content:  content,
		Encoding: call.Args["encoding"],
	})
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: fileWriteObservation(proposal), Mutated: true}, nil
}

func (h defaultHandlers) appendFile(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	if !request.ApproveWrites {
		err := errors.New("approval is required before appending workspace files")
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
	content, ok := call.Args["content"]
	if !ok {
		err := errors.New("content is required")
		return toolError(call, "high", err), err
	}
	proposal, err := h.deps.Workspace.ApplyFileAppend(root, workspacesvc.FileWriteRequest{
		RelPath:  relPath,
		Content:  content,
		Encoding: call.Args["encoding"],
	})
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: fileWriteObservation(proposal), Mutated: true}, nil
}

func (h defaultHandlers) copyFile(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	if result, err := requireWriteApproval(call, request, "copying workspace files"); err != nil {
		return result, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	source := firstArg(call, "sourceRelPath", "source", "from", "relPath")
	target := firstArg(call, "targetRelPath", "target", "to")
	if source == "" || target == "" {
		err := errors.New("sourceRelPath and targetRelPath are required")
		return toolError(call, "high", err), err
	}
	proposal, err := h.deps.Workspace.ApplyFileCopy(root, workspacesvc.FileCopyRequest{SourceRelPath: source, TargetRelPath: target})
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: fileOperationObservation(proposal), Mutated: true}, nil
}

func (h defaultHandlers) moveFile(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	if result, err := requireWriteApproval(call, request, "moving workspace files"); err != nil {
		return result, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	source := firstArg(call, "sourceRelPath", "source", "from", "relPath")
	target := firstArg(call, "targetRelPath", "target", "to")
	if source == "" || target == "" {
		err := errors.New("sourceRelPath and targetRelPath are required")
		return toolError(call, "high", err), err
	}
	proposal, err := h.deps.Workspace.ApplyFileMove(root, workspacesvc.FileMoveRequest{SourceRelPath: source, TargetRelPath: target})
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: fileOperationObservation(proposal), Mutated: true}, nil
}

func (h defaultHandlers) deleteFile(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	if result, err := requireWriteApproval(call, request, "deleting workspace files"); err != nil {
		return result, err
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
	proposal, err := h.deps.Workspace.ApplyFileDelete(root, relPath)
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: fileOperationObservation(proposal), Mutated: true}, nil
}

func (h defaultHandlers) applyPatch(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	if result, err := requireWriteApproval(call, request, "applying workspace patches"); err != nil {
		return result, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	patch := firstArg(call, "patch", "unifiedDiff", "diff")
	if patch == "" {
		err := errors.New("patch is required")
		return toolError(call, "high", err), err
	}
	proposal, err := h.deps.Workspace.ApplyUnifiedPatch(root, workspacesvc.UnifiedPatchRequest{Patch: patch})
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: unifiedPatchObservation(proposal), Mutated: true}, nil
}

func (h defaultHandlers) listRollbacks(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	records, err := h.deps.Workspace.ListRollbacks(root)
	if err != nil {
		return toolError(call, "low", err), err
	}
	lines := []string{fmt.Sprintf("%d rollback snapshot(s).", len(records))}
	for _, record := range records {
		lines = append(lines, fmt.Sprintf("- %s [%s] %s target=%s paths=%d created=%s", record.ID, record.Status, record.Action, record.Target, len(record.Entries), record.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	return toolOK(call, "low", strings.Join(lines, "\n")), nil
}

func (h defaultHandlers) rollbackFileMutation(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	if result, err := requireWriteApproval(call, request, "applying a rollback"); err != nil {
		return result, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	id := firstArg(call, "id", "rollbackId")
	if id == "" {
		err := errors.New("rollback id is required")
		return toolError(call, "high", err), err
	}
	result, err := h.deps.Workspace.ApplyRollback(root, id)
	if err != nil {
		return toolError(call, "high", err), err
	}
	lines := []string{
		result.Message,
		"Restored: " + strings.Join(result.Restored, ", "),
		"Removed: " + strings.Join(result.Removed, ", "),
	}
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: strings.Join(lines, "\n"), Mutated: true}, nil
}

func requireWriteApproval(call agent.ToolCall, request agent.Request, action string) (agent.ToolResult, error) {
	if request.ApproveWrites {
		return agent.ToolResult{}, nil
	}
	err := fmt.Errorf("approval is required before %s", action)
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: err.Error(), Error: err.Error()}, err
}

func fileWriteObservation(proposal workspacesvc.FileWriteProposal) string {
	lines := []string{
		proposal.Message,
		"Path: " + proposal.RelPath,
		"Action: " + proposal.Action,
		"Encoding: " + proposal.Encoding,
		"Rollback: " + proposal.RollbackID,
	}
	if proposal.Diff != "" {
		lines = append(lines, "Diff:\n"+proposal.Diff)
	}
	return strings.Join(lines, "\n")
}

func fileOperationObservation(proposal workspacesvc.FileOperationProposal) string {
	lines := []string{
		proposal.Message,
		"Action: " + proposal.Action,
		"Rollback: " + proposal.RollbackID,
	}
	if proposal.SourceRelPath != "" {
		lines = append(lines, "Source: "+proposal.SourceRelPath)
	}
	if proposal.TargetRelPath != "" {
		lines = append(lines, "Target: "+proposal.TargetRelPath)
	}
	if proposal.Size > 0 {
		lines = append(lines, fmt.Sprintf("Size: %d", proposal.Size))
	}
	return strings.Join(lines, "\n")
}

func unifiedPatchObservation(proposal workspacesvc.UnifiedPatchProposal) string {
	lines := []string{
		proposal.Message,
		fmt.Sprintf("Files: %d", proposal.FileCount),
		"Rollback: " + proposal.RollbackID,
	}
	for _, file := range proposal.Files {
		lines = append(lines, fmt.Sprintf("- %s %s", file.Action, file.RelPath))
		if file.Diff != "" {
			lines = append(lines, "Diff:\n"+file.Diff)
		}
	}
	return strings.Join(lines, "\n")
}
